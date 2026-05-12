# RFC 004: Follow Up the Latest Claude Code Statusline Schema

Status: Completed
Author: @moond4rk
Date: 2026-05-12

## Summary

RFCs 001-003 (Feb 2026) targeted the Claude Code statusline JSON schema as it was then. The official schema has since evolved. This RFC brings ccstatus up to date:

1. Reconcile the **semantic change** to `context_window.total_input_tokens` / `total_output_tokens` (as of Claude Code v2.1.132 these are "tokens currently in the context window from the most recent API response", not "cumulative session totals").
2. **Rework `block-timer`** to use the official 5-hour rate-limit window (`rate_limits.five_hour.resets_at`) as its primary source instead of conflating it with session wall-clock time.
3. Add **6 new widgets** for new JSON fields: `rate-limit-5h`, `rate-limit-7d`, `effort`, `thinking`, `session-name`, `added-dirs` (registry grows 37 -> 43).
4. Add `workspace.git_worktree` as the highest-priority source for the existing `git-worktree` widget (it covers any linked worktree, not just `--worktree` sessions).
5. Add optional `install` flags `--refresh` and `--hide-vim-indicator` that write `refreshInterval` / `hideVimModeIndicator` into Claude Code's `settings.json`.

`subagentStatusLine` support is explicitly out of scope (candidate for RFC 005). No ccstatus settings schema version bump is needed -- the changes are additive.

## Motivation

### 1. Existing fields changed meaning (v2.1.132)

The official docs now state: `context_window.total_input_tokens` / `total_output_tokens` are "Token counts currently in the context window, from the most recent API response. Input includes cache reads and writes. Before v2.1.132 these were cumulative session totals." And `total_input_tokens` = `input_tokens + cache_creation_input_tokens + cache_read_input_tokens`.

ccstatus's `tokens-input` / `tokens-output` / `tokens-total` widgets were built (RFC 001) around the old "cumulative session totals" meaning. Their `displayName` / `description` (e.g. "Total input token count") and default prefixes (`In: ` / `Out: ` / `Total: `) now mislead. Notably `tokens-input` is now effectively the same quantity as `context-length` (both = `input + cache_creation + cache_read`).

No crash, no wrong number -- but the labels/docs need to reflect reality.

### 2. `block-timer` conflates session duration with the 5-hour rate-limit window

`block-timer` currently computes `elapsed = cost.total_duration_ms` (wall-clock since session start) and renders it as "elapsed within a 5-hour block" (`2h15m/5h`). But session duration is not the rolling 5-hour rate-limit window -- a 2-hour-old session does not mean "3 hours left on the limit". The official `rate_limits.five_hour.resets_at` (Unix epoch seconds when the 5-hour window resets) is the authoritative source for that window. `block-timer` should prefer it, keeping the `cost.total_duration_ms` / JSONL heuristics as fallbacks for users without `rate_limits` (non-Pro/Max, or before the first API response).

### 3. New JSON fields with no widget

| Field | Meaning | Availability |
|-------|---------|--------------|
| `rate_limits.five_hour.{used_percentage,resets_at}` | 5-hour rate-limit % used (0-100) + reset epoch | Pro/Max only, after first API response; window may be independently absent |
| `rate_limits.seven_day.{used_percentage,resets_at}` | weekly rate-limit % used + reset epoch | same |
| `effort.level` | reasoning effort: `low`/`medium`/`high`/`xhigh`/`max` (live; reflects `/effort`) | absent when the model doesn't support effort |
| `thinking.enabled` | extended thinking on/off | always present |
| `session_name` | custom name from `--name` / `/rename` | absent if no custom name set |
| `workspace.added_dirs` | dirs added via `/add-dir` / `--add-dir` (string array) | empty array if none |
| `workspace.git_worktree` | worktree name for *any* linked worktree | absent in the main working tree |
| `worktree.{path,branch,original_cwd,original_branch}` | extra `--worktree` session info | already parsed into `WorktreeInfo`, no widget exposes them |

### 4. New `statusLine` config options

- `refreshInterval` (int, min 1): re-runs the status line command every N seconds *in addition to* event-driven updates. Recommended when the status line shows time-based data or when background subagents change git state while the main session is idle. ccstatus ships `session-clock`, `block-timer`, `api-duration`, plus git widgets -- all benefit.
- `hideVimModeIndicator` (bool): suppresses Claude Code's built-in `-- INSERT --` line. Users who enable ccstatus's `vim-mode` widget should set this so the mode isn't shown twice.

ccstatus's `install` command currently writes only `{type, command, padding: 0}`.

## Scope

### In Scope

1. `internal/status`: new `Session` / nested structs for the fields in section 3; new extractors for the rate-limit percentages.
2. New widgets: `rate-limit-5h`, `rate-limit-7d`, `effort`, `thinking`, `session-name`, `added-dirs`.
3. `block-timer` rework (prefer `rate_limits.five_hour.resets_at`).
4. `git-worktree`: add `workspace.git_worktree` as highest-priority source.
5. `tokens-input` / `tokens-output` / `tokens-total`: relabel for the new semantics (no behavior change).
6. `install`: `--refresh` / `--hide-vim-indicator` flags; extend `claude.StatusLine`.
7. Docs: README + project CLAUDE.md (and fix the existing widget-count mismatch -- CLAUDE.md says "36", README says "37"; actual is 37, will be 43).
8. Tests for all of the above.

### Out of Scope

- **`subagentStatusLine`** -- a separate Claude Code setting that renders a custom row body per subagent in the agent panel. Its input is different (base hook fields + `columns` + a `tasks[]` array with `id/name/type/status/description/label/startTime/tokenCount/tokenSamples/cwd`) and its output is different (one JSON line per row: `{"id","content"}`). It needs a new subcommand and a new (much simpler, per-task) rendering model. Worth doing -- as RFC 005.
- **OSC 8 clickable-link widget** (e.g. a `repo-link` widget). Minor; separate.
- **Cross-invocation git cache** (a `session_id`-keyed temp file with a TTL, like the official caching example). The current `GitCache` (added in PR #20) already de-dupes git calls within a single render; a process is short-lived and git commands have a 5s timeout. Low priority; separate.
- ccstatus **settings schema version bump** -- not needed (see section 4.8).

## Design

### 4.1 `Session` struct additions -- `internal/status/status.go`

```go
type Session struct {
    // ... existing fields ...
    SessionName string        `json:"session_name,omitempty"`
    Effort      *EffortInfo   `json:"effort,omitempty"`
    Thinking    *ThinkingInfo `json:"thinking,omitempty"`
    RateLimits  *RateLimits   `json:"rate_limits,omitempty"`
    // Worktree already exists
}

type Workspace struct {
    CurrentDir  string   `json:"current_dir,omitempty"`
    ProjectDir  string   `json:"project_dir,omitempty"`
    AddedDirs   []string `json:"added_dirs,omitempty"`
    GitWorktree string   `json:"git_worktree,omitempty"`
}

// EffortInfo holds the live reasoning-effort level. Absent when the model
// does not support the effort parameter.
type EffortInfo struct {
    Level string `json:"level,omitempty"` // low | medium | high | xhigh | max
}

// ThinkingInfo indicates whether extended thinking is enabled for the session.
type ThinkingInfo struct {
    Enabled bool `json:"enabled,omitempty"`
}

// RateLimits holds Claude.ai subscription rate-limit windows. Present only for
// Pro/Max subscribers after the first API response; each window may be absent.
type RateLimits struct {
    FiveHour *RateLimitWindow `json:"five_hour,omitempty"`
    SevenDay *RateLimitWindow `json:"seven_day,omitempty"`
}

// RateLimitWindow is one rate-limit window.
type RateLimitWindow struct {
    UsedPercentage *float64 `json:"used_percentage,omitempty"` // 0-100
    ResetsAt       *int64   `json:"resets_at,omitempty"`       // Unix epoch seconds
}
```

Existing `WorktreeInfo` (`name/path/branch/original_cwd/original_branch`) is unchanged; this RFC just notes that only `name` is currently surfaced by a widget.

### 4.2 New extractors -- `internal/status/format.go`

Same `(float64, bool)` shape as `ContextPercentage`, so they plug into the existing `percentageWidget` extractor signature:

```go
// FiveHourLimitPercent returns rate_limits.five_hour.used_percentage.
func FiveHourLimitPercent(data *Session) (float64, bool) { ... }

// SevenDayLimitPercent returns rate_limits.seven_day.used_percentage.
func SevenDayLimitPercent(data *Session) (float64, bool) { ... }
```

Reset-countdown math (`resets_at` -> "2h13m") stays in the widget layer, consistent with `block_timer.go` / `session_clock.go` (which is where `time` is already used).

### 4.3 New widgets -- registry grows 37 -> 43

**`rate-limit-5h`, `rate-limit-7d` -- new file `internal/widget/rate_limit.go`**

A `RateLimitWidget` parameterized by window (`five_hour` / `seven_day`). Display modes via `metadata["display"]` (mirrors `block-timer`):

| mode | output | source |
|------|--------|--------|
| `percent` (default) | `45%` | `used_percentage` |
| `bar` | `▓▓▓▓░░░░░░ 45%` (width via `metadata["barWidth"]`, default 10) | `used_percentage` |
| `reset` | `2h13m` | `resets_at - now`, clamped to >= 0 |
| `full` | `45% / 2h13m` | both |

Hidden (`""`) when the window or `used_percentage` is absent. `SupportsRawValue()` -> bare number from `used_percentage`. Defaults: color `yellow`; prefix `5h: ` / `7d: `; displayName "5h Rate Limit" / "7d Rate Limit"; description "5-hour rate-limit usage" / "7-day rate-limit usage". A shared `formatShortDuration(time.Duration)` helper is extracted (reuse for the `reset` mode; today `block_timer.formatBlockTime` and `session_clock.formatDuration` each roll their own).

**`effort`, `thinking`, `session-name` -- registry literals in `internal/widget/widget.go`** (same pattern as `output-style` / `vim-mode` / `agent-name`, which are `stringFieldWidget` literals -- no new files):

- `effort`: extract `d.Effort.Level` (nil -> `""`); color `magenta`; prefix `Effort: `; name "Reasoning Effort"; desc "Current reasoning effort level".
- `thinking`: extract `"on"` when `d.Thinking != nil && d.Thinking.Enabled`, else `""` (text-or-empty pattern, like `exceeds-200k`); color `magenta`; prefix `Think: `; name "Thinking"; desc "Extended thinking indicator".
- `session-name`: extract `d.SessionName` (empty -> hidden); color `white`; no prefix; name "Session Name"; desc "Custom session name set with --name / /rename".

**`added-dirs` -- new file `internal/widget/added_dirs.go`**

`AddedDirsWidget`: default renders `+N` (count of `workspace.added_dirs`); `metadata["display"]="list"` renders the joined basenames. Hidden when empty. Color `blue`; no prefix; name "Added Dirs"; desc "Directories added via /add-dir".

### 4.4 `block-timer` rework -- `internal/widget/block_timer.go`

`getElapsed` precedence becomes:

1. `rate_limits.five_hour.resets_at` (authoritative): `blockStart = resetsAt - 5h`; `elapsed = now - blockStart`, clamped to `[0, 5h]`.
2. `cost.total_duration_ms` (current behavior) -- fallback when `rate_limits` is absent.
3. JSONL transcript first-entry timestamp -- last-ditch fallback (unchanged).

`block-timer` stays a *time* widget ("how far into the current 5-hour window, by time"). *Usage* % is the new `rate-limit-5h` widget's job -- keeping them separate avoids changing `block-timer`'s output meaning. Update the docstring and `Description()` to say it now prefers the official 5-hour window. (Note: there is some overlap between `block-timer`'s `reset`-ish behavior and `rate-limit-5h`'s `reset` mode -- that's acceptable; `block-timer` also serves non-Pro/Max users via fallbacks.)

### 4.5 `git-worktree`: add `workspace.git_worktree` source -- `internal/widget/git_worktree.go`

Render precedence:

1. `ctx.Data.Workspace.GitWorktree` -- official, general (any linked worktree). **NEW.**
2. `ctx.Data.Worktree.Name` -- official, `--worktree` sessions only. (current primary)
3. `ctx.Git.Worktree()` -- `git rev-parse --git-dir` heuristic. (current fallback)

### 4.6 `tokens-input` / `tokens-output` / `tokens-total`: relabel

No behavior change (the data source is still the correct field). Update `displayName` / `description` in the `internal/widget/widget.go` registry:

- `tokens-input` -> "Input tokens currently in context (includes cache reads/writes)". Doc note (RFC + README): since v2.1.132 this is ~= `context-length`.
- `tokens-output` -> "Output tokens from the most recent API response".
- `tokens-total` -> "Total tokens currently in the context window (input + output)".

Rationale for keeping the widgets (rather than removing/aliasing `tokens-input`): avoids churning the default config and existing user configs; `tokens-input` (the official rollup) and `context-length` (ccstatus's manual sum of `current_usage.*`) can still differ in the early-session edge case where `total_input_tokens` is set but `current_usage` is null. Per CLAUDE.md "breaking changes are acceptable", a future RFC may deprecate `tokens-input` if the redundancy proves not worth it.

### 4.7 `install`: `--refresh` / `--hide-vim-indicator`

`internal/claude/claude.go`, `cmd/ccstatus/cmd_install.go`:

```go
type StatusLine struct {
    Type                 string `json:"type"`
    Command              string `json:"command"`
    Padding              int    `json:"padding"`
    RefreshInterval      *int   `json:"refreshInterval,omitempty"`      // unset -> omitted
    HideVimModeIndicator *bool  `json:"hideVimModeIndicator,omitempty"` // unset -> omitted
}

// Install registers ccstatus. refreshInterval <= 0 leaves the field unset.
func Install(refreshInterval int, hideVimMode bool) (string, error) { ... }
```

`cmd_install.go` gains `--refresh N` (int, default 0) and `--hide-vim-indicator` (bool, default false). Default `ccstatus install` behavior is unchanged (neither key written). Update `claude_test.go` callers for the new `Install` signature.

### 4.8 No settings schema version bump

`config.CurrentVersion` stays `4`. Adding new widget *types* is additive -- a `version: 4` config that references only existing widgets remains valid; the new widgets are opt-in. (RFC 003 added widgets the same way without a bump.) The `install` / `StatusLine` changes touch *Claude Code's* settings.json, not ccstatus's, and are additive there too.

## Configuration Compatibility

- Existing `~/.config/ccstatus/settings.json` files keep working unchanged.
- New widgets must be added by the user (or via a future `init` default -- see the optional item below).
- `ccstatus install` without flags writes the same JSON as before.
- ccstatus still ignores unknown fields in settings.json (forward-compatible).

**Optional (to decide):** add `rate-limit-5h` (in `percent` mode) to `DefaultSettings()` line 1. It auto-hides for non-Pro/Max users (extractor returns `ok=false`), so it's low-risk and surfaces the headline feature for the target audience.

## Implementation Plan

1. `internal/status/status.go`: add the structs in section 4.1.
2. `internal/status/format.go`: add `FiveHourLimitPercent` / `SevenDayLimitPercent` (4.2).
3. `internal/widget/block_timer.go`: rework `getElapsed` (4.4); extract `formatShortDuration`.
4. `internal/widget/rate_limit.go` (new): `RateLimitWidget` (4.3).
5. `internal/widget/added_dirs.go` (new): `AddedDirsWidget` (4.3).
6. `internal/widget/git_worktree.go`: add `workspace.git_worktree` source (4.5).
7. `internal/widget/widget.go`: register `rate-limit-5h`, `rate-limit-7d`, `added-dirs`, and the `effort` / `thinking` / `session-name` `stringFieldWidget` literals; relabel `tokens-*` (4.3, 4.6).
8. `internal/claude/claude.go`: extend `StatusLine`, change `Install` signature (4.7).
9. `cmd/ccstatus/cmd_install.go`: add `--refresh` / `--hide-vim-indicator` (4.7).
10. (Optional) `internal/config/config.go`: add `rate-limit-5h` to `DefaultSettings()`.
11. Tests:
    - `internal/status/status_test.go`: parse the new fields (present / absent / null).
    - `internal/status/format_test.go`: `FiveHourLimitPercent` / `SevenDayLimitPercent`.
    - `internal/widget/widget_test.go`: the new widgets (rate-limit-5h/7d in `percent` -- asserted exactly -- and `reset` / `full` -- asserted on format/sign with a fixed `resets_at`; effort; thinking on/off; session-name present/absent; added-dirs count/list/empty); updated `git-worktree` precedence; updated `block-timer` precedence (`rate_limits.five_hour.resets_at` -> elapsed).
    - `internal/claude/claude_test.go`: `Install(5, true)` writes both keys; `Install(0, false)` omits both.
    - Optional: a `cmd/ccstatus` test for the new install flags (also begins closing the existing `cmd/` 0%-coverage gap).
12. Docs: `README.md` (widgets table + `tokens-*` descriptions + `refreshInterval` / `hideVimModeIndicator` + widget count 43) and `CLAUDE.md` (fix "36" -> 43; add new widgets and JSON sources; update `tokens-*` lines).
13. Flip RFC 004 Status -> Completed.

## Verification

```bash
go build ./... && go test ./... && go vet ./...
golangci-lint run            # watch dupl on the new widgets -- reuse the generic templates
gofumpt -l -w . && goimports -w -local github.com/moond4rk/ccstatus .

# manual: pipe a sample with the new fields, using a config that includes the new widgets
echo '{"model":{"id":"claude-opus-4-7","display_name":"Opus"},"session_name":"my-feat","effort":{"level":"high"},"thinking":{"enabled":true},"rate_limits":{"five_hour":{"used_percentage":45,"resets_at":<now+7200>},"seven_day":{"used_percentage":30,"resets_at":<now+200000>}},"workspace":{"current_dir":"/x","added_dirs":["/y","/z"],"git_worktree":"feat-1"},"context_window":{"used_percentage":25,"context_window_size":200000}}' | ./ccstatus

./ccstatus widgets           # expect 43 entries incl. the 6 new ones
CLAUDE_CONFIG_DIR=/tmp/cc-test ./ccstatus install --refresh 5 --hide-vim-indicator
cat /tmp/cc-test/settings.json   # verify refreshInterval / hideVimModeIndicator present
CLAUDE_CONFIG_DIR=/tmp/cc-test2 ./ccstatus install
cat /tmp/cc-test2/settings.json  # verify neither key present
# replay a real `ccstatus dump` capture if one is available
```

## Follow-up changes (post design review)

The default config and a few widget details were iterated on after the design above:

- Default config is leaner — line 1 `model · effort | Ctx-% | git-branch git-worktree | +/- | $cost`, line 2 `cwd | rate-limits | session-clock` (plain separator, left-aligned); the now-redundant `tokens-input` / `tokens-output` and `cache-hit-rate` were dropped from the default (still available to add).
- `compactThreshold` default raised 60 → 90 (the `-40` right margin is for the near-full "context low" warning; switching at 60% was too eager).
- `session-cost` shows `$0.34` — the `Cost: ` prefix was removed from the widget (the `$` is self-evident).
- Added a combined `rate-limits` widget — `5h: 3% / 7d: 12%`, each window shown only when present, default prefix `Limit ` — alongside the per-window `rate-limit-5h` / `rate-limit-7d` (default prefixes `5h limit: ` / `7d limit: `, plus a `bar` display mode via `metadata.barWidth`). Registry: 37 → 44 widgets.
- `session-name` is intentionally not in the default config (a `--name` / `/rename` value can be long enough to overflow a line).
- Terminal-width detection — the `terminalWidth` setting and an ancestor-TTY fallback — was handled separately in #22, since Claude Code does not pass the column width to the main status line command.

## References

- Claude Code statusline documentation: https://code.claude.com/docs/en/statusline
- RFC 001 (architecture), RFC 002 (CLI refactor), RFC 003 (remaining widgets)
- Original TypeScript implementation: https://github.com/sirmalloc/ccstatusline
