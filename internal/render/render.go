// Package render implements the status line rendering pipeline.
package render

import (
	"strings"

	"github.com/moond4rk/ccstatus/internal/color"
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/widget"
)

const (
	separatorType     = "separator"
	flexSeparatorType = "flex-separator"

	// flexFullPadding is the padding subtracted in "full" and "full-until-compact" flex modes.
	flexFullPadding = 10
	// flexCompactPadding is the padding subtracted in "full-minus-40" mode and compact state.
	flexCompactPadding = 40
	// minFlexWidth is the floor for the resolved width so truncation always runs.
	minFlexWidth = 1
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
func joinWithFlex(colored []string, segments []segment, settings *config.Settings, ctx widget.RenderContext) string {
	var flexIdx []int
	for i := range segments {
		if segments[i].item.Type == flexSeparatorType {
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

	totalWidth := ctx.TerminalWidth
	if totalWidth <= 0 {
		// Unknown width: collapse each gap to a single space.
		return strings.Join(groups, " ")
	}

	used := 0
	for _, g := range groups {
		used += color.VisibleWidth(g)
	}
	free := totalWidth - used
	if free <= 0 {
		// No room to expand: butt the groups together.
		return strings.Join(groups, "")
	}

	// Distribute free width across the gaps; spread the remainder over the
	// leading gaps so the line fills exactly to totalWidth.
	base, extra := free/gaps, free%gaps
	var b strings.Builder
	for i, g := range groups {
		b.WriteString(g)
		if i < gaps {
			w := base
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
			isSep: items[i].Type == separatorType,
		}
		if seg.text != "" {
			prefix := items[i].Prefix
			suffix := items[i].Suffix
			if prefix == "" {
				prefix = w.DefaultPrefix()
			}
			if suffix == "" {
				suffix = w.DefaultSuffix()
			}
			seg.text = prefix + seg.text + suffix
			if v, ok := w.(widget.Valuer); ok && len(items[i].Thresholds) > 0 {
				seg.value, seg.hasValue = v.Value(&items[i], ctx)
			}
		}
		segments = append(segments, seg)
	}
	return segments
}

// cleanSeparators removes empty non-separator widgets and trims edge/consecutive separators.
// Flex separators are preserved and not treated as regular separators.
func cleanSeparators(segments []segment) []segment {
	// Remove empty non-separator widgets (but keep flex separators)
	var filtered []segment
	for _, seg := range segments {
		if seg.text == "" && !seg.isSep && seg.item.Type != flexSeparatorType {
			continue
		}
		filtered = append(filtered, seg)
	}

	// Remove leading separators
	for len(filtered) > 0 && filtered[0].isSep {
		filtered = filtered[1:]
	}
	// Remove trailing separators
	for len(filtered) > 0 && filtered[len(filtered)-1].isSep {
		filtered = filtered[:len(filtered)-1]
	}

	// Remove consecutive separators (keep first of each run)
	var result []segment
	for i, seg := range filtered {
		if seg.isSep && i > 0 && filtered[i-1].isSep {
			continue
		}
		result = append(result, seg)
	}
	return result
}

// applyColors wraps each segment text with ANSI color codes. A threshold the
// segment's value reaches restyles it; the global overrides still win. Flex
// separators are not colored (they are invisible spacing).
func applyColors(segments []segment, settings *config.Settings, ctx widget.RenderContext) []string {
	result := make([]string, 0, len(segments))
	for i, seg := range segments {
		if seg.item.Type == flexSeparatorType {
			result = append(result, "")
			continue
		}

		fg := resolveColor(seg, segments, i, settings, ctx)
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

// resolveColor determines the foreground color for a segment.
func resolveColor(seg segment, segments []segment, idx int, settings *config.Settings, ctx widget.RenderContext) string {
	if seg.item.Color != "" {
		return seg.item.Color
	}
	if seg.isSep && settings.InheritSeparatorColors {
		return inheritColor(segments, idx, ctx)
	}
	return defaultColor(seg.item.Type, ctx)
}

// inheritColor finds the color from the nearest non-separator widget before the separator.
func inheritColor(segments []segment, sepIdx int, ctx widget.RenderContext) string {
	for i := sepIdx - 1; i >= 0; i-- {
		if !segments[i].isSep {
			if segments[i].item.Color != "" {
				return segments[i].item.Color
			}
			return defaultColor(segments[i].item.Type, ctx)
		}
	}
	return ""
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
