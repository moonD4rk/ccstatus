# RFC 003: Remaining Widget Implementation

Status: Draft
Author: @moond4rk
Date: 2026-02-15

## Summary

This RFC tracks the remaining widgets to implement for feature parity with the TypeScript ccstatusline project, plus ccstatus-exclusive widgets from the Claude Code JSON API.

## Current State

ccstatus has 27 registered widgets. The TypeScript ccstatusline has 21 widgets. There is significant overlap but each project has unique widgets the other lacks.

## Comparison Matrix

| Widget | ccstatus (Go) | ccstatusline (TS) | Status |
|--------|:---:|:---:|--------|
| model | Y | Y | Done |
| version | Y | Y | Done |
| output-style | - | Y | **TODO** |
| session-id | - | Y (claude-session-id) | **TODO** |
| git-branch | Y | Y | Done |
| git-changes | Y | Y | Done (different format) |
| git-worktree | - | Y | **TODO** |
| tokens-input | Y | Y | Done |
| tokens-output | Y | Y | Done |
| tokens-cached | Y | Y | Done |
| tokens-total | Y | Y | Done |
| current-usage-input | Y | - | Done (Go exclusive) |
| current-usage-output | Y | - | Done (Go exclusive) |
| cache-creation | Y | - | Done (Go exclusive) |
| context-length | Y | Y | Done |
| context-percentage | Y | Y | Done |
| context-percentage-usable | Y | Y | Done |
| remaining-percentage | Y | - | Done (Go exclusive) |
| block-timer | - | Y | **TODO** (high effort) |
| session-clock | Y | Y | Done |
| session-cost | Y | Y | Done |
| api-duration | Y | - | Done (Go exclusive) |
| current-working-dir | Y | Y | Done |
| project-dir | Y | - | Done (Go exclusive) |
| transcript-path | Y | - | Done (Go exclusive) |
| terminal-width | - | Y | **TODO** |
| custom-text | Y | Y | Done |
| custom-command | - | Y | **TODO** (high value) |
| vim-mode | - | - | **TODO** (both planned) |
| agent-name | - | - | **TODO** (both planned) |
| exceeds-200k | - | - | **TODO** (both planned) |
| lines-changed | Y | - | Done (Go exclusive) |
| lines-added | Y | - | Done (Go exclusive) |
| lines-removed | Y | - | Done (Go exclusive) |
| separator | Y | Y | Done |
| flex-separator | Y | Y | Done |

### Summary

- **ccstatus-exclusive widgets (9)**: current-usage-input, current-usage-output, cache-creation, remaining-percentage, api-duration, project-dir, transcript-path, lines-changed/added/removed
- **ccstatusline-exclusive widgets (6)**: output-style, session-id, git-worktree, block-timer, terminal-width, custom-command
- **Both planned but neither implemented (3)**: vim-mode, agent-name, exceeds-200k

## Widgets to Implement

### Priority 1 - High Value

#### 1. `custom-command`

Execute arbitrary shell commands and display output. Core extensibility widget.

- **Source**: Shell command execution
- **Config fields**: `customCommand` (string), `timeout` (ms), `maxWidth` (int)
- **Format**: stdout of the command (first line, trimmed)
- **Complexity**: Medium - need subprocess execution with timeout, security considerations
- **TS reference**: Passes JSON input via stdin to the command, supports `preserveColors` metadata

#### 2. `output-style`

Display the currently configured output style.

- **Source**: `StatusJSON.OutputStyle.Name`
- **Format**: `text` / `json` / `stream-json`
- **Default color**: `brightBlack`
- **Complexity**: Low - single field read
- **Supports rawValue**: no

#### 3. `session-id`

Display the Claude Code session ID.

- **Source**: `StatusJSON.SessionID`
- **Format**: Full UUID (rawValue) or truncated (normal)
- **Default color**: `brightBlack`
- **Complexity**: Low - single field read
- **Supports rawValue**: yes (full vs truncated)

### Priority 2 - Useful

#### 4. `git-worktree`

Display the current git worktree name.

- **Source**: Git command (`git rev-parse --git-dir`)
- **Format**: Worktree name or empty
- **Default color**: `magenta`
- **Complexity**: Medium - need to detect worktree vs main repo
- **TS reference**: Has `hideNoGit` metadata option

#### 5. `terminal-width`

Display the current terminal width in columns. Primarily for debugging.

- **Source**: `RenderContext.TerminalWidth` (already available)
- **Format**: `120` (columns)
- **Default color**: `brightBlack`
- **Complexity**: Very low - data already in RenderContext
- **Supports rawValue**: no

#### 6. `block-timer`

Display time elapsed in current 5-hour Claude Code session block.

- **Source**: JSONL transcript file (parsed via `internal/jsonl/`)
- **Format**: Three display modes - time (`2h15m`), progress bar (`[=====>    ]`), progress-short (`45%`)
- **Default color**: `brightBlack`
- **Complexity**: High - requires JSONL parsing, block detection, multiple display modes
- **TS reference**: 3 modes toggled via metadata `display` field

### Priority 3 - Narrow Use Case

#### 7. `vim-mode`

Display the current vim mode when vim keybindings are enabled.

- **Source**: `StatusJSON.Vim.Mode`
- **Format**: `NORMAL` / `INSERT` / `VISUAL`
- **Default color**: `yellow`
- **Complexity**: Low - single field, only renders when vim is enabled
- **Note**: Field exists in StatusJSON but no widget yet in either project

#### 8. `agent-name`

Display the agent name when running with `--agent` flag.

- **Source**: `StatusJSON.Agent.Name`
- **Format**: Agent name string
- **Default color**: `cyan`
- **Complexity**: Low - single field, only renders in agent mode
- **Note**: Field exists in StatusJSON but no widget yet in either project

#### 9. `exceeds-200k`

Warning indicator when token count exceeds 200k threshold.

- **Source**: `StatusJSON.Exceeds200K`
- **Format**: Warning text (e.g., `>200k`) or empty
- **Default color**: `red`
- **Complexity**: Low - boolean field check
- **Note**: Field exists in StatusJSON but no widget yet in either project

## Notable Feature Differences

### ccstatusline (TS) has but ccstatus (Go) does not

1. **Display mode toggles**: context-percentage has `inverse` metadata for used/remaining toggle
2. **Fish-style path abbreviation**: current-working-dir supports fish-style (`~/D/g/m/ccstatus`) via metadata
3. **N-segment path truncation**: current-working-dir supports configurable segment count
4. **`hideNoGit` metadata**: git widgets can suppress "no git" display

### ccstatus (Go) has but ccstatusline (TS) does not

1. **lines-changed/added/removed**: Session-level code change metrics from Claude Code cost data
2. **current-usage-input/output**: Per-round token metrics
3. **cache-creation**: Cache creation token visibility
4. **remaining-percentage**: Explicit remaining context percentage
5. **api-duration**: API response time widget
6. **project-dir / transcript-path**: Additional environment info

## Implementation Order

Suggested implementation sequence:

1. `output-style` + `session-id` (trivial, same session)
2. `terminal-width` (trivial, data already available)
3. `vim-mode` + `agent-name` + `exceeds-200k` (low effort, complete JSON coverage)
4. `custom-command` (medium effort, high user value)
5. `git-worktree` (medium effort, requires git integration)
6. `block-timer` (high effort, requires JSONL parsing infrastructure)

After all widgets: 36 total registered widgets.
