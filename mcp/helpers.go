package mcp

import (
	"fmt"
	"strings"

	"github.com/81ueman/gobatfish/dataframe"
	"github.com/81ueman/gobatfish/datamodel"
)

// legacyNextHopColumns are deprecated next-hop columns dropped from route
// results, mirroring pybatfish.mcp.server.
var legacyNextHopColumns = []string{"Next_Hop_IP", "Next_Hop_Interface", "NextHopIp", "NextHopInterface"}

// ParseInterfaces parses a comma-separated "node[interface]" string.
//
// Bare node names are rejected with an error; use the deactivate_nodes /
// restore_nodes parameters for node-level operations.
func ParseInterfaces(interfaces string) ([]datamodel.Interface, error) {
	var out []datamodel.Interface
	for _, item := range strings.Split(interfaces, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.Contains(item, "[") && strings.HasSuffix(item, "]") {
			node, iface, _ := strings.Cut(strings.TrimSuffix(item, "]"), "[")
			out = append(out, datamodel.Interface{Hostname: strings.TrimSpace(node), Interface: strings.TrimSpace(iface)})
			continue
		}
		return nil, fmt.Errorf("invalid interface specifier %q: expected 'node[interface]' format. Use deactivate_nodes/restore_nodes for node-level operations", item)
	}
	return out, nil
}

// BuildHeaderConstraints builds header constraints from string parameters.
func BuildHeaderConstraints(dstIps, srcIps, applications, ipProtocols, srcPorts, dstPorts string) datamodel.HeaderConstraints {
	h := datamodel.HeaderConstraints{}
	if dstIps != "" {
		h.DstIps = dstIps
	}
	if srcIps != "" {
		h.SrcIps = srcIps
	}
	if applications != "" {
		h.Applications = applications
	}
	if ipProtocols != "" {
		h.IpProtocols = ipProtocols
	}
	if srcPorts != "" {
		h.SrcPorts = srcPorts
	}
	if dstPorts != "" {
		h.DstPorts = dstPorts
	}
	return h
}

// DropLegacyNextHop removes deprecated next-hop columns from a route data frame.
func DropLegacyNextHop(df *dataframe.DataFrame) *dataframe.DataFrame {
	var toDrop []string
	for _, column := range legacyNextHopColumns {
		if df.HasColumn(column) {
			toDrop = append(toDrop, column)
		}
	}
	if len(toDrop) == 0 {
		return df
	}
	return df.Drop(toDrop...)
}

// splitCSV splits a comma-separated string, dropping empty entries, and returns
// nil when the result is empty.
func splitCSV(s string) []string {
	var out []string
	for _, item := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(item); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// optionalString returns nil when s is empty, so callers can pass it to
// client option structs.
func optionalString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
