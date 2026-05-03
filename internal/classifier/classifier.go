package classifier

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jbrukh/bayesian"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

const modelFileName = "classifier.model"

type ClassifierManager struct {
	app        *pocketbase.PocketBase
	modelsDir  string
	classifier *bayesian.Classifier
	mu         sync.RWMutex
}

func NewClassifierManager(app *pocketbase.PocketBase, modelsDir string) *ClassifierManager {
	return &ClassifierManager{
		app:       app,
		modelsDir: modelsDir,
	}
}

func (m *ClassifierManager) Load() error {
	path := filepath.Join(m.modelsDir, modelFileName)
	c, err := LoadModel(path)
	if err != nil {
		log.Printf("classifier: no model found at %s (%v), will be created on first retrain", path, err)
		return err
	}
	m.mu.Lock()
	m.classifier = c
	m.mu.Unlock()
	log.Printf("classifier: loaded model with %d classes", len(c.Classes))
	return nil
}

func (m *ClassifierManager) ClassifyAndAssign(docID string, ocrText string) {
	if ocrText == "" {
		return
	}

	m.mu.RLock()
	c := m.classifier
	m.mu.RUnlock()

	if c == nil || len(c.Classes) < 2 {
		return
	}

	tokens := Tokenize(ocrText)
	if len(tokens) == 0 {
		return
	}

	scores, inx, strict := c.LogScores(tokens)

	if strict {
		log.Printf("classifier: strict mode for %s, skipping", docID)
		return
	}

	if inx < 0 || inx >= len(scores) {
		return
	}

	bestScore := scores[inx]
	bestIdx := inx

	if len(scores) > 1 {
		for i, s := range scores {
			if s > bestScore {
				bestScore = s
				bestIdx = i
			}
		}
	}

	secondBest := -9999.0
	for i, s := range scores {
		if i != bestIdx && s > secondBest {
			secondBest = s
		}
	}

	if bestScore < -5.0 {
		log.Printf("classifier: low confidence %.2f for %s, skipping", bestScore, docID)
		return
	}

	if len(scores) > 1 && (bestScore-secondBest) < 1.0 {
		log.Printf("classifier: uncertain (diff=%.2f) for %s, skipping", bestScore-secondBest, docID)
		return
	}

	subcategoryID := string(c.Classes[bestIdx])

	if subcategoryID == defaultOtherClass {
		log.Printf("classifier: classified as _other for %s, skipping", docID)
		return
	}

	log.Printf("classifier: document %s classified as %s (score=%.2f)", docID, subcategoryID, bestScore)

	err := m.app.RunInTransaction(func(txApp core.App) error {
		doc, err := txApp.FindRecordById("documents", docID)
		if err != nil {
			return fmt.Errorf("document %s not found: %w", docID, err)
		}

		sub, err := txApp.FindRecordById("subcategories", subcategoryID)
		if err != nil {
			return fmt.Errorf("subcategory %s not found: %w", subcategoryID, err)
		}

		doc.Set("subcategory_id", subcategoryID)
		doc.Set("category_id", sub.GetString("category_id"))
		return txApp.Save(doc)
	})
	if err != nil {
		log.Printf("classifier: failed to assign %s: %v", docID, err)
	}
}

func (m *ClassifierManager) Retrain(subcategoryID string) error {
	log.Printf("classifier: retraining model for subcategory %s", subcategoryID)

	docs, err := m.app.FindRecordsByFilter("documents",
		"subcategory_id = {:sid} && ocr_txt != ''", "", -1, 0,
		map[string]any{"sid": subcategoryID})
	if err != nil {
		return fmt.Errorf("query documents for retrain: %w", err)
	}

	if len(docs) == 0 {
		log.Printf("classifier: no documents with OCR for %s, skipping", subcategoryID)
		return nil
	}

	ocrTexts := make([]string, len(docs))
	for i, doc := range docs {
		ocrTexts[i] = doc.GetString("ocr_txt")
	}

	words := TopWords(ocrTexts, 1000)
	if len(words) == 0 {
		return nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	c := m.classifier

	if c == nil {
		c = bayesian.NewClassifier(bayesian.Class(subcategoryID), bayesian.Class(defaultOtherClass))
		c.Learn(words, bayesian.Class(subcategoryID))
	} else {
		found := false
		for _, cls := range c.Classes {
			if string(cls) == subcategoryID {
				found = true
				break
			}
		}

		trainingData := make(map[string][]string)
		for _, cls := range c.Classes {
			clsName := string(cls)
			if clsName == defaultOtherClass {
				continue
			}
			if clsName == subcategoryID {
				trainingData[clsName] = ocrTexts
			} else {
				wm := c.WordsByClass(cls)
				existingWords := make([]string, 0, len(wm))
				for w := range wm {
					existingWords = append(existingWords, w)
				}
				trainingData[clsName] = []string{strings.Join(existingWords, " ")}
			}
		}
		if !found {
			trainingData[subcategoryID] = ocrTexts
		}

		newC := rebuildClassifierFromData(trainingData, 1000)
		if newC == nil {
			return fmt.Errorf("rebuild classifier returned nil for %s", subcategoryID)
		}
		c = newC
	}

	m.classifier = c

	path := filepath.Join(m.modelsDir, modelFileName)
	if err := SaveModel(path, c); err != nil {
		return fmt.Errorf("save model: %w", err)
	}

	log.Printf("classifier: model saved with %d classes (%d words for %s)", len(c.Classes), len(words), subcategoryID)
	return nil
}

func rebuildClassifierFromData(trainingData map[string][]string, topN int) *bayesian.Classifier {
	if len(trainingData) == 0 {
		return nil
	}

	classNames := make([]bayesian.Class, 0, len(trainingData)+1)
	for subcatID := range trainingData {
		classNames = append(classNames, bayesian.Class(subcatID))
	}
	classNames = append(classNames, bayesian.Class(defaultOtherClass))

	c := bayesian.NewClassifier(classNames...)

	for subcatID, docs := range trainingData {
		words := TopWords(docs, topN)
		if len(words) > 0 {
			c.Learn(words, bayesian.Class(subcatID))
		}
	}

	return c
}
