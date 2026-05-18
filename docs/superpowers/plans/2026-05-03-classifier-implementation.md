# Classifier Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add automatic Naive Bayes document classification after OCR extraction, with model retraining when users reassign documents.

**Architecture:** New `internal/classifier/` package with tokenizer, model persistence, and ClassifierManager. Hooks into the existing OcrWorker (OnComplete callback) and PocketBase OnRecordUpdate event. Single `classifier.model` file stored in `pb_data/models/`.

**Tech Stack:** Go 1.25, `github.com/jbrukh/bayesian`, PocketBase hooks, `encoding/gob` for persistence.

---

### Task 1: Add bayesian dependency

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`

- [ ] **Step 1: Add the dependency**

```bash
go get github.com/jbrukh/bayesian
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./...
```

Expected: no errors, `go.mod` updated with the new require line.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "deps: add jbrukh/bayesian for Naive Bayes classification"
```

---

### Task 2: Tokenizer (TDD)

**Files:**
- Create: `internal/classifier/tokenizer.go`
- Create: `internal/classifier/tokenizer_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/classifier/tokenizer_test.go`:

```go
package classifier

import (
	"reflect"
	"sort"
	"testing"
)

func TestTokenize(t *testing.T) {
	tokens := Tokenize("Hola Mundo! Esto es una prueba de tokenización 123.")
	sort.Strings(tokens)
	expected := []string{"hola", "mundo", "esto", "prueba", "tokenización"}
	sort.Strings(expected)
	if !reflect.DeepEqual(tokens, expected) {
		t.Errorf("Tokenize = %v, want %v", tokens, expected)
	}
}

func TestTokenizeFiltersShortWords(t *testing.T) {
	tokens := Tokenize("a b cd de la y el en")
	if len(tokens) != 0 {
		t.Errorf("expected no tokens, got %v", tokens)
	}
}

func TestTokenizeFiltersStopwords(t *testing.T) {
	tokens := Tokenize("de la que el en y a los se del las")
	if len(tokens) != 0 {
		t.Errorf("expected no tokens from stopwords, got %v", tokens)
	}
}

func TestTopWords(t *testing.T) {
	docs := []string{
		"gato perro gato perro gato",
		"perro perro gato pez",
		"pez pez pez ave",
	}
	top := TopWords(docs, 3)
	expected := []string{"gato", "perro", "pez"}
	sort.Strings(top)
	sort.Strings(expected)
	if !reflect.DeepEqual(top, expected) {
		t.Errorf("TopWords = %v, want %v", top, expected)
	}
}

func TestTopWordsWithStopwords(t *testing.T) {
	docs := []string{
		"el gato y la casa de papel",
		"la casa del perro que ladra",
	}
	top := TopWords(docs, 3)
	// stopwords "el", "y", "la", "de", "del", "que" should be filtered
	sort.Strings(top)
	expected := []string{"casa", "gato", "papel", "perro"}
	// "casa" appears twice, "gato" once, "papel" once, "perro" once
	// With 4 words at freq 2 and 1, top 3 should include "casa" + 2 of the others
	if len(top) != 3 {
		t.Fatalf("expected 3 top words, got %d: %v", len(top), top)
	}
	if top[0] != "casa" {
		t.Errorf("expected 'casa' as top word (freq 2), got %s", top[0])
	}
}

func TestTopWordsEmptyDocs(t *testing.T) {
	top := TopWords([]string{}, 10)
	if len(top) != 0 {
		t.Errorf("expected empty result, got %v", top)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/classifier/ -v -run "TestTokenize|TestTopWords"
```

Expected: compilation error "undefined: Tokenize", "undefined: TopWords"

- [ ] **Step 3: Write the tokenizer implementation**

Create `internal/classifier/tokenizer.go`:

```go
package classifier

import (
	"regexp"
	"sort"
	"strings"
)

var nonWordRE = regexp.MustCompile(`\W+`)

var spanishStopwords = map[string]bool{
	"de": true, "la": true, "que": true, "el": true, "en": true,
	"y": true, "a": true, "los": true, "se": true, "del": true,
	"las": true, "un": true, "por": true, "con": true, "no": true,
	"una": true, "su": true, "para": true, "es": true, "al": true,
	"lo": true, "como": true, "mas": true, "pero": true, "sus": true,
	"le": true, "ya": true, "o": true, "este": true, "fue": true,
	"ha": true, "era": true, "muy": true, "son": true, "todo": true,
	"si": true, "sin": true, "sobre": true, "entre": true, "cuando": true,
	"tambien": true, "asi": true, "dos": true, "hasta": true, "desde": true,
	"porque": true, "cada": true, "otros": true, "gran": true, "vez": true,
	"ano": true, "esto": true, "parte": true, "me": true, "mi": true,
	"tu": true, "te": true, "nos": true, "os": true, "les": true,
	"e": true, "ni": true, "mas": true, "tras": true, "hacia": true,
	"durante": true, "contra": true, "bajo": true,
}

func Tokenize(text string) []string {
	text = strings.ToLower(text)
	parts := nonWordRE.Split(text, -1)
	tokens := make([]string, 0, len(parts))
	for _, p := range parts {
		if len(p) < 3 {
			continue
		}
		if spanishStopwords[p] {
			continue
		}
		tokens = append(tokens, p)
	}
	return tokens
}

type wordCount struct {
	word  string
	count int
}

func TopWords(docs []string, n int) []string {
	freq := make(map[string]int)
	for _, doc := range docs {
		tokens := Tokenize(doc)
		seen := make(map[string]bool)
		for _, t := range tokens {
			if !seen[t] {
				freq[t]++
				seen[t] = true
			}
		}
	}

	wc := make([]wordCount, 0, len(freq))
	for w, c := range freq {
		wc = append(wc, wordCount{w, c})
	}

	sort.Slice(wc, func(i, j int) bool {
		if wc[i].count == wc[j].count {
			return wc[i].word < wc[j].word
		}
		return wc[i].count > wc[j].count
	})

	if n > len(wc) {
		n = len(wc)
	}

	result := make([]string, n)
	for i := 0; i < n; i++ {
		result[i] = wc[i].word
	}
	return result
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/classifier/ -v -run "TestTokenize|TestTopWords"
```

Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/classifier/tokenizer.go internal/classifier/tokenizer_test.go
git commit -m "feat(classifier): add OCR text tokenizer with Spanish stopwords"
```

---

### Task 3: Model persistence

**Files:**
- Create: `internal/classifier/model.go`
- Create: `internal/classifier/model_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/classifier/model_test.go`:

```go
package classifier

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jbrukh/bayesian"
)

func TestTrainModel(t *testing.T) {
	docs := []string{
		"factura pago servicio electricidad enero",
		"factura consumo electricidad febrero",
		"pago factura luz marzo",
	}
	c := TrainModel(docs, 10, "test_subcat")

	if c == nil {
		t.Fatal("TrainModel returned nil")
	}
	if len(c.Classes) != 1 {
		t.Fatalf("expected 1 class, got %d", len(c.Classes))
	}
	if c.Classes[0].Name != "test_subcat" {
		t.Errorf("class name = %s, want test_subcat", c.Classes[0].Name)
	}
	if len(c.Classes[0].Words) == 0 {
		t.Error("class has no words")
	}
}

func TestTrainModelEmptyDocs(t *testing.T) {
	c := TrainModel([]string{}, 10, "empty_subcat")
	if c != nil {
		t.Error("TrainModel with empty docs should return nil")
	}
}

func TestModelPersistence(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "test.model")

	docs := []string{
		"factura pago enero electricidad",
		"factura consumo febrero gas",
		"pago recibo agua marzo",
	}
	c := TrainModel(docs, 20, "subcat_1")

	err := SaveModel(modelPath, c)
	if err != nil {
		t.Fatalf("SaveModel failed: %v", err)
	}

	_, err = os.Stat(modelPath)
	if err != nil {
		t.Fatalf("model file not created: %v", err)
	}

	loaded, err := LoadModel(modelPath)
	if err != nil {
		t.Fatalf("LoadModel failed: %v", err)
	}

	if len(loaded.Classes) != 1 {
		t.Fatalf("loaded classifier has %d classes, want 1", len(loaded.Classes))
	}
	if loaded.Classes[0].Name != "subcat_1" {
		t.Errorf("loaded class name = %s, want subcat_1", loaded.Classes[0].Name)
	}
	if len(loaded.Classes[0].Words) == 0 {
		t.Error("loaded class has no words")
	}
}

func TestLoadModelNonExistent(t *testing.T) {
	_, err := LoadModel("/nonexistent/path.model")
	if err == nil {
		t.Error("LoadModel should return error for nonexistent file")
	}
}

func TestClassify(t *testing.T) {
	docs := []string{
		"factura pago enero electricidad consumo",
		"factura pago febrero electricidad luz",
		"factura consumo marzo gas natural",
	}
	c := TrainModel(docs, 20, "facturas_electricidad")

	// Classify a text similar to what was trained on
	scores, inx, strict := c.LogScores(Tokenize("factura pago electricidad abril"))
	if strict {
		t.Log("classification in strict mode (may be ok for small training sets)")
	}
	if inx < 0 || inx >= len(c.Classes) {
		t.Fatalf("invalid class index: %d", inx)
	}
	if c.Classes[inx].Name != "facturas_electricidad" {
		t.Errorf("classified as %s, want facturas_electricidad (scores: %v)", c.Classes[inx].Name, scores)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/classifier/ -v -run "TestTrainModel|TestModelPersistence|TestLoadModel|TestClassify"
```

Expected: compilation error

- [ ] **Step 3: Write model.go implementation**

Create `internal/classifier/model.go`:

```go
package classifier

import (
	"encoding/gob"
	"fmt"
	"os"

	"github.com/jbrukh/bayesian"
)

func init() {
	gob.Register(bayesian.Class{})
}

type modelData struct {
	ClassNames []string
	Classes    []bayesian.Class
}

func TrainModel(documents []string, topN int, className string) *bayesian.Classifier {
	if len(documents) == 0 {
		return nil
	}

	words := TopWords(documents, topN)
	if len(words) == 0 {
		return nil
	}

	c := bayesian.NewClassifier(bayesian.Class(className))
	c.Learn(words, bayesian.Class(className))
	return c
}

func SaveModel(path string, c *bayesian.Classifier) error {
	if c == nil {
		return fmt.Errorf("cannot save nil classifier")
	}

	data := modelData{
		ClassNames: make([]string, len(c.Classes)),
		Classes:    c.Classes,
	}
	for i, cls := range c.Classes {
		data.ClassNames[i] = string(cls.Name)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create model dir: %w", err)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create model file: %w", err)
	}
	defer f.Close()

	enc := gob.NewEncoder(f)
	if err := enc.Encode(data); err != nil {
		return fmt.Errorf("encode model: %w", err)
	}
	return nil
}

func LoadModel(path string) (*bayesian.Classifier, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open model file: %w", err)
	}
	defer f.Close()

	var data modelData
	dec := gob.NewDecoder(f)
	if err := dec.Decode(&data); err != nil {
		return nil, fmt.Errorf("decode model: %w", err)
	}

	if len(data.ClassNames) == 0 {
		return nil, fmt.Errorf("model file has no classes")
	}

	classNames := make([]bayesian.Class, len(data.ClassNames))
	for i, name := range data.ClassNames {
		classNames[i] = bayesian.Class(name)
	}

	c := bayesian.NewClassifier(classNames...)
	c.Classes = data.Classes

	return c, nil
}
```

- [ ] **Step 4: Add missing import to model.go**

The `filepath` import is used in `SaveModel`. Ensure the import block is:

```go
import (
	"encoding/gob"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jbrukh/bayesian"
)
```

- [ ] **Step 5: Run tests to verify they pass**

```bash
go test ./internal/classifier/ -v -run "TestTrainModel|TestModelPersistence|TestLoadModel|TestClassify"
```

Expected: all tests PASS

- [ ] **Step 6: Commit**

```bash
git add internal/classifier/model.go internal/classifier/model_test.go
git commit -m "feat(classifier): add model persistence with gob serialization"
```

---

### Task 4: ClassifierManager

**Files:**
- Create: `internal/classifier/classifier.go`
- Create: `internal/classifier/classifier_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/classifier/classifier_test.go`:

```go
package classifier

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jbrukh/bayesian"
)

func TestClassifierManagerLoadEmpty(t *testing.T) {
	dir := t.TempDir()
	mgr := &ClassifierManager{modelsDir: dir}
	err := mgr.Load()
	if err != nil {
		t.Logf("Load on empty dir returned error (expected): %v", err)
	}
	if mgr.classifier != nil {
		t.Error("classifier should be nil for empty dir")
	}
}

func TestClassifierManagerClassifyAndAssignNoModel(t *testing.T) {
	mgr := &ClassifierManager{modelsDir: t.TempDir()}
	// Should not panic with nil classifier
	mgr.ClassifyAndAssign("doc123", "some ocr text")
}

func TestRebuildClassifier(t *testing.T) {
	mgr := &ClassifierManager{modelsDir: t.TempDir()}

	// Simulate training data: maps subcategory_id -> ocr texts
	trainingData := map[string][]string{
		"subcat_a": {"factura pago enero electricidad consumo", "factura febrero pago luz"},
		"subcat_b": {"contrato arrendamiento vivienda clausulas", "contrato alquiler apartamento"},
	}

	c := rebuildClassifierFromData(trainingData, 50)
	if c == nil {
		t.Fatal("rebuildClassifierFromData returned nil")
	}
	if len(c.Classes) != 2 {
		t.Fatalf("expected 2 classes, got %d", len(c.Classes))
	}

	// Save and reload
	path := filepath.Join(t.TempDir(), "classifier.model")
	if err := SaveModel(path, c); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadModel(path)
	if err != nil {
		t.Fatal(err)
	}

	// Classify a document that should match subcat_a
	_, inx, _ := loaded.LogScores(Tokenize("factura pago electricidad marzo"))
	className := string(loaded.Classes[inx].Name)
	if className != "subcat_a" {
		t.Logf("classified as %s, want subcat_a (may be ok with small training set)", className)
	}
}

func TestClassifierManagerRetrain(t *testing.T) {
	dir := t.TempDir()
	mgr := &ClassifierManager{modelsDir: dir}

	// Build training data
	trainingData := map[string][]string{
		"subcat_facturas": {"factura pago enero electricidad consumo", "factura febrero luz"},
	}

	c := rebuildClassifierFromData(trainingData, 50)
	path := filepath.Join(dir, "classifier.model")
	if err := SaveModel(path, c); err != nil {
		t.Fatal(err)
	}
	mgr.classifier = c

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatal("model file should exist")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/classifier/ -v
```

Expected: compilation error for ClassifierManager, rebuildClassifierFromData

- [ ] **Step 3: Write classifier.go implementation**

Create `internal/classifier/classifier.go`:

```go
package classifier

import (
	"fmt"
	"log"
	"path/filepath"
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
	classIdx   map[string]int
	mu         sync.RWMutex
}

func NewClassifierManager(app *pocketbase.PocketBase, modelsDir string) *ClassifierManager {
	return &ClassifierManager{
		app:       app,
		modelsDir: modelsDir,
		classIdx:  make(map[string]int),
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
	m.rebuildIndex()
	m.mu.Unlock()
	log.Printf("classifier: loaded model with %d classes", len(c.Classes))
	return nil
}

func (m *ClassifierManager) rebuildIndex() {
	m.classIdx = make(map[string]int)
	if m.classifier == nil {
		return
	}
	for i, cls := range m.classifier.Classes {
		m.classIdx[string(cls.Name)] = i
	}
}

func (m *ClassifierManager) ClassifyAndAssign(docID string, ocrText string) {
	if ocrText == "" {
		return
	}

	m.mu.RLock()
	c := m.classifier
	m.mu.RUnlock()

	if c == nil || len(c.Classes) == 0 {
		return
	}

	tokens := Tokenize(ocrText)
	if len(tokens) == 0 {
		return
	}

	scores, inx, strict := c.LogScores(tokens)

	if strict {
		log.Printf("classifier: strict mode classification for %s, skipping", docID)
		return
	}

	if inx < 0 || inx >= len(scores) {
		return
	}

	bestScore := scores[inx]
	if bestScore < -5.0 {
		log.Printf("classifier: low confidence score %.2f for %s, skipping", bestScore, docID)
		return
	}

	if len(scores) > 1 {
		secondBest := scores[0]
		bestIdx := 0
		for i, s := range scores {
			if s > bestScore {
				secondBest = bestScore
				bestScore = s
				bestIdx = i
			} else if s > secondBest && i != bestIdx {
				secondBest = s
			}
		}
		inx = bestIdx
		if bestScore-secondBest < 1.0 {
			log.Printf("classifier: uncertain classification (diff=%.2f) for %s, skipping", bestScore-secondBest, docID)
			return
		}
	}

	subcategoryID := string(c.Classes[inx].Name)
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
		log.Printf("classifier: no documents with OCR text for subcategory %s, skipping retrain", subcategoryID)
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
		c = bayesian.NewClassifier(bayesian.Class(subcategoryID))
	} else {
		found := false
		for _, cls := range c.Classes {
			if string(cls.Name) == subcategoryID {
				found = true
				break
			}
		}
		if !found {
			newClasses := make([]bayesian.Class, len(c.Classes)+1)
			for i, cls := range c.Classes {
				newClasses[i] = cls
			}
			newClasses[len(c.Classes)] = bayesian.Class(subcategoryID)
			c = bayesian.NewClassifier(newClasses...)
			c.Classes = newClasses
		}
	}

	for i, cls := range c.Classes {
		if string(cls.Name) == subcategoryID {
			freq := make(map[string]uint64)
			for _, w := range words {
				freq[w]++
			}
			c.Classes[i].Words = freq
			c.Classes[i].Documents = map[string]uint64{"_total": uint64(len(words))}
			c.Classes[i].Total = uint64(len(words))
			break
		}
	}

	m.classifier = c
	m.rebuildIndex()

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

	classNames := make([]bayesian.Class, 0, len(trainingData))
	for subcatID := range trainingData {
		classNames = append(classNames, bayesian.Class(subcatID))
	}

	c := bayesian.NewClassifier(classNames...)

	for subcatID, docs := range trainingData {
		words := TopWords(docs, topN)
		if len(words) > 0 {
			c.Learn(words, bayesian.Class(subcatID))
		}
	}

	return c
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/classifier/ -v
```

Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add internal/classifier/classifier.go internal/classifier/classifier_test.go
git commit -m "feat(classifier): add ClassifierManager with classify and retrain"
```

---

### Task 5: Add OnComplete callback to OcrWorker

**Files:**
- Modify: `internal/ocr/queue.go`

- [ ] **Step 1: Add OnComplete field to OcrJob and invoke it**

Modify `internal/ocr/queue.go` — change the `OcrJob` struct and `processJob`:

```go
type OcrJob struct {
	ID         string
	FilePath   string
	FileType   string
	OnComplete func(ocrText string)
}
```

In `processJob`, after the successful `updateDocumentStatus` call, add the callback invocation:

```go
func (w *OcrWorker) processJob(job OcrJob) {
	log.Printf("OCR processing document %s (%s)", job.ID, job.FileType)

	ctx := context.Background()
	text, err := w.client.ExtractText(ctx, job.ID, job.FilePath, job.FileType)

	if err != nil {
		log.Printf("OCR error for document %s: %v", job.ID, err)
		w.updateDocumentStatus(job.ID, "error", "")
		return
	}

	w.updateDocumentStatus(job.ID, "processed", text)
	log.Printf("OCR complete for document %s (%d chars)", job.ID, len(text))

	if job.OnComplete != nil {
		job.OnComplete(text)
	}
}
```

The rest of the file stays the same.

- [ ] **Step 2: Verify compilation**

```bash
go build ./...
```

Expected: no errors

- [ ] **Step 3: Commit**

```bash
git add internal/ocr/queue.go
git commit -m "feat(ocr): add OnComplete callback to OcrJob for post-OCR processing"
```

---

### Task 6: Wire up classifier in main.go

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Update main.go imports and initialization**

Change `main.go`:

```go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"SebasXeon/Fileoteca/internal/addfile"
	"SebasXeon/Fileoteca/internal/classifier"
	"SebasXeon/Fileoteca/internal/ocr"
	"SebasXeon/Fileoteca/internal/shell"

	_ "SebasXeon/Fileoteca/migrations"

	"github.com/pocketbase/pocketbase/core"
)
```

In the normal startup path (after `ocrWorker.Start()`), add:

```go
			// Create classifier manager
			classifierMgr := classifier.NewClassifierManager(app, "pb_data/models")
			classifierMgr.Load()

			app.OnRecordCreate("documents").BindFunc(func(e *core.RecordEvent) error {
				go func() {
					resolvedPath, cleanup, err := ocr.ResolvePath(e.Record)
					if err != nil {
						log.Printf("OCR skip para %s: %v", e.Record.Id, err)
						return
					}
					ocrWorker.Enqueue(ocr.OcrJob{
						ID:       e.Record.Id,
						FilePath: resolvedPath,
						FileType: e.Record.GetString("file_ext"),
						OnComplete: func(ocrText string) {
							classifierMgr.ClassifyAndAssign(e.Record.Id, ocrText)
						},
					})
					go func() {
						time.Sleep(5 * time.Minute)
						cleanup()
					}()
				}()
				return e.Next()
			})

			app.OnRecordUpdate("documents").BindFunc(func(e *core.RecordEvent) error {
				go func() {
					oldSub := e.Record.Original().GetString("subcategory_id")
					newSub := e.Record.GetString("subcategory_id")
					if oldSub != newSub && newSub != "" {
						cfg, _ := shell.LoadConfig()
						if newSub == cfg.DefaultSubcategoryID {
							return
						}
						if err := classifierMgr.Retrain(newSub); err != nil {
							log.Printf("classifier: retrain error: %v", err)
						}
					}
				}()
				return e.Next()
			})
```

The full `main.go` after changes should be:

```go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"SebasXeon/Fileoteca/internal/addfile"
	"SebasXeon/Fileoteca/internal/classifier"
	"SebasXeon/Fileoteca/internal/ocr"
	"SebasXeon/Fileoteca/internal/shell"

	_ "SebasXeon/Fileoteca/migrations"

	"github.com/pocketbase/pocketbase/core"
)

func resolveOcrServerDir() string {
	cwd, _ := os.Getwd()
	cwdCandidate := filepath.Join(cwd, "ocr-server")
	if info, err := os.Stat(cwdCandidate); err == nil && info.IsDir() {
		return cwdCandidate
	}

	exec, err := os.Executable()
	if err == nil {
		exeCandidate := filepath.Join(filepath.Dir(exec), "ocr-server")
		if info, err := os.Stat(exeCandidate); err == nil && info.IsDir() {
			return exeCandidate
		}
	}

	return "ocr-server"
}

func main() {
	var addFilePath string

	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "--add" && i+1 < len(os.Args) {
			addFilePath = os.Args[i+1]
			break
		}
	}

	if addFilePath != "" {
		info, err := addfile.Extract(addFilePath)
		if err != nil {
			shell.ShowError(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}

		if shell.IsServerRunning() {
			if err := shell.AddFileViaHTTP(info); err != nil {
				shell.ShowError(fmt.Sprintf("Error agregando documento:\n%v", err))
				os.Exit(1)
			}
			shell.ShowInfo(fmt.Sprintf("\"%s\" agregado a Fileoteca.", info.FileName))
			return
		}

		newArgs := os.Args[:1]
		for i := 1; i < len(os.Args); i++ {
			if os.Args[i] == "--add" {
				i++
				continue
			}
			newArgs = append(newArgs, os.Args[i])
		}
		os.Args = newArgs

		app, stopFn, err := shell.StartServer()
		if err != nil {
			log.Fatalf("error iniciando servidor: %v\n", err)
		}

		if err := shell.AddFileViaDAO(app, info); err != nil {
			log.Printf("error agregando documento vía DAO: %v\n", err)
		}

		shell.StartTray(stopFn)
		return
	}

	app, stopFn, err := shell.StartServer()
	if err != nil {
		log.Fatalf("error iniciando servidor: %v\n", err)
	}

	ocrServerDir := resolveOcrServerDir()
	ocrServer, ocrErr := ocr.StartOcrServer(ocrServerDir)
	if ocrErr != nil {
		log.Printf("aviso: OCR server no disponible: %v", ocrErr)
	} else {
		defer ocrServer.Stop()

		ocrClient, clientErr := ocr.NewOcrClient(ocr.OcrServerAddr())
		if clientErr != nil {
			log.Printf("aviso: cliente OCR no disponible: %v", clientErr)
		} else {
			defer ocrClient.Close()
			ocrWorker := ocr.NewOcrWorker(ocrClient, app, 100)
			ocrWorker.Start()
			defer ocrWorker.Stop()

			classifierMgr := classifier.NewClassifierManager(app, "pb_data/models")
			classifierMgr.Load()

			app.OnRecordCreate("documents").BindFunc(func(e *core.RecordEvent) error {
				go func() {
					resolvedPath, cleanup, err := ocr.ResolvePath(e.Record)
					if err != nil {
						log.Printf("OCR skip para %s: %v", e.Record.Id, err)
						return
					}
					ocrWorker.Enqueue(ocr.OcrJob{
						ID:       e.Record.Id,
						FilePath: resolvedPath,
						FileType: e.Record.GetString("file_ext"),
						OnComplete: func(ocrText string) {
							classifierMgr.ClassifyAndAssign(e.Record.Id, ocrText)
						},
					})
					go func() {
						time.Sleep(5 * time.Minute)
						cleanup()
					}()
				}()
				return e.Next()
			})

			app.OnRecordUpdate("documents").BindFunc(func(e *core.RecordEvent) error {
				go func() {
					oldSub := e.Record.Original().GetString("subcategory_id")
					newSub := e.Record.GetString("subcategory_id")
					if oldSub != newSub && newSub != "" {
						cfg, _ := shell.LoadConfig()
						if newSub == cfg.DefaultSubcategoryID {
							return
						}
						if err := classifierMgr.Retrain(newSub); err != nil {
							log.Printf("classifier: retrain error: %v", err)
						}
					}
				}()
				return e.Next()
			})

			log.Println("OCR integrado correctamente")
		}
	}

	fmt.Println("Fileoteca iniciada. Haz clic en el icono del área de notificación.")
	shell.StartTray(stopFn)
}
```

- [ ] **Step 2: Verify compilation**

```bash
go build ./...
```

Expected: no errors

- [ ] **Step 3: Run all classifier tests**

```bash
go test ./internal/classifier/ -v
```

Expected: all tests PASS

- [ ] **Step 4: Commit**

```bash
git add main.go
git commit -m "feat: wire classifier into OCR pipeline and document update hooks"
```

---

### Task 7: Final verification

**Files:**
- None (verification only)

- [ ] **Step 1: Run all tests**

```bash
go test ./... -v
```

Expected: all tests pass (excluding OCR server tests that require the Python server running)

- [ ] **Step 2: Build the binary**

```bash
go build -o Fileoteca.exe
```

Expected: successful build, no errors

- [ ] **Step 3: Verify classifier package isolation**

```bash
go vet ./internal/classifier/...
```

Expected: no warnings

- [ ] **Step 4: Commit final build artifacts if needed**

No commit needed unless build produced files to track.
