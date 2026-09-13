// Package datamodel mirrors pybatfish.datamodel: the Python data model objects
// that are exchanged with Batfish.
//
// Every element is a Go struct with exported fields whose json tags match the
// Batfish wire format. Elements expose Dict(), the analogue of the Python
// dict() method, and FromDict constructors. Because datamodel elements are sent
// as question parameters, Dict() preserves the same shape (including null
// values) as the Python objects.
package datamodel

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// DataModelElement is implemented by every datamodel element. It is the Go
// analogue of pybatfish's DataModelElement.dict().
type DataModelElement interface {
	Dict() map[string]any
}

// Dict returns a dictionary representation of v, recursively converting nested
// datamodel elements, slices and maps. It mirrors attr.asdict(recurse=True) and
// BfJsonEncoder.
func Dict(v any) map[string]any { return dictOf(v) }

func dictOf(v any) map[string]any {
	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return nil
	}
	t := rv.Type()
	out := make(map[string]any, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" { // unexported
			continue
		}
		name, skip := jsonFieldName(f)
		if skip {
			continue
		}
		out[name] = jsonValue(rv.Field(i).Interface())
	}
	return out
}

func jsonFieldName(f reflect.StructField) (string, bool) {
	tag := f.Tag.Get("json")
	if tag == "" {
		return f.Name, false
	}
	parts := strings.Split(tag, ",")
	if parts[0] == "-" {
		return "", true
	}
	if parts[0] == "" {
		return f.Name, false
	}
	return parts[0], false
}

// jsonValue recursively converts a value into a JSON-friendly representation.
func jsonValue(v any) any {
	if v == nil {
		return nil
	}
	switch v.(type) {
	case bool, string, int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64, float32, float64:
		return v
	}
	if dm, ok := v.(DataModelElement); ok {
		return dm.Dict()
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Interface:
		if rv.IsNil() {
			return nil
		}
		return jsonValue(rv.Elem().Interface())
	case reflect.Slice, reflect.Array:
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = jsonValue(rv.Index(i).Interface())
		}
		return out
	case reflect.Map:
		out := make(map[string]any, rv.Len())
		iter := rv.MapRange()
		for iter.Next() {
			out[fmt.Sprint(iter.Key().Interface())] = jsonValue(iter.Value().Interface())
		}
		return out
	case reflect.Struct:
		return dictOf(v)
	default:
		return v
	}
}

// ToJSONValue converts an arbitrary value into something encoding/json can
// marshal while preserving datamodel dict() semantics.
func ToJSONValue(v any) any { return jsonValue(v) }

// --- small typed accessors for FromDict helpers -----------------------------

func mapGet(m map[string]any, key string) (any, bool) {
	if m == nil {
		return nil, false
	}
	v, ok := m[key]
	return v, ok
}

func strField(m map[string]any, key string) string {
	v, ok := mapGet(m, key)
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func optStrField(m map[string]any, key string) *string {
	v, ok := mapGet(m, key)
	if !ok || v == nil {
		return nil
	}
	s := fmt.Sprint(v)
	return &s
}

func intField(m map[string]any, key string) int {
	v, _ := mapGet(m, key)
	return toInt(v)
}

func optIntField(m map[string]any, key string) *int {
	v, ok := mapGet(m, key)
	if !ok || v == nil {
		return nil
	}
	i := toInt(v)
	return &i
}

func boolField(m map[string]any, key string) bool {
	v, _ := mapGet(m, key)
	return toBool(v)
}

func sliceField(m map[string]any, key string) []any {
	v, ok := mapGet(m, key)
	if !ok || v == nil {
		return nil
	}
	if s, ok := v.([]any); ok {
		return s
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice || rv.Kind() == reflect.Array {
		out := make([]any, rv.Len())
		for i := range out {
			out[i] = rv.Index(i).Interface()
		}
		return out
	}
	return nil
}

func stringSliceField(m map[string]any, key string) []string {
	vals := sliceField(m, key)
	if vals == nil {
		return nil
	}
	out := make([]string, len(vals))
	for i, v := range vals {
		out[i] = fmt.Sprint(v)
	}
	return out
}

func mapField(m map[string]any, key string) map[string]any {
	v, ok := mapGet(m, key)
	if !ok || v == nil {
		return nil
	}
	if mm, ok := v.(map[string]any); ok {
		return mm
	}
	return nil
}

func toInt(v any) int {
	switch n := v.(type) {
	case nil:
		return 0
	case int:
		return n
	case int8:
		return int(n)
	case int16:
		return int(n)
	case int32:
		return int(n)
	case int64:
		return int(n)
	case uint:
		return int(n)
	case uint8:
		return int(n)
	case uint16:
		return int(n)
	case uint32:
		return int(n)
	case uint64:
		return int(n)
	case float32:
		return int(n)
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(n))
		if err != nil {
			return 0
		}
		return i
	default:
		return 0
	}
}

func toBool(v any) bool {
	switch b := v.(type) {
	case bool:
		return b
	case string:
		parsed, err := strconv.ParseBool(b)
		return err == nil && parsed
	default:
		if b, ok := v.(bool); ok {
			return b
		}
		return false
	}
}

// StringPtr returns a pointer to s, a convenience for building datamodel
// objects with optional string fields.
func StringPtr(s string) *string { return &s }

// IntPtr returns a pointer to i.
func IntPtr(i int) *int { return &i }

// BoolPtr returns a pointer to b.
func BoolPtr(b bool) *bool { return &b }
