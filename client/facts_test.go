package client

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/81ueman/gobatfish/datamodel/answer"
	"github.com/81ueman/gobatfish/question"
	"gopkg.in/yaml.v3"
)

type recordedCall struct {
	snapshot     string
	questionJSON string
}

type factStubSession struct {
	calls []recordedCall
}

func (s *factStubSession) GetSnapshot(snapshot *string) (string, error) {
	if snapshot != nil {
		return *snapshot, nil
	}
	return "snapshot", nil
}

func (s *factStubSession) AnswerQuestion(_ context.Context, questionStr, _ string, _ bool, snapshot string, _ *string, _ map[string]any) (answer.Result, error) {
	s.calls = append(s.calls, recordedCall{snapshot: snapshot, questionJSON: questionStr})
	return answer.NewTableAnswer(emptyTableDict())
}

func (s *factStubSession) FetchQuestionTemplates(context.Context, bool) (map[string]string, error) {
	return map[string]string{}, nil
}

func emptyTableDict() map[string]any {
	return map[string]any{"answerElements": []any{map[string]any{
		"metadata": map[string]any{"columnMetadata": []any{}},
		"rows":     []any{},
	}}}
}

var factQuestions = []string{
	"nodeProperties", "interfaceProperties", "bgpProcessConfiguration", "bgpPeerConfiguration",
	"ospfProcessConfiguration", "ospfAreaConfiguration", "ospfInterfaceConfiguration",
}

func newFactSession(t *testing.T, stub *factStubSession) *Session {
	t.Helper()
	bf := NewSession(SessionConfig{})
	qs := question.NewQuestions(stub)
	for _, name := range factQuestions {
		_, tmpl, err := question.LoadQuestionDict(map[string]any{"instance": map[string]any{
			"instanceName": name, "description": name,
			"variables": map[string]any{"nodes": map[string]any{"description": "nodes.", "type": "string"}},
		}}, stub)
		if err != nil {
			t.Fatal(err)
		}
		qs.Install(tmpl)
	}
	bf.Q = qs
	return bf
}

func decodeQuestionCall(t *testing.T, raw string) (string, map[string]any) {
	t.Helper()
	var q map[string]any
	if err := json.Unmarshal([]byte(raw), &q); err != nil {
		t.Fatal(err)
	}
	instance := q["instance"].(map[string]any)
	name := instance["instanceName"].(string)
	var nodes map[string]any
	if vars, ok := instance["variables"].(map[string]any); ok {
		if n, ok := vars["nodes"].(map[string]any); ok {
			nodes = n
		}
	}
	return name, nodes
}

func TestGetFactsQuestions(t *testing.T) {
	stub := &factStubSession{}
	bf := newFactSession(t, stub)
	if _, err := GetFacts(context.Background(), bf, "foo", nil); err != nil {
		t.Fatal(err)
	}
	if len(stub.calls) != len(factQuestions) {
		t.Fatalf("calls = %d, want %d", len(stub.calls), len(factQuestions))
	}
	seen := map[string]bool{}
	for _, call := range stub.calls {
		name, nodes := decodeQuestionCall(t, call.questionJSON)
		if nodes["value"] != "foo" {
			t.Fatalf("%s nodes = %v", name, nodes["value"])
		}
		for _, expected := range factQuestions {
			if strings.HasPrefix(name, "__"+expected+"_") {
				seen[expected] = true
			}
		}
	}
	for _, expected := range factQuestions {
		if !seen[expected] {
			t.Fatalf("question %s was not called", expected)
		}
	}
}

func TestGetFactsQuestionsSpecificSnapshot(t *testing.T) {
	stub := &factStubSession{}
	bf := newFactSession(t, stub)
	snapshot := "snapshot"
	if _, err := GetFacts(context.Background(), bf, "foo", &snapshot); err != nil {
		t.Fatal(err)
	}
	for _, call := range stub.calls {
		if call.snapshot != "snapshot" {
			t.Fatalf("snapshot = %q", call.snapshot)
		}
	}
}

func TestLoadFacts(t *testing.T) {
	dir := t.TempDir()
	version := "fake_version"
	writeYAML(t, filepath.Join(dir, "node1.yml"), encapsulateNodesFacts(map[string]any{"node1": "foo"}, version))
	writeYAML(t, filepath.Join(dir, "node2.yml"), encapsulateNodesFacts(map[string]any{"node2": "foo"}, version))

	facts, err := LoadFacts(dir)
	if err != nil {
		t.Fatal(err)
	}
	want := encapsulateNodesFacts(map[string]any{"node1": "foo", "node2": "foo"}, version)
	if !reflect.DeepEqual(facts, want) {
		t.Fatalf("facts = %v, want %v", facts, want)
	}
}

func TestLoadFactsBadDir(t *testing.T) {
	empty := t.TempDir()
	if _, err := LoadFacts(empty); err == nil || !strings.Contains(err.Error(), "No files present in specified directory") {
		t.Fatalf("err = %v", err)
	}
	file := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(file, []byte("foo"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadFacts(file); err == nil || !strings.Contains(err.Error(), "Not a directory") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadFactsMismatchVersion(t *testing.T) {
	dir := t.TempDir()
	writeYAML(t, filepath.Join(dir, "node1.yml"), encapsulateNodesFacts(map[string]any{"node1": "foo"}, "version1"))
	writeYAML(t, filepath.Join(dir, "node2.yml"), encapsulateNodesFacts(map[string]any{"node2": "foo"}, "version2"))
	if _, err := LoadFacts(dir); err == nil || !strings.Contains(err.Error(), "Input file version mismatch") {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateFacts(t *testing.T) {
	expected := map[string]any{"node1": map[string]any{"foo": 1}, "node2": map[string]any{"foo": 2}}
	actual := map[string]any{"node1": map[string]any{"foo": 1}, "node2": map[string]any{"foo": 2}, "node3": map[string]any{"foo": 3}}
	res := ValidateFacts(encapsulateNodesFacts(expected, "fake_version"), encapsulateNodesFacts(actual, "fake_version"), false)
	if len(res) != 0 {
		t.Fatalf("res = %v", res)
	}
}

func TestValidateFactsNotMatchingVersion(t *testing.T) {
	expected := map[string]any{"node1": map[string]any{"foo": 1}, "node2": map[string]any{"foo": 2}}
	actual := map[string]any{"node1": map[string]any{"foo": 1}, "node2": map[string]any{"foo": 2}, "node3": map[string]any{"foo": 3}}
	res := ValidateFacts(encapsulateNodesFacts(expected, "correct_version"), encapsulateNodesFacts(actual, "fake_version"), false)
	if len(res) != len(expected) {
		t.Fatalf("res = %v", res)
	}
	for node := range expected {
		want := map[string]any{"Version": map[string]any{"actual": "fake_version", "expected": "correct_version"}}
		if !reflect.DeepEqual(res[node], want) {
			t.Fatalf("%s = %v, want %v", node, res[node], want)
		}
	}
}

func TestValidateFactsNotMatchingData(t *testing.T) {
	expected := map[string]any{"node1": map[string]any{"foo": 1, "bar": 1, "baz": 1}, "node2": map[string]any{"foo": 2}}
	actual := map[string]any{"node1": map[string]any{"foo": 0, "bar": 1}, "node2": map[string]any{"foo": 2}, "node3": map[string]any{"foo": 3}}
	res := ValidateFacts(encapsulateNodesFacts(expected, "version"), encapsulateNodesFacts(actual, "version"), false)
	want := map[string]any{"node1": map[string]any{
		"foo": map[string]any{"expected": 1, "actual": 0},
		"baz": map[string]any{"expected": 1, "key_present": false},
	}}
	if !reflect.DeepEqual(res, want) {
		t.Fatalf("res = %v, want %v", res, want)
	}
}

func TestValidateFactsNoMatchingNode(t *testing.T) {
	expected := map[string]any{"node1": map[string]any{"foo": 1, "bar": 1, "baz": 1}, "node2": map[string]any{"foo": 2}}
	actual := map[string]any{"node1": map[string]any{"foo": 1, "bar": 1, "baz": 1}}
	res := ValidateFacts(encapsulateNodesFacts(expected, "version"), encapsulateNodesFacts(actual, "version"), false)
	want := map[string]any{"node2": map[string]any{"foo": map[string]any{"expected": 2, "key_present": false}}}
	if !reflect.DeepEqual(res, want) {
		t.Fatalf("res = %v, want %v", res, want)
	}
}

func TestValidateFactsVerbose(t *testing.T) {
	expected := map[string]any{"node1": map[string]any{"foo": 1, "bar": 1, "baz": 1}, "node2": map[string]any{"foo": 2}}
	actual := map[string]any{"node1": map[string]any{"foo": 0, "bar": 1}, "node2": map[string]any{"foo": 2}, "node3": map[string]any{"foo": 3}}
	res := ValidateFacts(encapsulateNodesFacts(expected, "version"), encapsulateNodesFacts(actual, "version"), true)
	want := map[string]any{
		"node1": map[string]any{
			"foo": map[string]any{"expected": 1, "actual": 0},
			"bar": map[string]any{"expected": 1, "actual": 1},
			"baz": map[string]any{"expected": 1, "key_present": false},
		},
		"node2": map[string]any{"foo": map[string]any{"expected": 2, "actual": 2}},
	}
	if !reflect.DeepEqual(res, want) {
		t.Fatalf("res = %v, want %v", res, want)
	}
}

func TestWriteFacts(t *testing.T) {
	dir := t.TempDir()
	nodes := map[string]any{"node1": "foo", "node2": "bar"}
	facts := encapsulateNodesFacts(nodes, "version")
	if err := WriteFacts(dir, facts); err != nil {
		t.Fatal(err)
	}
	for node, want := range nodes {
		data, err := os.ReadFile(filepath.Join(dir, node+".yml"))
		if err != nil {
			t.Fatal(err)
		}
		var raw map[string]any
		if err := yaml.Unmarshal(data, &raw); err != nil {
			t.Fatal(err)
		}
		gotNodes, version := unencapsulateFacts(raw)
		if version != "version" {
			t.Fatalf("%s version = %v", node, version)
		}
		gotMap, _ := gotNodes.(map[string]any)
		if !reflect.DeepEqual(gotMap[node], want) {
			t.Fatalf("%s facts = %v, want %v", node, gotMap[node], want)
		}
	}
}

func TestAssertDictSubsetEqual(t *testing.T) {
	actual := map[string]any{"key": "value", "parent_key": map[string]any{"nested_key": "nested_value"}, "list": []any{"foo"}, "empty_list": []any{}, "none": nil}
	expected := map[string]any{"key": "value", "parent_key": map[string]any{"nested_key": "nested_value"}, "list": []any{"foo"}, "empty_list": []any{}, "none": nil}
	if got := assertDictSubset(actual, expected, "", nil, false); len(got) != 0 {
		t.Fatalf("diffs = %v", got)
	}
}

func TestAssertDictSubsetSubset(t *testing.T) {
	actual := map[string]any{"key": "value", "key2": "value2", "parent_key": map[string]any{"nested_key": "nested_value", "nested_key2": "nested_value2"}}
	expected := map[string]any{"key": "value", "parent_key": map[string]any{"nested_key": "nested_value"}}
	if got := assertDictSubset(actual, expected, "", nil, false); len(got) != 0 {
		t.Fatalf("diffs = %v", got)
	}
}

func TestAssertDictSubsetNotEqual(t *testing.T) {
	actual := map[string]any{
		"key": "value", "key2": "value2",
		"parent_key":    map[string]any{"nested_key": "nested_value", "nested_key2": "nested_value2", "different_nested_key": "not_different_value"},
		"different_key": "not_different_value",
	}
	expected := map[string]any{
		"key":           "value",
		"parent_key":    map[string]any{"nested_key": "nested_value", "missing_nested_key": "missing_value", "different_nested_key": "different_value"},
		"missing_key":   "missing_value",
		"different_key": "different_value",
	}
	want := map[string]any{
		"parent_key.missing_nested_key":   map[string]any{"expected": "missing_value", "key_present": false},
		"parent_key.different_nested_key": map[string]any{"expected": "different_value", "actual": "not_different_value"},
		"missing_key":                     map[string]any{"expected": "missing_value", "key_present": false},
		"different_key":                   map[string]any{"expected": "different_value", "actual": "not_different_value"},
	}
	if got := assertDictSubset(actual, expected, "", nil, false); !reflect.DeepEqual(got, want) {
		t.Fatalf("diffs = %v, want %v", got, want)
	}
}

func writeYAML(t *testing.T, path string, v any) {
	t.Helper()
	data, err := yaml.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
