# RFC 005: Robustness, Official Schema Parity, and Subagent Status Lines

- Status: Completed
- Author: @moond4rk
- Date: 2026-10-17

## Summary

ccstatus v0.3.1 (2026-08-02) predates several additions to the official Claude Code status line schema: `prompt_cache`, `rate_limits.spend_limit`, `pr.kind` (GitLab merge requests), and a richer `subagentStatusLine` payload. Reviewing the codebase against the current docs also surfaced failure modes where the status line silently disappears. This RFC is the plan for v0.4.0 and v0.5.0, delivered as three PRs and a follow-up:

1. Foundation: toolchain cleanup (drop typos, Go 1.27, latest dependencies, gofumpt enforced in CI); robustness (never exit non-zero while rendering, tolerate schema drift, merge settings over defaults, keep OSC 8 links balanced when truncating); a lock-free, concurrent git layer.
2. Schema, widgets and rendering: model `prompt_cache`, `rate_limits.spend_limit` and `pr.kind`; widgets for them; fixes to `cache-hit-rate`, `rate-limits` and `pr`; threshold colors, progress bars and `merge`.
3. Subagents and tooling: `subagentStatusLine` support via `ccstatus subagent`; order-preserving atomic `install`, scoped `uninstall`, `dump --append`, `preview`.
4. Polish from daily use: the default layout and its shared usage color scale, `task-sparkline` drawn from per-tick growth, a bare `Fast`, and `maxWidth` on every widget.

## Motivation

### Schema changes since v0.3.1

| Change | Claude Code | ccstatus v0.3.1 |
| --- | --- | --- |
| `pr.kind: "mr"` for GitLab merge requests | v2.1.234 | Not parsed; merge requests render as `#N` |
| `prompt_cache` object | v2.1.251 (`last_miss_cause`, `miss_causes` in v2.1.260) | Not parsed |
| `rate_limits.spend_limit` (`used_percentage`, `resets_at`) | v2.1.251 | Not parsed; gateway users see no limit at all |
| `spend_limit.used_usd` / `limit_usd` / `period` | v2.1.284 | Not parsed |
| `subagentStatusLine` payload gains `effort` and `agentType` | v2.1.214, v2.1.293 | Setting not supported |
| Re-run when a rate-limit `resets_at` or the prompt cache `expires_at` passes | - | No change needed |
| `workspace.repo.owner` holds nested GitLab groups (`group/subgroup`) | v2.1.260 | Already works |
| `/clear` resets cost, `modelPricing`, usage above 100% fixed | v2.1.211, v2.1.243, v2.1.299 | No change needed |

### Problems found in review

All reproduced against v0.3.1.

1. Any error exits 1, which blanks every row: Claude Code shows the output only after the command exits 0. A trailing comma in the hand-edited settings.json, or a single field changing type (for example a fractional `resets_at`), removes the whole status line, and the error is visible only under `claude --debug`.
2. A settings file that omits fields gets zero values instead of defaults: without `colorLevel` there is no color, without `defaultPadding` widgets are glued together (`Opus|Ctx: 8%`), and `compactThreshold: 0` keeps `full-until-compact` permanently in its narrow mode.
3. `Truncate` can cut inside an OSC 8 hyperlink and never close it, so the ellipsis and whatever Claude Code draws next become part of the link.
4. `git status --porcelain` and `git diff --shortstat` rewrite `.git/index` when stat data is stale, taking `index.lock` and racing Claude's own `git add` / `git commit`. `GIT_OPTIONAL_LOCKS=0` prevents this for `status` but not for `diff`.
5. Git commands run serially with a 5s timeout each, while Claude Code cancels an in-flight status line command whenever a new update arrives.
6. `merge` is documented in the README but not implemented.
7. `cache-hit-rate` uses the last request's `current_usage`, which can differ wildly from the session ratio Claude Code now reports (13% versus 91% on the docs' sample payload).
8. `uninstall` deletes any `statusLine`, even one that no longer runs ccstatus, and `install` rewrites `~/.claude/settings.json` with sorted keys and a non-atomic write.
9. 5h/7d percentages are not clamped (`104%`), and the 7-day reset countdown renders as `111h12m`.

## Decisions

- Remove typos (spell checking) entirely: config file, workflow, and CI step.
- `cache-hit-rate` switches to `prompt_cache.hit_ratio`. This changes the reported number; the old formula remains the fallback for Claude Code versions without `prompt_cache`.
- Implement `merge` rather than removing it.
- A fresh `install` writes `refreshInterval: 30`, since the default layout includes `session-clock`.
- `prompt-cache` ships as an opt-in widget, not in the default layout.
- No Windows support in this RFC.

## Roadmap

| PR | Scope | Release |
| --- | --- | --- |
| 1 (#37) | Foundation: toolchain cleanup, robustness, git layer | v0.4.0 |
| 2 (#38) | Schema parity, new and updated widgets, rendering (thresholds, bars, merge) | v0.4.0 |
| 3 (#39) | Subagent status lines, tooling | v0.5.0 |
| 4 (#41) | Polish from daily use: default layout, `task-sparkline`, generic `maxWidth` | v0.5.0 |

The PRs land in order and each ships with tests. PR 3 builds on the lenient parser from PR 1 and the threshold colors from PR 2; PR 4 retunes the defaults of PRs 2 and 3.

## Design

### PR 1: Foundation

#### Toolchain cleanup

- Delete `.typos.toml`, `.github/workflows/typos.yml`, the CI spelling step, and the `typos` entry in CLAUDE.md.
- `go.mod`: `go 1.27.2`, every module at its latest version (supersedes Dependabot PR #36).
- CI: golangci-lint `v2.15`; enable the `gofumpt` formatter so CI enforces the formatting CLAUDE.md prescribes.
- README: `terminalWidth` no longer claims Claude Code withholds the width; it sets `COLUMNS` since v2.1.153.

#### Robustness

**Never blank the status line.** In render mode `ccstatus` always exits 0:

| Failure | Output |
| --- | --- |
| stdin is not JSON | One red line: `ccstatus: invalid input: <error>` |
| settings.json unreadable or fails to parse | A red first line such as `ccstatus: settings.json:3:41: invalid character ']' (run ccstatus validate)`, followed by the rows rendered with the built-in defaults |
| A field has an unexpected type | That field is dropped and noted on stderr; everything else renders |

`dump` goes through the same render path and reads the command's stdin. `validate`, `init`, `install` and `uninstall` keep their non-zero exit codes.

**Lenient parsing.** `status.Parse` decodes strictly first, which stays the fast path. If that fails with a type error, it decodes again leniently, walking the struct by reflection: every object field is decoded on its own, so a bad field only drops itself. A number with a fraction or a numeric string is coerced into an integer field, a numeric string into a float field, and a number into a string field. Custom unmarshalers such as `ModelField` still run.

**Settings over defaults.**

```go
s := DefaultSettings()
s.Lines = nil
if err := json.Unmarshal(data, &s); err != nil {
	return DefaultSettings(), err
}
if s.Lines == nil {
	s.Lines = DefaultSettings().Lines
}
```

An explicit `"lines": []` still renders nothing.

**Balanced links.** `Truncate` tracks whether it is inside an OSC 8 hyperlink (an `ESC ] 8 ; params ; URI` sequence with a non-empty URI opens one, an empty URI closes it) and emits `ESC ] 8 ; ; BEL` before the reset and ellipsis when it cuts inside one.

**Clamping.** 5h/7d `used_percentage` is clamped to 0-100 for text and bars. Spend limits are not clamped (see PR 2).

**validate.** Report JSON syntax errors as `line:col`. Also check `flexMode`, `colorLevel` (0-3), `compactThreshold` (0-100) and `metadata.display` values (widgets list their modes through an optional, unexported `displayModes() []string`, which `validate` reads through `widget.DisplayModes`). PR 2 adds `thresholds` checks, PR 3 adds widget scope checks for `subagentLine`, and PR 4 rejects a negative `maxWidth`.

#### Git layer

- Every git command runs with `GIT_OPTIONAL_LOCKS=0`.
- `Diff` becomes a single `git diff-index --numstat HEAD`: the same staged-plus-unstaged total as today, one process instead of two, and it never writes the index. `--numstat` also replaces `--shortstat`, whose summary line is localized and would not parse under a non-English locale. With an unborn HEAD it falls back to `git diff-files` plus `git diff --cached`.
- The per-command timeout drops from 5s to 2s.
- Before rendering, ccstatus collects the git facts the configured widgets need and fetches them concurrently; `GitCache` guards each fact with a `sync.Once`.
- `git-worktree` trusts the JSON when `version` is at least 2.1.98, which always sets `workspace.git_worktree` inside a linked worktree, and skips `git rev-parse`.
- Deferred: a cross-invocation cache keyed by `session_id`, to revisit if large repositories still render slowly.

### PR 2: Schema, widgets and rendering

#### Schema parity

```go
type Session struct {
	// ... existing fields ...
	PromptCache *PromptCache `json:"prompt_cache,omitempty"`
}

type RateLimits struct {
	FiveHour   *RateLimitWindow `json:"five_hour,omitempty"`
	SevenDay   *RateLimitWindow `json:"seven_day,omitempty"`
	SpendLimit *SpendLimit      `json:"spend_limit,omitempty"`
}

// SpendLimit is the Claude apps gateway spend limit. UsedPercentage can exceed
// 100; the USD fields and Period arrive on a separate schedule and may be absent.
type SpendLimit struct {
	RateLimitWindow
	UsedUSD  *float64 `json:"used_usd,omitempty"`
	LimitUSD *float64 `json:"limit_usd,omitempty"`
	Period   string   `json:"period,omitempty"` // daily | weekly | monthly
}

type PRInfo struct {
	Number      *int   `json:"number,omitempty"`
	URL         string `json:"url,omitempty"`
	ReviewState string `json:"review_state,omitempty"`
	Kind        string `json:"kind,omitempty"` // "mr" for GitLab merge requests, absent for GitHub
}

// PromptCache summarizes the main conversation's prompt cache. Timestamps are
// Unix epoch seconds; nullable fields are pointers.
type PromptCache struct {
	Warm                *bool          `json:"warm,omitempty"`
	CachingObserved     *bool          `json:"caching_observed,omitempty"`
	TTL                 string         `json:"ttl,omitempty"` // "5m" | "1h"
	ExpiresAt           *int64         `json:"expires_at,omitempty"`
	Requests            *int           `json:"requests,omitempty"`
	Misses              *int           `json:"misses,omitempty"`
	ExpectedRebuilds    *int           `json:"expected_rebuilds,omitempty"`
	HitRatio            *float64       `json:"hit_ratio,omitempty"` // 0-1
	CacheWriteTokens    *int           `json:"cache_write_tokens,omitempty"`
	MissRecacheTokens   *int           `json:"miss_recache_tokens,omitempty"`
	LastMissAt          *int64         `json:"last_miss_at,omitempty"`
	LastMissCause       *MissCause     `json:"last_miss_cause,omitempty"`
	MissCauses          map[string]int `json:"miss_causes,omitempty"`
	RecacheTokensIfCold *int           `json:"recache_tokens_if_cold,omitempty"`
}

// MissCause is the likely cause of the most recent cache miss.
type MissCause struct {
	Causes          []string `json:"causes,omitempty"` // tools_changed, system_prompt_changed, ttl_expired_5m, likely_server_side, ...
	ToolsAdded      *int     `json:"tools_added,omitempty"`
	ToolsRemoved    *int     `json:"tools_removed,omitempty"`
	SystemCharDelta *int     `json:"system_char_delta,omitempty"`
}
```

The docs' full sample payload is checked in as `internal/status/testdata/statusline_full.json` and becomes the shared fixture for parser and widget tests. `status.Parse` gets a fuzz target that must never panic.

#### New and updated widgets

| Widget | Source | Default output | `metadata.display` |
| --- | --- | --- | --- |
| `prompt-cache` (new) | `prompt_cache` | `warm 42m` / `cold` | `status` (default), `ttl`, `hit`, `misses`, `full` |
| `spend-limit` (new) | `rate_limits.spend_limit` | `$314.12 / $500.00`, or `63%` before the USD fields arrive | `amount` (default), `percent`, `bar`, `reset`, `full` |
| `session-lines` (new) | `cost.total_lines_added` / `total_lines_removed` | `+156/-23` | - |
| `cache-hit-rate` (changed) | `prompt_cache.hit_ratio`, falling back to `current_usage` | `91%` | - |
| `rate-limits` (changed) | Adds the spend limit | `5h: 24% / 7d: 41% / spend: $314.12/$500.00` | - |
| `pr` (changed) | `pr.kind` | `!1234 pending` for merge requests | - |

- `prompt-cache` is hidden when `prompt_cache` is absent or `caching_observed` is false. `status` shows `warm <time until expires_at>` (just `warm` when `expires_at` is null) or `cold`; a countdown that has run out renders `cold`. `misses` renders `2 misses (tools_changed)` from `last_miss_cause.causes`; `full` joins status, hit ratio and misses with `·`. The countdown only advances with a `refreshInterval`, but Claude Code re-runs the command at `expires_at` itself, so the warm-to-cold flip is immediate.
- `spend-limit` reuses the reset countdown through the embedded `RateLimitWindow` and the shared bar helper, but formats its own percentage, which is never clamped (being over the limit is meaningful). `full` mode adds the period (`$314.12 / $500.00 monthly, resets 12d`).
- `session-lines` counts Claude's edits this session; the existing `lines-*` widgets keep showing the uncommitted git diff.
- Durations of a day or more render with days (`4d15h`): reset countdowns, the session clock and API time share one formatter.

#### Rendering

**Threshold colors.**

```json
{
  "id": "4",
  "type": "context-percentage",
  "color": "green",
  "thresholds": [
    { "gte": 70, "color": "yellow" },
    { "gte": 90, "color": "red", "bold": true }
  ]
}
```

- `WidgetItem` gains `Thresholds []Threshold` with `gte`, `color`, `backgroundColor` and `bold`.
- Numeric widgets implement an optional interface:

  ```go
  // Valuer exposes the number a widget renders, for threshold colors.
  type Valuer interface {
  	Value(item *config.WidgetItem, ctx RenderContext) (float64, bool)
  }
  ```

  Implemented by the percentage widgets, the rate-limit widgets, `spend-limit`, `prompt-cache` (hit percentage in `hit` mode, minutes left otherwise), `session-cost`, `context-length`, the token widgets, and later `task-tokens` and `task-context`.

- The renderer applies the threshold with the highest `gte` that is at most the value; below the first threshold the item's normal color applies. `overrideForegroundColor` and `overrideBackgroundColor` still win.
- Widgets whose default color depends on state implement the optional `DynamicDefaultColor` interface (`DefaultColorFor(ctx RenderContext) string`), consulted when the item sets no color: `prompt-cache` is green while warm and `brightBlack` once cold, and `task-status` (PR 3) colors by task state.

**Bars.** `context-percentage`, `remaining-percentage`, `context-percentage-usable` and `cache-hit-rate` accept `metadata.display: "bar"` (`▓▓▓░░░░░░░ 32%`, width from `metadata.barWidth`), sharing one bar helper with the rate-limit widgets.

**merge.** `merge: true`, or the legacy `"no-padding"`, joins a widget to the next one without `defaultPadding`. Separators are widgets and are unaffected.

### PR 3: Subagents and tooling

#### Subagent status lines

`subagentStatusLine` runs a command on every refresh tick with the base hook fields, `columns` and `tasks[]` on stdin. The command prints one JSON line per row it overrides, `{"id": "<task id>", "content": "<row>"}`. An omitted id keeps Claude Code's default row; an empty `content` hides the row.

**Command.** `ccstatus subagent` reads the payload, renders `subagentLine` once per task at width `columns`, and prints the JSON lines. A task whose row renders empty is omitted, keeping the default row. On any error it prints nothing and exits 0, so Claude Code falls back to its default rows. It runs no git commands and reads no transcript. The payload goes through the same lenient decoder as the main status line, which now also decodes arrays element by element, so one malformed task drops only itself.

**Input.**

```go
type SubagentPayload struct {
	SessionID      string `json:"session_id,omitempty"`
	TranscriptPath string `json:"transcript_path,omitempty"`
	Cwd            string `json:"cwd,omitempty"`
	Columns        int    `json:"columns,omitempty"`
	Tasks          []Task `json:"tasks"`
}

type Task struct {
	ID                string      `json:"id"`
	Name              string      `json:"name,omitempty"`
	Type              string      `json:"type,omitempty"`      // local_agent
	AgentType         string      `json:"agentType,omitempty"` // v2.1.293+
	Status            string      `json:"status,omitempty"`    // running | completed | failed | killed
	Description       string      `json:"description,omitempty"`
	Label             string      `json:"label,omitempty"`
	StartTime         int64       `json:"startTime,omitempty"` // epoch milliseconds
	Model             string      `json:"model,omitempty"`
	Effort            *TaskEffort `json:"effort,omitempty"` // level string or numeric token budget
	ContextWindowSize *int        `json:"contextWindowSize,omitempty"`
	TokenCount        int         `json:"tokenCount,omitempty"`
	TokenSamples      []int       `json:"tokenSamples,omitempty"`
	Cwd               string      `json:"cwd,omitempty"`
}
```

**Widgets.** They read `RenderContext.Task` and render empty on the main status line.

| Widget | Output |
| --- | --- |
| `task-name` | `name`, else `agentType`, else `type` |
| `task-type` | `agentType` |
| `task-status` | `running` / `completed` / `failed` / `killed`, colored by state unless `color` is set |
| `task-label` | `label`, falling back to `description` |
| `task-elapsed` | Time since `startTime` (`45s`, `1m23s`, `1h02m`); empty unless the task is running, since the payload has no end time |
| `task-model` | Short model name from `model` (`Sonnet 5.5`); the raw id with `rawValue` |
| `task-effort` | `high`, or a token budget such as `32k` |
| `task-tokens` | `tokenCount` formatted like the token widgets (`45.2k`) |
| `task-context` | `tokenCount / contextWindowSize` as a percentage; supports `bar` and thresholds |
| `task-sparkline` | Tokens added between `tokenSamples` readings as `▂▅█▁`, running tasks only |
| `task-cwd` | Base name of `cwd` |

`separator`, `flex-separator` and `custom-text` work on both kinds of line. `validate` rejects main-line widgets in `subagentLine` and task widgets in `lines`.

**Config.** A new top-level `subagentLine`, one row of widget items; the built-in default (as revised in PR 4) applies when it is absent:

```json
"subagentLine": [
  { "id": "s1", "type": "task-name", "color": "cyan" },
  { "id": "s2", "type": "task-status" },
  { "id": "s3", "type": "task-elapsed" },
  { "id": "s4", "type": "separator" },
  { "id": "s5", "type": "task-model" },
  { "id": "s6", "type": "task-effort", "prefix": "· " },
  { "id": "s7", "type": "task-tokens" },
  { "id": "s8", "type": "task-context", "color": "brightBlack", "thresholds": [{ "gte": 30, "color": "white" }, { "gte": 60, "color": "yellow" }, { "gte": 90, "color": "red" }] },
  { "id": "s9", "type": "task-sparkline" },
  { "id": "s10", "type": "separator" },
  { "id": "s11", "type": "task-label" }
]
```

This renders like `reviewer running 1m23s | Sonnet 5.5 · high 45.2k 23% ▅▄▆▇██▇ | Reading internal/auth/session.go`; the label comes last so truncation cuts it first.

**Rendering.** A `render.RenderRow(items, settings, ctx, width)` entry point renders at exactly `width`, without a flex-mode margin since `columns` is already the usable width, and reuses separator cleanup, colors, thresholds, flex expansion and truncation.

**Install.** `ccstatus install --subagent` also writes `"subagentStatusLine": {"type": "command", "command": "ccstatus subagent"}`.

#### Tooling

- `install` and `uninstall` edit `~/.claude/settings.json` in place: the top-level object is decoded as ordered key and raw-value pairs, only `statusLine` / `subagentStatusLine` change, the order of keys and every other value are preserved, and the file is replaced atomically (temp file plus rename, keeping its mode). A symlinked `settings.json`, common with dotfiles managers, is resolved first so the rename replaces the target and the link survives.
- `uninstall` removes `statusLine` / `subagentStatusLine` only when their command runs ccstatus.
- A fresh `install` writes `refreshInterval: 30`; an existing value is kept; `--refresh 0` removes it.
- `dump --append` appends one JSON line per invocation (`ts`, `columns`, `lines`, `input`) instead of overwriting, so a whole session can be captured.
- `preview` renders the current settings against an embedded sample payload (the docs' full schema, from the new `internal/sample` package, with its timestamps moved to now) or `--input FILE`, at `--width N` or the terminal width; `preview --subagent` does the same for `subagentLine` with a sample task list. The docs payload moves from `internal/status/testdata` into `internal/sample` and stays the parser's test fixture.

### PR 4: Polish from daily use

- `task-sparkline` draws the tokens added between `tokenSamples` readings. Scaling the raw counts from zero drew a finished task (16 equal readings) and a fork (a large inherited context) as a full bar. `▁` marks a tick with no growth, any growth is at least `▂`, and a compaction counts as none. Like `task-elapsed`, it is empty unless the task is running.
- `fast-mode` renders a bare `Fast` instead of `Fast: on`.
- `maxWidth` caps the text of any widget, not just `custom-command`: the cut ends with `…`, the prefix and suffix are not counted, wide characters count as two columns, and a cut OSC 8 link is closed. `custom-command` drops its own rune cut in favor of it, and `validate` rejects negative values.

## Default Layout

- PR 2: `rate-limits` picks up the spend limit automatically; `prompt-cache`, `spend-limit` and `session-lines` are opt-in.
- PR 3: the `subagentLine` default.
- PR 4: `context-percentage` and `task-context` share one scale: a `brightBlack` base, white at 30, yellow at 60 and red at 90; `rate-limits` takes the same steps a shade brighter (white, then bright white at 30). Line 1 gains `fast-mode` after the effort, line 2 ends with `session-name` capped at 30 columns, and the subagent row shows `task-effort` after the model.

## Testing

- Shared fixture: the docs' full payload, with absent and null variants for every widget.
- Fuzz targets: `status.Parse` never panics; `render.Truncate` never exceeds the width and always leaves OSC 8 links balanced.
- Git integration tests assert that `.git/index` is not rewritten and that `Diff` matches the staged-plus-unstaged sum.
- `cmd/ccstatus` tests: malformed stdin, malformed settings and type drift all exit 0 (the first two with an error line, type drift by dropping the field with a note on stderr), which also lifts `cmd/` coverage from 44.6%.
- Subagent: golden JSON-lines output for a sample payload, omitted empty rows, and no output for invalid input.

## Out of Scope

- Windows builds; `custom-command` relies on `sh -c`.
- A cross-invocation git cache (see the PR 1 git layer).
- Sanitizing string fields in Claude Code's JSON.
- Feature parity with the TypeScript ccstatusline.

## Verification

```bash
go build ./... && go vet ./... && go test -race ./...
golangci-lint run

# Render the docs' sample payload
ccstatus preview
ccstatus preview --subagent

# Robustness: each prints an error line and exits 0
echo 'not json' | ccstatus; echo "exit=$?"

# Capture a real session: point statusLine.command at "ccstatus dump --append"
```

## References

- Status line documentation: https://code.claude.com/docs/en/statusline
- Hook common input fields (base of the subagent payload): https://code.claude.com/docs/en/hooks#common-input-fields
- Claude Code changelog: v2.1.153 (`COLUMNS` / `LINES`), v2.1.234 (GitLab merge requests), v2.1.251 (`prompt_cache`, `spend_limit`), v2.1.260 (cache miss causes), v2.1.284 (spend USD fields), v2.1.293 (`agentType`)
- RFC 004 (previous schema follow-up)
