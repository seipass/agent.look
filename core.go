package main

import (
	"fmt"
	"os"
	"strings"
)

const version = "0.1.0"
const schemaVersion = 1
const defaultLock = "agent.lock"

type Lock struct {
	SchemaVersion int             `json:"schema_version"`
	CreatedAt     string          `json:"created_at"`
	Agent         AgentRun        `json:"agent"`
	Prompt        *Artifact       `json:"prompt,omitempty"`
	Instructions  []Artifact      `json:"instructions,omitempty"`
	ExternalTools []ExternalTool  `json:"external_tools,omitempty"`
	Permissions   []string        `json:"permissions,omitempty"`
	Repository    *RepositoryInfo `json:"repository,omitempty"`
	Environment   EnvironmentInfo `json:"environment"`
	Tests         []TestResult    `json:"tests,omitempty"`
}

type AgentRun struct {
	Name       string   `json:"name,omitempty"`
	Model      string   `json:"model,omitempty"`
	Command    []string `json:"command,omitempty"`
	ExitCode   *int     `json:"exit_code,omitempty"`
	DurationMS int64    `json:"duration_ms,omitempty"`
}

type Artifact struct {
	Path    string `json:"path,omitempty"`
	SHA256  string `json:"sha256"`
	Content string `json:"content,omitempty"`
}

type ExternalTool struct {
	Source  string `json:"source"`
	Name    string `json:"name"`
	Command string `json:"command,omitempty"`
}

type RepositoryInfo struct {
	Root     string `json:"root,omitempty"`
	Branch   string `json:"branch,omitempty"`
	Commit   string `json:"commit,omitempty"`
	Dirty    bool   `json:"dirty"`
	DiffHash string `json:"diff_hash,omitempty"`
}

type EnvironmentInfo struct {
	OS         string          `json:"os"`
	Arch       string          `json:"arch"`
	Kernel     string          `json:"kernel,omitempty"`
	Timezone   string          `json:"timezone,omitempty"`
	Tools      map[string]Tool `json:"tools,omitempty"`
	EnvPresent []string        `json:"env_present,omitempty"`
}

type Tool struct {
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
}

type TestResult struct {
	Command      string `json:"command"`
	ExitCode     int    `json:"exit_code"`
	DurationMS   int64  `json:"duration_ms"`
	OutputSHA256 string `json:"output_sha256,omitempty"`
}

type Drift struct {
	Severity string `json:"severity"`
	Key      string `json:"key"`
	Recorded string `json:"recorded"`
	Current  string `json:"current"`
	Why      string `json:"why"`
}

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

var toolVersionArgs = map[string][]string{
	"git":     {"--version"},
	"go":      {"version"},
	"python":  {"--version"},
	"python3": {"--version"},
	"node":    {"--version"},
	"npm":     {"--version"},
	"pnpm":    {"--version"},
	"yarn":    {"--version"},
	"bun":     {"--version"},
	"deno":    {"--version"},
	"java":    {"-version"},
	"rustc":   {"--version"},
	"cargo":   {"--version"},
	"docker":  {"--version"},
	"claude":  {"--version"},
	"codex":   {"--version"},
	"gemini":  {"--version"},
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "agent.look:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		printHelp()
		return nil
	}
	switch args[0] {
	case "record":
		return recordCommand(args[1:])
	case "inspect":
		return inspectCommand(args[1:])
	case "verify":
		return verifyCommand(args[1:])
	case "replay":
		return replayCommand(args[1:])
	case "test":
		return testCommand(args[1:])
	case "version", "--version", "-v":
		fmt.Println(version)
		return nil
	case "help", "--help", "-h":
		printHelp()
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\nRun 'agent.look help' for usage", args[0])
	}
}

func printHelp() {
	fmt.Print(`agent.look records the conditions around an AI coding run in agent.lock.

Usage:
  agent.look record [options] -- <agent command...>
  agent.look inspect [agent.lock]
  agent.look verify [--json] [agent.lock]
  agent.look replay [--run] [--tests] [--force] [agent.lock]
  agent.look test [agent.lock]

Record options:
  --agent NAME           agent name, for example codex or claude
  --model NAME           model name
  --prompt TEXT          prompt text to embed in the lock file
  --prompt-file PATH     prompt file to embed in the lock file
  --permission VALUE     declared permission; repeatable
  --test COMMAND         verification command; repeatable
  -o PATH                output path (default agent.lock)

Examples:
  agent.look record --agent codex --model gpt-5.6 --prompt-file task.md \
    --test "go test ./..." -- codex exec "$(cat task.md)"

  agent.look verify
  agent.look replay
  agent.look replay --run

Replay never executes the recorded command unless --run is supplied.
`)
}
