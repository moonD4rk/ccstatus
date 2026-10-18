package claude

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
)

// document is a JSON object whose top-level members keep their order and their
// original values, so editing one key leaves the rest of settings.json alone.
type document struct {
	keys   []string
	values map[string]json.RawMessage
}

// parseDocument decodes a top-level JSON object; empty input is an empty object.
func parseDocument(data []byte) (*document, error) {
	doc := &document{values: make(map[string]json.RawMessage)}
	if len(bytes.TrimSpace(data)) == 0 {
		return doc, nil
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return nil, errors.New("settings.json is not a JSON object")
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, _ := tok.(string)
		var raw json.RawMessage
		if err := dec.Decode(&raw); err != nil {
			return nil, err
		}
		if _, dup := doc.values[key]; !dup {
			doc.keys = append(doc.keys, key)
		}
		doc.values[key] = raw
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, errors.New("settings.json has data after the top-level object")
	}
	return doc, nil
}

func (d *document) get(key string) (json.RawMessage, bool) {
	raw, ok := d.values[key]
	return raw, ok
}

// set encodes v under key, appending the key when it is new.
func (d *document) set(key string, v any) error {
	raw, err := marshalNoEscape(v)
	if err != nil {
		return err
	}
	if _, ok := d.values[key]; !ok {
		d.keys = append(d.keys, key)
	}
	d.values[key] = raw
	return nil
}

func (d *document) remove(key string) {
	delete(d.values, key)
	d.keys = slices.DeleteFunc(d.keys, func(k string) bool { return k == key })
}

// encode renders the document with two-space indentation and a final newline.
func (d *document) encode() ([]byte, error) {
	var compact bytes.Buffer
	compact.WriteByte('{')
	for i, key := range d.keys {
		if i > 0 {
			compact.WriteByte(',')
		}
		k, err := marshalNoEscape(key)
		if err != nil {
			return nil, err
		}
		compact.Write(k)
		compact.WriteByte(':')
		if err := json.Compact(&compact, d.values[key]); err != nil {
			return nil, err
		}
	}
	compact.WriteByte('}')
	var out bytes.Buffer
	if err := json.Indent(&out, compact.Bytes(), "", "  "); err != nil {
		return nil, err
	}
	out.WriteByte('\n')
	return out.Bytes(), nil
}

// marshalNoEscape is json.Marshal without HTML escaping, so characters such as
// "&" in a command are written as typed.
func marshalNoEscape(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// readDocument loads settings.json at path; a missing file is an empty object.
func readDocument(path string) (*document, error) {
	data, err := os.ReadFile(path) //nolint:gosec // path is Claude Code's own settings file
	if err != nil {
		if os.IsNotExist(err) {
			return parseDocument(nil)
		}
		return nil, err
	}
	return parseDocument(data)
}

func writeDocument(path string, doc *document) error {
	data, err := doc.encode()
	if err != nil {
		return err
	}
	return writeFileAtomic(path, data)
}

// writeFileAtomic replaces the file at path through a temp file and a rename, so
// an interrupted write never leaves a truncated settings.json. A symlink at path
// is followed, so a dotfiles-managed settings.json stays linked; the file keeps
// its permissions.
func writeFileAtomic(path string, data []byte) error {
	target, err := resolveSymlink(path)
	if err != nil {
		return err
	}
	dir := filepath.Dir(target)
	if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
		return mkErr
	}
	mode := os.FileMode(0o600)
	if info, statErr := os.Stat(target); statErr == nil {
		mode = info.Mode().Perm()
	}
	tmp, err := os.CreateTemp(dir, ".settings-*.json.tmp")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }() // no-op once renamed
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), target)
}

// resolveSymlink returns the file path refers to, following symlinks. A dangling
// symlink resolves to its target, so writing creates the target instead of
// replacing the link.
func resolveSymlink(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved, nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}
	link, err := os.Readlink(path)
	if err != nil {
		return path, nil //nolint:nilerr // not a symlink: the write creates path itself
	}
	if !filepath.IsAbs(link) {
		link = filepath.Join(filepath.Dir(path), link)
	}
	return link, nil
}
