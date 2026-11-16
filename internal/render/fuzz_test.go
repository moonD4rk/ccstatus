package render

import (
	"slices"
	"testing"
	"time"

	"github.com/moond4rk/ccstatus/internal/color"
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/git"
	"github.com/moond4rk/ccstatus/internal/sample"
	"github.com/moond4rk/ccstatus/internal/status"
	"github.com/moond4rk/ccstatus/internal/widget"
)

// fuzzGit answers the git widgets without running git.
type fuzzGit struct{ branch string }

func (g fuzzGit) Branch() string   { return g.branch }
func (fuzzGit) Changes() int       { return 3 }
func (fuzzGit) Diff() git.DiffStat { return git.DiffStat{Added: 42, Removed: 7} }
func (fuzzGit) Worktree() string   { return "wt" }

// fuzzItems lays out every widget type that runs no command, with options
// picked by the bits of layout and text as the free-form strings.
func fuzzItems(types []string, layout uint64, text string) []config.WidgetItem {
	items := make([]config.WidgetItem, 0, 2*len(types))
	for i, typ := range types {
		bits := layout >> (uint(i) % 48)
		item := config.WidgetItem{Type: typ, RawValue: bits&1 != 0, Merge: bits&2 != 0, MaxWidth: int(bits >> 2 & 15)}
		if modes, ok := widget.DisplayModes(typ); ok {
			item.Display = modes[int(bits>>6)%len(modes)]
			item.BarWidth = int(bits >> 8 & 31)
		}
		if bits&0x2000 != 0 {
			item.Prefix = &text
			item.Suffix = text
		}
		if widget.TakesThresholds(typ) {
			item.Thresholds = []config.Threshold{{GTE: float64(bits & 127), Color: "red", Bold: true}}
		}
		switch typ {
		case widget.TypeCustomText:
			item.CustomText = text
		case widget.TypeSeparator, "git-branch":
			item.Character = text
		}
		items = append(items, item)
		switch bits >> 14 & 7 {
		case 0:
			items = append(items, config.WidgetItem{Type: widget.TypeFlexSeparator})
		case 1, 2:
			items = append(items, config.WidgetItem{Type: widget.TypeSeparator})
		}
	}
	return items
}

// FuzzRender renders every widget, in layouts and with payloads the fuzzer
// picks, and checks the two promises of the pipeline: it does not panic, and a
// row never exceeds the width it was given.
func FuzzRender(f *testing.F) {
	now := time.Unix(1_800_000_000, 0)
	f.Add(sample.StatusLine(now), sample.Subagent(now), uint64(0), 120, "· ")
	f.Add(sample.StatusLine(now), sample.Subagent(now), ^uint64(0), 37, "\u26a0\ufe0f \U0001F468\u200d\U0001F469\u200d\U0001F467 中文")
	f.Add([]byte(`{"model":"claude-opus-4-7[1m]","context_window":{"used_percentage":150,"current_usage":{}}}`),
		[]byte(`{"columns":20,"tasks":[{"id":"a","status":"running","startTime":-5,"tokenSamples":[9,1,1e9],"model":"claude-"}]}`),
		uint64(0x5555_5555_5555_5555), 1, "\x1b]8;;https://example.com\x07link\x1b]8;;\x07")
	f.Add([]byte(`{"rate_limits":{"five_hour":{"used_percentage":-3,"resets_at":1},"spend_limit":{"used_percentage":1e300}},"cost":{"total_duration_ms":-1}}`),
		[]byte(`{"tasks":[{"id":"b","contextWindowSize":0,"tokenCount":-1,"effort":0}]}`), uint64(1)<<40, 0, "")

	mainTypes := slices.DeleteFunc(widget.Types(), func(t string) bool {
		return t == widget.TypeCustomCommand || widget.IsTaskWidget(t)
	})
	slices.Sort(mainTypes)
	taskTypes := slices.DeleteFunc(widget.Types(), func(t string) bool {
		return !widget.IsTaskWidget(t) && !widget.IsSharedWidget(t)
	})
	slices.Sort(taskTypes)

	f.Fuzz(func(t *testing.T, payload, tasks []byte, layout uint64, width int, text string) {
		if width < 0 || width > 400 || len(text) > 200 {
			return
		}
		settings := config.DefaultSettings()
		settings.FlexMode = config.FlexExact
		settings.ColorLevel = int(layout >> 60 & 3)
		settings.InheritSeparatorColors = layout>>62&1 != 0
		check := func(row string) {
			if width > 0 && color.VisibleWidth(row) > width {
				t.Fatalf("row is %d columns wide, over %d: %q", color.VisibleWidth(row), width, row)
			}
			PostProcess(row)
		}

		if session, _, err := status.Parse(payload); err == nil {
			ctx := widget.RenderContext{
				Data: session, TerminalWidth: width, Now: now,
				Git: widget.NewGitCacheWithProvider(fuzzGit{branch: text}), RawInput: payload,
			}
			check(RenderLine(fuzzItems(mainTypes, layout, text), &settings, ctx))
		}
		if rows, _, err := status.ParseSubagent(tasks); err == nil {
			for i := range rows.Tasks {
				ctx := widget.RenderContext{Task: &rows.Tasks[i], Now: now}
				check(RenderRow(fuzzItems(taskTypes, layout, text), &settings, ctx, width))
			}
		}
	})
}
