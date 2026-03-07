package executor

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/A-ndrey/flow/internal/flowfile"
)

type Parser interface {
	Parse([]byte) (map[string]any, error)
}

type Executor struct {
	data       map[string]any
	steps      []*step
	scriptName string
}

func Prepare(ff flowfile.Flowfile, script string) (*Executor, error) {
	e := Executor{
		data: map[string]any{
			"_env":   mergeEnv(ff.Environment),
			"_vars":  ff.Variables,
			"_steps": make(map[string]stepResult),
		},
		scriptName: script,
	}

	ffScript, ok := ff.Scripts[script]
	if !ok {
		return nil, fmt.Errorf("script %q not found", script)
	}

	e.steps = make([]*step, 0, len(ffScript))
	for i, scriptStep := range ffScript {
		ffStep, ok := ff.Steps[scriptStep.Step]
		if !ok {
			return nil, fmt.Errorf("step %q not found", scriptStep.Step)
		}

		printOutput := len(ffScript)-1 == i
		if scriptStep.PrintOutput != nil {
			printOutput = *scriptStep.PrintOutput
		}

		st, err := newStep(scriptStep.Step, ffStep, scriptStep.Variables, printOutput, scriptStep.Stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to prepare step %q: %w", scriptStep.Step, err)
		}

		e.steps = append(e.steps, st)
	}

	return &e, nil
}

func (e *Executor) Exec(silent bool) error {
	if !silent {
		fmt.Fprintf(os.Stderr, "\033[1;33mscript %q\033[0m\n", e.scriptName)
	}

	stepsResults := e.data["_steps"].(map[string]stepResult)

	for i, st := range e.steps {
		if err := st.execTemplates(e.data); err != nil {
			return fmt.Errorf("step %d failed execution template: %w", i, err)
		}

		result, err := st.run(silent)
		if err != nil {
			return fmt.Errorf("step %d failed: %w", i, err)
		}

		stepsResults[strconv.Itoa(i)] = result
		stepsResults["last"] = result
	}

	return nil
}

func mergeEnv(ffEnv map[string]string) map[string]string {
	for k, v := range ffEnv {
		os.Setenv(k, v)
	}

	envList := os.Environ()

	envMap := make(map[string]string, len(envList))
	for _, env := range envList {
		envKV := strings.SplitN(env, "=", 2)
		envMap[envKV[0]] = envKV[1]
	}

	return envMap
}
