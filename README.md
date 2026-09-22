# agent.look

**Record an AI coding run now. See what changed when you try to repeat it later.**

When you run a coding agent through `agent.look`, it writes an `agent.lock` file containing the conditions around that run:

- which agent and model you named
- the command that launched it
- the prompt you supplied
- project instruction files such as `AGENTS.md` and `CLAUDE.md`
- detected project tool configuration
- declared permissions
- Git commit, branch, and local-change state
- operating system and developer-tool versions
- checks you asked `agent.look` to run afterward

Later, `agent.look verify` compares the current project and machine with the recorded run and tells you what changed.

```text
Yesterday

agent.look record ... -- codex exec "Fix the flaky test"
        │
        ▼
    agent.lock
        │
        │   model: gpt-5.6
        │   Git:   91c8f2d
        │   Node:  v22.15.0
        │   AGENTS.md: 4d7a...
        │   ...
        │
        ▼
Today

agent.look verify
        │
        ▼
! Git commit changed
! Node v22.15.0 -> v24.1.0
! AGENTS.md changed
```

Think of `agent.lock` as a **record of the conditions around one AI coding run**.

It does not make AI output deterministic, restore a remote model to an old internal state, or rebuild an operating system for you. It records observable project and machine conditions, detects drift, and can re-run the recorded command after you review that drift.

## Install

```sh
go install github.com/seipass/agent.look@latest
```

Or build it from source:

```sh
git clone https://github.com/seipass/agent.look
cd agent.look
go build .
```

## Quick start

Run an agent through `agent.look`:

```sh
agent.look record \
  --agent codex \
  --model gpt-5.6 \
  -- codex exec "Fix the flaky test"
```

This runs the command after `--`, records its exit status and duration, inspects the surrounding project and machine, then writes `agent.lock`.

See what was recorded:

```sh
agent.look inspect
```

Example:

```text
agent.look schema 1
agent: codex
model: gpt-5.6
command: codex exec "Fix the flaky test"
repo: 91c8f2d4a61e @ main
dirty: false
instructions: 2
external tools: 3
permissions: 0
tests: 0
```

Later, in the same project:

```sh
agent.look verify
```

Example:

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

If nothing relevant changed, `verify` says the current environment matches the lock file.

## What is automatic and what you provide

`agent.look` can inspect the repository and local machine itself. It automatically records Git state, common project instruction files, common project-level external-tool configuration, operating-system information, developer-tool versions, and environment-variable names.

You provide information that `agent.look` cannot reliably discover from outside the agent process, such as:

- `--agent` for the agent name
- `--model` for the model name
- `--prompt` or `--prompt-file` when you want the prompt stored explicitly
- `--permission` for permissions you want recorded
- `--test` for checks that should be run and recorded

If the command after `--` contains the prompt itself, that command is also stored in `agent.lock`.

## Record without launching an agent

You can create a lock file from metadata and the current project state without executing an agent command:

```sh
agent.look record \
  --agent my-agent \
  --model my-model \
  --prompt "Fix issue #42"
```

This is useful when the agent was launched somewhere `agent.look` cannot wrap directly.

## Record checks

You can ask `agent.look` to run checks after the agent command:

```sh
agent.look record \
  --agent codex \
  --model gpt-5.6 \
  --test "go test ./..." \
  -- codex exec "Fix the flaky test"
```

The lock file records the check command, exit code, duration, and an output digest.

## Inspect the lock file

```sh
agent.look inspect
```

`agent.lock` is JSON and is intentionally small enough to inspect by hand.

It may contain prompt text and command arguments. Review it before publishing if those contain sensitive information.

Environment-variable values are never stored.

## Verify drift

```sh
agent.look verify
```

Machine-readable output:

```sh
agent.look verify --json
```

The verifier checks the current repository, instruction files, environment, and detected tool versions against the recorded run.

## Replay a recorded command

First inspect what would be executed:

```sh
agent.look replay
```

This does not execute the command.

To run the recorded command:

```sh
agent.look replay --run
```

If relevant recorded conditions have changed, replay stops and shows the differences first.

After reviewing them, you can explicitly continue:

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

A lock file can contain a command. Treat an `agent.lock` from someone else with the same care you would give a shell script. `replay --run` is the explicit operation that executes it.

## Detected instruction files

The current release recognizes:

- `AGENTS.md`
- `CLAUDE.md`
- `GEMINI.md`
- `.cursorrules`
- `.windsurfrules`
- `.github/copilot-instructions.md`
- files under `.cursor/rules/`

## Detected project tool configuration

The current release reads project-level server names and executable commands from:

- `.mcp.json`
- `mcp.json`
- `.cursor/mcp.json`
- `.vscode/mcp.json`

Server environment values and secrets are not copied into the lock file.

## What `agent.look` cannot reproduce

`agent.look` records observable inputs and local conditions. Some parts of an AI run can still differ later, including:

- nondeterministic model output
- a model provider changing the model behind the same name
- remote service state
- external data that changed after the original run
- system state that `agent.look` does not currently record

The goal is to make unexplained drift visible and to preserve enough context to investigate or repeat a run more deliberately.

## Status

Early release. The lock schema is versioned and may change before v1.0.

## License

MIT
