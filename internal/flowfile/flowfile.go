package flowfile

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Flowfile represents the configuration structure for a flow automation file.
// It defines environment variables, global variables, reusable steps, and executable scripts.
type Flowfile struct {
	// Environment variables that will be merged with OS environment variables.
	// Available in templates as {{._env}} and passed to step commands.
	// Values are used as-is, no template expansion.
	Environment map[string]string `yaml:"env"`

	// Global variables available across all steps in templates as {{._vars}}.
	// Values are used as-is, no template expansion.
	Variables map[string]string `yaml:"vars"`

	// Reusable step definitions that can be referenced and configured in scripts.
	// Each step defines a command with its configuration options.
	Steps map[string]Step `yaml:"steps"`

	// Named scripts composed of step executions that run sequentially.
	// By default, only the last step's output is printed to stdout.
	Scripts map[string]Script `yaml:"scripts"`
}

// Step defines a reusable command execution with configuration options.
type Step struct {
	// List of variable names that must be provided when this step is executed.
	// These variables must be defined in the script's Variables field.
	RequiredVariables []string `yaml:"required_vars"`

	// When true, the step requires stdin input to be configured in the script.
	// If true, the script must provide a Stdin value for this step.
	RequireStdin bool `yaml:"require_stdin"`

	// When true, the step runs in interactive mode and cannot have stdin input.
	// Interactive steps allow user input during execution.
	Interactive bool `yaml:"interactive"`

	// File mappings used by the step. Keys are template variable names that will be
	// available as {{._files.<filename>}} with temporary file paths as values.
	// File paths (values) are used as-is, no template expansion.
	// File contents are read, processed as templates, and saved to temporary files.
	// During validation, file paths are checked for existence as literal strings.
	Files map[string]string `yaml:"files"`

	// Command to execute. Format: "command arg1 arg2 ..."
	// The first word is treated as the command executable (no template expansion).
	// Remaining text are arguments (can contain templates).
	CMD string `yaml:"cmd"`

	// Parser for the command output. Valid values: "json", "yaml", or empty (no parsing).
	// Parsed output is available in templates: {{._steps.last.parsed}} for the last step,
	// or accessed by index for specific steps.
	// Raw output (unparsed) is available as {{._steps.last.raw}}.
	Parser string `yaml:"parser"`

	// When true, the step's output is not saved to memory.
	// By default, output is saved and accessible as {{._steps.last.raw}}.
	// This option reduces memory usage for steps with large output.
	IgnoreOutput bool `yaml:"ignore_output"`
}

// Script represents a sequence of step executions that run in order.
type Script []struct {
	// Input string passed to the step's stdin. Required if step has RequireStdin=true.
	// Must be nil for interactive steps.
	// Can contain templates.
	Stdin *string `yaml:"stdin"`

	// Step-specific variables available in templates as {{.key}}.
	// Required if the step has RequiredVariables defined.
	// Values can contain templates.
	Variables map[string]string `yaml:"vars"`

	// Name of the step to execute, referencing a step defined in Flowfile.Steps.
	Step string `yaml:"step"`

	// Overrides the default output printing behavior for this step.
	// By default, only the last step's output is printed.
	// Set to true to print this step's output, or false to suppress it.
	PrintOutput *bool `yaml:"print_output"`
}

// Template context available during execution:
// - {{._env}}: environment variables (map[string]string)
// - {{._vars}}: global variables from Flowfile (map[string]string)
// - {{._steps}}: step results (map with "last" key and numeric indices)
//   - {{._steps.last.parsed}}: parsed output of last step (map[string]any)
//   - {{._steps.last.raw}}: raw output of last step (string)
//   - Access specific step results by index (e.g., step 0, step 1)
// - {{._files.<filename>}}: temporary file paths for file-based steps
// - {{.key}}: step-specific variables (available in root context)
// Templates support sprig template functions (http://masterminds.github.io/sprig/)

// Validate checks all scripts in the Flowfile for configuration errors.
func (ff *Flowfile) Validate() error {
	for name := range ff.Scripts {
		if err := ff.ValidateScript(name); err != nil {
			return fmt.Errorf("script %q: %w", name, err)
		}
	}

	return nil
}

// ValidateScript checks a specific script for configuration errors.
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

// ListScripts returns a sorted list of all script names defined in the Flowfile.
func (ff *Flowfile) ListScripts() []string {
	scripts := make([]string, 0, len(ff.Scripts))
	for script := range ff.Scripts {
		scripts = append(scripts, script)
	}

	sort.Strings(scripts)

	return scripts
}
