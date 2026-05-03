package classifier

import (
	"regexp"
	"sort"
	"strings"
)

var separatorRE = regexp.MustCompile(`[^\p{L}\d]+`)

var spanishStopwords = map[string]bool{
	"de": true, "la": true, "que": true, "el": true, "en": true,
	"y": true, "a": true, "los": true, "se": true, "del": true,
	"las": true, "un": true, "por": true, "con": true, "no": true,
	"una": true, "su": true, "para": true, "es": true, "al": true,
	"lo": true, "como": true, "mas": true, "pero": true, "sus": true,
	"le": true, "ya": true, "o": true, "este": true, "fue": true,
	"ha": true, "era": true, "muy": true, "son": true, "todo": true,
	"si": true, "sin": true, "sobre": true, "entre": true, "cuando": true,
	"también": true, "así": true, "dos": true, "hasta": true, "desde": true,
	"porque": true, "cada": true, "otros": true, "gran": true, "vez": true,
	"año": true, "parte": true, "me": true, "mi": true,
	"tu": true, "te": true, "nos": true, "os": true, "les": true,
	"e": true, "ni": true, "tras": true, "hacia": true,
	"durante": true, "contra": true, "bajo": true,
}

func Tokenize(text string) []string {
	text = strings.ToLower(text)
	parts := separatorRE.Split(text, -1)
	tokens := make([]string, 0, len(parts))
	for _, p := range parts {
		if len(p) < 3 {
			continue
		}
		if isNumeric(p) {
			continue
		}
		if spanishStopwords[p] {
			continue
		}
		tokens = append(tokens, p)
	}
	return tokens
}

type wordCount struct {
	word  string
	count int
}

func TopWords(docs []string, n int) []string {
	if n <= 0 {
		return nil
	}
	freq := make(map[string]int)
	for _, doc := range docs {
		tokens := Tokenize(doc)
		seen := make(map[string]bool)
		for _, t := range tokens {
			if !seen[t] {
				freq[t]++
				seen[t] = true
			}
		}
	}

	wc := make([]wordCount, 0, len(freq))
	for w, c := range freq {
		wc = append(wc, wordCount{w, c})
	}

	sort.Slice(wc, func(i, j int) bool {
		if wc[i].count == wc[j].count {
			return wc[i].word < wc[j].word
		}
		return wc[i].count > wc[j].count
	})

	if n > len(wc) {
		n = len(wc)
	}

	result := make([]string, n)
	for i := 0; i < n; i++ {
		result[i] = wc[i].word
	}
	return result
}

func isNumeric(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
