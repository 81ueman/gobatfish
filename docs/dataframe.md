# Data frame

`TableAnswer.Frame()` returns a `*dataframe.DataFrame`: a small, dependency-free,
object-typed table that implements the subset of pandas semantics pybatfish
relies on.

Every column has `dtype=object`: a cell may hold `nil`, a primitive, a slice, or
a datamodel struct. A missing value stays `nil` (it is never converted to a
numeric NaN), and `nil` renders as `None`, matching an object-typed pandas
Series.

## pandas mapping

| pandas | gobatfish |
|---|---|
| `df["Node"]` | `df.Col("Node")` |
| `df["Node"] == "n1"` | `df.Col("Node").Eq("n1")` |
| `df[df["Node"] == "n1"]` | `df.Filter(df.Col("Node").Eq("n1"))` |
| `df.loc[0]["col"]` | `df.At(0, "col")` |
| `df.iloc[0]` | `df.Row(0)` |
| `len(df)` | `df.Len()` |
| `df.empty` | `df.Empty()` |
| `df.shape` | `df.Shape()` |
| `df.size` | `df.Size()` |
| `df.columns` | `df.Columns()` |
| `df.to_dict(orient="records")` | `df.Records()` |
| `df.to_string()` | `df.String()` |
| `df._repr_html_()` | `df.HTML()` |
| `df.sort_values(["x"])` | `df.SortValues("x")` |
| `df.duplicated(["x"], keep=False)` | `df.Duplicated([]string{"x"}, false)` |
| `df.groupby("x").filter(f)` | `df.GroupBy("x").Filter(f)` |
| `series.apply(f)` | `series.Map(f)` |
| `series.nunique()` | `series.NUnique()` |
| `series.dtype` | `series.Dtype()` (`"object"`) |

## Iterating

`Rows` returns an `iter.Seq[Row]`, so it works with the standard `range` loop:

```go
for row := range df.Rows() {
	fmt.Println(row.Str("Node"), row.Int("AS_Number"))
}
```

## Building a frame

```go
df := dataframe.FromRecords([]map[string]any{
	{"Node": "n1", "VRF": "default", "Network": "10.0.0.0/24"},
	{"Node": "n2", "VRF": "default", "Network": "10.0.0.0/24"},
})

// Select columns and reorder.
df.Select("Node", "Network")

// Filter with a boolean mask.
mask := df.Col("Node").Eq("n1")
only := df.Filter(mask)

// Map a column.
strs := df.Col("Network").Map(func(v any) any { return fmt.Sprint(v) })

// Group and filter groups.
kept := df.GroupBy("Network").Filter(func(g *dataframe.DataFrame) bool {
	return g.Len() > 1
})
```

## Serializing

```go
records := df.Records()       // []map[string]any
table := df.String()          // pandas-style string
html := df.HTML()             // HTML table
```
