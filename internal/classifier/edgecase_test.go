package classifier

import (
	"testing"
)

func TestAutoClassifyShortGenericDoc(t *testing.T) {
	// Test with very short/generic documents that share few words

	doc1 := `Universidad Asignatura Calculo Estudiante Juan Tarea Limites`
	doc2 := `Universidad Asignatura Calculo Estudiante Maria Tarea Derivadas`

	trainingData := map[string][]string{
		"subcat_calculo": {doc1},
	}
	c := rebuildClassifierFromData(trainingData, 1000, 1)

	tokens := Tokenize(doc2)
	t.Logf("Tokens: %v", tokens)
	t.Logf("Token count: %d", len(tokens))

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

func TestAutoClassifyWithStopwordsOnly(t *testing.T) {
	// Test document that only contains stopwords after tokenization

	doc1 := `El la de los las un una para es al lo como mas pero sus le ya o este fue ha era muy son todo si sin sobre entre cuando tambien asi dos hasta desde porque cada otros gran vez`

	trainingData := map[string][]string{
		"subcat_test": {doc1},
	}
	c := rebuildClassifierFromData(trainingData, 1000, 1)

	doc2 := `El la de los las un una para es al lo como mas pero sus le ya o este fue ha era muy son todo si sin sobre entre cuando tambien asi dos hasta desde porque cada otros gran vez`

	tokens := Tokenize(doc2)
	t.Logf("Token count: %d", len(tokens))

	if len(tokens) == 0 {
		t.Log("No tokens - ClassifyAndAssign would return early")
		return
	}

	scores, inx, strict := c.LogScores(tokens)
	t.Logf("Scores: %v", scores)
	t.Logf("Best: %s, strict: %v", c.Classes[inx], strict)
}
