package classifier

import (
	"fmt"
	"strings"
	"testing"

	"github.com/jbrukh/bayesian"
)

func TestProductionParamsPreventOtherWin(t *testing.T) {
	// Simulate the exact production scenario that was failing:
	// 200 unique words in one training document, classify a similar document.
	var words1 []string
	for i := 0; i < 200; i++ {
		words1 = append(words1, fmt.Sprintf("word%d", i))
	}
	doc1 := strings.Join(words1, " ")

	trainingData := map[string][]string{
		"subcat_large": {doc1},
	}
	// Use production params: maxWordsPerClass=100, minDocFreq=2
	c := rebuildClassifierFromData(trainingData, maxWordsPerClass, 2)

	if c == nil {
		t.Fatal("classifier is nil")
	}

	wordsA := c.WordsByClass(bayesian.Class("subcat_large"))
	wordsOther := c.WordsByClass(bayesian.Class("_other"))
	t.Logf("subcat_large words: %d, _other words: %d", len(wordsA), len(wordsOther))

	// Test document with 50 known words + 10 unknown
	var testWords []string
	for i := 0; i < 50; i++ {
		testWords = append(testWords, fmt.Sprintf("word%d", i))
	}
	for i := 200; i < 210; i++ {
		testWords = append(testWords, fmt.Sprintf("word%d", i))
	}
	doc2 := strings.Join(testWords, " ")

	tokens := Tokenize(doc2)
	scores, inx, strict := c.LogScores(tokens)
	t.Logf("Scores: %v", scores)
	t.Logf("Best: %s, strict: %v", c.Classes[inx], strict)

	best := c.Classes[inx]
	if string(best) == "_other" {
		t.Fatalf("BUG: document classified as _other with production params")
	}

	bestScore := scores[inx]
	secondBest := -9999.0
	for i, s := range scores {
		if i != inx && s > secondBest {
			secondBest = s
		}
	}
	diff := bestScore - secondBest
	t.Logf("Diff: %.4f", diff)
	if diff < confidenceThreshold {
		t.Fatalf("BUG: confidence diff %.4f below threshold %.2f", diff, confidenceThreshold)
	}
}

func TestProductionParamsWithTwoDocs(t *testing.T) {
	// Same as above but with 2 training docs so minDocFreq actually filters.
	var words1 []string
	for i := 0; i < 200; i++ {
		words1 = append(words1, fmt.Sprintf("word%d", i))
	}
	doc1 := strings.Join(words1, " ")

	var words2 []string
	for i := 0; i < 100; i++ {
		words2 = append(words2, fmt.Sprintf("word%d", i))
	}
	for i := 200; i < 250; i++ {
		words2 = append(words2, fmt.Sprintf("word%d", i))
	}
	doc2 := strings.Join(words2, " ")

	trainingData := map[string][]string{
		"subcat_large": {doc1, doc2},
	}
	c := rebuildClassifierFromData(trainingData, maxWordsPerClass, 2)

	wordsA := c.WordsByClass(bayesian.Class("subcat_large"))
	wordsOther := c.WordsByClass(bayesian.Class("_other"))
	t.Logf("subcat_large words: %d, _other words: %d", len(wordsA), len(wordsOther))

	// Classify a test doc similar to training
	var testWords []string
	for i := 0; i < 80; i++ {
		testWords = append(testWords, fmt.Sprintf("word%d", i))
	}
	docTest := strings.Join(testWords, " ")

	tokens := Tokenize(docTest)
	_, inx, strict := c.LogScores(tokens)
	t.Logf("Best: %s, strict: %v", c.Classes[inx], strict)

	if string(c.Classes[inx]) == "_other" {
		t.Fatalf("BUG: classified as _other")
	}
}
