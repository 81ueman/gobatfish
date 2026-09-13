package client

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/81ueman/gobatfish/dataframe"
	"github.com/81ueman/gobatfish/datamodel"
	"github.com/81ueman/gobatfish/datamodel/answer"
	"github.com/81ueman/gobatfish/exception"
	"github.com/81ueman/gobatfish/question"
)

// UnestablishedOSPFSSessionStatusSpec matches OSPF statuses other than
// ESTABLISHED.
const UnestablishedOSPFSSessionStatusSpec = "/^(?!ESTABLISHED).*$/"

// Asserts contains assertions for a given Session.
type Asserts struct {
	session *Session
}

func jsonMarshal(v any) ([]byte, error) { return json.Marshal(v) }

func base64StdEncode(data []byte) string { return base64.StdEncoding.EncodeToString(data) }

func getQuestionObject(session *Session, name string) (*question.Question, error) {
	q, err := session.Q.Get(name)
	if err != nil {
		return nil, exception.NewBatfishErrorf("%s question was not found", name)
	}
	return q, nil
}

func askQuestionFrame(ctx context.Context, session *Session, name string, vars map[string]any, snapshot *string) (*dataframe.DataFrame, error) {
	q, err := getQuestionObject(session, name)
	if err != nil {
		return nil, err
	}
	for k, v := range vars {
		q.Set(k, v)
	}
	result, err := q.Answer(ctx, question.AnswerOptions{Snapshot: snapshot})
	if err != nil {
		return nil, err
	}
	table, ok := result.Table()
	if !ok {
		return nil, exception.NewBatfishErrorf("%s did not return a table answer", name)
	}
	return table.Frame(), nil
}

// raiseCommon implements soft/hard assertion raising.
func raiseCommon(errText string, soft bool) (bool, error) {
	if soft {
		log.Printf("WARNING: %s", errText)
		return false, nil
	}
	return false, exception.NewBatfishAssertError(errText)
}

func formatDF(df *dataframe.DataFrame, dfFormat string) (string, error) {
	switch dfFormat {
	case "table":
		return df.String(), nil
	case "records":
		return fmt.Sprint(df.Records()), nil
	default:
		return "", fmt.Errorf("unknown df_format %s. Should be 'table' or 'records'", dfFormat)
	}
}

// AssertZeroResults asserts that no results were returned.
func AssertZeroResults(a any, soft bool) (bool, error) { return AssertNumResults(a, 0, soft) }

// AssertNumResults asserts that an exact number of results were returned.
func AssertNumResults(a any, num int, soft bool) (bool, error) {
	var actual int
	switch t := a.(type) {
	case *dataframe.DataFrame:
		actual = t.Len()
	case *answer.TableAnswer:
		actual = t.Frame().Len()
	case answer.Answer:
		summary, ok := t["summary"].(map[string]any)
		if !ok {
			return false, fmt.Errorf("unrecognized answer type")
		}
		actual = toIntValue(summary["numResults"])
	default:
		return false, fmt.Errorf("unrecognized answer type")
	}
	if actual != num {
		return raiseCommon(fmt.Sprintf("Expected %d results, found: %d\nFull answer:\n%v", num, actual, a), soft)
	}
	return true, nil
}

// AssertHasRoute asserts that a particular route is present.
func AssertHasRoute(routes any, expectedRoute map[string]any, node, vrf string, soft bool) (bool, error) {
	if df, ok := routes.(*dataframe.DataFrame); ok {
		return assertHasRouteDF(df, expectedRoute, node, vrf, soft)
	}
	if dict, ok := routes.(map[string]map[string][]map[string]any); ok {
		return assertHasRouteDict(dict, expectedRoute, node, vrf, soft)
	}
	return false, fmt.Errorf("'routes' is neither a DataFrame nor a dictionary")
}

func assertHasRouteDF(df *dataframe.DataFrame, expectedRoute map[string]any, node, vrf string, soft bool) (bool, error) {
	nodeRoutes := df.Filter(df.Col("Node").Eq(node))
	if nodeRoutes.Len() == 0 {
		return false, exception.NewBatfishAssertErrorf("No node: %s", node)
	}
	vrfRoutes := nodeRoutes.Filter(nodeRoutes.Col("VRF").Eq(vrf))
	if vrfRoutes.Len() == 0 {
		return false, exception.NewBatfishAssertErrorf("No VRF: %s on node %s", vrf, node)
	}
	for _, r := range vrfRoutes.Records() {
		if isDictMatch(r, expectedRoute) {
			return true, nil
		}
	}
	return raiseCommon(fmt.Sprintf("No route matches for %v on node %s, VRF %s", expectedRoute, node, vrf), soft)
}

func assertHasRouteDict(routes map[string]map[string][]map[string]any, expectedRoute map[string]any, node, vrf string, soft bool) (bool, error) {
	nodeRoutes, ok := routes[node]
	if !ok {
		return false, exception.NewBatfishAssertErrorf("No node: %s", node)
	}
	vrfRoutes, ok := nodeRoutes[vrf]
	if !ok {
		return false, exception.NewBatfishAssertErrorf("No VRF: %s on node %s", vrf, node)
	}
	for _, r := range vrfRoutes {
		if isDictMatch(r, expectedRoute) {
			return true, nil
		}
	}
	return raiseCommon(fmt.Sprintf("No route matches for %v on node %s, VRF %s", expectedRoute, node, vrf), soft)
}

// AssertHasNoRoute asserts that a particular route is NOT present.
func AssertHasNoRoute(routes any, expectedRoute map[string]any, node, vrf string, soft bool) (bool, error) {
	if df, ok := routes.(*dataframe.DataFrame); ok {
		return assertHasNoRouteDF(df, expectedRoute, node, vrf, soft)
	}
	if dict, ok := routes.(map[string]map[string][]map[string]any); ok {
		return assertHasNoRouteDict(dict, expectedRoute, node, vrf, soft)
	}
	return false, fmt.Errorf("'routes' is neither a DataFrame nor a dictionary")
}

func assertHasNoRouteDF(df *dataframe.DataFrame, expectedRoute map[string]any, node, vrf string, soft bool) (bool, error) {
	nodeRoutes := df.Filter(df.Col("Node").Eq(node))
	if nodeRoutes.Len() == 0 {
		log.Printf("WARNING: No node: %s", node)
		return true, nil
	}
	vrfRoutes := nodeRoutes.Filter(nodeRoutes.Col("VRF").Eq(vrf))
	if vrfRoutes.Len() == 0 {
		log.Printf("WARNING: No VRF: %s on node %s", vrf, node)
		return true, nil
	}
	var matches []map[string]any
	for _, r := range vrfRoutes.Records() {
		if isDictMatch(r, expectedRoute) {
			matches = append(matches, r)
		}
	}
	if len(matches) > 0 {
		return raiseCommon(fmt.Sprintf("Found route(s) that match, when none were expected:\n%v", matches), soft)
	}
	return true, nil
}

func assertHasNoRouteDict(routes map[string]map[string][]map[string]any, expectedRoute map[string]any, node, vrf string, soft bool) (bool, error) {
	nodeRoutes, ok := routes[node]
	if !ok {
		log.Printf("WARNING: No node: %s", node)
		return true, nil
	}
	vrfRoutes, ok := nodeRoutes[vrf]
	if !ok {
		log.Printf("WARNING: No VRF: %s on node %s", vrf, node)
		return true, nil
	}
	var matches []map[string]any
	for _, r := range vrfRoutes {
		if isDictMatch(r, expectedRoute) {
			matches = append(matches, r)
		}
	}
	if len(matches) > 0 {
		return raiseCommon(fmt.Sprintf("Found route(s) that match, when none were expected:\n%v", matches), soft)
	}
	return true, nil
}

// AssertFilterDenies checks whether a filter denies a specified set of flows.
func AssertFilterDenies(ctx context.Context, filters string, headers datamodel.HeaderConstraints, startLocation *string, soft bool, snapshot *string, session *Session, dfFormat string) (bool, error) {
	if session == nil {
		return false, fmt.Errorf("session must be provided. Preferably, use Session.Asserts rather than this function")
	}
	vars := map[string]any{"filters": filters, "headers": headers, "action": "permit"}
	if startLocation != nil {
		vars["startLocation"] = *startLocation
	}
	df, err := askQuestionFrame(ctx, session, "searchFilters", vars, snapshot)
	if err != nil {
		return false, err
	}
	if df.Len() > 0 {
		formatted, _ := formatDF(df, dfFormat)
		return raiseCommon(fmt.Sprintf("Found a flow that was permitted, when expected to be denied\n%s", formatted), soft)
	}
	return true, nil
}

// AssertFilterHasNoUnreachableLines checks that a filter has no unreachable lines.
func AssertFilterHasNoUnreachableLines(ctx context.Context, filters string, soft bool, snapshot *string, session *Session, dfFormat string) (bool, error) {
	if session == nil {
		return false, fmt.Errorf("session must be provided. Preferably, use Session.Asserts rather than this function")
	}
	df, err := askQuestionFrame(ctx, session, "filterLineReachability", map[string]any{"filters": filters}, snapshot)
	if err != nil {
		return false, err
	}
	if df.Len() > 0 {
		formatted, _ := formatDF(df, dfFormat)
		return raiseCommon(fmt.Sprintf("Found unreachable filter line(s), when none were expected\n%s", formatted), soft)
	}
	return true, nil
}

// AssertFilterPermits checks whether a filter permits a specified set of flows.
func AssertFilterPermits(ctx context.Context, filters string, headers datamodel.HeaderConstraints, startLocation *string, soft bool, snapshot *string, session *Session, dfFormat string) (bool, error) {
	if session == nil {
		return false, fmt.Errorf("session must be provided. Preferably, use Session.Asserts rather than this function")
	}
	vars := map[string]any{"filters": filters, "headers": headers, "action": "deny"}
	if startLocation != nil {
		vars["startLocation"] = *startLocation
	}
	df, err := askQuestionFrame(ctx, session, "searchFilters", vars, snapshot)
	if err != nil {
		return false, err
	}
	if df.Len() > 0 {
		formatted, _ := formatDF(df, dfFormat)
		return raiseCommon(fmt.Sprintf("Found a flow that was denied, when expected to be permitted\n%s", formatted), soft)
	}
	return true, nil
}

// AssertFlowsFail checks that the specified flows fail.
func AssertFlowsFail(ctx context.Context, startLocation string, headers datamodel.HeaderConstraints, soft bool, snapshot *string, session *Session, dfFormat string) (bool, error) {
	if session == nil {
		return false, fmt.Errorf("session must be provided. Preferably, use Session.Asserts rather than this function")
	}
	vars := map[string]any{
		"pathConstraints": datamodel.PathConstraints{StartLocation: datamodel.StringPtr(startLocation)},
		"headers":         headers,
		"actions":         "success",
	}
	df, err := askQuestionFrame(ctx, session, "reachability", vars, snapshot)
	if err != nil {
		return false, err
	}
	if df.Len() > 0 {
		formatted, _ := formatDF(df, dfFormat)
		return raiseCommon(fmt.Sprintf("Found a flow that succeed, when expected to fail\n%s", formatted), soft)
	}
	return true, nil
}

// AssertFlowsSucceed checks that the specified flows succeed.
func AssertFlowsSucceed(ctx context.Context, startLocation string, headers datamodel.HeaderConstraints, soft bool, snapshot *string, session *Session, dfFormat string) (bool, error) {
	if session == nil {
		return false, fmt.Errorf("session must be provided. Preferably, use Session.Asserts rather than this function")
	}
	vars := map[string]any{
		"pathConstraints": datamodel.PathConstraints{StartLocation: datamodel.StringPtr(startLocation)},
		"headers":         headers,
		"actions":         "failure",
	}
	df, err := askQuestionFrame(ctx, session, "reachability", vars, snapshot)
	if err != nil {
		return false, err
	}
	if df.Len() > 0 {
		formatted, _ := formatDF(df, dfFormat)
		return raiseCommon(fmt.Sprintf("Found a flow that failed, when expected to succeed\n%s", formatted), soft)
	}
	return true, nil
}

// AssertNoIncompatibleBGPSessions asserts there are no incompatible BGP sessions.
func AssertNoIncompatibleBGPSessions(ctx context.Context, nodes, remoteNodes, status *string, snapshot *string, soft bool, session *Session, dfFormat string) (bool, error) {
	if session == nil {
		return false, fmt.Errorf("session must be provided. Preferably, use Session.Asserts rather than this function")
	}
	vars := map[string]any{}
	if status != nil {
		vars["status"] = *status
	}
	if nodes != nil {
		vars["nodes"] = *nodes
	}
	if remoteNodes != nil {
		vars["remoteNodes"] = *remoteNodes
	}
	df, err := askQuestionFrame(ctx, session, "bgpSessionCompatibility", vars, snapshot)
	if err != nil {
		return false, err
	}
	if status == nil {
		ignored := []any{"UNIQUE_MATCH", "DYNAMIC_MATCH", "UNKNOWN_REMOTE"}
		df = df.Filter(df.Col("Configured_Status").NotIn(ignored...))
	}
	if df.Len() > 0 {
		formatted, _ := formatDF(df, dfFormat)
		return raiseCommon(fmt.Sprintf("Found incompatible BGP session(s), when none were expected\n%s", formatted), soft)
	}
	return true, nil
}

// AssertNoIncompatibleOSPFSessions asserts there are no incompatible OSPF sessions.
func AssertNoIncompatibleOSPFSessions(ctx context.Context, nodes, remoteNodes *string, snapshot *string, soft bool, session *Session, dfFormat string) (bool, error) {
	if session == nil {
		return false, fmt.Errorf("session must be provided. Preferably, use Session.Asserts rather than this function")
	}
	vars := map[string]any{"statuses": UnestablishedOSPFSSessionStatusSpec}
	if nodes != nil {
		vars["nodes"] = *nodes
	}
	if remoteNodes != nil {
		vars["remoteNodes"] = *remoteNodes
	}
	df, err := askQuestionFrame(ctx, session, "ospfSessionCompatibility", vars, snapshot)
	if err != nil {
		return false, err
	}
	if df.Len() > 0 {
		formatted, _ := formatDF(df, dfFormat)
		return raiseCommon(fmt.Sprintf("Found OSPF session(s) that were not established, when none were expected\n%s", formatted), soft)
	}
	return true, nil
}

// AssertNoUnestablishedBGPSessions asserts there are no compatible but
// unestablished BGP sessions.
func AssertNoUnestablishedBGPSessions(ctx context.Context, nodes, remoteNodes *string, snapshot *string, soft bool, session *Session, dfFormat string) (bool, error) {
	if session == nil {
		return false, fmt.Errorf("session must be provided. Preferably, use Session.Asserts rather than this function")
	}
	vars := map[string]any{"status": "NOT_ESTABLISHED"}
	if nodes != nil {
		vars["nodes"] = *nodes
	}
	if remoteNodes != nil {
		vars["remoteNodes"] = *remoteNodes
	}
	df, err := askQuestionFrame(ctx, session, "bgpSessionStatus", vars, snapshot)
	if err != nil {
		return false, err
	}
	if df.Len() > 0 {
		formatted, _ := formatDF(df, dfFormat)
		return raiseCommon(fmt.Sprintf("Found compatible BGP session(s) that were not established, when none were expected\n%s", formatted), soft)
	}
	return true, nil
}

// AssertNoUndefinedReferences asserts there are no undefined references.
func AssertNoUndefinedReferences(ctx context.Context, snapshot *string, soft bool, session *Session, dfFormat string) (bool, error) {
	if session == nil {
		return false, fmt.Errorf("session must be provided. Preferably, use Session.Asserts rather than this function")
	}
	df, err := askQuestionFrame(ctx, session, "undefinedReferences", nil, snapshot)
	if err != nil {
		return false, err
	}
	if df.Len() > 0 {
		formatted, _ := formatDF(df, dfFormat)
		return raiseCommon(fmt.Sprintf("Found undefined reference(s), when none were expected\n%s", formatted), soft)
	}
	return true, nil
}

// AssertNoDuplicateRouterIDs asserts there are no duplicate router IDs.
func AssertNoDuplicateRouterIDs(ctx context.Context, snapshot *string, nodes *string, protocols []string, soft bool, session *Session, dfFormat string, ignoreSameNode bool) (bool, error) {
	if session == nil {
		return false, fmt.Errorf("session must be provided. Preferably, use Session.Asserts rather than this function")
	}
	vars := map[string]any{}
	if nodes != nil {
		vars["nodes"] = *nodes
	}
	supported := map[string]bool{"bgp": true, "ospf": true}
	protocolsToFetch := map[string]bool{}
	if protocols == nil {
		protocolsToFetch = supported
	} else {
		for _, p := range protocols {
			protocolsToFetch[strings.ToLower(p)] = true
		}
	}
	for p := range protocolsToFetch {
		if !supported[p] {
			return false, fmt.Errorf("unsupported protocols supplied: %s", p)
		}
	}
	foundDuplicates := false
	duplicateResults := ""
	for protocol := range protocolsToFetch {
		df, err := askQuestionFrame(ctx, session, protocol+"ProcessConfiguration", vars, snapshot)
		if err != nil {
			return false, err
		}
		dup, err := duplicateRouterIDs(df, ignoreSameNode)
		if err != nil {
			return false, err
		}
		if !dup.Empty() {
			foundDuplicates = true
			formatted, _ := formatDF(dup, dfFormat)
			duplicateResults += fmt.Sprintf("%s: %s\n", strings.ToUpper(protocol), formatted)
		}
	}
	if foundDuplicates {
		return raiseCommon(fmt.Sprintf("Found duplicate router-id(s), when none were expected\n%s", duplicateResults), soft)
	}
	return true, nil
}

func duplicateRouterIDs(df *dataframe.DataFrame, ignoreSameNode bool) (*dataframe.DataFrame, error) {
	if ignoreSameNode {
		return df.GroupBy("Router_ID").Filter(func(g *dataframe.DataFrame) bool {
			uniqueNodes := g.Col("Node").NUnique()
			return uniqueNodes > 1 && uniqueNodes != g.Len()
		}), nil
	}
	dup := df.Filter(df.Duplicated([]string{"Router_ID"}, false))
	return dup.SortValues("Router_ID"), nil
}

// AssertNoForwardingLoops asserts there are no forwarding loops.
func AssertNoForwardingLoops(ctx context.Context, snapshot *string, soft bool, session *Session, dfFormat string) (bool, error) {
	if session == nil {
		return false, fmt.Errorf("session must be provided. Preferably, use Session.Asserts rather than this function")
	}
	df, err := askQuestionFrame(ctx, session, "detectLoops", nil, snapshot)
	if err != nil {
		return false, err
	}
	if df.Len() > 0 {
		formatted, _ := formatDF(df, dfFormat)
		return raiseCommon(fmt.Sprintf("Found forwarding loops, when none were expected\n%s", formatted), soft)
	}
	return true, nil
}

// isDictMatch reports whether expected is a subset of actual, comparing lists
// order-insensitively (mirroring deepdiff with ignore_order=True).
func isDictMatch(actual, expected map[string]any) bool {
	for k, v := range expected {
		av, ok := actual[k]
		if !ok || !valueMatches(av, v) {
			return false
		}
	}
	return true
}

func valueMatches(actual, expected any) bool {
	switch ev := expected.(type) {
	case map[string]any:
		am, ok := actual.(map[string]any)
		return ok && isDictMatch(am, ev)
	case []any:
		av, ok := actual.([]any)
		if !ok || len(av) != len(ev) {
			return false
		}
		used := make([]bool, len(av))
		for _, e := range ev {
			found := false
			for i, a := range av {
				if used[i] {
					continue
				}
				if valueMatches(a, e) {
					used[i] = true
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(actual, expected)
	}
}

func toIntValue(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

// --- Session.Asserts convenience methods ------------------------------------

// AssertFilterDenies checks whether a filter denies a specified set of flows.
func (a *Asserts) AssertFilterDenies(ctx context.Context, filters string, headers datamodel.HeaderConstraints, startLocation *string, soft bool, snapshot *string, dfFormat string) (bool, error) {
	return AssertFilterDenies(ctx, filters, headers, startLocation, soft, snapshot, a.session, dfFormat)
}

// AssertFilterHasNoUnreachableLines checks that a filter has no unreachable lines.
func (a *Asserts) AssertFilterHasNoUnreachableLines(ctx context.Context, filters string, soft bool, snapshot *string, dfFormat string) (bool, error) {
	return AssertFilterHasNoUnreachableLines(ctx, filters, soft, snapshot, a.session, dfFormat)
}

// AssertFilterPermits checks whether a filter permits a specified set of flows.
func (a *Asserts) AssertFilterPermits(ctx context.Context, filters string, headers datamodel.HeaderConstraints, startLocation *string, soft bool, snapshot *string, dfFormat string) (bool, error) {
	return AssertFilterPermits(ctx, filters, headers, startLocation, soft, snapshot, a.session, dfFormat)
}

// AssertFlowsFail checks that the specified flows fail.
func (a *Asserts) AssertFlowsFail(ctx context.Context, startLocation string, headers datamodel.HeaderConstraints, soft bool, snapshot *string, dfFormat string) (bool, error) {
	return AssertFlowsFail(ctx, startLocation, headers, soft, snapshot, a.session, dfFormat)
}

// AssertFlowsSucceed checks that the specified flows succeed.
func (a *Asserts) AssertFlowsSucceed(ctx context.Context, startLocation string, headers datamodel.HeaderConstraints, soft bool, snapshot *string, dfFormat string) (bool, error) {
	return AssertFlowsSucceed(ctx, startLocation, headers, soft, snapshot, a.session, dfFormat)
}

// AssertNoIncompatibleBGPSessions asserts there are no incompatible BGP sessions.
func (a *Asserts) AssertNoIncompatibleBGPSessions(ctx context.Context, nodes, remoteNodes, status *string, soft bool, snapshot *string, dfFormat string) (bool, error) {
	return AssertNoIncompatibleBGPSessions(ctx, nodes, remoteNodes, status, snapshot, soft, a.session, dfFormat)
}

// AssertNoIncompatibleOSPFSessions asserts there are no incompatible OSPF sessions.
func (a *Asserts) AssertNoIncompatibleOSPFSessions(ctx context.Context, nodes, remoteNodes *string, soft bool, snapshot *string, dfFormat string) (bool, error) {
	return AssertNoIncompatibleOSPFSessions(ctx, nodes, remoteNodes, snapshot, soft, a.session, dfFormat)
}

// AssertNoUnestablishedBGPSessions asserts there are no unestablished BGP sessions.
func (a *Asserts) AssertNoUnestablishedBGPSessions(ctx context.Context, nodes, remoteNodes *string, soft bool, snapshot *string, dfFormat string) (bool, error) {
	return AssertNoUnestablishedBGPSessions(ctx, nodes, remoteNodes, snapshot, soft, a.session, dfFormat)
}

// AssertNoUndefinedReferences asserts there are no undefined references.
func (a *Asserts) AssertNoUndefinedReferences(ctx context.Context, soft bool, snapshot *string, dfFormat string) (bool, error) {
	return AssertNoUndefinedReferences(ctx, snapshot, soft, a.session, dfFormat)
}

// AssertNoDuplicateRouterIDs asserts there are no duplicate router IDs.
func (a *Asserts) AssertNoDuplicateRouterIDs(ctx context.Context, snapshot *string, nodes *string, protocols []string, soft bool, dfFormat string, ignoreSameNode bool) (bool, error) {
	return AssertNoDuplicateRouterIDs(ctx, snapshot, nodes, protocols, soft, a.session, dfFormat, ignoreSameNode)
}

// AssertNoForwardingLoops asserts there are no forwarding loops.
func (a *Asserts) AssertNoForwardingLoops(ctx context.Context, snapshot *string, soft bool, dfFormat string) (bool, error) {
	return AssertNoForwardingLoops(ctx, snapshot, soft, a.session, dfFormat)
}
