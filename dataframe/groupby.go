package dataframe

import "sort"

// GroupBy groups the rows of a DataFrame by one or more columns.
type GroupBy struct {
	df      *DataFrame
	columns []string
	order   []string
	keys    map[string]map[string]any
	indices map[string][]int
}

// GroupBy returns a GroupBy for the given columns.
func (d *DataFrame) GroupBy(columns ...string) *GroupBy {
	g := &GroupBy{
		df:      d,
		columns: columns,
		keys:    make(map[string]map[string]any),
		indices: make(map[string][]int),
	}
	for i := 0; i < d.nrows; i++ {
		key := d.rowKey(columns, i)
		if _, ok := g.indices[key]; !ok {
			g.order = append(g.order, key)
			k := make(map[string]any, len(columns))
			for _, name := range columns {
				k[name] = d.At(i, name)
			}
			g.keys[key] = k
		}
		g.indices[key] = append(g.indices[key], i)
	}
	return g
}

// Keys returns the group keys in first-appearance order.
func (g *GroupBy) Keys() []map[string]any {
	out := make([]map[string]any, 0, len(g.order))
	for _, k := range g.order {
		out = append(out, g.keys[k])
	}
	return out
}

// Groups returns one DataFrame per group, in first-appearance order.
func (g *GroupBy) Groups() []*DataFrame {
	out := make([]*DataFrame, 0, len(g.order))
	for _, k := range g.order {
		out = append(out, g.df.take(g.indices[k]))
	}
	return out
}

// Filter returns the rows belonging to groups for which f returns true. Rows
// are returned in their original order, matching pandas GroupBy.filter.
func (g *GroupBy) Filter(f func(*DataFrame) bool) *DataFrame {
	var keep []int
	for _, k := range g.order {
		idx := g.indices[k]
		if f(g.df.take(idx)) {
			keep = append(keep, idx...)
		}
	}
	sort.Ints(keep)
	return g.df.take(keep)
}
