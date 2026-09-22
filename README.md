# agent.look

**A lockfile for AI coding.**

`agent.look` records the conditions of an AI coding run in one `agent.lock` file, so you can see what changed when yesterday's run cannot be reproduced today.

```text
AI coding run
     │
     ├── model
     ├── prompt
     ├── instructions
     ├── tools
     ├── permissions
     ├── Git state
     ├── environment
     └── checks
          │
          ▼
      agent.lock
```

## Try it in 30 seconds

Record a run:

```sh
agent.look record --agent codex --model gpt-5.6 -- codex exec "Fix the flaky test"
```

See what was recorded:

```sh
agent.look inspect
```

```text
agent.look schema 1
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

Later, check whether the conditions changed:

```sh
agent.look verify
```

```text
! HIGH   repository.commit
  recorded: 91c8f2d4a61e...
  current:  3a612de71a4c...
  why:      Source revision changed.

~ MEDIUM tool.node.version
  recorded: v22.15.0
  current:  v24.1.0
  why:      Tool version changed.
```

That's the whole idea: **one AI coding run, one inspectable lockfile.**

## Install

```sh
go install github.com/seipass/agent.look@latest
```

Or build it locally:

```sh
git clone https://github.com/seipass/agent.look
cd agent.look
go build .
```

## What goes into `agent.lock`

| Area | Recorded |
| --- | --- |
| Agent | name, model, launch command, exit code, duration |
| Prompt | supplied text or prompt-file contents |
| Instructions | hashes and contents of common repository instruction files |
| External tools | project-level server names and executable commands from supported configuration files |
| Permissions | permissions declared for the run |
| Repository | commit, branch, working-tree state, local-change digest |
| Environment | operating system, CPU, kernel, timezone, tool versions, executable paths |
| Checks | commands, exit codes, duration, output digests |

Environment variable **values are never stored**.

## Record checks too

```sh
agent.look record \
  --agent codex \
  --model gpt-5.6 \
  --test "go test ./..." \
  -- codex exec "Fix the flaky test"
```

The check result becomes part of `agent.lock`.

Run the recorded checks again later:

```sh
agent.look test
```

## Replay

Preview the recorded command:

```sh
agent.look replay
```

Execute it only when you explicitly ask:

```sh
agent.look replay --run
```

Replay stops when `agent.look` detects environment drift. After reviewing the differences, you can override that check:

```sh
agent.look replay --run --force
```

Run the recorded checks afterward:

```sh
agent.look replay --run --tests
```

## Safety

An `agent.lock` file can contain prompt text and command arguments. Review it before publishing if those contain sensitive information.

`replay --run` can execute the command stored in the file. Treat lockfiles from strangers with the same caution as shell scripts.

## What `agent.look` can reproduce

`agent.look` records the parts of a coding run it can observe locally and detects when they drift.

It does **not** freeze a hosted AI model, remote service, network response, or hidden provider-side state. A matching `agent.lock` means the recorded local conditions still match; it is not a promise that an external model will return identical output.

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

Server environment values and secrets are not copied into the lockfile.

## Machine-readable verification

```sh
agent.look verify --json
```

## Status

Early release. The lock schema is versioned and may change before v1.0.

## License

MIT
