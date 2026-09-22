package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCapturePrompt(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "prompt.txt")
	if err := os.WriteFile(path, []byte("fix it"), 0o600); err != nil {
		t.Fatal(err)
	}
	a, err := capturePrompt("", path)
	if err != nil {
		t.Fatal(err)
	}
	if a.Content != "fix it" {
		t.Fatalf("unexpected content %q", a.Content)
	}
	if a.SHA256 == "" {
		t.Fatal("expected hash")
	}
}

func TestSortedUnique(t *testing.T) {
	got := sortedUnique([]string{"network", "write", "network", ""})
	if len(got) != 2 || got[0] != "network" || got[1] != "write" {
		t.Fatalf("unexpected values: %#v", got)
	}
}

func TestLockRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.lock")
	original := Lock{SchemaVersion: 1, Agent: AgentRun{Name: "test", Command: []string{"echo", "hi"}}, Environment: EnvironmentInfo{OS: "linux", Arch: "amd64"}}
	if err := writeLock(path, original); err != nil {
		t.Fatal(err)
	}
	got, err := readLock(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Agent.Name != "test" || len(got.Agent.Command) != 2 {
		t.Fatalf("bad round trip: %#v", got)
	}
}
