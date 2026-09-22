package main

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

func verify(lock Lock) []Drift {
	currentRepo := captureRepository()
	currentEnv := captureEnvironment()
	currentInstructions := captureInstructions()
	var drifts []Drift
	add := func(severity, key, old, current, why string) {
		if old != current {
			drifts = append(drifts, Drift{Severity: severity, Key: key, Recorded: printable(old), Current: printable(current), Why: why})
		}
	}
	add("high", "environment.os", lock.Environment.OS, currentEnv.OS, "Operating system changed.")
	add("high", "environment.arch", lock.Environment.Arch, currentEnv.Arch, "CPU architecture changed.")
	add("medium", "environment.kernel", lock.Environment.Kernel, currentEnv.Kernel, "Kernel version changed.")
	add("medium", "environment.timezone", lock.Environment.Timezone, currentEnv.Timezone, "Timezone changed.")

	if lock.Repository != nil {
		if currentRepo == nil {
			add("high", "repository", "git repository", "not a git repository", "Replay is outside the recorded repository.")
		} else {
			add("high", "repository.commit", lock.Repository.Commit, currentRepo.Commit, "Source revision changed.")
			add("high", "repository.dirty", fmt.Sprint(lock.Repository.Dirty), fmt.Sprint(currentRepo.Dirty), "Working-tree state changed.")
			if lock.Repository.Dirty && currentRepo.Dirty {
				add("high", "repository.diff", lock.Repository.DiffHash, currentRepo.DiffHash, "Uncommitted changes differ.")
			}
		}
	}

	names := map[string]bool{}
	for n := range lock.Environment.Tools {
		names[n] = true
	}
	for n := range currentEnv.Tools {
		names[n] = true
	}
	toolNames := make([]string, 0, len(names))
	for n := range names {
		toolNames = append(toolNames, n)
	}
	sort.Strings(toolNames)
	for _, name := range toolNames {
		old, oldOK := lock.Environment.Tools[name]
		now, nowOK := currentEnv.Tools[name]
		if oldOK != nowOK {
			add("high", "tool."+name+".present", fmt.Sprint(oldOK), fmt.Sprint(nowOK), "A recorded executable is missing or newly present.")
			continue
		}
		if oldOK {
			add("medium", "tool."+name+".version", old.Version, now.Version, "Tool version changed.")
			add("low", "tool."+name+".path", old.Path, now.Path, "Executable path changed.")
		}
	}

	oldEnv := stringSet(lock.Environment.EnvPresent)
	nowEnv := stringSet(currentEnv.EnvPresent)
	for key := range oldEnv {
		if !nowEnv[key] {
			add("high", "env."+key+".present", "true", "false", "An environment variable that existed during recording is missing.")
		}
	}

	oldInst := map[string]string{}
	nowInst := map[string]string{}
	for _, a := range lock.Instructions {
		oldInst[a.Path] = a.SHA256
	}
	for _, a := range currentInstructions {
		nowInst[a.Path] = a.SHA256
	}
	paths := map[string]bool{}
	for p := range oldInst {
		paths[p] = true
	}
	for p := range nowInst {
		paths[p] = true
	}
	pathList := make([]string, 0, len(paths))
	for p := range paths {
		pathList = append(pathList, p)
	}
	sort.Strings(pathList)
	for _, p := range pathList {
		add("high", "instruction."+p, oldInst[p], nowInst[p], "Agent instruction file changed.")
	}

	rank := map[string]int{"high": 0, "medium": 1, "low": 2}
	sort.SliceStable(drifts, func(i, j int) bool {
		if rank[drifts[i].Severity] != rank[drifts[j].Severity] {
			return rank[drifts[i].Severity] < rank[drifts[j].Severity]
		}
		return drifts[i].Key < drifts[j].Key
	})
	return drifts
}

func printDrift(drifts []Drift) {
	if len(drifts) == 0 {
		fmt.Println("✓ Current environment matches agent.lock.")
		return
	}
	fmt.Printf("Found %d differences from agent.lock\n\n", len(drifts))
	for _, d := range drifts {
		icon := "·"
		switch d.Severity {
		case "high":
			icon = "!"
		case "medium":
			icon = "~"
		}
		fmt.Printf("%s %-6s %s\n", icon, strings.ToUpper(d.Severity), d.Key)
		fmt.Printf("  recorded: %s\n", truncate(d.Recorded, 140))
		fmt.Printf("  current:  %s\n", truncate(d.Current, 140))
		fmt.Printf("  why:      %s\n\n", d.Why)
	}
}

func executeTest(command string) TestResult {
	start := time.Now()
	out, code := runShellCapture(command)
	return TestResult{Command: command, ExitCode: code, DurationMS: time.Since(start).Milliseconds(), OutputSHA256: hashBytes(out)}
}

func runRecordedTests(lock Lock) error {
	if len(lock.Tests) == 0 {
		return errors.New("lock file has no recorded tests")
	}
	failed := 0
	for _, old := range lock.Tests {
		now := executeTest(old.Command)
		status := "PASS"
		if now.ExitCode != 0 {
			status = "FAIL"
			failed++
		}
		changed := ""
		if old.OutputSHA256 != "" && old.OutputSHA256 != now.OutputSHA256 {
			changed = " output-changed"
		}
		fmt.Printf("%s %s (%d ms)%s\n", status, old.Command, now.DurationMS, changed)
	}
	if failed > 0 {
		return fmt.Errorf("%d recorded test(s) failed", failed)
	}
	return nil
}
