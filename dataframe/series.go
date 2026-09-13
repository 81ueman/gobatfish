// Package dataframe provides a small, dependency free, object-typed data frame
// for gobatfish.
//
// It is not a general purpose pandas replacement. It implements the subset of
// pandas.DataFrame semantics that pybatfish relies on, most importantly an
// "object" dtype in which every cell can hold an arbitrary Go value (including
// nil and nested datamodel structs).
package dataframe

import (
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// Series is an ordered, named column of values. Every Series has dtype
// "object": values may be nil, bool, integer, float, string, slices, maps or
// arbitrary structs. It corresponds to a pandas.Series with dtype=object.
type Series struct {
	name   string
	values []any
}

// NewSeries creates a Series with the given name and values.
func NewSeries(name string, values []any) *Series {
	cp := make([]any, len(values))
	copy(cp, values)
	return &Series{name: name, values: cp}
}

// Name returns the series name.
func (s *Series) Name() string { return s.name }

// Dtype always returns "object", matching the dtype pybatfish forces on table
// answers.
func (s *Series) Dtype() string { return "object" }

// Len returns the number of values.
func (s *Series) Len() int { return len(s.values) }

// At returns the value at position i, or nil if out of range.
func (s *Series) At(i int) any {
	if i < 0 || i >= len(s.values) {
		return nil
	}
	return s.values[i]
}

// Values returns a copy of the underlying values.
func (s *Series) Values() []any {
	cp := make([]any, len(s.values))
	copy(cp, s.values)
	return cp
}

// Eq returns a boolean Series that is true where the value equals v.
func (s *Series) Eq(v any) *Series {
	return s.mask(func(x any) bool { return Equal(x, v) })
}

// Ne returns a boolean Series that is true where the value does not equal v.
func (s *Series) Ne(v any) *Series {
	return s.mask(func(x any) bool { return !Equal(x, v) })
}

// In returns a boolean Series that is true where the value is one of vs.
func (s *Series) In(vs ...any) *Series {
	return s.mask(func(x any) bool {
		for _, v := range vs {
			if Equal(x, v) {
				return true
			}
		}
		return false
	})
}

// NotIn returns a boolean Series that is true where the value is none of vs.
func (s *Series) NotIn(vs ...any) *Series {
	return s.mask(func(x any) bool {
		for _, v := range vs {
			if Equal(x, v) {
				return false
			}
		}
		return true
	})
}

// IsNull returns a boolean Series that is true where the value is nil.
func (s *Series) IsNull() *Series {
	return s.mask(func(x any) bool { return x == nil })
}

// NotNull returns a boolean Series that is true where the value is non-nil.
func (s *Series) NotNull() *Series {
	return s.mask(func(x any) bool { return x != nil })
}

// Mask returns a boolean Series that is true where f returns true.
func (s *Series) Mask(f func(any) bool) *Series { return s.mask(f) }

func (s *Series) mask(f func(any) bool) *Series {
	out := make([]any, len(s.values))
	for i, v := range s.values {
		out[i] = f(v)
	}
	return &Series{name: s.name, values: out}
}

// Map applies f to every value and returns a new Series. Map corresponds to
// pandas Series.apply.
func (s *Series) Map(f func(any) any) *Series {
	out := make([]any, len(s.values))
	for i, v := range s.values {
		out[i] = f(v)
	}
	return &Series{name: s.name, values: out}
}

// Apply is an alias for Map, matching pandas Series.apply.
func (s *Series) Apply(f func(any) any) *Series { return s.Map(f) }

// NUnique returns the number of distinct values, ignoring nil.
func (s *Series) NUnique() int {
	seen := make(map[string]struct{}, len(s.values))
	for _, v := range s.values {
		if v == nil {
			continue
		}
		seen[hashKey(v)] = struct{}{}
	}
	return len(seen)
}

// Unique returns the distinct non-nil values in first-seen order.
func (s *Series) Unique() []any {
	seen := make(map[string]struct{}, len(s.values))
	var out []any
	for _, v := range s.values {
		if v == nil {
			continue
		}
		k := hashKey(v)
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, v)
	}
	return out
}

// String renders the Series the way pandas renders an object Series index.
func (s *Series) String() string {
	parts := make([]string, len(s.values))
	for i, v := range s.values {
		parts[i] = renderValue(v)
	}
	return strings.Join(parts, "\n")
}

// Equal reports whether two cell values are equal, treating numeric types as
// comparable across widths and using deep equality for containers.
func Equal(a, b any) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if af, aok := asFloat(a); aok {
		if bf, bok := asFloat(b); bok {
			return af == bf
		}
	}
	return reflect.DeepEqual(a, b)
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case int:
		return float64(n), true
	case int8:
		return float64(n), true
	case int16:
		return float64(n), true
	case int32:
		return float64(n), true
	case int64:
		return float64(n), true
	case uint:
		return float64(n), true
	case uint8:
		return float64(n), true
	case uint16:
		return float64(n), true
	case uint32:
		return float64(n), true
	case uint64:
		return float64(n), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

func isNumber(v any) bool {
	_, ok := asFloat(v)
	return ok
}

// hashKey returns a canonical key for a value, used for grouping and uniqueness.
func hashKey(v any) string {
	if v == nil {
		return "\x00nil"
	}
	switch t := v.(type) {
	case string:
		return "s:" + t
	case bool:
		return "b:" + strconv.FormatBool(t)
	}
	if f, ok := asFloat(v); ok {
		return "n:" + strconv.FormatFloat(f, 'g', -1, 64)
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("p:%p", v)
	}
	return "j:" + string(b)
}

// compareValues defines a total order for sorting. Numeric values compare
// numerically; strings lexicographically; everything else by its rendered form.
func compareValues(a, b any) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return 1 // nils sort last, matching pandas na_position='last'
	}
	if b == nil {
		return -1
	}
	if af, aok := asFloat(a); aok {
		if bf, bok := asFloat(b); bok {
			switch {
			case af < bf:
				return -1
			case af > bf:
				return 1
			default:
				return 0
			}
		}
	}
	as, aok := a.(string)
	bs, bok := b.(string)
	if aok && bok {
		return strings.Compare(as, bs)
	}
	return strings.Compare(renderValue(a), renderValue(b))
}

// renderValue renders a cell value the way pandas renders it in an object
// column: nil becomes "None".
func renderValue(v any) string {
	if v == nil {
		return "None"
	}
	switch t := v.(type) {
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		if math.IsNaN(t) {
			return "NaN"
		}
		return strconv.FormatFloat(t, 'g', -1, 64)
	case float32:
		return strconv.FormatFloat(float64(t), 'g', -1, 32)
	default:
		if f, ok := asFloat(v); ok {
			return strconv.FormatFloat(f, 'g', -1, 64)
		}
		return fmt.Sprint(v)
	}
}

func copyAnySlice(in []any) []any {
	out := make([]any, len(in))
	copy(out, in)
	return out
}

func sortedStrings(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}
