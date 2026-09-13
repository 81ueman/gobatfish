package dataframe

import (
	"testing"
)

func TestDtypeObjectAndNil(t *testing.T) {
	df := New([]string{"col1", "col2"}, map[string][]any{
		"col1": {"v1", "v2", "v3", "v4"},
		"col2": {1, nil, -1, "-1"},
	})
	if got := df.Col("col1").Dtype(); got != "object" {
		t.Fatalf("col1 dtype = %q, want object", got)
	}
	if got := df.Col("col2").Dtype(); got != "object" {
		t.Fatalf("col2 dtype = %q, want object", got)
	}
	if v := df.At(1, "col2"); v != nil {
		t.Fatalf("col2[1] = %v, want nil", v)
	}
	if got := renderValue(df.At(1, "col2")); got != "None" {
		t.Fatalf("rendered nil = %q, want None", got)
	}
}

func TestEmptyAndColumns(t *testing.T) {
	df := Empty("col1")
	if !df.Empty() {
		t.Fatal("expected empty frame")
	}
	if got := df.Columns(); len(got) != 1 || got[0] != "col1" {
		t.Fatalf("columns = %v", got)
	}
}

func TestFilterEq(t *testing.T) {
	df := FromRecords([]map[string]any{
		{"Node": "n1", "VRF": "vrf1", "Network": "10.10.10.0/24"},
		{"Node": "n1", "VRF": "vrf2", "Network": "20.20.20.0/24"},
	})
	got := df.Filter(df.Col("Node").Eq("n1"))
	if got.Len() != 2 {
		t.Fatalf("len = %d, want 2", got.Len())
	}
	got = df.Filter(df.Col("VRF").Eq("vrf2"))
	if got.Len() != 1 || got.At(0, "Network") != "20.20.20.0/24" {
		t.Fatalf("unexpected filter result: %v", got.Records())
	}
}

func TestNotIn(t *testing.T) {
	df := FromRecords([]map[string]any{
		{"Configured_Status": "UNIQUE_MATCH"},
		{"Configured_Status": "NO_MATCH"},
		{"Configured_Status": "DYNAMIC_MATCH"},
	})
	mask := df.Col("Configured_Status").NotIn("UNIQUE_MATCH", "DYNAMIC_MATCH", "UNKNOWN_REMOTE")
	got := df.Filter(mask)
	if got.Len() != 1 || got.At(0, "Configured_Status") != "NO_MATCH" {
		t.Fatalf("unexpected NotIn result: %v", got.Records())
	}
}

func TestDuplicated(t *testing.T) {
	df := FromRecords([]map[string]any{
		{"Router_ID": "1.1.1.1", "VRF": "vrf1"},
		{"Router_ID": "1.1.1.2", "VRF": "vrf2"},
		{"Router_ID": "1.1.1.1", "VRF": "vrf3"},
	})
	dup := df.Filter(df.Duplicated([]string{"Router_ID"}, false))
	if dup.Len() != 2 {
		t.Fatalf("duplicated rows = %d, want 2", dup.Len())
	}
}

func TestGroupByFilter(t *testing.T) {
	df := FromRecords([]map[string]any{
		{"Node": "n1", "Router_ID": "1.1.1.1"},
		{"Node": "n2", "Router_ID": "1.1.1.1"},
		{"Node": "n3", "Router_ID": "1.1.1.2"},
	})
	// Keep groups whose Router_ID appears on more than one distinct node.
	got := df.GroupBy("Router_ID").Filter(func(g *DataFrame) bool {
		return g.Col("Node").NUnique() > 1
	})
	if got.Len() != 2 {
		t.Fatalf("groupby filter rows = %d, want 2: %v", got.Len(), got.Records())
	}
}

func TestSortValues(t *testing.T) {
	df := FromRecords([]map[string]any{
		{"Router_ID": "1.1.1.2"},
		{"Router_ID": "1.1.1.1"},
	})
	got := df.SortValues("Router_ID")
	if got.At(0, "Router_ID") != "1.1.1.1" || got.At(1, "Router_ID") != "1.1.1.2" {
		t.Fatalf("unexpected sort: %v", got.Records())
	}
}

func TestRecordsAndTake(t *testing.T) {
	df := FromRecords([]map[string]any{{"a": 1}, {"a": 2}})
	recs := df.Records()
	if len(recs) != 2 || recs[0]["a"] != 1 {
		t.Fatalf("records = %v", recs)
	}
}

func TestStringSelfConsistent(t *testing.T) {
	df := FromRecords([]map[string]any{{"col1": "a", "col2": 1}, {"col1": "b", "col2": 2}})
	s := df.String()
	if s == "" {
		t.Fatal("empty string representation")
	}
	if s != df.Copy().String() {
		t.Fatal("string representation is not deterministic")
	}
}
