package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/A-ndrey/flow/internal/executor"
	"github.com/A-ndrey/flow/internal/flowfile"
)

func main() {
	var (
		listFlag     bool
		silentFlag   bool
		validateFlag bool
	)
	flag.BoolVar(&listFlag, "list", false, "list available scripts")
	flag.BoolVar(&silentFlag, "silent", false, "disable command echoing")
	flag.BoolVar(&validateFlag, "validate", false, "validate whole flowfile")
	flag.Parse()

	filename, err := flowfile.Find()
	if err != nil {
		failf("Can't find flowfile: %v\n", err)
	}

	ff, err := flowfile.Read(filename)
	if err != nil {
		failf("Failed to read %s: %v\n", filename, err)
	}

	if validateFlag {
		if err := ff.Validate(); err != nil {
			failf("File %q is invalid: %v\n", filename, err)
		}
		fmt.Println("OK")
		return
	}

	if listFlag {
		printScripts(ff)
		return
	}

	scriptName := flag.Arg(0)

	if err := ff.ValidateScript(scriptName); err != nil {
		failf("Script %q is invalid: %v\n", scriptName, err)
	}

	e, err := executor.Prepare(ff, scriptName)
	if err != nil {
		failf("Failed to prepare executor: %v\n", err)
	}

	if err = e.Exec(silentFlag); err != nil {
		failf("Failed to execute script: %v\n", err)
	}
}

func failf(format string, a ...any) {
	fmt.Fprintf(os.Stderr, format, a...)
	os.Exit(1)
}

func printScripts(ff flowfile.Flowfile) {
	for _, script := range ff.ListScripts() {
		fmt.Println(script)
	}
}
