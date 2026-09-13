package dataframe

import (
	"iter"
	"sort"
)

// DataFrame is a column oriented table. Every column is object-typed, so any Go
// value may be stored in a cell. It corresponds to a pandas.DataFrame with
// dtype=object.
type DataFrame struct {
	columns []string
	cols    map[string]*Series
	nrows   int
}

// New creates a DataFrame from an explicit column order and column data. Columns
// listed in order but absent from data are created empty; data keys not listed
// in order are appended in sorted order.
func New(order []string, data map[string][]any) *DataFrame {
	df := &DataFrame{cols: make(map[string]*Series), nrows: 0}
	seen := make(map[string]bool, len(order))
	for _, name := range order {
		if seen[name] {
			continue
		}
		seen[name] = true
		df.addColumn(name, data[name])
	}
	var extra []string
	for name := range data {
		if !seen[name] {
			extra = append(extra, name)
		}
	}
	for _, name := range sortedStrings(extra) {
		df.addColumn(name, data[name])
	}
	return df
}

// Empty creates a DataFrame with the given columns and no rows.
func Empty(columns ...string) *DataFrame {
	return New(columns, map[string][]any{})
}

// FromRecords creates a DataFrame from row-oriented records. If columns are
// provided they define the column order; otherwise the union of keys is used in
// sorted order.
func FromRecords(records []map[string]any, columns ...string) *DataFrame {
	order := columns
	if len(order) == 0 {
		seen := map[string]bool{}
		for _, r := range records {
			for k := range r {
				if !seen[k] {
					seen[k] = true
					order = append(order, k)
				}
			}
		}
		order = sortedStrings(order)
	}
	data := make(map[string][]any, len(order))
	for _, name := range order {
		col := make([]any, len(records))
		for i, r := range records {
			col[i] = r[name]
		}
		data[name] = col
	}
	return New(order, data)
}

// FromSeries creates a DataFrame from the given Series. Columns are ordered by
// the order in which the series are passed.
func FromSeries(series ...*Series) *DataFrame {
	df := &DataFrame{cols: make(map[string]*Series)}
	for _, s := range series {
		df.addColumn(s.name, s.values)
	}
	return df
}

func (d *DataFrame) addColumn(name string, values []any) {
	if d.cols == nil {
		d.cols = make(map[string]*Series)
	}
	if _, exists := d.cols[name]; !exists {
		d.columns = append(d.columns, name)
	}
	d.cols[name] = NewSeries(name, values)
	if len(values) > d.nrows {
		d.nrows = len(values)
	}
}

// Columns returns the column names in order.
func (d *DataFrame) Columns() []string {
	return append([]string(nil), d.columns...)
}

// HasColumn reports whether a column exists.
func (d *DataFrame) HasColumn(name string) bool {
	_, ok := d.cols[name]
	return ok
}

// Col returns a copy of the named column, or nil if it does not exist.
func (d *DataFrame) Col(name string) *Series {
	s, ok := d.cols[name]
	if !ok {
		return nil
	}
	return NewSeries(s.name, s.values)
}

// Len returns the number of rows.
func (d *DataFrame) Len() int { return d.nrows }

// Shape returns (rows, columns).
func (d *DataFrame) Shape() (int, int) { return d.nrows, len(d.columns) }

// Size returns rows*columns, matching pandas DataFrame.size.
func (d *DataFrame) Size() int { return d.nrows * len(d.columns) }

// Empty reports whether the frame has no rows.
func (d *DataFrame) Empty() bool { return d.nrows == 0 }

// At returns the value at the given row and column, or nil if absent.
func (d *DataFrame) At(row int, column string) any {
	s, ok := d.cols[column]
	if !ok {
		return nil
	}
	return s.At(row)
}

// Row returns a view of row i. It panics if i is out of range.
func (d *DataFrame) Row(i int) Row {
	if i < 0 || i >= d.nrows {
		panic("dataframe: row index out of range")
	}
	return &frameRow{df: d, i: i}
}

// Rows returns an iterator over the rows of the frame.
func (d *DataFrame) Rows() iter.Seq[Row] {
	return func(yield func(Row) bool) {
		for i := 0; i < d.nrows; i++ {
			if !yield(&frameRow{df: d, i: i}) {
				return
			}
		}
	}
}

// Records returns the frame as a slice of row maps, mirroring
// pandas.DataFrame.to_dict(orient="records").
func (d *DataFrame) Records() []map[string]any {
	out := make([]map[string]any, d.nrows)
	for i := 0; i < d.nrows; i++ {
		r := make(map[string]any, len(d.columns))
		for _, name := range d.columns {
			r[name] = d.cols[name].At(i)
		}
		out[i] = r
	}
	return out
}

// Copy returns a shallow copy of the frame.
func (d *DataFrame) Copy() *DataFrame {
	out := &DataFrame{
		columns: append([]string(nil), d.columns...),
		cols:    make(map[string]*Series, len(d.cols)),
		nrows:   d.nrows,
	}
	for name, s := range d.cols {
		out.cols[name] = NewSeries(s.name, s.values)
	}
	return out
}

// Filter returns the rows where mask is true. If mask is nil or has a different
// length than the frame, a copy of the frame is returned.
func (d *DataFrame) Filter(mask *Series) *DataFrame {
	if mask == nil || mask.Len() != d.nrows {
		return d.Copy()
	}
	out := &DataFrame{cols: make(map[string]*Series), columns: append([]string(nil), d.columns...)}
	for _, name := range d.columns {
		col := d.cols[name]
		values := make([]any, 0, d.nrows)
		for i := 0; i < d.nrows; i++ {
			if b, ok := mask.At(i).(bool); ok && b {
				values = append(values, col.At(i))
			}
		}
		out.cols[name] = NewSeries(name, values)
	}
	if len(d.columns) > 0 {
		out.nrows = out.cols[d.columns[0]].Len()
	}
	return out
}

// FilterFunc returns the rows for which f returns true.
func (d *DataFrame) FilterFunc(f func(Row) bool) *DataFrame {
	out := &DataFrame{cols: make(map[string]*Series), columns: append([]string(nil), d.columns...)}
	for _, name := range d.columns {
		out.cols[name] = NewSeries(name, nil)
	}
	for i := 0; i < d.nrows; i++ {
		if !f(&frameRow{df: d, i: i}) {
			continue
		}
		for _, name := range d.columns {
			s := out.cols[name]
			s.values = append(s.values, d.cols[name].At(i))
		}
	}
	if len(d.columns) > 0 {
		out.nrows = out.cols[d.columns[0]].Len()
	}
	return out
}

// Select returns a new frame containing only the named columns, in the given
// order. Missing columns are created empty.
func (d *DataFrame) Select(names ...string) *DataFrame {
	return d.Reindex(names)
}

// Reindex returns a new frame whose columns are exactly the given names in the
// given order. Columns absent from the receiver are created filled with nil.
func (d *DataFrame) Reindex(columns []string) *DataFrame {
	out := &DataFrame{columns: append([]string(nil), columns...), cols: make(map[string]*Series, len(columns))}
	for _, name := range columns {
		if s, ok := d.cols[name]; ok {
			out.cols[name] = NewSeries(name, s.values)
		} else {
			values := make([]any, d.nrows)
			out.cols[name] = NewSeries(name, values)
		}
	}
	out.nrows = d.nrows
	return out
}

// Drop returns a copy of the frame without the named columns.
func (d *DataFrame) Drop(names ...string) *DataFrame {
	drop := make(map[string]bool, len(names))
	for _, n := range names {
		drop[n] = true
	}
	var kept []string
	for _, c := range d.columns {
		if !drop[c] {
			kept = append(kept, c)
		}
	}
	return d.Reindex(kept)
}

// SortValues returns a new frame sorted by the given columns. Nil values sort
// last. The sort is stable.
func (d *DataFrame) SortValues(columns ...string) *DataFrame {
	idx := make([]int, d.nrows)
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool {
		for _, name := range columns {
			s := d.cols[name]
			if s == nil {
				continue
			}
			c := compareValues(s.At(idx[a]), s.At(idx[b]))
			if c != 0 {
				return c < 0
			}
		}
		return false
	})
	return d.take(idx)
}

// Duplicated returns a boolean mask marking duplicate rows with respect to the
// given columns. When keep is true only all but the first occurrence of each
// duplicate is marked; when false every occurrence is marked. It mirrors
// pandas.DataFrame.duplicated(keep=False|"first").
func (d *DataFrame) Duplicated(columns []string, keep bool) *Series {
	keys := make([]string, d.nrows)
	counts := make(map[string]int, d.nrows)
	for i := 0; i < d.nrows; i++ {
		keys[i] = d.rowKey(columns, i)
		counts[keys[i]]++
	}
	seen := make(map[string]int, d.nrows)
	out := make([]any, d.nrows)
	for i := 0; i < d.nrows; i++ {
		if counts[keys[i]] <= 1 {
			out[i] = false
			continue
		}
		if keep {
			seen[keys[i]]++
			out[i] = seen[keys[i]] > 1
		} else {
			out[i] = true
		}
	}
	return &Series{name: "", values: out}
}

func (d *DataFrame) rowKey(columns []string, i int) string {
	key := ""
	for _, name := range columns {
		if s, ok := d.cols[name]; ok {
			key += hashKey(s.At(i))
		}
		key += "\x1f"
	}
	return key
}

func (d *DataFrame) take(idx []int) *DataFrame {
	out := &DataFrame{columns: append([]string(nil), d.columns...), cols: make(map[string]*Series, len(d.cols))}
	for _, name := range d.columns {
		src := d.cols[name]
		values := make([]any, len(idx))
		for j, i := range idx {
			values[j] = src.At(i)
		}
		out.cols[name] = NewSeries(name, values)
	}
	if len(d.columns) > 0 {
		out.nrows = out.cols[d.columns[0]].Len()
	}
	return out
}

// Concat appends the rows of the other frames to the receiver, matching by
// column name.
func (d *DataFrame) Concat(others ...*DataFrame) *DataFrame {
	out := d.Copy()
	for _, other := range others {
		for _, name := range other.columns {
			if !out.HasColumn(name) {
				out.columns = append(out.columns, name)
				out.cols[name] = NewSeries(name, make([]any, out.nrows))
			}
		}
		total := out.nrows + other.nrows
		for _, name := range out.columns {
			values := make([]any, 0, total)
			values = append(values, out.cols[name].values...)
			if s, ok := other.cols[name]; ok {
				values = append(values, s.values...)
			} else {
				values = append(values, make([]any, other.nrows)...)
			}
			out.cols[name] = NewSeries(name, values)
		}
		out.nrows = total
	}
	return out
}
