package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
)

func writeLock(path string, lock Lock) error {
	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o600)
}

func readLock(path string) (Lock, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Lock{}, err
	}
	var lock Lock
	if err := json.Unmarshal(data, &lock); err != nil {
		return Lock{}, err
	}
	if lock.SchemaVersion != schemaVersion {
		return Lock{}, fmt.Errorf("unsupported schema version %d", lock.SchemaVersion)
	}
	return lock, nil
}

func runExactCommand(command []string) (int, error) {
	if len(command) == 0 {
		return 0, nil
	}
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), err
	}
	return 1, err
}

func runShellCapture(command string) ([]byte, int) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/d", "/s", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}
	out, err := cmd.CombinedOutput()
	if err == nil {
		return out, 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return out, exitErr.ExitCode()
	}
	return out, 1
}

func kernelVersion() string {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "ver")
	} else {
		cmd = exec.Command("uname", "-sr")
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.Join(strings.Fields(string(out)), " ")
}

func commandOutput(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func firstNonEmptyLine(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			return strings.TrimSpace(line)
		}
	}
	return ""
}

func hashBytes(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func sortedUnique(items []string) []string {
	set := map[string]bool{}
	for _, item := range items {
		if strings.TrimSpace(item) != "" {
			set[item] = true
		}
	}
	out := make([]string, 0, len(set))
	for item := range set {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func stringSet(items []string) map[string]bool {
	out := make(map[string]bool, len(items))
	for _, item := range items {
		out[item] = true
	}
	return out
}

func sanitizePath(s string) string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		if s == home {
			return "~"
		}
		if strings.HasPrefix(s, home+string(os.PathSeparator)) {
			return "~" + strings.TrimPrefix(s, home)
		}
	}
	return s
}

func shellDisplay(parts []string) string {
	out := make([]string, len(parts))
	for i, p := range parts {
		if p == "" || strings.ContainsAny(p, " \t\n\"'") {
			out[i] = fmt.Sprintf("%q", p)
		} else {
			out[i] = p
		}
	}
	return strings.Join(out, " ")
}

func singleOptionalPath(args []string, fallback string) (string, error) {
	if len(args) == 0 {
		return fallback, nil
	}
	if len(args) == 1 {
		return args[0], nil
	}
	return "", errors.New("too many positional arguments")
}

func valueOr(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}
func printable(v string) string {
	if v == "" {
		return "(missing)"
	}
	return v
}
func shortCommit(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return valueOr(s, "(none)")
}
func shortHash(s string) string {
	if len(s) > 12 {
		return s[:12]
	}
	return s
}
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
