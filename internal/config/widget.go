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
}

// IsMerged returns true if this widget should merge with the adjacent widget.
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

// MergeNoPadding returns true if merge mode is "no-padding".
func (w *WidgetItem) MergeNoPadding() bool {
	s, ok := w.Merge.(string)
	return ok && s == "no-padding"
}
