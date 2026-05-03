package classifier

import (
	"os"
	"path/filepath"
	"testing"
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
	mgr.ClassifyAndAssign("doc123", "some ocr text")
}

func TestRebuildClassifier(t *testing.T) {
	trainingData := map[string][]string{
		"subcat_a": {"factura pago enero electricidad consumo", "factura febrero pago luz"},
		"subcat_b": {"contrato arrendamiento vivienda clausulas", "contrato alquiler apartamento"},
	}

	c := rebuildClassifierFromData(trainingData, 50)
	if c == nil {
		t.Fatal("rebuildClassifierFromData returned nil")
	}
	if len(c.Classes) < 2 {
		t.Fatalf("expected at least 2 classes, got %d", len(c.Classes))
	}

	path := filepath.Join(t.TempDir(), "classifier.model")
	if err := SaveModel(path, c); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadModel(path)
	if err != nil {
		t.Fatal(err)
	}

	_, inx, _ := loaded.LogScores(Tokenize("factura pago electricidad marzo"))
	if inx >= 0 && inx < len(loaded.Classes) {
		t.Logf("classified as %s", loaded.Classes[inx])
	}
}

func TestClassifierManagerRetrain(t *testing.T) {
	dir := t.TempDir()
	mgr := &ClassifierManager{modelsDir: dir}

	trainingData := map[string][]string{
		"subcat_facturas": {"factura pago enero electricidad consumo", "factura febrero luz"},
	}

	c := rebuildClassifierFromData(trainingData, 50)
	path := filepath.Join(dir, "classifier.model")
	if err := SaveModel(path, c); err != nil {
		t.Fatal(err)
	}
	mgr.classifier = c

	if _, err := os.Stat(path); err != nil {
		t.Fatal("model file should exist")
	}
}
