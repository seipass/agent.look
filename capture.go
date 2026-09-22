package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

func capturePrompt(text, path string) (*Artifact, error) {
	if text == "" && path == "" {
		return nil, nil
	}
	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read prompt file: %w", err)
		}
		return &Artifact{Path: sanitizePath(path), SHA256: hashBytes(data), Content: string(data)}, nil
	}
	data := []byte(text)
	return &Artifact{SHA256: hashBytes(data), Content: text}, nil
}

func captureInstructions() []Artifact {
	candidates := []string{
		"AGENTS.md", "CLAUDE.md", "GEMINI.md", ".cursorrules", ".windsurfrules",
		filepath.Join(".github", "copilot-instructions.md"),
	}
	var files []string
	for _, p := range candidates {
		if stat, err := os.Stat(p); err == nil && !stat.IsDir() {
			files = append(files, p)
		}
	}
	for _, dir := range []string{filepath.Join(".cursor", "rules")} {
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d == nil {
				return nil
			}
			if !d.IsDir() {
				files = append(files, path)
			}
			return nil
		})
	}
	files = sortedUnique(files)
	out := make([]Artifact, 0, len(files))
	for _, p := range files {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		out = append(out, Artifact{Path: filepath.ToSlash(p), SHA256: hashBytes(data)})
	}
	return out
}

func captureExternalTools() []ExternalTool {
	candidates := []string{".mcp.json", "mcp.json", filepath.Join(".cursor", "mcp.json"), filepath.Join(".vscode", "mcp.json")}
	var out []ExternalTool
	for _, p := range candidates {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var root map[string]any
		if json.Unmarshal(data, &root) != nil {
			continue
		}
		var servers map[string]any
		for _, key := range []string{"mcpServers", "servers"} {
			if m, ok := root[key].(map[string]any); ok {
				servers = m
				break
			}
		}
		names := make([]string, 0, len(servers))
		for name := range servers {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			command := ""
			if config, ok := servers[name].(map[string]any); ok {
				if c, ok := config["command"].(string); ok {
					command = c
				}
			}
			out = append(out, ExternalTool{Source: filepath.ToSlash(p), Name: name, Command: command})
		}
	}
	return out
}

func captureRepository() *RepositoryInfo {
	root, err := commandOutput("git", "rev-parse", "--show-toplevel")
	if err != nil {
		return nil
	}
	branch, _ := commandOutput("git", "branch", "--show-current")
	commit, _ := commandOutput("git", "rev-parse", "HEAD")
	status, _ := commandOutput("git", "status", "--porcelain=v1", "--untracked-files=normal")
	diffText, _ := commandOutput("git", "diff", "HEAD")
	info := &RepositoryInfo{Root: sanitizePath(root), Branch: branch, Commit: commit, Dirty: strings.TrimSpace(status) != ""}
	if info.Dirty {
		info.DiffHash = hashBytes([]byte(diffText + "\n" + status))
	}
	return info
}

func captureEnvironment() EnvironmentInfo {
	env := EnvironmentInfo{
		OS: runtime.GOOS, Arch: runtime.GOARCH, Kernel: kernelVersion(), Timezone: time.Now().Location().String(),
		Tools: make(map[string]Tool),
	}
	for name, args := range toolVersionArgs {
		path, err := exec.LookPath(name)
		if err != nil {
			continue
		}
		data, _ := exec.Command(path, args...).CombinedOutput()
		env.Tools[name] = Tool{Path: sanitizePath(path), Version: firstNonEmptyLine(string(data))}
	}
	for _, pair := range os.Environ() {
		key, _, ok := strings.Cut(pair, "=")
		if ok && key != "" {
			env.EnvPresent = append(env.EnvPresent, key)
		}
	}
	sort.Strings(env.EnvPresent)
	return env
}
