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

// maxWordsPerClass limits how many distinct words each class learns.
// Keeping this bounded prevents Laplace smoothing from penalising the class
// too heavily for unknown words.
const maxWordsPerClass = 100

// confidenceThreshold is the minimum log-score difference required between
// the best class and the second-best to accept an auto-classification.
const confidenceThreshold = 0.5

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

// ClassifyAndAssign attempts to classify a document and assign it automatically.
// It skips documents that already have a non-default subcategory (manual assignment).
func (m *ClassifierManager) ClassifyAndAssign(docID string, ocrText string, defaultSubcategoryID string) {
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

	if len(scores) > 1 && (bestScore-secondBest) < confidenceThreshold {
		log.Printf("classifier: uncertain classification (diff=%.2f) for %s, skipping", bestScore-secondBest, docID)
		return
	}

	if !strict && len(scores) > 1 {
		log.Printf("classifier: non-strict classification for %s, skipping", docID)
		return
	}

	subcategoryID := string(c.Classes[bestIdx])

	if subcategoryID == defaultOtherClass {
		log.Printf("classifier: classified as _other for %s, skipping", docID)
		return
	}

	log.Printf("classifier: document %s classified as %s (score=%.2f, diff=%.2f)", docID, subcategoryID, bestScore, bestScore-secondBest)

	err := m.app.RunInTransaction(func(txApp core.App) error {
		doc, err := txApp.FindRecordById("documents", docID)
		if err != nil {
			return fmt.Errorf("document %s not found: %w", docID, err)
		}

		// Skip if the user already assigned a non-default subcategory manually.
		currentSub := doc.GetString("subcategory_id")
		if currentSub != "" && currentSub != defaultSubcategoryID {
			log.Printf("classifier: document %s already manually assigned to %s, skipping auto-assign", docID, currentSub)
			return nil
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

// Retrain rebuilds the classifier from scratch using ALL subcategories that
// have documents with OCR text. This keeps frequencies accurate and avoids
// the vocabulary explosion caused by incremental Laplace smoothing.
func (m *ClassifierManager) Retrain(triggerSubcategoryID string) error {
	log.Printf("classifier: retraining model (triggered by %s)", triggerSubcategoryID)

	// Fetch all distinct subcategories that have at least one document with OCR text.
	allDocs, err := m.app.FindRecordsByFilter("documents",
		"ocr_txt != '' && subcategory_id != ''", "", -1, 0, nil)
	if err != nil {
		return fmt.Errorf("query all documents for retrain: %w", err)
	}

	if len(allDocs) == 0 {
		log.Printf("classifier: no documents with OCR found, skipping retrain")
		return nil
	}

	// Group OCR texts by subcategory.
	trainingData := make(map[string][]string)
	for _, doc := range allDocs {
		subID := doc.GetString("subcategory_id")
		if subID == "" {
			continue
		}
		trainingData[subID] = append(trainingData[subID], doc.GetString("ocr_txt"))
	}

	if len(trainingData) == 0 {
		log.Printf("classifier: no training data after grouping, skipping retrain")
		return nil
	}

	newC := rebuildClassifierFromData(trainingData, maxWordsPerClass, 2)
	if newC == nil {
		return fmt.Errorf("rebuild classifier returned nil")
	}

	m.mu.Lock()
	m.classifier = newC
	m.mu.Unlock()

	path := filepath.Join(m.modelsDir, modelFileName)
	if err := SaveModel(path, newC); err != nil {
		return fmt.Errorf("save model: %w", err)
	}

	log.Printf("classifier: model saved with %d classes (%d subcategories trained)", len(newC.Classes), len(trainingData))
	return nil
}

func rebuildClassifierFromData(trainingData map[string][]string, topN int, minDocs int) *bayesian.Classifier {
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
		// Adaptive minDocFreq: require words to appear in at least 2 documents
		// only when we have 3+ training docs. Otherwise keep all words.
		mdf := minDocs
		if len(docs) < 3 {
			mdf = 1
		}
		words := topWordsWithMinDocFreq(docs, topN, mdf)
		if len(words) > 0 {
			c.Learn(words, bayesian.Class(subcatID))
		}
	}

	c.Learn(Tokenize(spanishBaselineText), bayesian.Class(defaultOtherClass))

	return c
}

// topWordsWithMinDocFreq extracts the top N words that appear in at least
// minDocs distinct documents. This removes hapax legomena (words appearing
// only once) which are pure noise for Naive Bayes with Laplace smoothing.
func topWordsWithMinDocFreq(docs []string, n int, minDocs int) []string {
	if n <= 0 {
		return nil
	}

	// Count in how many distinct documents each word appears.
	docFreq := make(map[string]int)
	for _, doc := range docs {
		tokens := Tokenize(doc)
		seen := make(map[string]bool)
		for _, t := range tokens {
			if !seen[t] {
				docFreq[t]++
				seen[t] = true
			}
		}
	}

	// Filter to words meeting minimum document frequency.
	filtered := make(map[string]int)
	for w, count := range docFreq {
		if count >= minDocs {
			filtered[w] = count
		}
	}

	// Sort by frequency descending, then alphabetically.
	list := make([]wc, 0, len(filtered))
	for w, c := range filtered {
		list = append(list, wc{w, c})
	}

	// If after filtering we have nothing but we have docs, fall back to
	// regular TopWords so we don't end up with an empty class.
	if len(list) == 0 && len(docs) > 0 {
		return TopWords(docs, n)
	}

	// Sort descending by count
	sortWords(list)

	if n > len(list) {
		n = len(list)
	}
	result := make([]string, n)
	for i := 0; i < n; i++ {
		result[i] = list[i].word
	}
	return result
}

type wc struct {
	word  string
	count int
}

func sortWords(list []wc) {
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].count > list[i].count ||
				(list[j].count == list[i].count && list[j].word < list[i].word) {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
}

var spanishBaselineText = `
	documento pagina archivo fecha numero nombre direccion telefono email
	informacion datos contenido texto seccion capitulo indice titulo
	tabla figura imagen grafico lista item elemento valor unidad
	codigo referencia anexo adjunto copia original version revision
	proyecto informe reporte resumen analisis estudio caso ejemplo
	desarrollo implementacion proceso procedimiento funcion metodo
	sistema aplicacion programa software hardware dispositivo equipo
	servicio producto cliente usuario persona empresa organizacion
	general especifico particular comun basico avanzado principal
	calle ciudad pais provincia municipio comunidad region zona
	nacional internacional local federal estatal publico privado
	ministerio departamento division oficina secretaria direccion
	gerencia administracion gestion coordinacion supervision
	presidente director gerente jefe encargado responsable
	empleado trabajador funcionario personal staff equipo
	obra construccion edificio inmueble terreno solar parcela
	vehiculo automovil motocicleta camion furgoneta ciclomotor
	banco cuenta tarjeta credito debito prestamo hipoteca
	seguro poliza cobertura siniestro reclamacion indemnizacion
	hospital clinica medico enfermera paciente tratamiento diagnostico
	colegio instituto universidad academia centro educativo
	alumno estudiante profesor docente tutor maestro
	asignatura materia curso ciclo modulo trimestre semestre
	evaluacion examen prueba control ejercicio cuestionario
	nota calificacion puntos porcentaje aprobado suspenso
	ingreso egreso gasto ingreso presupuesto balance cuenta
	impuesto tributo recargo recaudacion declaracion modelo
	contrato acuerdo convenio pacto clausula anexo adhesion
	orden pedido compra venta factura recibo albaran
	solicitud demanda instancia recurso apelacion reclamacion
	resolucion sentencia auto fallo dictamen providencia
	diligencia acta certificado escritura testimonio copia
	registro inscripcion anotacion asiento partida folio
	libro registro archivo expediente carpeta legajo
`
