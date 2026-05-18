package classifier

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jbrukh/bayesian"
)

func TestLargeVocabularyVsOther(t *testing.T) {
	// Test with a subcategory that has many unique words
	// This can cause _other to win because of Laplace smoothing

	// Create a document with 200 unique words
	var words1 []string
	for i := 0; i < 200; i++ {
		words1 = append(words1, fmt.Sprintf("word%d", i))
	}
	doc1 := strings.Join(words1, " ")

	trainingData := map[string][]string{
		"subcat_large": {doc1},
	}
	c := rebuildClassifierFromData(trainingData, 1000, 1)

	t.Logf("subcat_large words: %d", len(c.WordsByClass(bayesian.Class("subcat_large"))))
	t.Logf("_other words: %d", len(c.WordsByClass(bayesian.Class("_other"))))

	// Test document with 50 words from subcat_large and 10 new words
	var testWords []string
	for i := 0; i < 50; i++ {
		testWords = append(testWords, fmt.Sprintf("word%d", i))
	}
	for i := 200; i < 210; i++ {
		testWords = append(testWords, fmt.Sprintf("word%d", i))
	}
	doc2 := strings.Join(testWords, " ")

	tokens := Tokenize(doc2)
	t.Logf("Test doc tokens: %d", len(tokens))

	scores, inx, strict := c.LogScores(tokens)
	t.Logf("Scores: %v", scores)
	t.Logf("Best: %s, strict: %v", c.Classes[inx], strict)

	bestScore := scores[inx]
	secondBest := -9999.0
	for i, s := range scores {
		if i != inx && s > secondBest {
			secondBest = s
		}
	}
	diff := bestScore - secondBest
	t.Logf("Diff: %.4f", diff)
}

func TestRealisticAssignmentLargeVocab(t *testing.T) {
	// Simulate a university assignment with many unique words
	// The OCR text might have 100+ unique words

	doc1 := `
		Universidad Nacional de Ingenieria
		Facultad de Ciencias
		Departamento de Matematicas
		
		Asignatura: Calculo Diferencial e Integral
		Codigo: MAT101
		Profesor: Dr. Carlos Rodriguez
		Periodo: Primavera 2024
		
		Estudiante: Juan Pablo Martinez
		Matricula: 202401234
		Seccion: A
		
		TAREA NUMERO 3
		
		Instrucciones: Resolver cada problema mostrando todo el procedimiento.
		Valor total: 100 puntos
		Fecha de entrega: 15 de marzo de 2024
		
		Problema 1 (20 puntos):
		Calcular la derivada de las siguientes funciones:
		a) f(x) = x^3 - 2x^2 + 5x - 7
		b) g(x) = sen(x) * cos(x)
		c) h(x) = ln(x^2 + 1)
		
		Problema 2 (20 puntos):
		Encontrar los puntos criticos de la funcion f(x) = x^4 - 4x^3 + 10
		y clasificarlos como maximos, minimos o puntos de silla.
		
		Problema 3 (20 puntos):
		Un agricultor tiene 200 metros de cerca para construir un corral
		rectangular. Determinar las dimensiones que maximizan el area.
		
		Problema 4 (20 puntos):
		Calcular las siguientes integrales indefinidas:
		a) integral de x^2 dx
		b) integral de e^x dx
		c) integral de 1/x dx
		
		Problema 5 (20 puntos):
		Demostrar que si f es continua en [a,b] y derivable en (a,b),
		entonces existe c en (a,b) tal que f'(c) = (f(b)-f(a))/(b-a).
		
		Bibliografia:
		Stewart, J. (2015). Calculo de una variable. Cengage Learning.
		Larson, R. (2017). Calculo. McGraw-Hill Education.
		
		Nota: Esta tarea es individual. Cualquier copia sera sancionada.
	`

	doc2 := `
		Universidad Nacional de Ingenieria
		Facultad de Ciencias
		Departamento de Matematicas
		
		Asignatura: Calculo Diferencial e Integral
		Codigo: MAT101
		Profesor: Dr. Carlos Rodriguez
		Periodo: Primavera 2024
		
		Estudiante: Maria Fernanda Lopez
		Matricula: 202405678
		Seccion: A
		
		TAREA NUMERO 4
		
		Instrucciones: Resolver cada problema mostrando todo el procedimiento.
		Valor total: 100 puntos
		Fecha de entrega: 22 de marzo de 2024
		
		Problema 1 (25 puntos):
		Calcular los limites siguientes usando la regla de LHopital:
		a) limite cuando x tiende a 0 de sen(x)/x
		b) limite cuando x tiende a infinito de ln(x)/x
		
		Problema 2 (25 puntos):
		Dada la funcion f(x,y) = x^2 + y^2 - 2x - 4y + 5,
		encontrar sus puntos criticos y clasificarlos.
		
		Problema 3 (25 puntos):
		Calcular la integral definida de 0 a 1 de x^3 dx.
		Interpretar el resultado geometricamente.
		
		Problema 4 (25 puntos):
		Demostrar la convergencia de la serie suma de 1/n^2.
		
		Bibliografia:
		Stewart, J. (2015). Calculo de una variable. Cengage Learning.
		Larson, R. (2017). Calculo. McGraw-Hill Education.
		
		Nota: Esta tarea es individual. Cualquier copia sera sancionada.
	`

	trainingData := map[string][]string{
		"subcat_calculo": {doc1},
	}
	c := rebuildClassifierFromData(trainingData, 1000, 1)

	tokens1 := Tokenize(doc1)
	tokens2 := Tokenize(doc2)
	t.Logf("Doc1 tokens: %d, Doc2 tokens: %d", len(tokens1), len(tokens2))

	// Count shared words
	set1 := make(map[string]bool)
	for _, t := range tokens1 {
		set1[t] = true
	}
	shared := 0
	for _, t := range tokens2 {
		if set1[t] {
			shared++
		}
	}
	t.Logf("Shared tokens: %d", shared)

	scores, inx, strict := c.LogScores(tokens2)
	t.Logf("Scores: %v", scores)
	t.Logf("Best: %s, strict: %v", c.Classes[inx], strict)

	bestScore := scores[inx]
	secondBest := -9999.0
	for i, s := range scores {
		if i != inx && s > secondBest {
			secondBest = s
		}
	}
	diff := bestScore - secondBest
	t.Logf("Diff: %.4f", diff)
}
