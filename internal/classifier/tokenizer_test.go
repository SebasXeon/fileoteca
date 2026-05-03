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
	sort.Strings(top)
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
