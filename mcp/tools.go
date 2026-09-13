package mcp

import (
	"context"
	"fmt"

	"github.com/81ueman/gobatfish/client"
	"github.com/81ueman/gobatfish/datamodel"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func sessionName(s string) string {
	if s == "" {
		return "default"
	}
	return s
}

// addTool registers a typed MCP tool.
func addTool[In, Out any](s *Server, name, description string, fn func(context.Context, *mcpsdk.CallToolRequest, In) (Out, error)) {
	mcpsdk.AddTool(s.mcp, &mcpsdk.Tool{Name: name, Description: description},
		func(ctx context.Context, req *mcpsdk.CallToolRequest, in In) (*mcpsdk.CallToolResult, Out, error) {
			out, err := fn(ctx, req, in)
			return nil, out, err
		})
}

// addTableTool registers a table-returning MCP tool.
func addTableTool[In any](s *Server, name, description string, fn func(context.Context, *mcpsdk.CallToolRequest, In) (TableResult, error)) {
	addTool(s, name, description, fn)
}

// --- inputs -----------------------------------------------------------------

type emptyInput struct{}

type sessionInput struct {
	Session string `json:"session,omitempty" jsonschema:"named session to use (default 'default')"`
}

type networkInput struct {
	Network string `json:"network" jsonschema:"name of the network"`
	Session string `json:"session,omitempty" jsonschema:"named session to use (default 'default')"`
}

type snapshotInput struct {
	Network  string `json:"network" jsonschema:"name of the network"`
	Snapshot string `json:"snapshot" jsonschema:"name of the snapshot"`
	Session  string `json:"session,omitempty" jsonschema:"named session to use (default 'default')"`
}

// analysisInput is the shared input for table-returning analysis tools. It is a
// superset of the parameters used across those tools; individual tools use the
// subset relevant to them.
type analysisInput struct {
	Network  string `json:"network" jsonschema:"name of the network"`
	Snapshot string `json:"snapshot" jsonschema:"name of the snapshot"`

	Session           string `json:"session,omitempty" jsonschema:"named session to use (default 'default')"`
	ReferenceSnapshot string `json:"reference_snapshot,omitempty" jsonschema:"reference snapshot for differential questions"`

	Nodes       string `json:"nodes,omitempty" jsonschema:"node specifier"`
	RemoteNodes string `json:"remote_nodes,omitempty" jsonschema:"remote node specifier"`
	Vrfs        string `json:"vrfs,omitempty" jsonschema:"VRF specifier"`
	Protocols   string `json:"protocols,omitempty" jsonschema:"routing protocol(s), e.g. 'bgp,ospf'"`

	NetworkPrefix   string `json:"network_prefix,omitempty" jsonschema:"prefix to filter routes by"`
	PrefixMatchType string `json:"prefix_match_type,omitempty" jsonschema:"EXACT, LONGEST_PREFIX_MATCH, LONGER_PREFIXES or SHORTER_PREFIXES"`
	Status          string `json:"status,omitempty" jsonschema:"status specifier to filter by"`
	Properties      string `json:"properties,omitempty" jsonschema:"regex of properties to include"`
	Interfaces      string `json:"interfaces,omitempty" jsonschema:"interface specifier"`
	Filters         string `json:"filters,omitempty" jsonschema:"filter specifier"`
	Names           string `json:"names,omitempty" jsonschema:"regex to filter structure names"`
	Types           string `json:"types,omitempty" jsonschema:"regex to filter structure types"`
	Filename        string `json:"filename,omitempty" jsonschema:"include only structures defined in this file"`

	StartLocation   string `json:"start_location,omitempty" jsonschema:"source location specifier"`
	SourceLocations string `json:"src_locations,omitempty" jsonschema:"source location specifier"`
	SrcIPs          string `json:"src_ips,omitempty" jsonschema:"source IP address or prefix"`
	DstIPs          string `json:"dst_ips,omitempty" jsonschema:"destination IP address or prefix"`
	Applications    string `json:"applications,omitempty" jsonschema:"application specifier"`
	IPProtocols     string `json:"ip_protocols,omitempty" jsonschema:"IP protocol(s)"`
	SrcPorts        string `json:"src_ports,omitempty" jsonschema:"source port(s)"`
	DstPorts        string `json:"dst_ports,omitempty" jsonschema:"destination port(s)"`
	Actions         string `json:"actions,omitempty" jsonschema:"disposition filter"`
	Action          string `json:"action,omitempty" jsonschema:"filter action, 'permit' or 'deny'"`

	AggregateDuplicates bool `json:"aggregate_duplicates,omitempty" jsonschema:"aggregate duplicate warnings"`
	DuplicatesOnly      bool `json:"duplicates_only,omitempty" jsonschema:"return only duplicated IPs"`
}

// --- management tools -------------------------------------------------------

type registerSessionInput struct {
	Name   string         `json:"name" jsonschema:"name for the session"`
	Type   string         `json:"type,omitempty" jsonschema:"session type entry point name (default 'bf')"`
	Params map[string]any `json:"params,omitempty" jsonschema:"session constructor parameters"`
}

type initSnapshotInput struct {
	Network      string `json:"network" jsonschema:"name of the network"`
	SnapshotPath string `json:"snapshot_path" jsonschema:"local path to a snapshot directory or zip file"`
	SnapshotName string `json:"snapshot_name,omitempty" jsonschema:"optional snapshot name"`
	Overwrite    bool   `json:"overwrite,omitempty" jsonschema:"overwrite an existing snapshot"`
	Session      string `json:"session,omitempty" jsonschema:"named session to use (default 'default')"`
}

type initSnapshotFromTextInput struct {
	Network      string `json:"network" jsonschema:"name of the network"`
	ConfigText   string `json:"config_text" jsonschema:"raw configuration text"`
	Filename     string `json:"filename,omitempty" jsonschema:"filename to use inside the snapshot"`
	SnapshotName string `json:"snapshot_name,omitempty" jsonschema:"optional snapshot name"`
	Platform     string `json:"platform,omitempty" jsonschema:"RANCID platform string"`
	Overwrite    bool   `json:"overwrite,omitempty" jsonschema:"overwrite an existing snapshot"`
	Session      string `json:"session,omitempty" jsonschema:"named session to use (default 'default')"`
}

type forkSnapshotInput struct {
	Network              string `json:"network" jsonschema:"name of the network"`
	BaseSnapshot         string `json:"base_snapshot" jsonschema:"snapshot to fork from"`
	NewSnapshot          string `json:"new_snapshot,omitempty" jsonschema:"name for the forked snapshot"`
	DeactivateNodes      string `json:"deactivate_nodes,omitempty" jsonschema:"comma-separated node names to deactivate"`
	DeactivateInterfaces string `json:"deactivate_interfaces,omitempty" jsonschema:"comma-separated 'node[interface]' pairs to deactivate"`
	RestoreNodes         string `json:"restore_nodes,omitempty" jsonschema:"comma-separated node names to restore"`
	RestoreInterfaces    string `json:"restore_interfaces,omitempty" jsonschema:"comma-separated 'node[interface]' pairs to restore"`
	Overwrite            bool   `json:"overwrite,omitempty" jsonschema:"overwrite an existing snapshot"`
	Session              string `json:"session,omitempty" jsonschema:"named session to use (default 'default')"`
}

func registerManagementTools(s *Server) {
	addTool(s, "register_session",
		"Register a new named session for use with all other tools.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in registerSessionInput) (RegisteredResult, error) {
			type_ := in.Type
			if type_ == "" {
				type_ = "bf"
			}
			if _, err := s.registry.Register(ctx, in.Name, type_, in.Params); err != nil {
				return RegisteredResult{}, err
			}
			return RegisteredResult{Registered: in.Name, Type: type_}, nil
		})

	addTool(s, "list_sessions", "List all registered session names and their types.",
		func(context.Context, *mcpsdk.CallToolRequest, emptyInput) (SessionsResult, error) {
			return SessionsResult{Sessions: s.registry.Names()}, nil
		})

	addTool(s, "list_networks", "List all available networks on the Batfish server.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in sessionInput) (NetworksResult, error) {
			session, err := s.registry.Get(ctx, sessionName(in.Session))
			if err != nil {
				return NetworksResult{}, err
			}
			networks, err := session.ListNetworks(ctx)
			if err != nil {
				return NetworksResult{}, err
			}
			if networks == nil {
				networks = []string{}
			}
			return NetworksResult{Networks: networks}, nil
		})

	addTool(s, "set_network", "Create or select a network on the Batfish server.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in networkInput) (NetworkResult, error) {
			session, err := s.registry.Get(ctx, sessionName(in.Session))
			if err != nil {
				return NetworkResult{}, err
			}
			name, err := session.SetNetwork(ctx, in.Network)
			if err != nil {
				return NetworkResult{}, err
			}
			return NetworkResult{Network: name}, nil
		})

	addTool(s, "delete_network", "Delete a network from the Batfish server.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in networkInput) (DeletedResult, error) {
			session, err := s.registry.Get(ctx, sessionName(in.Session))
			if err != nil {
				return DeletedResult{}, err
			}
			if err := session.DeleteNetwork(ctx, in.Network); err != nil {
				return DeletedResult{}, err
			}
			return DeletedResult{Deleted: in.Network}, nil
		})

	addTool(s, "list_snapshots", "List all snapshots within a network.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in networkInput) (SnapshotListResult, error) {
			session, err := s.registry.mgmt(ctx, sessionName(in.Session), in.Network)
			if err != nil {
				return SnapshotListResult{}, err
			}
			snapshots, err := session.ListSnapshots(ctx, false)
			if err != nil {
				return SnapshotListResult{}, err
			}
			out := make([]string, 0, len(snapshots))
			for _, snapshot := range snapshots {
				out = append(out, fmt.Sprint(snapshot))
			}
			return SnapshotListResult{Snapshots: out}, nil
		})

	addTool(s, "init_snapshot", "Initialize a new snapshot from a local directory or zip file.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in initSnapshotInput) (SnapshotResult, error) {
			session, err := s.registry.mgmt(ctx, sessionName(in.Session), in.Network)
			if err != nil {
				return SnapshotResult{}, err
			}
			name, err := session.InitSnapshot(ctx, in.SnapshotPath, client.InitSnapshotOptions{
				Name:      in.SnapshotName,
				Overwrite: in.Overwrite,
			})
			if err != nil {
				return SnapshotResult{}, err
			}
			return SnapshotResult{Snapshot: name}, nil
		})

	addTool(s, "init_snapshot_from_text", "Initialize a single-device snapshot from configuration text.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in initSnapshotFromTextInput) (SnapshotResult, error) {
			session, err := s.registry.mgmt(ctx, sessionName(in.Session), in.Network)
			if err != nil {
				return SnapshotResult{}, err
			}
			name, err := session.InitSnapshotFromText(ctx, in.ConfigText, client.InitSnapshotFromTextOptions{
				Filename:  in.Filename,
				Name:      in.SnapshotName,
				Platform:  in.Platform,
				Overwrite: in.Overwrite,
			})
			if err != nil {
				return SnapshotResult{}, err
			}
			return SnapshotResult{Snapshot: name}, nil
		})

	addTool(s, "delete_snapshot", "Delete a snapshot from a network.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in snapshotInput) (DeletedResult, error) {
			session, err := s.registry.mgmt(ctx, sessionName(in.Session), in.Network)
			if err != nil {
				return DeletedResult{}, err
			}
			if err := session.DeleteSnapshot(ctx, in.Snapshot); err != nil {
				return DeletedResult{}, err
			}
			return DeletedResult{Deleted: in.Snapshot}, nil
		})

	addTool(s, "fork_snapshot", "Fork an existing snapshot, optionally deactivating or restoring nodes/interfaces.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in forkSnapshotInput) (SnapshotResult, error) {
			session, err := s.registry.mgmt(ctx, sessionName(in.Session), in.Network)
			if err != nil {
				return SnapshotResult{}, err
			}
			deactivateInterfaces, err := ParseInterfaces(in.DeactivateInterfaces)
			if err != nil {
				return SnapshotResult{}, err
			}
			restoreInterfaces, err := ParseInterfaces(in.RestoreInterfaces)
			if err != nil {
				return SnapshotResult{}, err
			}
			name, err := session.ForkSnapshot(ctx, in.BaseSnapshot, client.ForkSnapshotOptions{
				Name:                 in.NewSnapshot,
				Overwrite:            in.Overwrite,
				DeactivateNodes:      splitCSV(in.DeactivateNodes),
				DeactivateInterfaces: deactivateInterfaces,
				RestoreNodes:         splitCSV(in.RestoreNodes),
				RestoreInterfaces:    restoreInterfaces,
			})
			if err != nil {
				return SnapshotResult{}, err
			}
			return SnapshotResult{Snapshot: name}, nil
		})
}

// --- analysis tools ---------------------------------------------------------

func registerAnalysisTools(s *Server) {
	addTableTool(s, "get_parse_warnings", "Get warnings from parsing the snapshot configurations.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			vars := map[string]any{}
			if in.AggregateDuplicates {
				vars["aggregateDuplicates"] = true
			}
			return s.ask(ctx, in.Session, in.Network, in.Snapshot, "parseWarning", vars, nil)
		})

	addTableTool(s, "get_init_issues", "Get issues encountered when processing the snapshot.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			return s.ask(ctx, in.Session, in.Network, in.Snapshot, "initIssues", nil, nil)
		})

	addTableTool(s, "get_file_parse_status", "Get the parse status of each file in the snapshot.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			return s.ask(ctx, in.Session, in.Network, in.Snapshot, "fileParseStatus", nil, nil)
		})

	addTableTool(s, "get_vi_conversion_status", "Get the vendor-independent conversion status for each node.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			return s.ask(ctx, in.Session, in.Network, in.Snapshot, "viConversionStatus", nil, nil)
		})

	addTableTool(s, "get_vi_conversion_warnings", "Get warnings from vendor-independent model conversion.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			return s.ask(ctx, in.Session, in.Network, in.Snapshot, "viConversionWarning", nil, nil)
		})

	addTableTool(s, "run_traceroute", "Simulate a traceroute from a location to a destination IP address.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			vars := map[string]any{
				"startLocation": in.StartLocation,
				"headers":       BuildHeaderConstraints(in.DstIPs, in.SrcIPs, in.Applications, in.IPProtocols, in.SrcPorts, in.DstPorts),
			}
			return s.ask(ctx, in.Session, in.Network, in.Snapshot, "traceroute", vars, nil)
		})

	addTableTool(s, "run_bidirectional_traceroute", "Simulate a bidirectional traceroute (forward + reverse paths).",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			vars := map[string]any{
				"startLocation": in.StartLocation,
				"headers":       BuildHeaderConstraints(in.DstIPs, in.SrcIPs, in.Applications, in.IPProtocols, in.SrcPorts, in.DstPorts),
			}
			return s.ask(ctx, in.Session, in.Network, in.Snapshot, "bidirectionalTraceroute", vars, nil)
		})

	addTableTool(s, "check_reachability", "Check reachability between network locations.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			vars := map[string]any{
				"headers": BuildHeaderConstraints(in.DstIPs, in.SrcIPs, in.Applications, in.IPProtocols, in.SrcPorts, in.DstPorts),
			}
			if in.SourceLocations != "" {
				vars["pathConstraints"] = datamodel.PathConstraints{StartLocation: optionalString(in.SourceLocations)}
			}
			if in.Actions != "" {
				vars["actions"] = in.Actions
			}
			return s.ask(ctx, in.Session, in.Network, in.Snapshot, "reachability", vars, nil)
		})

	addTableTool(s, "analyze_acl", "Identify unreachable (shadowed) lines in ACLs and firewall rules.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			vars := map[string]any{}
			if in.Filters != "" {
				vars["filters"] = in.Filters
			}
			if in.Nodes != "" {
				vars["nodes"] = in.Nodes
			}
			return s.ask(ctx, in.Session, in.Network, in.Snapshot, "filterLineReachability", vars, nil)
		})

	addTableTool(s, "search_filters", "Search for flows that match specific filter (ACL/firewall) criteria.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			vars := map[string]any{
				"headers": BuildHeaderConstraints(in.DstIPs, in.SrcIPs, in.Applications, in.IPProtocols, in.SrcPorts, in.DstPorts),
			}
			if in.Filters != "" {
				vars["filters"] = in.Filters
			}
			if in.Nodes != "" {
				vars["nodes"] = in.Nodes
			}
			if in.Action != "" {
				vars["action"] = in.Action
			}
			return s.ask(ctx, in.Session, in.Network, in.Snapshot, "searchFilters", vars, nil)
		})

	addTableTool(s, "get_routes", "Retrieve the routing table (RIB) from one or more devices.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			return s.routesResult(ctx, in, "routes")
		})

	addTableTool(s, "compare_routes", "Compare routing tables between two snapshots to identify route changes.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			return s.routesResult(ctx, in, "routes")
		})

	addTableTool(s, "get_bgp_rib", "Retrieve the BGP RIB (Routing Information Base) from devices.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			return s.routesResult(ctx, in, "bgpRib")
		})

	addTableTool(s, "get_evpn_rib", "Retrieve the EVPN RIB (Routing Information Base) from devices.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			return s.routesResult(ctx, in, "evpnRib")
		})

	addTableTool(s, "get_bgp_session_status", "Get the status of BGP sessions in a snapshot.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			vars := map[string]any{}
			if in.Nodes != "" {
				vars["nodes"] = in.Nodes
			}
			if in.RemoteNodes != "" {
				vars["remoteNodes"] = in.RemoteNodes
			}
			if in.Status != "" {
				vars["status"] = in.Status
			}
			return s.ask(ctx, in.Session, in.Network, in.Snapshot, "bgpSessionStatus", vars, nil)
		})

	addTableTool(s, "get_bgp_session_compatibility", "Check BGP session compatibility between peers.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			vars := map[string]any{}
			if in.Nodes != "" {
				vars["nodes"] = in.Nodes
			}
			if in.RemoteNodes != "" {
				vars["remoteNodes"] = in.RemoteNodes
			}
			if in.Status != "" {
				vars["status"] = in.Status
			}
			return s.ask(ctx, in.Session, in.Network, in.Snapshot, "bgpSessionCompatibility", vars, nil)
		})

	addTableTool(s, "get_bgp_peer_configuration", "Get configuration settings for BGP peerings.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			vars := map[string]any{}
			if in.Nodes != "" {
				vars["nodes"] = in.Nodes
			}
			if in.Properties != "" {
				vars["properties"] = in.Properties
			}
			return s.ask(ctx, in.Session, in.Network, in.Snapshot, "bgpPeerConfiguration", vars, nil)
		})

	addTableTool(s, "get_bgp_process_configuration", "Get configuration settings of BGP processes.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			vars := map[string]any{}
			if in.Nodes != "" {
				vars["nodes"] = in.Nodes
			}
			if in.Properties != "" {
				vars["properties"] = in.Properties
			}
			return s.ask(ctx, in.Session, in.Network, in.Snapshot, "bgpProcessConfiguration", vars, nil)
		})

	addTableTool(s, "get_node_properties", "Retrieve configuration properties of network nodes (routers/switches).",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			vars := map[string]any{}
			if in.Nodes != "" {
				vars["nodes"] = in.Nodes
			}
			if in.Properties != "" {
				vars["properties"] = in.Properties
			}
			return s.ask(ctx, in.Session, in.Network, in.Snapshot, "nodeProperties", vars, nil)
		})

	addTableTool(s, "get_interface_properties", "Retrieve configuration properties of network interfaces.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			vars := map[string]any{}
			if in.Nodes != "" {
				vars["nodes"] = in.Nodes
			}
			if in.Interfaces != "" {
				vars["interfaces"] = in.Interfaces
			}
			if in.Properties != "" {
				vars["properties"] = in.Properties
			}
			return s.ask(ctx, in.Session, in.Network, in.Snapshot, "interfaceProperties", vars, nil)
		})

	addTableTool(s, "get_ip_owners", "Get the mapping of IP addresses to network interfaces.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			return s.ask(ctx, in.Session, in.Network, in.Snapshot, "ipOwners", map[string]any{"duplicatesOnly": in.DuplicatesOnly}, nil)
		})

	addTableTool(s, "compare_filters", "Compare ACL/firewall filter behavior between two snapshots.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			vars := map[string]any{}
			if in.Filters != "" {
				vars["filters"] = in.Filters
			}
			if in.Nodes != "" {
				vars["nodes"] = in.Nodes
			}
			return s.ask(ctx, in.Session, in.Network, in.Snapshot, "compareFilters", vars, optionalString(in.ReferenceSnapshot))
		})

	addTableTool(s, "get_undefined_references", "Find undefined references in device configurations.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			vars := map[string]any{}
			if in.Nodes != "" {
				vars["nodes"] = in.Nodes
			}
			return s.ask(ctx, in.Session, in.Network, in.Snapshot, "undefinedReferences", vars, nil)
		})

	addTableTool(s, "get_defined_structures", "List named structures defined in the network configurations.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			vars := map[string]any{}
			if in.Nodes != "" {
				vars["nodes"] = in.Nodes
			}
			if in.Names != "" {
				vars["names"] = in.Names
			}
			if in.Types != "" {
				vars["types"] = in.Types
			}
			if in.Filename != "" {
				vars["filename"] = in.Filename
			}
			return s.ask(ctx, in.Session, in.Network, in.Snapshot, "definedStructures", vars, nil)
		})

	addTableTool(s, "get_unused_structures", "Find structures that are defined but never referenced.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			vars := map[string]any{}
			if in.Nodes != "" {
				vars["nodes"] = in.Nodes
			}
			return s.ask(ctx, in.Session, in.Network, in.Snapshot, "unusedStructures", vars, nil)
		})

	addTableTool(s, "detect_loops", "Detect forwarding loops in the network snapshot.",
		func(ctx context.Context, _ *mcpsdk.CallToolRequest, in analysisInput) (TableResult, error) {
			return s.ask(ctx, in.Session, in.Network, in.Snapshot, "detectLoops", nil, nil)
		})
}

// routesResult answers a routing question and drops deprecated next-hop columns.
func (s *Server) routesResult(ctx context.Context, in analysisInput, question string) (TableResult, error) {
	vars := map[string]any{}
	if in.Nodes != "" {
		vars["nodes"] = in.Nodes
	}
	if in.Vrfs != "" {
		vars["vrfs"] = in.Vrfs
	}
	if in.NetworkPrefix != "" {
		vars["network"] = in.NetworkPrefix
	}
	if in.Protocols != "" {
		vars["protocols"] = in.Protocols
	}
	if in.PrefixMatchType != "" {
		vars["prefixMatchType"] = in.PrefixMatchType
	}
	if in.Status != "" {
		vars["status"] = in.Status
	}
	df, err := s.askFrame(ctx, in.Session, in.Network, in.Snapshot, question, vars, optionalString(in.ReferenceSnapshot))
	if err != nil {
		return TableResult{}, err
	}
	return NewTableResult(DropLegacyNextHop(df)), nil
}
