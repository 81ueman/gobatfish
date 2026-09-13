package answer

import (
	"reflect"
	"testing"

	"github.com/81ueman/gobatfish/datamodel"
)

func TestBlankQuestionName(t *testing.T) {
	if got := NewAnswer(map[string]any{}).QuestionName(); got != nil {
		t.Fatalf("question name = %v", got)
	}
	if got := NewAnswer(map[string]any{"question": map[string]any{}}).QuestionName(); got != nil {
		t.Fatalf("question name = %v", got)
	}
	if got := NewAnswer(map[string]any{"question": map[string]any{"instance": map[string]any{}}}).QuestionName(); got != nil {
		t.Fatalf("question name = %v", got)
	}
}

func TestQuestionName(t *testing.T) {
	a := NewAnswer(map[string]any{
		"answerElements": []any{},
		"question":       map[string]any{"instance": map[string]any{"instanceName": "q_name"}},
	})
	if got := a.QuestionName(); got == nil || *got != "q_name" {
		t.Fatalf("question name = %v", got)
	}
}

func TestParseJSONWithSchemaSelfDescribing(t *testing.T) {
	got, err := ParseJSONWithSchema("SelfDescribing", map[string]any{"schema": "Integer", "value": 23})
	if err != nil || got != 23 {
		t.Fatalf("got %v, %v", got, err)
	}
	got, err = ParseJSONWithSchema("SelfDescribing", map[string]any{"schema": "Integer"})
	if err != nil || got != nil {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestParseJSONWithSchemaInteger(t *testing.T) {
	for _, in := range []any{"0", 0, -1, "-1"} {
		got, err := ParseJSONWithSchema("Integer", in)
		if err != nil {
			t.Fatal(err)
		}
		if got != 0 && got != -1 {
			t.Fatalf("got %v for %v", got, in)
		}
	}
}

func TestIterableSchema(t *testing.T) {
	if IsIterableSchema("x") || IsIterableSchema("") || IsIterableSchema("Integer") || IsIterableSchema("None") {
		t.Fatal("false positive iterable schema")
	}
	for _, s := range []string{"List<Integer>", "List<None>", "Set<String>", "set<String>", "list<acltrace>"} {
		if !IsIterableSchema(s) {
			t.Fatalf("expected iterable schema for %s", s)
		}
	}
}

func TestGetBaseSchema(t *testing.T) {
	cases := map[string]string{
		"List<Integer>":            "Integer",
		"Set<Integer>":             "Integer",
		"List<List<List<String>>>": "List<List<String>>",
		"invalid":                  "invalid",
		"Integer":                  "Integer",
	}
	for in, want := range cases {
		if got := GetBaseSchema(in); got != want {
			t.Fatalf("GetBaseSchema(%q) = %q, want %q", in, got, want)
		}
	}
	if got := GetBaseSchema(GetBaseSchema("List<Set<string>>")); got != "string" {
		t.Fatalf("nested base schema = %q", got)
	}
}

func TestColumnMetadata(t *testing.T) {
	cm, err := NewColumnMetadata(map[string]any{
		"name": "col1", "schema": "Node", "description": "itsme", "isKey": false, "isValue": false,
	})
	if err != nil || cm.Name != "col1" || cm.Schema != "Node" || cm.Description != "itsme" || cm.IsKey || cm.IsValue {
		t.Fatalf("column metadata = %+v, %v", cm, err)
	}
	if _, err := NewColumnMetadata(map[string]any{"noName": "col1", "schema": "Node"}); err == nil {
		t.Fatal("expected error for missing name")
	}
	if _, err := NewColumnMetadata(map[string]any{"name": "col1"}); err == nil {
		t.Fatal("expected error for missing schema")
	}
	cm, _ = NewColumnMetadata(map[string]any{"name": "col1", "schema": "Node"})
	if cm.Description == "" || !cm.IsKey || !cm.IsValue {
		t.Fatalf("optional fields not defaulted: %+v", cm)
	}
}

func TestTableAnswerValidation(t *testing.T) {
	if _, err := NewTableAnswer(map[string]any{}); err == nil {
		t.Fatal("expected error for missing answer elements")
	}
	if _, err := NewTableAnswer(map[string]any{"answerElements": []any{}}); err == nil {
		t.Fatal("expected error for empty answer elements")
	}
	if _, err := NewTableAnswer(map[string]any{"answerElements": []any{map[string]any{"nometadata": map[string]any{}}}}); err == nil {
		t.Fatal("expected error for missing metadata")
	}
}

func TestTableAnswerDeserialization(t *testing.T) {
	ta, err := NewTableAnswer(map[string]any{
		"answerElements": []any{map[string]any{
			"metadata": map[string]any{"columnMetadata": []any{
				map[string]any{"name": "col1", "schema": "String"},
				map[string]any{"name": "col2", "schema": "String"},
			}},
			"rows": []any{map[string]any{"col1": "value1", "col2": "value2"}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ta.Metadata.ColumnMetadata) != 2 || ta.Metadata.ColumnMetadata[0].Name != "col1" {
		t.Fatalf("metadata = %+v", ta.Metadata)
	}
	if len(ta.Rows) != 1 || ta.Rows[0]["col1"] != "value1" || ta.Rows[0]["col2"] != "value2" {
		t.Fatalf("rows = %v", ta.Rows)
	}
	if ta.Frame().Empty() {
		t.Fatal("expected non-empty frame")
	}
}

func TestTableAnswerNoRows(t *testing.T) {
	ta, err := NewTableAnswer(map[string]any{
		"answerElements": []any{map[string]any{
			"metadata": map[string]any{"columnMetadata": []any{map[string]any{"name": "col1", "schema": "Node"}}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ta.Rows) != 0 || !ta.Frame().Empty() {
		t.Fatalf("expected empty table: %v", ta)
	}
	if !reflect.DeepEqual(ta.Frame().Columns(), ta.Metadata.GetColumnNames()) {
		t.Fatalf("columns = %v, want %v", ta.Frame().Columns(), ta.Metadata.GetColumnNames())
	}
}

func TestTableAnswerExcludedRows(t *testing.T) {
	ta, err := NewTableAnswer(map[string]any{
		"answerElements": []any{map[string]any{
			"metadata": map[string]any{"columnMetadata": []any{map[string]any{"name": "col1", "schema": "String"}}},
			"excludedRows": []any{map[string]any{
				"exclusionName": "myEx", "rows": []any{map[string]any{"col1": "stringValue"}},
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(ta.ExcludedRows) != 1 {
		t.Fatalf("excluded rows = %v", ta.ExcludedRows)
	}
	frame, err := ta.ExcludedFrame("myEx")
	if err != nil || frame.Size() != 1 {
		t.Fatalf("excluded frame size = %d, %v", frame.Size(), err)
	}
	if _, err := ta.ExcludedFrame("missing"); err == nil {
		t.Fatal("expected error for missing exclusion")
	}
}

func TestTableAnswerImmutableLists(t *testing.T) {
	ta, err := NewTableAnswer(map[string]any{
		"answerElements": []any{map[string]any{
			"metadata": map[string]any{"columnMetadata": []any{map[string]any{"name": "col1", "schema": "List<String>"}}},
			"rows":     []any{map[string]any{"col1": []any{"e1", "e2", "e3"}}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := ta.Frame().At(0, "col1").(datamodel.ListWrapper); !ok {
		t.Fatalf("expected ListWrapper, got %T", ta.Frame().At(0, "col1"))
	}
}

func TestTableAnswerDtypeObject(t *testing.T) {
	ta, err := NewTableAnswer(map[string]any{
		"answerElements": []any{map[string]any{
			"metadata": map[string]any{"columnMetadata": []any{
				map[string]any{"name": "col1", "schema": "String"},
				map[string]any{"name": "col2", "schema": "Integer"},
			}},
			"rows": []any{
				map[string]any{"col1": "v1", "col2": 1},
				map[string]any{"col1": "v2", "col2": nil},
				map[string]any{"col1": "v3", "col2": -1},
				map[string]any{"col1": "v4", "col2": "-1"},
			},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	df := ta.Frame()
	if df.Col("col1").Dtype() != "object" || df.Col("col2").Dtype() != "object" {
		t.Fatal("expected object dtype")
	}
	if df.At(1, "col2") != nil {
		t.Fatalf("col2[1] = %v, want nil", df.At(1, "col2"))
	}
}

func TestIsTableAns(t *testing.T) {
	d := map[string]any{
		"answerElements": []any{map[string]any{
			"metadata": map[string]any{"columnMetadata": []any{map[string]any{"name": "col1", "schema": "List<String>"}}},
			"rows":     []any{map[string]any{"col1": []any{"e1"}}},
		}},
	}
	if IsTableAns(d) {
		t.Fatal("expected non-table answer")
	}
	d["answerElements"].([]any)[0].(map[string]any)["class"] = "org.batfish.datamodel.table.TableAnswerElement"
	if !IsTableAns(d) {
		t.Fatal("expected table answer")
	}
}
