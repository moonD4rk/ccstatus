// Package config handles ccstatus configuration loading, saving, and defaults.
package config

// WidgetItem represents a single widget in the status line configuration.
type WidgetItem struct {
	Type            string `json:"type"`
	Color           string `json:"color,omitempty"`
	BackgroundColor string `json:"backgroundColor,omitempty"`
	Bold            bool   `json:"bold,omitempty"`
	// Prefix replaces the widget's default prefix, such as "Ctx: ". Nil keeps
	// the default; an empty string removes it.
	Prefix *string `json:"prefix,omitempty"`
	Suffix string  `json:"suffix,omitempty"`
	// Character is the text of a separator, or an icon before the git-branch name.
	Character string `json:"character,omitempty"`
	RawValue  bool   `json:"rawValue,omitempty"`
	// Display selects one of the widget's display modes, such as "bar".
	Display string `json:"display,omitempty"`
	// BarWidth is the width, in columns, of a "bar" display.
	BarWidth int `json:"barWidth,omitempty"`
	MaxWidth int `json:"maxWidth,omitempty"`
	// Merge joins this widget to the next one without the default padding.
	Merge          bool        `json:"merge,omitempty"`
	CustomText     string      `json:"customText,omitempty"`
	CommandPath    string      `json:"commandPath,omitempty"`
	Timeout        int         `json:"timeout,omitempty"`
	PreserveColors bool        `json:"preserveColors,omitempty"`
	Thresholds     []Threshold `json:"thresholds,omitempty"`
}

// Threshold restyles a numeric widget once its value reaches GTE. Empty colors
// keep the widget's normal colors.
type Threshold struct {
	GTE             float64 `json:"gte"`
	Color           string  `json:"color,omitempty"`
	BackgroundColor string  `json:"backgroundColor,omitempty"`
	Bold            bool    `json:"bold,omitempty"`
}

// ThresholdFor returns the threshold with the highest GTE that value reaches,
// and false when value is below every threshold.
func (w *WidgetItem) ThresholdFor(value float64) (Threshold, bool) {
	var best Threshold
	found := false
	for _, t := range w.Thresholds {
		if value >= t.GTE && (!found || t.GTE >= best.GTE) {
			best, found = t, true
		}
	}
	return best, found
}
