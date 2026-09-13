package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/81ueman/gobatfish/dataframe"
	"github.com/81ueman/gobatfish/datamodel"
	"github.com/81ueman/gobatfish/datamodel/answer"
	"github.com/81ueman/gobatfish/exception"
	"github.com/81ueman/gobatfish/question"
)

type assertCall struct {
	question string
	snapshot string
	vars     map[string]any
}

type assertStub struct {
	answers map[string]answer.Result
	calls   []assertCall
}

func newAssertStub() *assertStub {
	return &assertStub{answers: map[string]answer.Result{}}
}

func (s *assertStub) GetSnapshot(snapshot *string) (string, error) {
	if snapshot != nil {
		return *snapshot, nil
	}
	return "snapshot", nil
}

func (s *assertStub) AnswerQuestion(_ context.Context, questionStr, _ string, _ bool, snapshot string, _ *string, _ map[string]any) (answer.Result, error) {
	var q map[string]any
	if err := json.Unmarshal([]byte(questionStr), &q); err != nil {
		return nil, err
	}
	instance, _ := q["instance"].(map[string]any)
	name := parseQuestionName(fmt.Sprint(instance["instanceName"]))
	vars := map[string]any{}
	if vs, ok := instance["variables"].(map[string]any); ok {
		for k, v := range vs {
			if m, ok := v.(map[string]any); ok {
				if value, hasValue := m["value"]; hasValue {
					vars[k] = value
				}
			}
		}
	}
	s.calls = append(s.calls, assertCall{question: name, snapshot: snapshot, vars: vars})
	if a, ok := s.answers[name]; ok {
		return a, nil
	}
	return emptyTableAnswer(), nil
}

func (s *assertStub) FetchQuestionTemplates(context.Context, bool) (map[string]string, error) {
	return map[string]string{}, nil
}

func (s *assertStub) last(t *testing.T, name string) assertCall {
	t.Helper()
	for i := len(s.calls) - 1; i >= 0; i-- {
		if s.calls[i].question == name {
			return s.calls[i]
		}
	}
	t.Fatalf("no call recorded for %s", name)
	return assertCall{}
}

func parseQuestionName(instanceName string) string {
	if strings.HasPrefix(instanceName, "__") {
		rest := instanceName[2:]
		if i := strings.LastIndex(rest, "_"); i >= 0 {
			return rest[:i]
		}
		return rest
	}
	return instanceName
}

var assertQuestionNames = []string{
	"searchFilters", "filterLineReachability", "reachability", "bgpSessionCompatibility",
	"ospfSessionCompatibility", "bgpSessionStatus", "undefinedReferences", "detectLoops",
	"ospfProcessConfiguration", "bgpProcessConfiguration",
}

func assertVariableDefs() map[string]any {
	types := map[string]string{"headers": "headerConstraint", "pathConstraints": "pathConstraint"}
	names := []string{"filters", "headers", "action", "startLocation", "pathConstraints", "actions", "nodes", "remoteNodes", "status", "statuses"}
	out := make(map[string]any, len(names))
	for _, name := range names {
		t := "string"
		if ty, ok := types[name]; ok {
			t = ty
		}
		out[name] = map[string]any{"description": name + ".", "type": t, "optional": true}
	}
	return out
}

func newAssertSession(t *testing.T, stub *assertStub) *Session {
	t.Helper()
	bf := NewSession(SessionConfig{})
	qs := question.NewQuestions(stub)
	for _, name := range assertQuestionNames {
		_, tmpl, err := question.LoadQuestionDict(map[string]any{"instance": map[string]any{
			"instanceName": name, "description": name, "variables": assertVariableDefs(),
		}}, stub)
		if err != nil {
			t.Fatal(err)
		}
		qs.Install(tmpl)
	}
	bf.Q = qs
	return bf
}

func tableAnswer(t *testing.T, columns []any, rows []any) *answer.TableAnswer {
	t.Helper()
	ta, err := answer.NewTableAnswer(map[string]any{"answerElements": []any{map[string]any{
		"metadata": map[string]any{"columnMetadata": columns},
		"rows":     rows,
	}}})
	if err != nil {
		t.Fatal(err)
	}
	return ta
}

func emptyTableAnswer() *answer.TableAnswer {
	ta, err := answer.NewTableAnswer(map[string]any{"answerElements": []any{map[string]any{
		"metadata": map[string]any{"columnMetadata": []any{}},
		"rows":     []any{},
	}}})
	if err != nil {
		panic(err)
	}
	return ta
}

func foundTable(t *testing.T) *answer.TableAnswer {
	t.Helper()
	return tableAnswer(t, []any{
		map[string]any{"name": "Flow", "schema": "String"},
		map[string]any{"name": "More", "schema": "String"},
	}, []any{map[string]any{"Flow": "found", "More": "data"}})
}

func normalizedDict(t *testing.T, hc datamodel.HeaderConstraints) map[string]any {
	t.Helper()
	normalized, err := hc.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	return normalized.Dict()
}

func TestRaiseCommonDefault(t *testing.T) {
	_, err := raiseCommon("foobar", false)
	var assertErr *exception.BatfishAssertError
	if err == nil || !errors.As(err, &assertErr) || !strings.Contains(err.Error(), "foobar") {
		t.Fatalf("err = %v", err)
	}
	if _, err := raiseCommon("foobaragain", false); err == nil || !strings.Contains(err.Error(), "foobaragain") {
		t.Fatalf("err = %v", err)
	}
}

func TestRaiseCommonWarn(t *testing.T) {
	pass, err := raiseCommon("foobar", true)
	if pass || err != nil {
		t.Fatalf("pass = %v, err = %v", pass, err)
	}
}

func TestFormatDFTable(t *testing.T) {
	df := dataframe.FromRecords([]map[string]any{{"col1": 1, "col2": 3}, {"col1": 2, "col2": 4}})
	got, err := formatDF(df, "table")
	if err != nil || got != df.String() {
		t.Fatalf("got %q, err %v", got, err)
	}
}

func TestFormatDFRecords(t *testing.T) {
	df := dataframe.FromRecords([]map[string]any{{"col1": 1, "col2": 3}, {"col1": 2, "col2": 4}})
	got, err := formatDF(df, "records")
	if err != nil || got != fmt.Sprint(df.Records()) {
		t.Fatalf("got %q, err %v", got, err)
	}
}

func TestFilterDenies(t *testing.T) {
	headers := datamodel.HeaderConstraints{SrcIps: "1.1.1.1"}
	stub := newAssertStub()
	bf := newAssertSession(t, stub)

	if _, err := AssertFilterDenies(context.Background(), "filter", headers, nil, false, nil, bf, "table"); err != nil {
		t.Fatal(err)
	}
	call := stub.last(t, "searchFilters")
	if call.vars["filters"] != "filter" || call.vars["action"] != "permit" {
		t.Fatalf("vars = %v", call.vars)
	}
	if !reflect.DeepEqual(call.vars["headers"], normalizedDict(t, headers)) {
		t.Fatalf("headers = %v", call.vars["headers"])
	}
	if _, ok := call.vars["startLocation"]; ok {
		t.Fatalf("unexpected startLocation: %v", call.vars)
	}

	ta := foundTable(t)
	stub.answers["searchFilters"] = ta
	start := "Ethernet1"
	_, err := AssertFilterDenies(context.Background(), "filter", headers, &start, false, nil, bf, "table")
	if err == nil || !strings.Contains(err.Error(), ta.Frame().String()) {
		t.Fatalf("err = %v", err)
	}
	if got := stub.last(t, "searchFilters").vars["startLocation"]; got != "Ethernet1" {
		t.Fatalf("startLocation = %v", got)
	}
}

func TestFilterDeniesFromSession(t *testing.T) {
	headers := datamodel.HeaderConstraints{SrcIps: "1.1.1.1"}
	stub := newAssertStub()
	bf := newAssertSession(t, stub)

	if _, err := bf.Asserts.AssertFilterDenies(context.Background(), "filter", headers, nil, false, nil, "table"); err != nil {
		t.Fatal(err)
	}
	ta := foundTable(t)
	stub.answers["searchFilters"] = ta
	start := "Ethernet1"
	if _, err := bf.Asserts.AssertFilterDenies(context.Background(), "filter", headers, &start, false, nil, "table"); err == nil {
		t.Fatal("expected assertion failure")
	}
}

func TestFilterHasNoUnreachableLines(t *testing.T) {
	stub := newAssertStub()
	bf := newAssertSession(t, stub)
	if _, err := AssertFilterHasNoUnreachableLines(context.Background(), "filter1", false, nil, bf, "table"); err != nil {
		t.Fatal(err)
	}
	ta := foundTable(t)
	stub.answers["filterLineReachability"] = ta
	_, err := AssertFilterHasNoUnreachableLines(context.Background(), "filter1", false, nil, bf, "table")
	if err == nil || !strings.Contains(err.Error(), ta.Frame().String()) {
		t.Fatalf("err = %v", err)
	}
	if _, err := bf.Asserts.AssertFilterHasNoUnreachableLines(context.Background(), "filter1", false, nil, "table"); err == nil {
		t.Fatal("expected failure from session method")
	}
}

func TestFilterPermits(t *testing.T) {
	headers := datamodel.HeaderConstraints{SrcIps: "1.1.1.1"}
	stub := newAssertStub()
	bf := newAssertSession(t, stub)
	if _, err := AssertFilterPermits(context.Background(), "filter", headers, nil, false, nil, bf, "table"); err != nil {
		t.Fatal(err)
	}
	call := stub.last(t, "searchFilters")
	if call.vars["action"] != "deny" {
		t.Fatalf("action = %v", call.vars["action"])
	}
	ta := foundTable(t)
	stub.answers["searchFilters"] = ta
	start := "Ethernet1"
	_, err := AssertFilterPermits(context.Background(), "filter", headers, &start, false, nil, bf, "table")
	if err == nil || !strings.Contains(err.Error(), ta.Frame().String()) {
		t.Fatalf("err = %v", err)
	}
	if _, err := bf.Asserts.AssertFilterPermits(context.Background(), "filter", headers, nil, false, nil, "table"); err == nil {
		t.Fatal("expected failure from session method")
	}
}

func TestFlowsFail(t *testing.T) {
	headers := datamodel.HeaderConstraints{SrcIps: "1.1.1.1"}
	stub := newAssertStub()
	bf := newAssertSession(t, stub)
	if _, err := AssertFlowsFail(context.Background(), "node1", headers, false, nil, bf, "table"); err != nil {
		t.Fatal(err)
	}
	call := stub.last(t, "reachability")
	if call.vars["actions"] != "success" {
		t.Fatalf("actions = %v", call.vars["actions"])
	}
	pc, ok := call.vars["pathConstraints"].(map[string]any)
	if !ok || pc["startLocation"] != "node1" {
		t.Fatalf("pathConstraints = %v", call.vars["pathConstraints"])
	}
	stub.answers["reachability"] = foundTable(t)
	if _, err := bf.Asserts.AssertFlowsFail(context.Background(), "node1", headers, false, nil, "table"); err == nil {
		t.Fatal("expected failure")
	}
}

func TestFlowsSucceed(t *testing.T) {
	headers := datamodel.HeaderConstraints{SrcIps: "1.1.1.1"}
	stub := newAssertStub()
	bf := newAssertSession(t, stub)
	if _, err := AssertFlowsSucceed(context.Background(), "node1", headers, false, nil, bf, "table"); err != nil {
		t.Fatal(err)
	}
	if got := stub.last(t, "reachability").vars["actions"]; got != "failure" {
		t.Fatalf("actions = %v", got)
	}
	stub.answers["reachability"] = foundTable(t)
	if _, err := bf.Asserts.AssertFlowsSucceed(context.Background(), "node1", headers, false, nil, "table"); err == nil {
		t.Fatal("expected failure")
	}
}

func TestNoIncompatibleBGPSessions(t *testing.T) {
	stub := newAssertStub()
	bf := newAssertSession(t, stub)
	nodes, remote, status := "nodes", "remote_nodes", ".*"
	if _, err := AssertNoIncompatibleBGPSessions(context.Background(), &nodes, &remote, &status, nil, false, bf, "table"); err != nil {
		t.Fatal(err)
	}
	call := stub.last(t, "bgpSessionCompatibility")
	if call.vars["nodes"] != "nodes" || call.vars["remoteNodes"] != "remote_nodes" || call.vars["status"] != ".*" {
		t.Fatalf("vars = %v", call.vars)
	}
	stub.answers["bgpSessionCompatibility"] = foundTable(t)
	if _, err := bf.Asserts.AssertNoIncompatibleBGPSessions(context.Background(), &nodes, &remote, &status, false, nil, "table"); err == nil {
		t.Fatal("expected failure")
	}
}

func TestNoIncompatibleOSPFSessions(t *testing.T) {
	stub := newAssertStub()
	bf := newAssertSession(t, stub)
	nodes, remote := "nodes", "remote_nodes"
	if _, err := AssertNoIncompatibleOSPFSessions(context.Background(), &nodes, &remote, nil, false, bf, "table"); err != nil {
		t.Fatal(err)
	}
	call := stub.last(t, "ospfSessionCompatibility")
	if call.vars["statuses"] != UnestablishedOSPFSSessionStatusSpec {
		t.Fatalf("statuses = %v", call.vars["statuses"])
	}
	stub.answers["ospfSessionCompatibility"] = foundTable(t)
	if _, err := bf.Asserts.AssertNoIncompatibleOSPFSessions(context.Background(), &nodes, &remote, false, nil, "table"); err == nil {
		t.Fatal("expected failure")
	}
}

func TestNoUnestablishedBGPSessions(t *testing.T) {
	stub := newAssertStub()
	bf := newAssertSession(t, stub)
	nodes, remote := "nodes", "remote_nodes"
	if _, err := AssertNoUnestablishedBGPSessions(context.Background(), &nodes, &remote, nil, false, bf, "table"); err != nil {
		t.Fatal(err)
	}
	if got := stub.last(t, "bgpSessionStatus").vars["status"]; got != "NOT_ESTABLISHED" {
		t.Fatalf("status = %v", got)
	}
	stub.answers["bgpSessionStatus"] = foundTable(t)
	if _, err := bf.Asserts.AssertNoUnestablishedBGPSessions(context.Background(), &nodes, &remote, false, nil, "table"); err == nil {
		t.Fatal("expected failure")
	}
}

func TestNoUndefinedReferences(t *testing.T) {
	stub := newAssertStub()
	bf := newAssertSession(t, stub)
	if _, err := AssertNoUndefinedReferences(context.Background(), nil, false, bf, "table"); err != nil {
		t.Fatal(err)
	}
	stub.answers["undefinedReferences"] = foundTable(t)
	if _, err := bf.Asserts.AssertNoUndefinedReferences(context.Background(), false, nil, "table"); err == nil {
		t.Fatal("expected failure")
	}
}

func TestNoDuplicateRouterIDsOSPF(t *testing.T) {
	stub := newAssertStub()
	bf := newAssertSession(t, stub)
	unique := tableAnswer(t, []any{
		map[string]any{"name": "Router_ID", "schema": "String"},
		map[string]any{"name": "Node", "schema": "String"},
		map[string]any{"name": "VRF", "schema": "String"},
	}, []any{
		map[string]any{"Router_ID": "1.1.1.1", "Node": "n1", "VRF": "vrf1"},
		map[string]any{"Router_ID": "1.1.1.2", "Node": "n2", "VRF": "vrf2"},
	})
	duplicate := tableAnswer(t, []any{
		map[string]any{"name": "Router_ID", "schema": "String"},
		map[string]any{"name": "Node", "schema": "String"},
		map[string]any{"name": "VRF", "schema": "String"},
	}, []any{
		map[string]any{"Router_ID": "1.1.1.1", "Node": "n1", "VRF": "vrf1"},
		map[string]any{"Router_ID": "1.1.1.1", "Node": "n2", "VRF": "vrf2"},
	})

	stub.answers["ospfProcessConfiguration"] = unique
	if _, err := AssertNoDuplicateRouterIDs(context.Background(), nil, nil, []string{"ospf"}, false, bf, "table", false); err != nil {
		t.Fatal(err)
	}
	stub.answers["ospfProcessConfiguration"] = duplicate
	if _, err := AssertNoDuplicateRouterIDs(context.Background(), nil, nil, []string{"ospf"}, false, bf, "table", false); err == nil {
		t.Fatal("expected failure")
	}
}

func TestNoDuplicateRouterIDsBGP(t *testing.T) {
	stub := newAssertStub()
	bf := newAssertSession(t, stub)
	unique := tableAnswer(t, []any{
		map[string]any{"name": "Router_ID", "schema": "String"},
		map[string]any{"name": "Node", "schema": "String"},
		map[string]any{"name": "VRF", "schema": "String"},
	}, []any{
		map[string]any{"Router_ID": "1.1.1.1", "Node": "n1", "VRF": "vrf1"},
		map[string]any{"Router_ID": "1.1.1.2", "Node": "n2", "VRF": "vrf2"},
	})
	duplicate := tableAnswer(t, []any{
		map[string]any{"name": "Router_ID", "schema": "String"},
		map[string]any{"name": "Node", "schema": "String"},
		map[string]any{"name": "VRF", "schema": "String"},
	}, []any{
		map[string]any{"Router_ID": "1.1.1.1", "Node": "n1", "VRF": "vrf1"},
		map[string]any{"Router_ID": "1.1.1.1", "Node": "n2", "VRF": "vrf2"},
	})
	stub.answers["bgpProcessConfiguration"] = unique
	if _, err := bf.Asserts.AssertNoDuplicateRouterIDs(context.Background(), nil, nil, []string{"bgp"}, false, "table", false); err != nil {
		t.Fatal(err)
	}
	stub.answers["bgpProcessConfiguration"] = duplicate
	if _, err := bf.Asserts.AssertNoDuplicateRouterIDs(context.Background(), nil, nil, []string{"bgp"}, false, "table", false); err == nil {
		t.Fatal("expected failure")
	}
}

func TestNoForwardingLoops(t *testing.T) {
	stub := newAssertStub()
	bf := newAssertSession(t, stub)
	if _, err := AssertNoForwardingLoops(context.Background(), nil, false, bf, "table"); err != nil {
		t.Fatal(err)
	}
	stub.answers["detectLoops"] = foundTable(t)
	if _, err := bf.Asserts.AssertNoForwardingLoops(context.Background(), nil, false, "table"); err == nil {
		t.Fatal("expected failure")
	}
}

func TestGetQuestionObject(t *testing.T) {
	stub := newAssertStub()
	bf := newAssertSession(t, stub)
	if _, err := getQuestionObject(bf, "searchFilters"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err := getQuestionObject(bf, "qMissing")
	if err == nil || !strings.Contains(err.Error(), "qMissing question was not found") {
		t.Fatalf("err = %v", err)
	}
}

func TestHasRouteUnsupportedType(t *testing.T) {
	_, err := AssertHasRoute("bad type", map[string]any{}, "n1", "default", false)
	if err == nil || !strings.Contains(err.Error(), "'routes' is neither a Pandas DataFrame nor a dictionary") {
		t.Fatalf("err = %v", err)
	}
}

func TestHasRouteDataFrame(t *testing.T) {
	routes := dataframe.FromRecords([]map[string]any{
		{"Node": "n1", "VRF": "vrf1", "Network": "10.10.10.0/24"},
		{"Node": "n1", "VRF": "vrf2", "Network": "20.20.20.0/24"},
	})
	if _, err := AssertHasRoute(routes, map[string]any{}, "missing_node", "default", false); err == nil || !strings.Contains(err.Error(), "No node") {
		t.Fatalf("err = %v", err)
	}
	if _, err := AssertHasRoute(routes, map[string]any{}, "n1", "missing_vrf", false); err == nil || !strings.Contains(err.Error(), "No VRF") {
		t.Fatalf("err = %v", err)
	}
	if _, err := AssertHasRoute(routes, map[string]any{"Network": "30.30.30.30/32"}, "n1", "vrf1", false); err == nil || !strings.Contains(err.Error(), "No route") {
		t.Fatalf("err = %v", err)
	}
	if _, err := AssertHasRoute(routes, map[string]any{"Network": "10.10.10.0/24"}, "n1", "vrf2", false); err == nil || !strings.Contains(err.Error(), "No route") {
		t.Fatalf("err = %v", err)
	}
	if _, err := AssertHasRoute(routes, map[string]any{"Network": "10.10.10.0/24"}, "n1", "vrf1", false); err != nil {
		t.Fatal(err)
	}
}

func TestHasRouteDict(t *testing.T) {
	routes := map[string]map[string][]map[string]any{
		"n1": {
			"vrf1": {{"Network": "10.10.10.0/24"}},
			"vrf2": {{"Network": "20.20.20.0/24"}},
		},
	}
	if _, err := AssertHasRoute(routes, map[string]any{}, "missing_node", "default", false); err == nil || !strings.Contains(err.Error(), "No node") {
		t.Fatalf("err = %v", err)
	}
	if _, err := AssertHasRoute(routes, map[string]any{}, "n1", "missing_vrf", false); err == nil || !strings.Contains(err.Error(), "No VRF") {
		t.Fatalf("err = %v", err)
	}
	if _, err := AssertHasRoute(routes, map[string]any{"Network": "30.30.30.30/32"}, "n1", "vrf1", false); err == nil || !strings.Contains(err.Error(), "No route") {
		t.Fatalf("err = %v", err)
	}
	if _, err := AssertHasRoute(routes, map[string]any{"Network": "10.10.10.0/24"}, "n1", "vrf2", false); err == nil || !strings.Contains(err.Error(), "No route") {
		t.Fatalf("err = %v", err)
	}
	if _, err := AssertHasRoute(routes, map[string]any{"Network": "10.10.10.0/24"}, "n1", "vrf1", false); err != nil {
		t.Fatal(err)
	}
}

func TestHasNoRouteUnsupportedType(t *testing.T) {
	_, err := AssertHasNoRoute("bad type", map[string]any{}, "n1", "default", false)
	if err == nil || !strings.Contains(err.Error(), "'routes' is neither a Pandas DataFrame nor a dictionary") {
		t.Fatalf("err = %v", err)
	}
}

func TestHasNoRouteDataFrame(t *testing.T) {
	routes := dataframe.FromRecords([]map[string]any{
		{"Node": "n1", "VRF": "vrf1", "Network": "10.10.10.0/24"},
		{"Node": "n1", "VRF": "vrf2", "Network": "20.20.20.0/24"},
	})
	if _, err := AssertHasNoRoute(routes, map[string]any{}, "missing_node", "default", false); err != nil {
		t.Fatal(err)
	}
	if _, err := AssertHasNoRoute(routes, map[string]any{}, "n1", "missing_vrf", false); err != nil {
		t.Fatal(err)
	}
	if _, err := AssertHasNoRoute(routes, map[string]any{"Network": "30.30.30.30/32"}, "n1", "vrf1", false); err != nil {
		t.Fatal(err)
	}
	if _, err := AssertHasNoRoute(routes, map[string]any{"Network": "10.10.10.0/24"}, "n1", "vrf2", false); err != nil {
		t.Fatal(err)
	}
	if _, err := AssertHasNoRoute(routes, map[string]any{"Network": "10.10.10.0/24"}, "n1", "vrf1", false); err == nil || !strings.Contains(err.Error(), "Found route(s)") {
		t.Fatalf("err = %v", err)
	}
}

func TestHasNoRouteDict(t *testing.T) {
	routes := map[string]map[string][]map[string]any{
		"n1": {
			"vrf1": {{"Network": "10.10.10.0/24"}},
			"vrf2": {{"Network": "20.20.20.0/24"}},
		},
	}
	if _, err := AssertHasNoRoute(routes, map[string]any{}, "missing_node", "default", false); err != nil {
		t.Fatal(err)
	}
	if _, err := AssertHasNoRoute(routes, map[string]any{}, "n1", "missing_vrf", false); err != nil {
		t.Fatal(err)
	}
	if _, err := AssertHasNoRoute(routes, map[string]any{"Network": "30.30.30.30/32"}, "n1", "vrf1", false); err != nil {
		t.Fatal(err)
	}
	if _, err := AssertHasNoRoute(routes, map[string]any{"Network": "10.10.10.0/24"}, "n1", "vrf2", false); err != nil {
		t.Fatal(err)
	}
	if _, err := AssertHasNoRoute(routes, map[string]any{"Network": "10.10.10.0/24"}, "n1", "vrf1", false); err == nil || !strings.Contains(err.Error(), "Found route(s)") {
		t.Fatalf("err = %v", err)
	}
}
