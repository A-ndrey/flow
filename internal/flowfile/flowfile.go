package flowfile

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

type Flowfile struct {
	Environment map[string]string `yaml:"env"`
	Variables   map[string]string `yaml:"vars"`
	Steps       map[string]Step   `yaml:"steps"`
	Scripts     map[string]Script `yaml:"scripts"`
}

type Step struct {
	RequiredVariables []string          `yaml:"required_vars"`
	RequireStdin      bool              `yaml:"require_stdin"`
	Interactive       bool              `yaml:"interactive"`
	Files             map[string]string `yaml:"files"`
	CMD               string            `yaml:"cmd"`
	Parser            string            `yaml:"parser"`
	IgnoreOutput      bool              `yaml:"ignore_output"`
}

type Script []struct {
	Stdin       *string           `yaml:"stdin"`
	Variables   map[string]string `yaml:"vars"`
	Step        string            `yaml:"step"`
	PrintOutput *bool             `yaml:"print_output"`
}

func (ff *Flowfile) Validate() error {
	for name := range ff.Scripts {
		if err := ff.ValidateScript(name); err != nil {
			return fmt.Errorf("script %q: %w", name, err)
		}
	}

	return nil
}

func (ff *Flowfile) ValidateScript(name string) error {
	script, ok := ff.Scripts[name]
	if !ok {
		return fmt.Errorf("no such script")
	}

	for _, chain := range script {
		ffStep, ok := ff.Steps[chain.Step]
		if !ok {
			return fmt.Errorf("unknown step %q", chain.Step)
		}

		if strings.TrimSpace(ffStep.CMD) == "" {
			return fmt.Errorf("step %q: command not specified", chain.Step)
		}

		if ffStep.Interactive && chain.Stdin != nil {
			return fmt.Errorf("step %q: interactive step cannot have stdin", chain.Step)
		}

		if ffStep.RequireStdin && chain.Stdin == nil {
			return fmt.Errorf("step %q requires stdin", chain.Step)
		}

		var missedVariables []string
		for _, rv := range ffStep.RequiredVariables {
			if _, ok := chain.Variables[rv]; !ok {
				missedVariables = append(missedVariables, rv)
			}
		}

		if len(missedVariables) > 0 {
			return fmt.Errorf("step %q requires variables: %v", chain.Step, missedVariables)
		}

		for fileName, filePath := range ffStep.Files {
			if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("step %q: filename %q: file %q not found", chain.Step, fileName, filePath)
			}
		}
	}

	return nil
}

func (ff *Flowfile) ListScripts() []string {
	scripts := make([]string, 0, len(ff.Scripts))
	for script := range ff.Scripts {
		scripts = append(scripts, script)
	}

	sort.Strings(scripts)

	return scripts
}
