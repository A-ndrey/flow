package parser

import (
	"encoding/json"

	"go.yaml.in/yaml/v3"
)

type JSON struct{}

func (p JSON) Parse(data []byte) (map[string]any, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var result map[string]any

	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

type YAML struct{}

func (p YAML) Parse(data []byte) (map[string]any, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var result map[string]any

	if err := yaml.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

type Nop struct{}

func (p Nop) Parse(_ []byte) (map[string]any, error) {
	return nil, nil
}
