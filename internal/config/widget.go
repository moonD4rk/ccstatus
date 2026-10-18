// Package config handles ccstatus configuration loading, saving, and defaults.
package config

// WidgetItem represents a single widget in the status line configuration.
type WidgetItem struct {
	ID              string            `json:"id"`
	Type            string            `json:"type"`
	Color           string            `json:"color,omitempty"`
	BackgroundColor string            `json:"backgroundColor,omitempty"`
	Bold            bool              `json:"bold,omitempty"`
	Prefix          string            `json:"prefix,omitempty"`
	Suffix          string            `json:"suffix,omitempty"`
	Character       string            `json:"character,omitempty"`
	RawValue        bool              `json:"rawValue,omitempty"`
	CustomText      string            `json:"customText,omitempty"`
	CommandPath     string            `json:"commandPath,omitempty"`
	MaxWidth        int               `json:"maxWidth,omitempty"`
	PreserveColors  bool              `json:"preserveColors,omitempty"`
	Timeout         int               `json:"timeout,omitempty"`
	Merge           any               `json:"merge,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	Thresholds      []Threshold       `json:"thresholds,omitempty"`
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

// IsMerged reports whether this widget joins the next one without padding.
// true and the legacy "no-padding" mean the same thing.
func (w *WidgetItem) IsMerged() bool {
	if w.Merge == nil {
		return false
	}
	if b, ok := w.Merge.(bool); ok {
		return b
	}
	if s, ok := w.Merge.(string); ok {
		return s == "no-padding"
	}
	return false
}
