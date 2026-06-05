package eval

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"gopkg.in/yaml.v3"
)

type corpusFile struct {
	Vectors []corpusVector `yaml:"vectors"`
}

type corpusVector struct {
	ID          string         `yaml:"id"`
	Category    string         `yaml:"category"`
	Description string         `yaml:"description"`
	Input       string         `yaml:"input"`
	Variables   map[string]any `yaml:"variables"`
	Expected    map[string]any `yaml:"expected"`
}

func TestConformance(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "design", "gert", "conformance", "tv-*.yaml"))
	if err != nil {
		t.Fatalf("discover conformance corpus: %v", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatal("discover conformance corpus: no tv-*.yaml files found")
	}

	vectorsByFile := make(map[string][]corpusVector, len(files))
	total := 0
	for _, file := range files {
		vectors, err := loadCorpusVectors(file)
		if err != nil {
			t.Fatalf("load %s: %v", file, err)
		}
		vectorsByFile[file] = vectors
		total += len(vectors)
	}

	fmt.Printf("GERT conformance: %d files, %d vectors discovered, 0 passing, %d pending\n", len(files), total, total)

	for _, file := range files {
		file := file
		vectors := vectorsByFile[file]
		t.Run(filepath.Base(file), func(t *testing.T) {
			for _, vector := range vectors {
				vector := vector
				t.Run(vector.ID, func(t *testing.T) {
					t.Skip("runtime implementation pending")
				})
			}
		})
	}
}

func loadCorpusVectors(path string) ([]corpusVector, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var corpus corpusFile
	if err := yaml.Unmarshal(data, &corpus); err != nil {
		return nil, err
	}
	if len(corpus.Vectors) == 0 {
		return nil, fmt.Errorf("no vectors")
	}
	return corpus.Vectors, nil
}
