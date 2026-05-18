# Classifier Design — Naive Bayes Document Classification

**Date:** 2026-05-03
**Status:** Approved

---

## Overview

Add automatic document classification to Fileoteca using Naive Bayes. When OCR extracts text from a document, the backend classifies it into the best-matching subcategory. If no match is found, the document stays in the default "Sin categorizar / General" subcategory. The model retrains every time a user manually assigns a document to a subcategory.

---

## Package Structure

New package: `internal/classifier/`

```
internal/classifier/
  classifier.go    — ClassifierManager: load/unload models, classify text, retrain
  model.go         — model functions: train, persist to .model (gob), load
  tokenizer.go     — OCR text tokenization + top-N word selection
```

---

## Library

`github.com/jbrukh/bayesian` — Multinomial Naive Bayes classifier with native `gob` serialization support.

---

## Model Architecture

**Single model file:** `pb_data/models/classifier.model`

One `bayesian.Classifier` with N classes, one per subcategory (keyed by `subcategory_id`).

> **Note:** The `model_name` field on the `subcategories` table is no longer used. With a unified model, the class key is the `subcategory_id` directly. The field is kept in the schema for backwards compatibility but is ignored by the classifier.

```
Classifier
  +-- Class "subcat_abc123" — top-1000 frequent words from docs in that subcategory
  +-- Class "subcat_def456" — ...
  +-- Class "subcat_ghi789" — ...
  +-- ...
```

Classification runs against ALL classes at once; `bayesian` returns the most probable class natively.

---

## ClassifierManager API

```go
type ClassifierManager struct {
    app        *pocketbase.PocketBase
    modelsDir  string
    classifier *bayesian.Classifier
    classIdx   map[string]int  // subcategory_id -> class index
    mu         sync.RWMutex
}

func NewClassifierManager(app *pocketbase.PocketBase, dir string) *ClassifierManager
func (m *ClassifierManager) Load() error
func (m *ClassifierManager) ClassifyAndAssign(docID, ocrText string)
func (m *ClassifierManager) Retrain(subcategoryID string) error
```

---

## Tokenization (`tokenizer.go`)

- Lowercase
- Split by `\W+` (non-word characters)
- Filter out words of 1-2 characters
- Filter out Spanish stopwords: de, la, que, el, en, y, a, los, se, del, las, un, por, con, no, una, su, para, es, al, lo, como, mas, pero, sus, le, ya, o, este, fue, ha, era, muy, son, todo, si, sin, sobre, entre, cuando, tambien, asi, dos, hasta, desde, porque, cada, otros, gran, vez, ano, esto, parte

---

## Data Flows

### Flow A: Automatic Classification (on document creation)

```
1. Document created → POST /api/collections/documents/records
2. OnRecordCreate hook fires → OCR worker enqueues job
3. OCR worker extracts text → saves ocr_txt, status="processed"
4. [NEW] OnComplete callback → ClassifierManager.ClassifyAndAssign()
5. Tokenize ocr_text → run classifier.LogScores()
6. If highest score > threshold → update category_id + subcategory_id
7. If ≤ threshold or empty text → stays in "Sin categorizar / General"
```

### Flow B: Retraining (on document update)

```
1. User moves document to new subcategory → PATCH /api/collections/documents/records/{id}
2. OnRecordUpdate hook detects subcategory_id changed
3. Retrain(subcategoryID):
   a. Query all documents with this subcategory_id where ocr_txt != ""
   b. Concatenate all ocr_txt, tokenize, compute top-1000 words by frequency
   c. Update or create the class for this subcategory in the Classifier
   d. Serialize to pb_data/models/classifier.model via gob
```

**Note:** Only the DESTINATION subcategory is retrained. The source subcategory's model becomes slightly stale until it receives new documents.

---

## Confidence Threshold

`bayesian.LogScores()` returns log-probabilities for each class. Higher is more likely.

- If the difference between the top score and the second-best score is small (< 1.0), the result is considered uncertain → no assignment.
- If the top score itself is very low (< -5.0), the result is unreliable → no assignment.

This avoids false positives from under-trained models.

---

## Hook Integration

### OnRecordCreate ("documents")

Added AFTER the existing OCR hook. The `OcrJob` struct gains an `OnComplete func(string)` callback. The worker calls it after `updateDocumentStatus()`.

### OnRecordUpdate ("documents")

New hook. Detects `subcategory_id` change by comparing `e.Record.Original().GetString("subcategory_id")` vs `e.Record.GetString("subcategory_id")`. If changed and the new subcategory is NOT the default "General", triggers `Retrain()`.

---

## Model Persistence

- Format: Go `gob` encoding (native Go serialization, zero dependencies)
- File: `pb_data/models/classifier.model`
- Loaded once at startup via `ClassifierManager.Load()`
- Saved after every `Retrain()` call

---

## Edge Cases

| Case | Behavior |
|------|----------|
| Document with no OCR text (unreadable image, error) | `ocr_txt` empty → `ClassifyAndAssign()` skips |
| Model doesn't exist yet (first run) | `Load()` returns nil classifier → no classification until user trains by moving docs |
| Subcategory with 0 documents | `Retrain()` doesn't create a class for it. Existing class is left untouched. |
| OCR server down | OnComplete never called (worker sets status="error") → no classification |
| User moves doc to General (default) | `Retrain` checks: if `newSub == defaultSubcategoryID`, skip (General needs no model) |
| Corrupt .model file | `gob` decode error → `Load()` logs warning, starts with nil classifier, regenerates on next retrain |
| Concurrent classify + retrain | `sync.RWMutex`: multiple concurrent reads (classify), write (retrain) blocks reads briefly |
| Two documents created simultaneously | Each classified independently, RWMutex allows concurrent reads |

---

## Testing

### Unit Tests (`internal/classifier/classifier_test.go`)

- `TestTokenize` — lowercasing, stopword removal, min-length filter
- `TestTopWords` — word frequency across multiple documents
- `TestTrainModel` — train with 3 documents, verify correct words in classifier
- `TestClassify` — train 2 subcategories, classify known text, verify result
- `TestClassifyEmpty` — empty or irrelevant text → no assignment
- `TestModelPersistence` — train, save via gob, load, verify same classification

### Integration Tests

- `TestClassifyAfterOCR` — create doc with known OCR text, verify automatic classification
- `TestRetrainOnUpdate` — create doc, move to different subcategory, verify model updated
