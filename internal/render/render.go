// Package render implements the status line rendering pipeline.
package render

import (
	"strings"

	"github.com/moond4rk/ccstatus/internal/color"
	"github.com/moond4rk/ccstatus/internal/config"
	"github.com/moond4rk/ccstatus/internal/widget"
)

const separatorType = "separator"

// segment holds a rendered widget's text and metadata for the pipeline.
type segment struct {
	text  string
	item  *config.WidgetItem
	isSep bool
}

// RenderLine renders a single line of widgets into an ANSI-colored string.
func RenderLine(items []config.WidgetItem, settings *config.Settings, ctx widget.RenderContext) string {
	segments := renderWidgets(items, ctx, settings)
	segments = cleanSeparators(segments)
	if len(segments) == 0 {
		return ""
	}

	colored := applyColors(segments, settings)
	line := strings.Join(colored, settings.DefaultPadding)

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

// renderWidgets renders each widget item and collects segments.
func renderWidgets(items []config.WidgetItem, ctx widget.RenderContext, settings *config.Settings) []segment {
	var segments []segment
	for i := range items {
		w := widget.Get(items[i].Type)
		if w == nil {
			continue
		}
		text := w.Render(&items[i], ctx, settings)
		segments = append(segments, segment{
			text:  text,
			item:  &items[i],
			isSep: items[i].Type == separatorType,
		})
	}
	return segments
}

// cleanSeparators removes empty non-separator widgets and trims edge/consecutive separators.
func cleanSeparators(segments []segment) []segment {
	// Remove empty non-separator widgets
	var filtered []segment
	for _, seg := range segments {
		if seg.text == "" && !seg.isSep {
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

// applyColors wraps each segment text with ANSI color codes.
func applyColors(segments []segment, settings *config.Settings) []string {
	result := make([]string, 0, len(segments))
	for i, seg := range segments {
		fg := resolveColor(seg, segments, i, settings)
		bg := seg.item.BackgroundColor
		bold := seg.item.Bold || settings.GlobalBold

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
func resolveColor(seg segment, segments []segment, idx int, settings *config.Settings) string {
	if seg.item.Color != "" {
		return seg.item.Color
	}
	if seg.isSep && settings.InheritSeparatorColors {
		return inheritColor(segments, idx)
	}
	if w := widget.Get(seg.item.Type); w != nil {
		return w.DefaultColor()
	}
	return ""
}

// inheritColor finds the color from the nearest non-separator widget before the separator.
func inheritColor(segments []segment, sepIdx int) string {
	for i := sepIdx - 1; i >= 0; i-- {
		if !segments[i].isSep {
			if segments[i].item.Color != "" {
				return segments[i].item.Color
			}
			if w := widget.Get(segments[i].item.Type); w != nil {
				return w.DefaultColor()
			}
		}
	}
	return ""
}
