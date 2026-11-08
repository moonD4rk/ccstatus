package widget

import "github.com/moond4rk/ccstatus/internal/config"

// hyperlink wraps text in an OSC 8 link to url. It returns text unchanged when
// url is empty or the settings turn hyperlinks off; terminals without OSC 8
// support ignore the sequence and show the text.
func hyperlink(settings *config.Settings, url, text string) string {
	if url == "" || (settings != nil && !settings.Hyperlinks) {
		return text
	}
	return "\x1b]8;;" + url + "\x07" + text + "\x1b]8;;\x07"
}
