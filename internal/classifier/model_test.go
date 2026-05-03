package classifier

import (
	"os"
	"path/filepath"
	"testing"
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
	if len(c.Classes) != 2 {
		t.Fatalf("expected 2 classes, got %d", len(c.Classes))
	}
	var found bool
	for _, cls := range c.Classes {
		if string(cls) == "test_subcat" {
			found = true
			words := c.WordsByClass(cls)
			if len(words) == 0 {
				t.Error("class has no words")
			}
		}
	}
	if !found {
		t.Error("class test_subcat not found in classifier")
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

	if len(loaded.Classes) != 2 {
		t.Fatalf("loaded classifier has %d classes, want 2", len(loaded.Classes))
	}
	var found bool
	for _, cls := range loaded.Classes {
		if string(cls) == "subcat_1" {
			found = true
			words := loaded.WordsByClass(cls)
			if len(words) == 0 {
				t.Error("loaded class has no words")
			}
		}
	}
	if !found {
		t.Error("class subcat_1 not found in loaded classifier")
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

	scores, inx, strict := c.LogScores(Tokenize("factura pago electricidad abril"))
	if strict {
		t.Log("classification in strict mode (may be ok for small training sets)")
	}
	if inx < 0 || inx >= len(c.Classes) {
		t.Fatalf("invalid class index: %d", inx)
	}
	if string(c.Classes[inx]) != "facturas_electricidad" {
		t.Errorf("classified as %s, want facturas_electricidad (scores: %v)", c.Classes[inx], scores)
	}
}
