# RFC 006: Docs Parity Follow-ups

- Status: Accepted
- Author: @moond4rk
- Date: 2026-11-08

## Summary

A review of ccstatus against the current status line docs (Claude Code v2.1.319, 2026-11-06) found the schema fully covered: every field of the docs' example payload and every subagent task field is parsed, and `install` writes every `statusLine` key. What remains are behaviors the docs describe that ccstatus does not use yet, plus a toolchain refresh. Two PRs:

1. Subagent rows: hide rows by task status, and `dump --subagent` to capture the real payload.
2. Main status line and tooling: a `hyperlinks` setting, a linked `repo` widget, an `install` hint for the vim indicator, Go 1.27.3 and current dependencies.

## Motivation

| Docs behavior | ccstatus before |
| --- | --- |
| An empty `content` hides a subagent row | Empty rows are omitted, which keeps Claude Code's default row; a row cannot be hidden |
| The subagent payload carries the base hook fields (`prompt_id`, `permission_mode`, `scratchpad_dir`, `hook_event_name`, ...) without an exact list | `dump` captures only the main payload, so what arrives cannot be checked |
| A clickable repository link is one of the docs' examples; Terminal.app, tmux and SSH can mangle OSC 8 sequences | `pr` always links, `repo` never does, and links cannot be turned off |
| `hideVimModeIndicator` assumes the status line shows `vim.mode` | The default layout has no `vim-mode`, so `install --hide-vim-indicator` hides the mode entirely |

## Design

### PR 1: Subagent rows

- `subagentHideStatuses` (string list, default none) names the task statuses whose rows are hidden: `ccstatus subagent` prints `{"id": ..., "content": ""}` for them. Rows that render empty are still omitted, so Claude Code keeps its default row.
- `validate` rejects entries other than the documented `running`, `completed`, `failed` and `killed`, which catches typos such as `complete`. The statuses become constants in `internal/status`.
- `preview --subagent` leaves hidden rows out.
- `dump --subagent` stands in for `ccstatus subagent`: it saves the payload and prints the rows. It writes `/tmp/ccstatus-subagent-dump.json` (`.jsonl` with `--append`), so it never overwrites the main dump.
- The base hook fields stay unmodeled. Nothing renders them, and `permission_mode`, the one candidate for a widget, is session-wide, so it would repeat on every row.

### PR 2: Main status line and tooling

- `hyperlinks` (bool, default `true`) controls every OSC 8 link ccstatus emits; `false` suits terminals that print the escape sequences.
- `repo` links `owner/name` to `https://{host}/{owner}/{name}` when host, owner and name are all present. Raw mode stays plain, as it does for `pr`.
- `install --hide-vim-indicator` prints a hint when no line has a `vim-mode` widget.
- Go 1.27.3 and the latest module versions. golangci-lint v2.15 and the CI actions are already current.

## Out of Scope

- A cross-invocation git cache: a full render, git included, takes 20-30 ms.
- Windows builds and feature parity with the TypeScript ccstatusline (see RFC 005).

## Verification

```bash
go build ./... && go vet ./... && go test -race ./...
golangci-lint run

# With "subagentHideStatuses": ["completed"], prints {"id":"t1","content":""}
echo '{"columns":80,"tasks":[{"id":"t1","status":"completed"}]}' | ccstatus subagent

# Capture the real subagent payload: point subagentStatusLine.command at
# "ccstatus dump --subagent --append", then read /tmp/ccstatus-subagent-dump.jsonl
```

## References

- Status line documentation: https://code.claude.com/docs/en/statusline
- Hook common input fields (base of the subagent payload): https://code.claude.com/docs/en/hooks#common-input-fields
- RFC 005 (previous plan)
