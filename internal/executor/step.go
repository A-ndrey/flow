package executor

import (
	"bytes"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/A-ndrey/flow/internal/flowfile"
	"github.com/A-ndrey/flow/internal/parser"
	sprig "github.com/go-task/slim-sprig/v3"
)

type step struct {
	printOutput  bool
	ignoreOutput bool
	interactive  bool
	stepName     string
	command      string
	stdin        *string
	parser       Parser
	variables    map[string]string
	files        map[string]string
}

func newStep(stepName string, ffs flowfile.Step, vars map[string]string, printOutput bool, stdin *string) (*step, error) {
	st := step{
		printOutput:  printOutput,
		interactive:  ffs.Interactive,
		ignoreOutput: ffs.IgnoreOutput,
		stepName:     stepName,
		command:      strings.TrimSpace(ffs.CMD),
		variables:    vars,
		stdin:        stdin,
		files:        ffs.Files,
	}

	switch strings.ToLower(strings.TrimSpace(ffs.Parser)) {
	case "json":
		st.parser = parser.JSON{}
	case "yaml":
		st.parser = parser.YAML{}
	default:
		st.parser = parser.Nop{}
	}

	return &st, nil
}

func (s *step) execTemplates(data map[string]any) error {
	for k, v := range s.variables {
		res, err := tmplApply(v, data)
		if err != nil {
			return fmt.Errorf("failed to apply template for variable %s: %w", k, err)
		}
		s.variables[k] = res
	}

	data = s.mergeVarsAndData(data)

	files := make(map[string]string)
	for filename, path := range s.files {
		f, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %q: %w", filename, err)
		}
		defer f.Close()

		filecontent, err := io.ReadAll(f)
		if err != nil {
			return fmt.Errorf("failed to read file %q: %w", filename, err)
		}

		res, err := tmplApply(string(filecontent), data)
		if err != nil {
			return fmt.Errorf("failed to apply template for file %q: %w", filename, err)
		}

		tmp, err := os.CreateTemp("", filename)
		if err != nil {
			return fmt.Errorf("failed to create temp file for %q: %w", filename, err)
		}
		defer tmp.Close()

		if _, err := tmp.WriteString(res); err != nil {
			return fmt.Errorf("failed to write into temp file for %q: %w", filename, err)
		}

		files[filename] = tmp.Name()
	}

	data["_files"] = files

	if s.stdin != nil {
		stdin, err := tmplApply(*s.stdin, data)
		if err != nil {
			return fmt.Errorf("failed to apply template for stdin: %w", err)
		}

		s.stdin = &stdin
	}

	cmd, err := tmplApply(s.command, data)
	if err != nil {
		return fmt.Errorf("failed to apply template for command: %w", err)
	}
	s.command = cmd

	return nil
}

func (s *step) run(silent bool) (stepResult, error) {
	if !silent {
		fmt.Fprintf(os.Stderr, "\033[1;33mstep %q: %s\033[0m\n", s.stepName, s.command)
	}

	splittedCMD := strings.Split(s.command, " ")

	var cmd *exec.Cmd
	if len(splittedCMD) > 1 {
		cmd = exec.Command(splittedCMD[0], splittedCMD[1:]...)
	} else {
		cmd = exec.Command(splittedCMD[0])
	}

	var sb strings.Builder

	var writers []io.Writer
	if s.printOutput {
		writers = append(writers, os.Stdout)
	}
	if !s.ignoreOutput {
		writers = append(writers, &sb)
	}

	cmd.Stdout = io.MultiWriter(writers...)
	cmd.Stderr = os.Stderr

	if s.interactive {
		cmd.Stdin = os.Stdin
	} else if s.stdin != nil {
		cmd.Stdin = bytes.NewBufferString(*s.stdin)
	}

	if err := cmd.Run(); err != nil {
		return stepResult{}, err
	}

	raw := sb.String()

	parsed, err := s.parser.Parse([]byte(strings.Trim(raw, `"`)))
	if err != nil {
		return stepResult{}, err
	}

	result := newStepResult()
	result.setRaw(raw)
	result.setParsed(parsed)

	return result, nil
}

func (s *step) mergeVarsAndData(data map[string]any) map[string]any {
	res := make(map[string]any, len(data)+len(s.variables))

	maps.Copy(res, data)

	for k, v := range s.variables {
		res[k] = v
	}

	return res
}

type stepResult map[string]any

func newStepResult() stepResult {
	return make(stepResult)
}

func (r stepResult) setRaw(raw string) {
	r["raw"] = raw
}

func (r stepResult) raw() string {
	return r["raw"].(string)
}

func (r stepResult) setParsed(parsed map[string]any) {
	r["parsed"] = parsed
}

func (r stepResult) parsed() map[string]any {
	return r["parsed"].(map[string]any)
}

func tmplApply(text string, data map[string]any) (string, error) {
	tmpl, err := template.New("apply").Funcs(sprig.FuncMap()).Parse(text)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	tmpl = tmpl.Option("missingkey=error")

	var sb strings.Builder
	if err := tmpl.Execute(&sb, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return sb.String(), nil
}
