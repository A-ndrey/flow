package flowfile

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

func Find() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current working directory: %w", err)
	}

	patterns := []string{
		"[Ff]lowfile.yaml",
		"[Ff]lowfile.yml",
	}

	for {
		for _, p := range patterns {
			matches, err := filepath.Glob(filepath.Join(currentDir, p))
			if err == nil && len(matches) > 0 {
				return matches[0], nil
			}
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break
		}

		currentDir = parent
	}

	return "", fmt.Errorf("file not found")
}

func Read(path string) (Flowfile, error) {
	file, err := os.Open(path)
	if err != nil {
		return Flowfile{}, fmt.Errorf("can't open file %s: %w", path, err)
	}
	defer file.Close()

	flowfile := Flowfile{}
	if err := yaml.NewDecoder(file).Decode(&flowfile); err != nil {
		return Flowfile{}, fmt.Errorf("can't decode file %s: %w", path, err)
	}

	return flowfile, nil
}
