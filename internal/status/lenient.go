package status

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"
)

var unmarshalerType = reflect.TypeFor[json.Unmarshaler]()

// parseLenient decodes the JSON object in data into the struct pointed to by v,
// strictly when possible and otherwise through decodeLenient. Input that is not
// a JSON object (or null) is an error.
func parseLenient(data []byte, v any) (dropped []string, err error) {
	err = json.Unmarshal(data, v)
	if err == nil {
		return nil, nil
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(data, &fields) != nil {
		return nil, err
	}
	rv := reflect.ValueOf(v).Elem()
	rv.SetZero()
	return decodeLenient(data, rv, ""), nil
}

// decodeLenient decodes raw into the addressable value v. A value that fails
// strict decoding is coerced when the intent is unambiguous and dropped
// otherwise; objects and arrays are decoded member by member so a bad member
// drops only itself. It returns the JSON paths of dropped values.
func decodeLenient(raw json.RawMessage, v reflect.Value, path string) []string {
	if json.Unmarshal(raw, v.Addr().Interface()) == nil {
		return nil
	}
	v.SetZero()
	switch {
	case v.Kind() == reflect.Pointer:
		elem := reflect.New(v.Type().Elem())
		dropped := decodeLenient(raw, elem.Elem(), path)
		if !slices.Contains(dropped, path) {
			v.Set(elem)
		}
		return dropped
	case v.Kind() == reflect.Struct && !reflect.PointerTo(v.Type()).Implements(unmarshalerType):
		return decodeFields(raw, v, path)
	case v.Kind() == reflect.Slice:
		return decodeElements(raw, v, path)
	case coerce(raw, v):
		return nil
	default:
		return []string{path}
	}
}

// decodeElements decodes a JSON array into the slice v one element at a time,
// leaving out elements that cannot be decoded at all.
func decodeElements(raw json.RawMessage, v reflect.Value, path string) []string {
	var elems []json.RawMessage
	if json.Unmarshal(raw, &elems) != nil {
		return []string{path}
	}
	out := reflect.MakeSlice(v.Type(), 0, len(elems))
	var dropped []string
	for i, e := range elems {
		elemPath := fmt.Sprintf("%s[%d]", path, i)
		elem := reflect.New(v.Type().Elem()).Elem()
		d := decodeLenient(e, elem, elemPath)
		dropped = append(dropped, d...)
		if !slices.Contains(d, elemPath) {
			out = reflect.Append(out, elem)
		}
	}
	v.Set(out)
	return dropped
}

// decodeFields decodes a JSON object into the struct v one field at a time.
func decodeFields(raw json.RawMessage, v reflect.Value, path string) []string {
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil {
		return []string{path}
	}
	var dropped []string
	t := v.Type()
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		switch {
		case name == "-":
			continue
		case name == "" && f.Anonymous && f.Type.Kind() == reflect.Struct:
			// Embedded struct: its fields are promoted into this object.
			dropped = append(dropped, decodeFields(raw, v.Field(i), path)...)
			continue
		case name == "":
			name = f.Name
		}
		if msg, ok := fields[name]; ok {
			dropped = append(dropped, decodeLenient(msg, v.Field(i), joinPath(path, name))...)
		}
	}
	return dropped
}

// coerce stores raw into v when only the JSON type is off: a fractional or
// quoted number into an integer, a quoted number into a float, a number or
// boolean into a string, a quoted boolean into a boolean.
func coerce(raw json.RawMessage, v reflect.Value) bool {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || raw[0] == '{' || raw[0] == '[' {
		return false
	}
	text := string(raw)
	var quoted string
	if json.Unmarshal(raw, &quoted) == nil {
		text = strings.TrimSpace(quoted)
	}
	switch v.Kind() {
	case reflect.String:
		v.SetString(text)
		return true
	case reflect.Bool:
		b, err := strconv.ParseBool(text)
		if err != nil {
			return false
		}
		v.SetBool(b)
		return true
	default:
		return coerceNumber(text, v)
	}
}

// coerceNumber stores the finite number in text into a numeric v, truncating
// toward zero for integer kinds.
func coerceNumber(text string, v reflect.Value) bool {
	f, err := strconv.ParseFloat(text, 64)
	if err != nil || math.IsInf(f, 0) || math.IsNaN(f) {
		return false
	}
	switch {
	case v.CanInt():
		if f < math.MinInt64 || f >= math.MaxInt64 || v.OverflowInt(int64(f)) {
			return false
		}
		v.SetInt(int64(f))
	case v.CanUint():
		if f < 0 || f >= math.MaxUint64 || v.OverflowUint(uint64(f)) {
			return false
		}
		v.SetUint(uint64(f))
	case v.CanFloat():
		if v.OverflowFloat(f) {
			return false
		}
		v.SetFloat(f)
	default:
		return false
	}
	return true
}

func joinPath(path, name string) string {
	if path == "" {
		return name
	}
	return path + "." + name
}
