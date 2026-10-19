# RFC 003: Remaining Widget Implementation

Status: Completed
Author: @moond4rk
Date: 2026-02-15

## Summary

This RFC tracked the remaining widgets to implement for feature parity with the TypeScript ccstatusline project, plus ccstatus-exclusive widgets from the Claude Code JSON API.

All widgets are now implemented. ccstatus has 37 registered widgets, achieving 100% coverage of the official Claude Code JSON schema.

## Comparison Matrix

| Widget | ccstatus (Go) | ccstatusline (TS) | Status |
|--------|:---:|:---:|--------|
| model | Y | Y | Done |
| version | Y | Y | Done |
| output-style | Y | Y | Done |
| session-id | Y | Y (claude-session-id) | Done |
| git-branch | Y | Y | Done |
| git-changes | Y | Y | Done (different format) |
| git-worktree | Y | Y | Done |
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
| cache-hit-rate | Y | - | Done (Go exclusive) |
| block-timer | Y | Y | Done |
| session-clock | Y | Y | Done |
| session-cost | Y | Y | Done |
| api-duration | Y | - | Done (Go exclusive) |
| current-working-dir | Y | Y | Done |
| project-dir | Y | - | Done (Go exclusive) |
| transcript-path | Y | - | Done (Go exclusive) |
| terminal-width | Y | Y | Done |
| custom-text | Y | Y | Done |
| custom-command | Y | Y | Done |
| vim-mode | Y | - | Done (Go exclusive) |
| agent-name | Y | - | Done (Go exclusive) |
| exceeds-200k | Y | - | Done (Go exclusive) |
| lines-changed | Y | - | Done (Go exclusive) |
| lines-added | Y | - | Done (Go exclusive) |
| lines-removed | Y | - | Done (Go exclusive) |
| separator | Y | Y | Done |
| flex-separator | Y | Y | Done |

### Summary

- **Total widgets**: 37 registered
- **ccstatus-exclusive widgets (14)**: current-usage-input, current-usage-output, cache-creation, remaining-percentage, cache-hit-rate, api-duration, project-dir, transcript-path, lines-changed, lines-added, lines-removed, vim-mode, agent-name, exceeds-200k
- **Shared with ccstatusline (23)**: model, version, output-style, session-id, git-branch, git-changes, git-worktree, tokens-input, tokens-output, tokens-cached, tokens-total, context-length, context-percentage, context-percentage-usable, block-timer, session-clock, session-cost, current-working-dir, terminal-width, custom-text, custom-command, separator, flex-separator

## Implementation Notes

### Generic Widget Patterns

To avoid code duplication flagged by the `dupl` linter, several generic widget types were created:

- **`tokenWidget`** - Parameterized by extractor function. Used for 7 token widgets (tokens-input, tokens-output, tokens-cached, tokens-total, current-usage-input, current-usage-output, cache-creation).
- **`percentageWidget`** - Parameterized by extractor function. Used for 3 widgets (context-percentage, remaining-percentage, cache-hit-rate).
- **`stringFieldWidget`** - Parameterized by extractor, color, name, description. Used for 3 widgets (output-style, vim-mode, agent-name).

### cache-hit-rate

Calculates the cache read token ratio as a percentage. Formula: `cache_read_input_tokens / (input_tokens + cache_creation_input_tokens + cache_read_input_tokens) * 100`. Default color: cyan. Default prefix: "Cache: ".

### block-timer

Uses `cost.total_duration_ms` as primary data source (available from Claude Code JSON). Falls back to JSONL transcript parsing (`internal/jsonl/`) when duration data is unavailable. Supports three display modes via `metadata["display"]`:
- `"time"` (default): `2h15m/5h`
- `"progress"`: `[=====>    ] 50%`
- `"percentage"`: `50%`

### custom-command

Executes shell commands via `sh -c` with configurable timeout (default 3s). Pipes full JSON session data to stdin. Supports `preserveColors` to retain ANSI escape sequences, `maxWidth` to truncate output.

### git-worktree

Detects linked worktrees by checking if `git rev-parse --git-dir` returns a path containing `/worktrees/`. Returns the worktree name or empty string for main working tree.

## Notable Feature Differences

### ccstatusline (TS) has but ccstatus (Go) does not

1. **Display mode toggles**: context-percentage has `inverse` metadata for used/remaining toggle
2. **Fish-style path abbreviation**: current-working-dir supports fish-style (`~/D/g/m/ccstatus`) via metadata
3. **N-segment path truncation**: current-working-dir supports configurable segment count
4. **`hideNoGit` metadata**: git widgets can suppress "no git" display

### ccstatus (Go) has but ccstatusline (TS) does not

1. **lines-changed/added/removed**: Real git diff stats (uncommitted changes)
2. **current-usage-input/output**: Per-round token metrics
3. **cache-creation**: Cache creation token visibility
4. **remaining-percentage**: Explicit remaining context percentage
5. **cache-hit-rate**: Cache read token ratio as percentage
6. **api-duration**: API response time widget
7. **project-dir / transcript-path**: Additional environment info
8. **vim-mode / agent-name / exceeds-200k**: From official Claude Code JSON fields
