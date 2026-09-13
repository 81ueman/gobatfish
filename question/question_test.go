package question

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/81ueman/gobatfish/datamodel"
	"github.com/81ueman/gobatfish/datamodel/answer"
)

type fakeSession struct{}

func (fakeSession) GetSnapshot(snapshot *string) (string, error) { return "snapshot", nil }
func (fakeSession) AnswerQuestion(context.Context, string, string, bool, string, *string, map[string]any) (answer.Result, error) {
	return nil, nil
}
func (fakeSession) FetchQuestionTemplates(context.Context, bool) (map[string]string, error) {
	return map[string]string{}, nil
}

const testQuestionName = "testQuestionName"

func testQuestionDict() map[string]any {
	return map[string]any{
		"instance": map[string]any{
			"instanceName": testQuestionName,
			"description":  "a test question",
			"variables": map[string]any{
				"var1": map[string]any{
					"description": "desc1",
					"type":        "type1",
					"value":       "val1",
					"displayName": "display1",
				},
			},
		},
	}
}

func TestValidateMinLength(t *testing.T) {
	numbers := map[string]any{
		"minElements": 0, "minLength": 4, "type": "string",
		"value": []any{"one", "three", "four", "five", "seven"},
	}
	sample := map[string]any{"instance": map[string]any{"variables": map[string]any{"numbers": numbers}}}
	want := "\n   Length of value: 'one' for element : 0 of parameter: 'numbers' below minimum length: 4\n"
	err := validate(sample)
	if err == nil || err.Error() != want {
		t.Fatalf("error = %q, want %q", err, want)
	}
}

func TestValidateComparator(t *testing.T) {
	sample := map[string]any{"instance": map[string]any{"comparators": map[string]any{"type": "comparator", "value": ">"}}}
	if err := validate(sample); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAllowedValues(t *testing.T) {
	variable := map[string]any{
		"allowedValues": []any{"obsolete value"}, "type": "string", "value": "v1",
		"values": []any{map[string]any{"name": "v1"}},
	}
	sample := map[string]any{"instance": map[string]any{"variables": map[string]any{"v": variable}}}
	if err := validate(sample); err != nil {
		t.Fatal(err)
	}
	variable["value"] = "obsolete value"
	want := "\n   Value: 'obsolete value' is not among allowed values ['v1'] of parameter: 'v'\n"
	if err := validate(sample); err == nil || err.Error() != want {
		t.Fatalf("error = %q, want %q", err, want)
	}
}

func TestValidateOldAllowedValues(t *testing.T) {
	variable := map[string]any{"allowedValues": []any{"v1"}, "type": "string", "value": "v1"}
	sample := map[string]any{"instance": map[string]any{"variables": map[string]any{"v": variable}}}
	if err := validate(sample); err != nil {
		t.Fatal(err)
	}
	variable["value"] = "bad value"
	want := "\n   Value: 'bad value' is not among allowed values ['v1'] of parameter: 'v'\n"
	if err := validate(sample); err == nil || err.Error() != want {
		t.Fatalf("error = %q, want %q", err, want)
	}
}

func TestValidateAllowedValuesList(t *testing.T) {
	variable := map[string]any{
		"minElements": 0, "allowedValues": []any{"obsolete value"}, "type": "string",
		"value": []any{"v1"}, "values": []any{map[string]any{"name": "v1"}},
	}
	sample := map[string]any{"instance": map[string]any{"variables": map[string]any{"v": variable}}}
	if err := validate(sample); err != nil {
		t.Fatal(err)
	}
	variable["value"] = []any{"obsolete value"}
	want := "\n   Value: 'obsolete value' is not among allowed values ['v1'] of parameter: 'v'\n"
	if err := validate(sample); err == nil || err.Error() != want {
		t.Fatalf("error = %q, want %q", err, want)
	}
}

func TestComputeDocstring(t *testing.T) {
	if got := computeDocstring("foo", []string{}, map[string]any{}); got != "foo" {
		t.Fatalf("docstring = %q", got)
	}
}

func TestComputeVarHelp(t *testing.T) {
	cases := []struct {
		name string
		data map[string]any
		want string
	}{
		{
			"default falsy",
			map[string]any{"optional": true, "description": "Desc", "type": "boolean", "value": false},
			":param v: Desc\n\n    Default value: ``False``\n:type v: boolean",
		},
		{
			"no allowed values",
			map[string]any{"optional": true, "description": "Desc", "type": "boolean"},
			":param v: Desc\n:type v: boolean",
		},
		{
			"allowed values",
			map[string]any{"optional": true, "description": "variable description", "type": "boolean",
				"values": []any{map[string]any{"name": "v1", "description": "v1 description"}, map[string]any{"name": "v2", "description": "v2 description"}}},
			":param v: variable description\n    Allowed values:\n\n    * v1: v1 description\n    * v2: v2 description\n:type v: boolean",
		},
		{
			"old allowed values",
			map[string]any{"allowedValues": []any{"v1"}, "optional": true, "description": "variable description", "type": "boolean"},
			":param v: variable description\n    Allowed values:\n\n    * v1\n:type v: boolean",
		},
	}
	for _, c := range cases {
		if got := computeVarHelp("v", c.data); got != c.want {
			t.Fatalf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestProcessVariables(t *testing.T) {
	if got, _ := processVariables("foo", nil, nil); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
	variables := map[string]any{
		"c": map[string]any{"description": "c description", "optional": true, "type": "boolean"},
		"d": map[string]any{"description": "d description", "optional": false, "type": "boolean"},
		"a": map[string]any{"description": "a description", "optional": true, "type": "boolean"},
		"b": map[string]any{"description": "b description", "optional": false, "type": "boolean"},
	}
	if got, _ := processVariables("foo", variables, nil); !reflect.DeepEqual(got, []string{"b", "d", "a", "c"}) {
		t.Fatalf("default order = %v", got)
	}
	if got, _ := processVariables("foo", variables, []string{"d", "c", "b"}); !reflect.DeepEqual(got, []string{"b", "d", "a", "c"}) {
		t.Fatalf("invalid order = %v", got)
	}
	if got, _ := processVariables("foo", variables, []string{"d", "c", "b", "a"}); !reflect.DeepEqual(got, []string{"d", "c", "b", "a"}) {
		t.Fatalf("valid order = %v", got)
	}
}

func TestHasValidOrderedVariableNames(t *testing.T) {
	variables := map[string]any{"a": map[string]any{}, "b": map[string]any{}, "c": map[string]any{}}
	if hasValidOrderedVariableNames(nil, variables) {
		t.Fatal("expected false for empty")
	}
	if hasValidOrderedVariableNames([]string{"a"}, variables) || hasValidOrderedVariableNames([]string{"a", "c"}, variables) {
		t.Fatal("expected false for incomplete")
	}
	if hasValidOrderedVariableNames([]string{"a", "c", "b", "b"}, variables) {
		t.Fatal("expected false for duplicate")
	}
	if hasValidOrderedVariableNames([]string{"a", "c", "b", "d"}, variables) {
		t.Fatal("expected false for extraneous")
	}
	if !hasValidOrderedVariableNames([]string{"a", "c", "b"}, variables) {
		t.Fatal("expected true")
	}
}

func TestLoadQuestionsFromDir(t *testing.T) {
	dir := t.TempDir()
	data, _ := json.Marshal(testQuestionDict())
	if err := os.WriteFile(filepath.Join(dir, testQuestionName+".json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded := LoadQuestionsFromDir(dir, fakeSession{})
	if len(loaded) != 1 || loaded[testQuestionName] == nil {
		t.Fatalf("loaded = %v", loaded)
	}
	// Fault tolerance to bad questions.
	if err := os.WriteFile(filepath.Join(dir, "badq.json"), []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	loaded = LoadQuestionsFromDir(dir, fakeSession{})
	if len(loaded) != 1 {
		t.Fatalf("loaded = %v", loaded)
	}
}

func TestListQuestions(t *testing.T) {
	dir := t.TempDir()
	data, _ := json.Marshal(testQuestionDict())
	if err := os.WriteFile(filepath.Join(dir, testQuestionName+".json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	qs := NewQuestions(fakeSession{})
	if err := qs.Load(context.Background(), dir); err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0)
	for _, q := range qs.List() {
		names = append(names, q["name"].(string))
	}
	if !reflect.DeepEqual(names, []string{testQuestionName}) {
		t.Fatalf("names = %v", names)
	}
}

func TestMakeCheck(t *testing.T) {
	_, tmpl, err := LoadQuestionDict(testQuestionDict(), fakeSession{})
	if err != nil {
		t.Fatal(err)
	}
	q := tmpl.New().MakeCheck()
	want := datamodel.Assertion{Type: datamodel.AssertionCountEquals, Expect: 0}.Dict()
	if got := q.Dict()["assertion"]; !reflect.DeepEqual(got, want) {
		t.Fatalf("assertion = %v, want %v", got, want)
	}
}

func TestQuestionJSONWithDatamodel(t *testing.T) {
	dict := map[string]any{"instance": map[string]any{
		"instanceName": "q", "description": "d",
		"variables": map[string]any{"headers": map[string]any{"description": "h.", "type": "headerConstraint", "value": "x"}},
	}}
	_, tmpl, err := LoadQuestionDict(dict, fakeSession{})
	if err != nil {
		t.Fatal(err)
	}
	q := tmpl.New().Set("headers", datamodel.HeaderConstraints{SrcIps: "1.1.1.1"})
	s, err := q.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s, `"srcIps": "1.1.1.1"`) || !strings.Contains(s, `"dstIps": null`) {
		t.Fatalf("question json = %s", s)
	}
}

func TestQuestionName(t *testing.T) {
	_, tmpl, err := LoadQuestionDict(testQuestionDict(), fakeSession{})
	if err != nil {
		t.Fatal(err)
	}
	if got := tmpl.New().Set("question_name", "manually set").Name(); got != "manually set" {
		t.Fatalf("name = %q", got)
	}
	// Default name mirrors the Python "__<name>_<uuid>" convention.
	if got := tmpl.New().Name(); !strings.HasPrefix(got, "__"+testQuestionName+"_") {
		t.Fatalf("default name = %q", got)
	}
}
