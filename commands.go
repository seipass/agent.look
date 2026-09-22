package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func recordCommand(args []string) error {
	fs := flag.NewFlagSet("record", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	agent := fs.String("agent", "", "agent name")
	model := fs.String("model", "", "model name")
	promptText := fs.String("prompt", "", "prompt text to embed")
	promptFile := fs.String("prompt-file", "", "prompt file to embed")
	output := fs.String("o", defaultLock, "output lock file")
	var permissions stringList
	var tests stringList
	fs.Var(&permissions, "permission", "declared permission; repeatable")
	fs.Var(&tests, "test", "verification command; repeatable")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *promptText != "" && *promptFile != "" {
		return errors.New("use either --prompt or --prompt-file, not both")
	}

	prompt, err := capturePrompt(*promptText, *promptFile)
	if err != nil {
		return err
	}

	command := fs.Args()
	lock := Lock{
		SchemaVersion: schemaVersion,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
		Agent:         AgentRun{Name: *agent, Model: *model, Command: command},
		Prompt:        prompt,
		Instructions:  captureInstructions(),
		ExternalTools: captureExternalTools(),
		Permissions:   sortedUnique(permissions),
		Repository:    captureRepository(),
		Environment:   captureEnvironment(),
	}

	if len(command) > 0 {
		start := time.Now()
		code, err := runExactCommand(command)
		lock.Agent.DurationMS = time.Since(start).Milliseconds()
		lock.Agent.ExitCode = &code
		if err != nil {
			fmt.Fprintf(os.Stderr, "agent command exited with code %d; lock file will still be written\n", code)
		}
	}

	for _, test := range tests {
		lock.Tests = append(lock.Tests, executeTest(test))
	}

	if err := writeLock(*output, lock); err != nil {
		return err
	}
	fmt.Printf("Recorded %s\n", *output)
	fmt.Printf("  instructions: %d\n", len(lock.Instructions))
	fmt.Printf("  external tools: %d\n", len(lock.ExternalTools))
	fmt.Printf("  tests: %d\n", len(lock.Tests))
	if lock.Repository != nil && lock.Repository.Dirty {
		fmt.Println("  warning: repository had uncommitted changes; exact replay requires the same working tree")
	}
	return nil
}

func inspectCommand(args []string) error {
	path, err := singleOptionalPath(args, defaultLock)
	if err != nil {
		return err
	}
	lock, err := readLock(path)
	if err != nil {
		return err
	}
	fmt.Printf("agent.look schema %d\n", lock.SchemaVersion)
	fmt.Printf("created: %s\n", lock.CreatedAt)
	fmt.Printf("agent: %s\n", valueOr(lock.Agent.Name, "(unspecified)"))
	fmt.Printf("model: %s\n", valueOr(lock.Agent.Model, "(unspecified)"))
	if len(lock.Agent.Command) > 0 {
		fmt.Printf("command: %s\n", shellDisplay(lock.Agent.Command))
	}
	if lock.Agent.ExitCode != nil {
		fmt.Printf("exit: %d (%d ms)\n", *lock.Agent.ExitCode, lock.Agent.DurationMS)
	}
	if lock.Repository != nil {
		fmt.Printf("repo: %s @ %s\n", shortCommit(lock.Repository.Commit), valueOr(lock.Repository.Branch, "detached"))
		fmt.Printf("dirty: %t\n", lock.Repository.Dirty)
	}
	if lock.Prompt != nil {
		fmt.Printf("prompt: %s (%s)\n", valueOr(lock.Prompt.Path, "embedded"), shortHash(lock.Prompt.SHA256))
	}
	fmt.Printf("instructions: %d\n", len(lock.Instructions))
	fmt.Printf("external tools: %d\n", len(lock.ExternalTools))
	fmt.Printf("permissions: %s\n", valueOr(strings.Join(lock.Permissions, ", "), "(none declared)"))
	fmt.Printf("tests: %d\n", len(lock.Tests))
	return nil
}

func verifyCommand(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	jsonOut := fs.Bool("json", false, "print machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	path, err := singleOptionalPath(fs.Args(), defaultLock)
	if err != nil {
		return err
	}
	lock, err := readLock(path)
	if err != nil {
		return err
	}
	drifts := verify(lock)
	if *jsonOut {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(drifts)
	}
	printDrift(drifts)
	return nil
}

func replayCommand(args []string) error {
	fs := flag.NewFlagSet("replay", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	runIt := fs.Bool("run", false, "execute the recorded agent command")
	runTests := fs.Bool("tests", false, "run recorded verification commands after replay")
	force := fs.Bool("force", false, "run despite detected drift")
	if err := fs.Parse(args); err != nil {
		return err
	}
	path, err := singleOptionalPath(fs.Args(), defaultLock)
	if err != nil {
		return err
	}
	lock, err := readLock(path)
	if err != nil {
		return err
	}

	drifts := verify(lock)
	printDrift(drifts)
	if len(lock.Agent.Command) == 0 {
		if *runTests {
			return runRecordedTests(lock)
		}
		return errors.New("lock file has no recorded agent command")
	}

	fmt.Printf("Recorded command: %s\n", shellDisplay(lock.Agent.Command))
	if !*runIt {
		fmt.Println("Not executed. Add --run to replay it.")
		return nil
	}
	if len(drifts) > 0 && !*force {
		return errors.New("environment drift detected; use --force to run anyway")
	}

	code, execErr := runExactCommand(lock.Agent.Command)
	if execErr != nil {
		return fmt.Errorf("recorded command exited with code %d", code)
	}
	if *runTests {
		return runRecordedTests(lock)
	}
	return nil
}

func testCommand(args []string) error {
	path, err := singleOptionalPath(args, defaultLock)
	if err != nil {
		return err
	}
	lock, err := readLock(path)
	if err != nil {
		return err
	}
	return runRecordedTests(lock)
}
