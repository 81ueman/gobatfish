package dataframe

import (
	"fmt"
	"strconv"
)

// Row is a read-only view of a single row of a DataFrame.
type Row interface {
	// Get returns the raw value of the named column, or nil if absent.
	Get(name string) any
	// Str returns the value rendered as a string (empty string for nil).
	Str(name string) string
	// Int returns the value as an int64 when possible.
	Int(name string) (int64, bool)
	// Float returns the value as a float64 when possible.
	Float(name string) (float64, bool)
	// Bool returns the value as a bool when possible.
	Bool(name string) bool
	// Has reports whether the column exists.
	Has(name string) bool
	// Names returns the column names.
	Names() []string
	// Index returns the row index.
	Index() int
	// Map returns a copy of the row as a map.
	Map() map[string]any
}

type frameRow struct {
	df *DataFrame
	i  int
}

// Get returns the raw value of the named column.
func (r *frameRow) Get(name string) any { return r.df.At(r.i, name) }

// Str returns the value rendered as a string.
func (r *frameRow) Str(name string) string {
	v := r.df.At(r.i, name)
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return renderValue(v)
}

// Int returns the value as an int64 when possible.
func (r *frameRow) Int(name string) (int64, bool) {
	v := r.df.At(r.i, name)
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int64:
		return n, true
	case int32:
		return int64(n), true
	case float64:
		return int64(n), true
	case string:
		i, err := strconv.ParseInt(n, 10, 64)
		return i, err == nil
	default:
		return 0, false
	}
}

// Float returns the value as a float64 when possible.
func (r *frameRow) Float(name string) (float64, bool) {
	v := r.df.At(r.i, name)
	if f, ok := asFloat(v); ok {
		return f, true
	}
	if s, ok := v.(string); ok {
		f, err := strconv.ParseFloat(s, 64)
		return f, err == nil
	}
	return 0, false
}

// Bool returns the value as a bool when possible.
func (r *frameRow) Bool(name string) bool {
	v := r.df.At(r.i, name)
	switch b := v.(type) {
	case bool:
		return b
	case string:
		parsed, err := strconv.ParseBool(b)
		return err == nil && parsed
	default:
		return false
	}
}

// Has reports whether the column exists.
func (r *frameRow) Has(name string) bool { return r.df.HasColumn(name) }

// Names returns the column names.
func (r *frameRow) Names() []string { return r.df.Columns() }

// Index returns the row index.
func (r *frameRow) Index() int { return r.i }

// Map returns a copy of the row as a map.
func (r *frameRow) Map() map[string]any {
	out := make(map[string]any, len(r.df.columns))
	for _, name := range r.df.columns {
		out[name] = r.df.At(r.i, name)
	}
	return out
}

// String implements fmt.Stringer.
func (r *frameRow) String() string { return fmt.Sprint(r.Map()) }
