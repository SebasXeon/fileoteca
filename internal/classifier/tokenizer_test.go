package classifier

import (
	"reflect"
	"testing"
)

func TestTokenize(t *testing.T) {
	tokens := Tokenize("Hola Mundo! Esto es una prueba de tokenización 123.")
	expected := []string{"hola", "mundo", "esto", "prueba", "tokenización"}
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
	expected := []string{"casa", "gato", "ladra"}
	if !reflect.DeepEqual(top, expected) {
		t.Errorf("TopWords = %v, want %v", top, expected)
	}
}

func TestTopWordsEmptyDocs(t *testing.T) {
	top := TopWords([]string{}, 10)
	if len(top) != 0 {
		t.Errorf("expected empty result, got %v", top)
	}
}

func TestTopWordsNTooLarge(t *testing.T) {
	docs := []string{"gato perro gato"}
	top := TopWords(docs, 10)
	expected := []string{"gato", "perro"}
	if !reflect.DeepEqual(top, expected) {
		t.Errorf("TopWords = %v, want %v", top, expected)
	}
}

func TestTopWordsNegativeN(t *testing.T) {
	top := TopWords([]string{"gato perro"}, -1)
	if top != nil {
		t.Errorf("expected nil for negative n, got %v", top)
	}
}
