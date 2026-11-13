// Package render implements the status line rendering pipeline.
package render

import (
	"strings"

	"github.com/moond4rk/ccstatus/internal/color"
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/widget"
)

const (
	// flexFullPadding is the padding subtracted in "full" and "full-until-compact" flex modes.
	flexFullPadding = 10
	// flexCompactPadding is the padding subtracted in "full-minus-40" mode and compact state.
	flexCompactPadding = 40
	// minFlexWidth is the floor for the resolved width so truncation always runs.
	minFlexWidth = 1
	// maxRowWidth caps the width a row is laid out to, so a runaway terminalWidth,
	// COLUMNS or --width cannot make flex padding allocate without bound.
	maxRowWidth = 10_000
)

// segment holds a rendered widget's text and metadata for the pipeline. value is
// the widget's number when the item has thresholds and the widget exposes one.
type segment struct {
	text     string
	item     *config.WidgetItem
	isSep    bool
	value    float64
	hasValue bool
}

// RenderLine renders a single line of widgets into an ANSI-colored string.
func RenderLine(items []config.WidgetItem, settings *config.Settings, ctx widget.RenderContext) string {
	// Apply flex mode width adjustment to match Claude Code's actual display area.
	if ctx.TerminalWidth > 0 && settings != nil {
		contextPct := 0.0
		if ctx.Data != nil && ctx.Data.ContextWindow != nil && ctx.Data.ContextWindow.UsedPercentage != nil {
			contextPct = *ctx.Data.ContextWindow.UsedPercentage
		}
		ctx.TerminalWidth = CalculateFlexWidth(ctx.TerminalWidth, settings.FlexMode, settings.CompactThreshold, contextPct)
	}
	return renderRow(items, settings, ctx)
}

// RenderRow renders items at exactly width columns, without the flex-mode
// margin, for rows whose usable width is already known, such as the subagent
// rows Claude Code sizes with its "columns" field. A width of 0 disables
// truncation and flex expansion.
func RenderRow(items []config.WidgetItem, settings *config.Settings, ctx widget.RenderContext, width int) string {
	ctx.TerminalWidth = width
	return renderRow(items, settings, ctx)
}

// renderRow renders items at ctx.TerminalWidth.
func renderRow(items []config.WidgetItem, settings *config.Settings, ctx widget.RenderContext) string {
	ctx.TerminalWidth = min(ctx.TerminalWidth, maxRowWidth)
	segments := renderWidgets(items, ctx, settings)
	segments = cleanSeparators(segments)
	if len(segments) == 0 {
		return ""
	}

	colored := applyColors(segments, settings, ctx)
	line := joinWithFlex(colored, segments, settings, ctx)

	if ctx.TerminalWidth > 0 {
		line = Truncate(line, ctx.TerminalWidth)
	}
	return line
}

// PostProcess applies practical workarounds to a rendered line.
// - Replaces spaces with non-breaking spaces (U+00A0) for VSCode compatibility
// - Prepends ANSI reset to override Claude Code dim attribute
// - Returns empty string for lines with no visible content
func PostProcess(line string) string {
	stripped := color.StripANSI(line)
	if strings.TrimSpace(stripped) == "" {
		return ""
	}
	line = strings.ReplaceAll(line, " ", "\u00A0")
	line = "\x1b[0m" + line
	return line
}

// CalculateFlexWidth resolves the available terminal width based on flex mode.
// The result is clamped to a positive minimum: a non-positive width would make
// RenderLine skip truncation entirely (its guard is width > 0), letting the line
// overflow — strictly worse than a narrow-but-bounded width.
func CalculateFlexWidth(detected int, flexMode string, compactThreshold int, contextPct float64) int {
	var width int
	switch flexMode {
	case "full":
		width = detected - flexFullPadding
	case "full-minus-40":
		width = detected - flexCompactPadding
	case "full-until-compact":
		if contextPct >= float64(compactThreshold) {
			width = detected - flexCompactPadding
		} else {
			width = detected - flexFullPadding
		}
	default:
		return detected
	}
	return max(width, minFlexWidth)
}

// joinWithFlex joins colored segments, expanding every flex separator. With N
// flex separators the row splits into N+1 groups; the leftover terminal width is
// spread evenly across the N gaps so a left/center/right layout aligns correctly.
// Neighboring groups always keep one column between them: a row with no room to
// align is joined with single spaces and left for the caller to truncate.
func joinWithFlex(colored []string, segments []segment, settings *config.Settings, ctx widget.RenderContext) string {
	var flexIdx []int
	for i := range segments {
		if segments[i].item.Type == widget.TypeFlexSeparator {
			flexIdx = append(flexIdx, i)
		}
	}
	if len(flexIdx) == 0 {
		return joinPadded(colored, segments, settings.DefaultPadding)
	}

	// Split into the groups between flex separators, each padding-joined.
	groups := make([]string, 0, len(flexIdx)+1)
	prev := 0
	for _, idx := range flexIdx {
		groups = append(groups, joinPadded(colored[prev:idx], segments[prev:idx], settings.DefaultPadding))
		prev = idx + 1
	}
	groups = append(groups, joinPadded(colored[prev:], segments[prev:], settings.DefaultPadding))
	gaps := len(flexIdx)

	// minGap reserves a column after each group that has another one to its right.
	minGap := make([]int, gaps)
	var shown []string
	used, reserved, last := 0, 0, -1
	for i, g := range groups {
		if g == "" {
			continue
		}
		shown = append(shown, g)
		used += color.VisibleWidth(g)
		if last >= 0 {
			minGap[last] = 1
			reserved++
		}
		last = i
	}

	free := ctx.TerminalWidth - used - reserved
	if ctx.TerminalWidth <= 0 || free < 0 {
		return strings.Join(shown, " ")
	}

	// Distribute free width across the gaps; spread the remainder over the
	// leading gaps so the line fills exactly to the terminal width.
	base, extra := free/gaps, free%gaps
	var b strings.Builder
	for i, g := range groups {
		b.WriteString(g)
		if i < gaps {
			w := minGap[i] + base
			if i < extra {
				w++
			}
			b.WriteString(strings.Repeat(" ", w))
		}
	}
	return b.String()
}

// joinPadded joins colored segments with padding, except after a segment whose
// item sets merge.
func joinPadded(colored []string, segments []segment, padding string) string {
	var b strings.Builder
	for i, c := range colored {
		if i > 0 && !segments[i-1].item.IsMerged() {
			b.WriteString(padding)
		}
		b.WriteString(c)
	}
	return b.String()
}

// renderWidgets renders each widget item and collects segments.
func renderWidgets(items []config.WidgetItem, ctx widget.RenderContext, settings *config.Settings) []segment {
	var segments []segment
	for i := range items {
		w := widget.Get(items[i].Type)
		if w == nil {
			continue
		}
		seg := segment{
			text:  w.Render(&items[i], ctx, settings),
			item:  &items[i],
			isSep: items[i].Type == widget.TypeSeparator,
		}
		if seg.text != "" {
			seg.text = truncateWidget(seg.text, items[i].MaxWidth)
			prefix := items[i].Prefix
			if prefix == "" {
				prefix = widget.DefaultPrefix(w)
			}
			seg.text = prefix + seg.text + items[i].Suffix
			if v, ok := w.(widget.Valuer); ok && len(items[i].Thresholds) > 0 {
				seg.value, seg.hasValue = v.Value(&items[i], ctx)
			}
		}
		segments = append(segments, seg)
	}
	return segments
}

// cleanSeparators drops widgets that rendered empty and the separators left
// without a widget on both sides. A flex separator ends a group like the edge of
// the row does, so a separator never sits next to one; of a run of separators
// the first is kept.
func cleanSeparators(segments []segment) []segment {
	var result []segment
	var pending *segment // a separator waiting for a widget to follow it
	inGroup := false     // the current flex group already holds a widget
	for i := range segments {
		seg := &segments[i]
		switch {
		case seg.item.Type == widget.TypeFlexSeparator:
			result = append(result, *seg)
			pending, inGroup = nil, false
		case seg.isSep:
			if inGroup && pending == nil {
				pending = seg
			}
		case seg.text != "":
			if pending != nil {
				result = append(result, *pending)
				pending = nil
			}
			result = append(result, *seg)
			inGroup = true
		}
	}
	return result
}

// applyColors wraps each segment text with ANSI color codes. A threshold the
// segment's value reaches restyles it; the global overrides still win. Flex
// separators are not colored (they are invisible spacing).
func applyColors(segments []segment, settings *config.Settings, ctx widget.RenderContext) []string {
	result := make([]string, 0, len(segments))
	widgetFG := "" // foreground of the nearest widget to the left, thresholds applied
	for _, seg := range segments {
		if seg.item.Type == widget.TypeFlexSeparator {
			result = append(result, "")
			continue
		}

		fg := seg.item.Color
		if fg == "" {
			if seg.isSep && settings.InheritSeparatorColors {
				fg = widgetFG
			} else {
				fg = defaultColor(seg.item.Type, ctx)
			}
		}
		bg := seg.item.BackgroundColor
		bold := seg.item.Bold || settings.GlobalBold

		if seg.hasValue {
			if t, ok := seg.item.ThresholdFor(seg.value); ok {
				if t.Color != "" {
					fg = t.Color
				}
				if t.BackgroundColor != "" {
					bg = t.BackgroundColor
				}
				bold = bold || t.Bold
			}
		}
		if !seg.isSep {
			widgetFG = fg
		}

		if settings.OverrideForegroundColor != "" {
			fg = settings.OverrideForegroundColor
		}
		if settings.OverrideBackgroundColor != "" {
			bg = settings.OverrideBackgroundColor
		}

		colored := color.Apply(seg.text, fg, bg, bold, settings.ColorLevel)
		result = append(result, colored)
	}
	return result
}

// defaultColor returns a widget type's default color, preferring the
// data-dependent one when the widget provides it.
func defaultColor(widgetType string, ctx widget.RenderContext) string {
	w := widget.Get(widgetType)
	if w == nil {
		return ""
	}
	if dyn, ok := w.(widget.DynamicDefaultColor); ok {
		if c := dyn.DefaultColorFor(ctx); c != "" {
			return c
		}
	}
	return w.DefaultColor()
}
