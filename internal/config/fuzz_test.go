package config

import (
	"bytes"
	"encoding/json"
	"testing"
)

// FuzzParse checks that Parse never panics and that settings it accepts
// survive being saved and loaded again: explicit empties included, nothing is
// lost or defaulted on the way.
func FuzzParse(f *testing.F) {
	defaults, err := json.Marshal(DefaultSettings())
	if err != nil {
		f.Fatal(err)
	}
	f.Add(defaults)
	f.Add([]byte(`{"lines":[[{"type":"context-percentage","prefix":"","display":"bar","barWidth":20,"merge":true}]]}`))
	f.Add([]byte(`{"lines":[],"subagentLine":[],"defaultPadding":"","defaultSeparator":"","colorLevel":0}`))
	f.Add([]byte(`{"version":4,"lines":[[{"id":"1","type":"model","metadata":{"display":"bar"},"merge":"no-padding"}]]}`))
	f.Add([]byte(`{"lines":[[{"type":"model","thresholds":[{"gte":1e308,"color":"#fff"},{"gte":-0.5}]}]]}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{"lines":[[`))

	f.Fuzz(func(t *testing.T, data []byte) {
		settings, err := Parse(data)
		if err != nil {
			return
		}
		saved, err := json.Marshal(settings)
		if err != nil {
			t.Fatalf("accepted settings do not marshal: %v", err)
		}
		reloaded, err := Parse(saved)
		if err != nil {
			t.Fatalf("saved settings do not load: %v\n%s", err, saved)
		}
		again, err := json.Marshal(reloaded)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(saved, again) {
			t.Fatalf("settings changed on a save and load:\n%s\n%s", saved, again)
		}
	})
}
