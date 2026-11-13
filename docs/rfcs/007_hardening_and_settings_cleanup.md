# RFC 007: Hardening and Settings Cleanup

- Status: Completed
- Author: @moond4rk
- Date: 2026-11-13

## Summary

A review of ccstatus after RFC 006 found the project healthy: lint clean, 80-97% test coverage, and the status line schema fully covered. It also found bugs that reproduce on the render path, leftovers from the interactive TUI this project never had, and a `validate` that accepts most mistakes a hand-edited `settings.json` can hold. Four PRs, each building on the one before:

1. Fixes. No setting changes.
2. Internal cleanup. No setting changes.
3. The settings schema. The one breaking PR, with a validator that explains each break.
4. Colors, a JSON Schema, and tooling.

## Motivation

| Finding | Before |
| --- | --- |
| `custom-command` kills only the shell at its timeout | `sleep 3; echo late` with `timeout: 300` blocks the status line for 3 s |
| Nothing recovers a panic while rendering | `barWidth: "9000000000000000000"` exits 2, and Claude Code blanks the status line |
| Separators next to a `flex-separator` are kept | `Opus \|        \| $0.01` when the widgets between them render empty |
| An emoji such as U+26A0 U+FE0F is measured one column wide | A widget capped with `maxWidth` overflows by a column per emoji |
| A `flex-separator` with no room left adds no space | `AAAA` and `BBBB` render as `AAAABBBB` |
| `ccstatus` reads stdin even when it is a terminal | Run by hand, it hangs until end-of-file |
| `git-branch` asks for `rev-parse --abbrev-ref HEAD` | `HEAD` on a detached HEAD, nothing before the first commit |
| `task-model` only reads plain model IDs | `claude-sonnet-4-5@20250929` renders `Sonnet 4`; Bedrock IDs render unchanged |
| `validate` decodes loosely and checks few fields | A misspelled `colour`, an unknown `metadata` key and `rawValue` on a widget without a raw mode all pass |
| `prefix: ""` means "use the default" | A default prefix such as `Ctx: ` cannot be removed |
| `id` is required but never read; `version` is written but never read | Hand-maintained noise in every settings file |
| `block-timer` falls back to the session duration | A 7-hour session shows `5h/5h` on accounts without rate limits |
| `colorLevel` 1, 2 and 3 behave the same | Only the 16 named colors exist |

## Design

### PR 1: Fixes

- External commands run through `internal/proc`, which starts them in their own process group and kills the group at the deadline. A descendant that still holds the output pipe delays the result by at most 100 ms, and what the command itself printed is kept.
- Rendering recovers a panic into a red `ccstatus: internal error` line and logs the stack on stderr; `ccstatus subagent` prints no rows instead. `barWidth` is capped at 100 and a row at 10,000 columns.
- A `flex-separator` ends a group the way the edge of the row does: separators beside it are dropped, and neighboring groups always keep one column between them, so a row with no room is truncated rather than joined.
- Width is counted per grapheme cluster, and a symbol followed by U+FE0F counts as two columns. Truncation cuts between clusters with the same measure.
- With a terminal on stdin, `ccstatus`, `ccstatus subagent` and `ccstatus dump` print their help.
- `git-branch` uses `git symbolic-ref --short -q HEAD`, which also names a branch with no commits, and falls back to the short commit hash on a detached HEAD.
- `task-model` reads model IDs wrapped by a provider: a Bedrock prefix and `-v1:0` suffix, a Vertex `@date` suffix, a gateway `anthropic/` prefix.
- Smaller fixes: `session-cost` renders `$0.00` rather than `$0.0000`; `task-elapsed` formats hours like every other duration (`1h2m`); an inherited separator color follows the threshold color of the widget before it; with no home directory, commands fail instead of using paths relative to the working directory.

### PR 2: Internal cleanup

- `Widget` shrinks to `Render`, `DefaultColor` and `Description`. A default prefix and a raw mode become optional interfaces; `DisplayName`, `DefaultSuffix` and `noAffix` go, as nothing reads them.
- `block-timer` reads only `rate_limits.five_hour.resets_at` and is hidden without it. The session-duration and transcript fallbacks described a different quantity, and `internal/jsonl` goes with them.
- Terminal width is `terminalWidth`, then `COLUMNS`, then the attached terminal, then 80. The `/dev/tty` and `ps` ancestor probes predate the `COLUMNS` contract of Claude Code v2.1.153.
- `RenderContext.Now` carries the render time, so widgets never read the clock and a whole render is reproducible in tests. `preview` uses one instant, in whole seconds, for its sample and its render; before, a countdown could read a minute short.
- The formatters widgets share move into one file, and the `separator` and `flex-separator` type names are defined once.

### PR 3: Settings schema

| Change | Migration |
| --- | --- |
| `id` and `version` are removed | Delete them |
| `metadata.display` and `metadata.barWidth` become `display` (string) and `barWidth` (number) | Move the two keys up and drop the quotes around the number |
| `merge` is a boolean | `"no-padding"` becomes `true` |
| `prefix` distinguishes absent from empty | `"prefix": ""` now removes a default prefix |
| `block-timer` takes `time`, `bar` and `percent`, honors `barWidth` and supports thresholds | `progress` becomes `bar`, `percentage` becomes `percent` |
| `flexMode` gains `exact`: the terminal width with no margin | For Claude Code's fullscreen rendering, where notifications have their own row |

Rendering stays lenient: it ignores keys it does not know, so a file from an earlier version still renders. `validate` reads the file strictly instead and reports, with its location, every unknown key (naming the replacement for a removed one and the key a typo resembles), every key set twice, every option that has no effect on its widget (`rawValue`, `display`, `barWidth`, `character`, `customText`, `commandPath`, `timeout`, `preserveColors`), a `custom-text` or `custom-command` without its text or command, a threshold without `gte`, and out-of-range numbers. Without a settings file it says that the defaults are in use.

### PR 4: Colors, JSON Schema and tooling

- Colors also take `#rrggbb` (or `#rgb`) and a palette index `"0"`-`"255"`. `colorLevel` 1 emits the 16 ANSI colors, 2 the 256-color palette and 3 true color; a color the level cannot express is mapped to the nearest one it can. Named colors are the same at every level, so existing settings render as before. `fatih/color` is dropped.
- `ccstatus schema` prints a JSON Schema (draft-07) generated from the settings types and the widget registry: the widget types with what each shows, per type its display modes and the options that have no effect on it, the color forms and the numeric ranges. It is committed as `schema.json`, a test keeps it current, and `init` writes a `$schema` reference to it.
- `preview` renders the git widgets from sample data, since the sample payload's directories do not exist. `install` warns when `ccstatus` is not on `PATH`, and keeps a command that names the binary by absolute path. `pr` is colored by review state.
- Saving settings keeps an explicit empty `defaultPadding`, `defaultSeparator` and `subagentLine`; they used to come back as their defaults.
- Release and CI: `homebrew_casks` replaces the deprecated `brews`, with a post-install step that clears the quarantine attribute on macOS because the binary is not notarized; the release workflow runs the tests and `goreleaser check` and mints the tap token before it pushes the tag; CI checks `go mod tidy` and runs `govulncheck`, and drops the format step the linter already covers. Fuzz tests cover `config.Parse`, `validate` and the render pipeline.

After the first release with the cask, the tap needs one manual change: delete `Formula/ccstatus.rb` and add a `tap_migrations.json` with `{ "ccstatus": "ccstatus" }`, so existing installs move to the cask on upgrade.

## Out of Scope

- `lte` thresholds, comments in `settings.json`, and more git widgets.
- Sanitizing strings from the Claude Code payload.
- Windows builds.

## Verification

```bash
go build ./... && go vet ./... && go test -race ./...
golangci-lint run

# PR 1: returns within the timeout and renders the rest of the line
echo '{}' | ccstatus   # with a custom-command of "sleep 3; echo late" and "timeout": 300

# PR 3: lists every problem with its location and exits non-zero
ccstatus validate
```

## References

- Status line documentation: https://code.claude.com/docs/en/statusline
- RFC 006 (previous plan)
