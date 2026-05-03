package classifier

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jbrukh/bayesian"
)

const defaultOtherClass = "_other"

func TrainModel(documents []string, topN int, className string) *bayesian.Classifier {
	if len(documents) == 0 {
		return nil
	}

	words := TopWords(documents, topN)
	if len(words) == 0 {
		return nil
	}

	class := bayesian.Class(className)
	c := bayesian.NewClassifier(class, bayesian.Class(defaultOtherClass))
	c.Learn(words, class)
	return c
}

func SaveModel(path string, c *bayesian.Classifier) error {
	if c == nil {
		return fmt.Errorf("cannot save nil classifier")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create model dir: %w", err)
	}

	if err := c.WriteToFile(path); err != nil {
		return fmt.Errorf("write model: %w", err)
	}
	return nil
}

func LoadModel(path string) (*bayesian.Classifier, error) {
	c, err := bayesian.NewClassifierFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("load model: %w", err)
	}
	return c, nil
}
