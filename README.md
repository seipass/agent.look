# agent.look

**A lock file for AI coding runs.**

Package managers have lock files so tomorrow uses the same dependencies as today. `agent.look` does the same kind of job for AI-assisted development: it records the conditions around a coding run in one `agent.lock` file.

```text
$ agent.look inspect
agent.look schema 1
created: 2026-09-22T05:00:00Z
agent: codex
model: gpt-5.6
command: codex exec "Fix the flaky test"
repo: 91c8f2d4a61e @ main
dirty: false
instructions: 2
external tools: 3
permissions: network, write
tests: 1
```

Later:

```text
$ agent.look verify
! HIGH   repository.commit
  recorded: 91c8f2d4a61e...
  current:  3a612de71a4c...
  why:      Source revision changed.

~ MEDIUM tool.node.version
  recorded: v22.15.0
  current:  v24.1.0
  why:      Tool version changed.
```

## Record a run

```sh
agent.look record \
  --agent codex \
  --model gpt-5.6 \
  --prompt-file task.md \
  --permission write \
  --permission network \
  --test "go test ./..." \
  -- codex exec "$(cat task.md)"
```

This writes `agent.lock` after the command and recorded checks finish.

You can also record metadata without executing an agent command:

```sh
agent.look record --agent my-agent --model my-model --prompt "Fix issue #42"
```

## What goes into `agent.lock`

- agent name and model
- the exact command used to launch it
- prompt text or prompt-file contents when supplied
- hashes of common instruction files such as `AGENTS.md`, `CLAUDE.md`, and repository Copilot instructions
- project-level MCP server names and commands from common configuration files
- declared permissions
- Git commit, branch, working-tree state, and a digest of local changes
- operating system, CPU architecture, kernel, timezone, developer-tool versions, and executable paths
- names of environment variables that were present
- verification commands, exit codes, duration, and output digests

Environment variable **values are never stored**.

`agent.lock` can contain prompt text and command arguments. Review it before publishing if those contain sensitive information.

## Verify drift

```sh
agent.look verify
```

Machine-readable output:

```sh
agent.look verify --json
```

The verifier checks the current repository, instruction files, environment, and tool versions against the recorded run.

## Replay

See what would be replayed:

```sh
agent.look replay
```

The recorded agent command is **not executed by default**. To execute it:

```sh
agent.look replay --run
```

Replay stops when it detects environment drift. If you have reviewed the differences and still want to continue:

```sh
agent.look replay --run --force
```

Run the recorded checks afterward:

```sh
agent.look replay --run --tests
```

Or run only the recorded checks:

```sh
agent.look test
```

Treat lock files from strangers like shell scripts. `replay --run` can execute the command stored in the file.

## Detected instruction files

The first release recognizes:

- `AGENTS.md`
- `CLAUDE.md`
- `GEMINI.md`
- `.cursorrules`
- `.windsurfrules`
- `.github/copilot-instructions.md`
- files under `.cursor/rules/`

## Detected external-tool configuration

The first release reads project-level server names and executable commands from:

- `.mcp.json`
- `mcp.json`
- `.cursor/mcp.json`
- `.vscode/mcp.json`

It does not copy server environment values or secrets into the lock file.

## Install

With Go:

```sh
go install github.com/seipass/agent.look@latest
```

Or build from source:

```sh
git clone https://github.com/seipass/agent.look
cd agent.look
go build .
```

## Status

Early release. The lock schema is versioned and intentionally small enough to inspect by hand. Expect changes before v1.0.

## License

MIT
