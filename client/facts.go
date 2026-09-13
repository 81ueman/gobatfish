package client

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/81ueman/gobatfish/datamodel"
	"github.com/81ueman/gobatfish/datamodel/answer"
	"github.com/81ueman/gobatfish/question"
	"gopkg.in/yaml.v3"
)

// BatfishFactVersion is the version of the fact format.
const BatfishFactVersion = "batfish_v0"

// nodePropertiesReorg maps property names to new parent keys.
var nodePropertiesReorg = map[string]string{
	"DNS_Servers":              "DNS",
	"DNS_Source_Interface":     "DNS",
	"IKE_Phase1_Keys":          "IPsec",
	"IKE_Phase1_Policies":      "IPsec",
	"IKE_Phase1_Proposals":     "IPsec",
	"IPsec_Peer_Configs":       "IPsec",
	"IPsec_Phase2_Policies":    "IPsec",
	"IPsec_Phase2_Proposals":   "IPsec",
	"NTP_Servers":              "NTP",
	"NTP_Source_Interface":     "NTP",
	"Logging_Servers":          "Syslog",
	"Logging_Source_Interface": "Syslog",
	"SNMP_Source_Interface":    "SNMP",
	"SNMP_Trap_Servers":        "SNMP",
	"TACACS_Servers":           "TACACS",
	"TACACS_Source_Interface":  "TACACS",
}

// GetFacts collects facts for nodes matching the nodes specifier.
func GetFacts(ctx context.Context, session *Session, nodesSpecifier string, snapshot *string) (map[string]any, error) {
	args := map[string]any{}
	if nodesSpecifier != "" {
		args["nodes"] = nodesSpecifier
	}
	answerOpts := snapshot

	nodeProps, err := answerQuestionFrame(ctx, session, "nodeProperties", args, answerOpts)
	if err != nil {
		return nil, err
	}
	ifaceProps, err := answerQuestionFrame(ctx, session, "interfaceProperties", args, answerOpts)
	if err != nil {
		return nil, err
	}
	bgpProc, err := answerQuestionFrame(ctx, session, "bgpProcessConfiguration", args, answerOpts)
	if err != nil {
		return nil, err
	}
	bgpPeer, err := answerQuestionFrame(ctx, session, "bgpPeerConfiguration", args, answerOpts)
	if err != nil {
		return nil, err
	}
	ospfProc, err := answerQuestionFrame(ctx, session, "ospfProcessConfiguration", args, answerOpts)
	if err != nil {
		return nil, err
	}
	ospfArea, err := answerQuestionFrame(ctx, session, "ospfAreaConfiguration", args, answerOpts)
	if err != nil {
		return nil, err
	}
	ospfIface, err := answerQuestionFrame(ctx, session, "ospfInterfaceConfiguration", args, answerOpts)
	if err != nil {
		return nil, err
	}

	facts := processFacts(nodeProps, ifaceProps, bgpProc, bgpPeer, ospfProc, ospfArea, ospfIface)
	return map[string]any{"nodes": facts, "version": BatfishFactVersion}, nil
}

func questionAnswerOptions(snapshot *string) map[string]any {
	out := map[string]any{}
	if snapshot != nil {
		out["snapshot"] = *snapshot
	}
	return out
}

func answerQuestionFrame(ctx context.Context, session *Session, name string, vars map[string]any, snapshot *string) (*answer.TableAnswer, error) {
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
		return nil, fmt.Errorf("%s did not return a table answer", name)
	}
	return table, nil
}

func processFacts(nodeProps, ifaceProps, bgpProc, bgpPeer, ospfProc, ospfArea, ospfIface *answer.TableAnswer) map[string]any {
	out := processNodes(nodeProps)
	processInterfaces(out, ifaceProps)
	processBGPProcesses(out, bgpProc)
	processBGPPeers(out, bgpPeer)
	processOSPFProcesses(out, ospfProc)
	processOSPFAreas(out, ospfArea)
	processOSPFInterfaces(out, ospfIface)
	return convertListWrapper(out).(map[string]any)
}

func convertListWrapper(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = convertListWrapper(val)
		}
		return out
	case datamodel.ListWrapper:
		return []any(t)
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = convertListWrapper(val)
		}
		return out
	default:
		return v
	}
}

func processNodes(nodeProps *answer.TableAnswer) map[string]any {
	out := map[string]any{}
	for _, record := range nodeProps.Frame().Records() {
		node := fmt.Sprint(record["Node"])
		delete(record, "Node")
		delete(record, "Interfaces")
		removeConstructedNames(record)
		out[node] = record
		reorgDict(record, nodePropertiesReorg)
	}
	return out
}

func removeConstructedNames(d map[string]any) {
	for k, v := range d {
		if list, ok := v.(datamodel.ListWrapper); ok {
			filtered := make([]any, 0, len(list))
			for _, item := range list {
				if s, ok := item.(string); ok && strings.HasPrefix(s, "~") {
					continue
				}
				filtered = append(filtered, item)
			}
			d[k] = datamodel.NewListWrapper(filtered)
		}
	}
}

func reorgDict(d map[string]any, reorg map[string]string) {
	for key, parent := range reorg {
		val, ok := d[key]
		if !ok {
			continue
		}
		parentMap, ok := d[parent].(map[string]any)
		if !ok {
			parentMap = map[string]any{}
			d[parent] = parentMap
		}
		parentMap[key] = val
		delete(d, key)
	}
}

func processInterfaces(nodeDict map[string]any, ifaceProps *answer.TableAnswer) {
	for _, record := range ifaceProps.Frame().Records() {
		iface, ok := record["Interface"].(datamodel.Interface)
		if !ok {
			continue
		}
		nodeVal, ok := nodeDict[iface.Hostname]
		if !ok {
			continue
		}
		node := nodeVal.(map[string]any)
		if _, ok := node["Interfaces"]; !ok {
			node["Interfaces"] = map[string]any{}
		}
		delete(record, "Interface")
		node["Interfaces"].(map[string]any)[iface.Interface] = record
	}
}

func processBGPProcesses(nodeDict map[string]any, bgpProc *answer.TableAnswer) {
	for _, record := range bgpProc.Frame().Records() {
		node := fmt.Sprint(record["Node"])
		delete(record, "Node")
		record["Neighbors"] = map[string]any{}
		nodeDict[node].(map[string]any)["BGP"] = record
	}
}

func processBGPPeers(nodeDict map[string]any, bgpPeer *answer.TableAnswer) {
	for _, record := range bgpPeer.Frame().Records() {
		node := fmt.Sprint(record["Node"])
		delete(record, "Node")
		ip := fmt.Sprint(record["Remote_IP"])
		nodeDict[node].(map[string]any)["BGP"].(map[string]any)["Neighbors"].(map[string]any)[ip] = record
	}
}

func processOSPFProcesses(nodeDict map[string]any, ospf *answer.TableAnswer) {
	for _, record := range ospf.Frame().Records() {
		node := fmt.Sprint(record["Node"])
		proc := fmt.Sprint(record["Process_ID"])
		delete(record, "Node")
		delete(record, "Process_ID")
		orCreateOSPFProcesses(nodeDict, node)[proc] = map[string]any{
			"VRF":                 record["VRF"],
			"Reference_Bandwidth": record["Reference_Bandwidth"],
			"Router_ID":           record["Router_ID"],
		}
	}
}

func processOSPFAreas(nodeDict map[string]any, ospf *answer.TableAnswer) {
	for _, record := range ospf.Frame().Records() {
		node := fmt.Sprint(record["Node"])
		proc := fmt.Sprint(record["Process_ID"])
		area := fmt.Sprint(record["Area"])
		delete(record, "Node")
		delete(record, "Process_ID")
		delete(record, "Area")
		orCreateOSPFAreas(nodeDict, node, proc)[area] = map[string]any{"Area_Type": record["Area_Type"]}
	}
}

func processOSPFInterfaces(nodeDict map[string]any, ospf *answer.TableAnswer) {
	for _, record := range ospf.Frame().Records() {
		iface, ok := record["Interface"].(datamodel.Interface)
		if !ok {
			continue
		}
		node := iface.Hostname
		proc := fmt.Sprint(record["Process_ID"])
		area := fmt.Sprint(record["OSPF_Area_Name"])
		delete(record, "Interface")
		delete(record, "Process_ID")
		delete(record, "OSPF_Area_Name")
		orCreateOSPFInterfaces(nodeDict, node, proc, area)[iface.Interface] = map[string]any{
			"Enabled":        record["OSPF_Enabled"],
			"Passive":        record["OSPF_Passive"],
			"Cost":           record["OSPF_Cost"],
			"Dead_Interval":  record["OSPF_Dead_Interval"],
			"Hello_Interval": record["OSPF_Hello_Interval"],
			"Network_Type":   record["OSPF_Network_Type"],
		}
	}
}

func orCreateOSPFProcesses(nodeDict map[string]any, node string) map[string]any {
	nodeMap, ok := nodeDict[node].(map[string]any)
	if !ok {
		nodeMap = map[string]any{}
		nodeDict[node] = nodeMap
	}
	ospf, ok := nodeMap["OSPF"].(map[string]any)
	if !ok {
		ospf = map[string]any{}
		nodeMap["OSPF"] = ospf
	}
	procs, ok := ospf["Processes"].(map[string]any)
	if !ok {
		procs = map[string]any{}
		ospf["Processes"] = procs
	}
	return procs
}

func orCreateOSPFAreas(nodeDict map[string]any, node, process string) map[string]any {
	procs := orCreateOSPFProcesses(nodeDict, node)
	procMap, ok := procs[process].(map[string]any)
	if !ok {
		procMap = map[string]any{}
		procs[process] = procMap
	}
	areas, ok := procMap["Areas"].(map[string]any)
	if !ok {
		areas = map[string]any{}
		procMap["Areas"] = areas
	}
	return areas
}

func orCreateOSPFInterfaces(nodeDict map[string]any, node, process, area string) map[string]any {
	areas := orCreateOSPFAreas(nodeDict, node, process)
	areaMap, ok := areas[area].(map[string]any)
	if !ok {
		areaMap = map[string]any{}
		areas[area] = areaMap
	}
	ifaces, ok := areaMap["Interfaces"].(map[string]any)
	if !ok {
		ifaces = map[string]any{}
		areaMap["Interfaces"] = ifaces
	}
	return ifaces
}

// LoadFacts loads facts from YAML files in the specified directory.
func LoadFacts(inputDirectory string) (map[string]any, error) {
	out := map[string]any{"version": nil, "nodes": map[string]any{}}
	outNodes := out["nodes"].(map[string]any)
	entries, err := os.ReadDir(inputDirectory)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no files present in specified directory")
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(inputDirectory, entry.Name()))
		if err != nil {
			return nil, err
		}
		var dict map[string]any
		if err := yaml.Unmarshal(data, &dict); err != nil {
			return nil, err
		}
		nodes, version := unencapsulateFacts(dict)
		versionText := BatfishFactVersion
		if version != nil && fmt.Sprint(version) != "" {
			versionText = fmt.Sprint(version)
		}
		if out["version"] != nil && versionText != fmt.Sprint(out["version"]) {
			return nil, fmt.Errorf("input file version mismatch")
		}
		out["version"] = versionText
		if nodesMap, ok := nodes.(map[string]any); ok {
			for node, facts := range nodesMap {
				outNodes[node] = facts
			}
		}
	}
	return out, nil
}

// ValidateFacts returns a map of node to non-matching facts.
func ValidateFacts(expected, actual map[string]any, verbose bool) map[string]any {
	failures := map[string]any{}
	expectedFacts, _ := expected["nodes"].(map[string]any)
	actualFacts, _ := actual["nodes"].(map[string]any)
	expectedVersion := expected["version"]
	actualVersion := actual["version"]
	if fmt.Sprint(expectedVersion) != fmt.Sprint(actualVersion) {
		for n := range expectedFacts {
			failures[n] = map[string]any{"Version": map[string]any{"expected": expectedVersion, "actual": actualVersion}}
		}
		return failures
	}
	for node, expectedNode := range expectedFacts {
		actualNode, _ := actualFacts[node].(map[string]any)
		expectedMap, _ := expectedNode.(map[string]any)
		if res := assertDictSubset(actualNode, expectedMap, "", nil, verbose); len(res) > 0 {
			failures[node] = res
		}
	}
	return failures
}

// WriteFacts writes facts to YAML files in the supplied output directory.
func WriteFacts(outputDirectory string, facts map[string]any) error {
	nodes, version := unencapsulateFacts(facts)
	if version == nil || fmt.Sprint(version) == "" {
		return fmt.Errorf("version must be specified to write facts")
	}
	nodesMap, _ := nodes.(map[string]any)
	if err := os.MkdirAll(outputDirectory, 0o755); err != nil {
		return err
	}
	for node := range nodesMap {
		filepath := filepath.Join(outputDirectory, node+".yml")
		encapsulated := encapsulateNodesFacts(map[string]any{node: nodesMap[node]}, fmt.Sprint(version))
		data, err := yaml.Marshal(encapsulated)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func encapsulateNodesFacts(nodesFacts map[string]any, version string) map[string]any {
	return map[string]any{"nodes": nodesFacts, "version": version}
}

func unencapsulateFacts(facts map[string]any) (any, any) {
	return facts["nodes"], facts["version"]
}

func assertDictSubset(actual, expected map[string]any, prefix string, diffs map[string]any, verbose bool) map[string]any {
	if diffs == nil {
		diffs = map[string]any{}
	}
	for k, expectedVal := range expected {
		keyName := prefix + k
		actualVal, ok := actual[k]
		if !ok {
			diffs[keyName] = map[string]any{"key_present": false, "expected": expectedVal}
			continue
		}
		expectedMap, expectedIsMap := expectedVal.(map[string]any)
		if expectedIsMap {
			actualMap, _ := actualVal.(map[string]any)
			assertDictSubset(actualMap, expectedMap, keyName+".", diffs, verbose)
			continue
		}
		if !valueMatches(actualVal, expectedVal) || verbose {
			diffs[keyName] = map[string]any{"expected": expectedVal, "actual": actualVal}
		}
	}
	return diffs
}

// ExtractFacts extracts and returns facts about the specified nodes, optionally
// writing them to outputDirectory.
func (s *Session) ExtractFacts(ctx context.Context, nodes string, outputDirectory string, snapshot *string) (map[string]any, error) {
	facts, err := GetFacts(ctx, s, nodes, snapshot)
	if err != nil {
		return nil, err
	}
	if outputDirectory != "" {
		info, statErr := os.Stat(outputDirectory)
		if statErr == nil && !info.IsDir() {
			return nil, fmt.Errorf("cannot write facts to file, must be a directory: %s", outputDirectory)
		}
		if err := WriteFacts(outputDirectory, facts); err != nil {
			return nil, err
		}
	}
	return facts, nil
}

// ValidateFacts validates loaded expected facts against actual facts.
func (s *Session) ValidateFacts(ctx context.Context, expectedFacts string, snapshot *string) (map[string]any, error) {
	actual, err := GetFacts(ctx, s, "/.*/", snapshot)
	if err != nil {
		return nil, err
	}
	expected, err := LoadFacts(expectedFacts)
	if err != nil {
		return nil, err
	}
	return ValidateFacts(expected, actual, false), nil
}
