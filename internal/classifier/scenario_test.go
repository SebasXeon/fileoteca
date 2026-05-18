package classifier

import (
	"strings"
	"testing"

	"github.com/jbrukh/bayesian"
)

func TestAutoClassifyScenario(t *testing.T) {
	// Simulate user's scenario:
	// 1. Upload and assign a university assignment
	// 2. Upload a similar assignment
	// 3. Check if it gets classified

	// Document 1: University assignment (already assigned)
	doc1 := `
		Universidad Nacional
		Asignatura: Calculo I
		Estudiante: Juan Perez
		Fecha: 15 de marzo de 2024
		
		Tarea 1: Limites y Continuidad
		
		Resolver los siguientes problemas:
		1. Calcular el limite de la funcion f(x) = x^2 cuando x tiende a 2
		2. Determinar si la funcion es continua en el punto x = 3
		3. Demostrar que el limite existe usando la definicion epsilon-delta
		
		Desarrollo:
		Para el problema 1, aplicamos las propiedades de los limites...
		
		Conclusion: Los limites son fundamentales para el analisis matematico.
	`

	// Document 2: Similar university assignment (new upload)
	doc2 := `
		Universidad Nacional
		Asignatura: Calculo I
		Estudiante: Maria Garcia
		Fecha: 22 de marzo de 2024
		
		Tarea 2: Derivadas
		
		Resolver los siguientes problemas:
		1. Calcular la derivada de f(x) = x^3 + 2x
		2. Encontrar la recta tangente en el punto x = 1
		3. Aplicar la regla de la cadena
		
		Desarrollo:
		Para el problema 1, usamos la definicion de derivada...
		
		Conclusion: Las derivadas permiten analizar el comportamiento de las funciones.
	`

	// Step 1: Train with doc1
	trainingData := map[string][]string{
		"subcat_calculo": {doc1},
	}
	c := rebuildClassifierFromData(trainingData, 1000, 1)
	if c == nil {
		t.Fatal("failed to build classifier")
	}

	t.Logf("Classifier has %d classes", len(c.Classes))
	for _, cls := range c.Classes {
		words := c.WordsByClass(cls)
		t.Logf("Class %s has %d words", cls, len(words))
	}

	// Step 2: Classify doc2
	tokens := Tokenize(doc2)
	t.Logf("Document 2 has %d tokens", len(tokens))

	scores, inx, strict := c.LogScores(tokens)
	t.Logf("Scores: %v", scores)
	t.Logf("Best index: %d, strict: %v", inx, strict)
	if inx >= 0 && inx < len(c.Classes) {
		t.Logf("Classified as: %s", c.Classes[inx])
	}

	// Step 3: Check confidence threshold
	bestScore := scores[inx]
	secondBest := -9999.0
	bestIdx := inx
	for i, s := range scores {
		if i != bestIdx && s > secondBest {
			secondBest = s
		}
	}
	diff := bestScore - secondBest
	t.Logf("Score difference: %.4f", diff)

	if diff < 1.0 {
		t.Logf("WARNING: Classification rejected by threshold (diff=%.4f < 1.0)", diff)
	}

	// Step 4: Simulate retrain with doc1 and doc2 both assigned
	trainingData2 := map[string][]string{
		"subcat_calculo": {doc1, doc2},
	}
	c2 := rebuildClassifierFromData(trainingData2, 1000, 1)

	// Step 5: Classify doc2 again with 2 training docs
	scores2, inx2, strict2 := c2.LogScores(tokens)
	t.Logf("After retrain with 2 docs - Scores: %v", scores2)
	t.Logf("After retrain - Best index: %d, strict: %v", inx2, strict2)

	bestScore2 := scores2[inx2]
	secondBest2 := -9999.0
	bestIdx2 := inx2
	for i, s := range scores2 {
		if i != bestIdx2 && s > secondBest2 {
			secondBest2 = s
		}
	}
	diff2 := bestScore2 - secondBest2
	t.Logf("After retrain - Score difference: %.4f", diff2)

	if diff2 < 1.0 {
		t.Logf("WARNING: Classification rejected by threshold after retrain (diff=%.4f < 1.0)", diff2)
	}
}

func TestWordsByClassDegradation(t *testing.T) {
	// Test that WordsByClass loses frequency information

	trainingData := map[string][]string{
		"subcat_a": {"factura pago electricidad consumo enero", "factura pago electricidad consumo febrero"},
		"subcat_b": {"contrato arrendamiento vivienda clausulas", "contrato alquiler apartamento vivienda"},
	}
	c := rebuildClassifierFromData(trainingData, 1000, 1)

	// Check original word frequencies
	wordsA := c.WordsByClass(bayesian.Class("subcat_a"))
	t.Logf("Original words in subcat_a: %v", wordsA)

	// Simulate retrain of subcat_b using WordsByClass for subcat_a
	wm := c.WordsByClass(bayesian.Class("subcat_a"))
	existingWords := make([]string, 0, len(wm))
	for w := range wm {
		existingWords = append(existingWords, w)
	}

	trainingData2 := map[string][]string{
		"subcat_a": {strings.Join(existingWords, " ")},
		"subcat_b": {"contrato arrendamiento vivienda clausulas", "contrato alquiler apartamento vivienda", "contrato renta casa"},
	}
	c2 := rebuildClassifierFromData(trainingData2, 1000, 1)

	wordsA2 := c2.WordsByClass(bayesian.Class("subcat_a"))
	t.Logf("After retrain words in subcat_a: %v", wordsA2)

	// Compare probabilities for a key word
	if orig, ok := wordsA["electricidad"]; ok {
		if new, ok2 := wordsA2["electricidad"]; ok2 {
			t.Logf("electricidad: original=%.4f, after_retrain=%.4f", orig, new)
			if new != orig {
				t.Logf("BUG: Word probability changed after retrain!")
			}
		}
	}
}
