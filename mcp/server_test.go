package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/81ueman/gobatfish/client"
	"github.com/81ueman/gobatfish/dataframe"
	"github.com/81ueman/gobatfish/datamodel"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

type askCall struct {
	question string
	vars     map[string]any
	ref      *string
}

type fakeSession struct {
	mu           sync.Mutex
	networks     []string
	snapshots    []string
	frames       map[string]*dataframe.DataFrame
	asks         []askCall
	setNetworks  []string
	setSnapshots []string
	initOptions  client.InitSnapshotOptions
	fromText     client.InitSnapshotFromTextOptions
	forkOptions  client.ForkSnapshotOptions
	deletedSnap  string
	closed       bool
}

func newFakeSession() *fakeSession {
	return &fakeSession{frames: map[string]*dataframe.DataFrame{}}
}

func (f *fakeSession) ListNetworks(context.Context) ([]string, error) { return f.networks, nil }

func (f *fakeSession) SetNetwork(_ context.Context, name string) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.setNetworks = append(f.setNetworks, name)
	return name, nil
}

func (f *fakeSession) DeleteNetwork(context.Context, string) error { return nil }

func (f *fakeSession) ListSnapshots(context.Context, bool) ([]any, error) {
	out := make([]any, len(f.snapshots))
	for i, s := range f.snapshots {
		out[i] = s
	}
	return out, nil
}

func (f *fakeSession) SetSnapshot(_ context.Context, name string, _ *int) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.setSnapshots = append(f.setSnapshots, name)
	return name, nil
}

func (f *fakeSession) InitSnapshot(_ context.Context, _ string, opts client.InitSnapshotOptions) (string, error) {
	f.initOptions = opts
	return "snap-init", nil
}

func (f *fakeSession) InitSnapshotFromText(_ context.Context, _ string, opts client.InitSnapshotFromTextOptions) (string, error) {
	f.fromText = opts
	return "snap-text", nil
}

func (f *fakeSession) DeleteSnapshot(_ context.Context, name string) error {
	f.deletedSnap = name
	return nil
}

func (f *fakeSession) ForkSnapshot(_ context.Context, base string, opts client.ForkSnapshotOptions) (string, error) {
	f.forkOptions = opts
	return "snap-fork", nil
}

func (f *fakeSession) Ask(_ context.Context, question string, vars map[string]any, _ *string, ref *string) (*dataframe.DataFrame, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.asks = append(f.asks, askCall{question: question, vars: vars, ref: ref})
	if df, ok := f.frames[question]; ok {
		return df, nil
	}
	return dataframe.Empty(), nil
}

func (f *fakeSession) Close() { f.closed = true }

func (f *fakeSession) lastAsk(t *testing.T) askCall {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.asks) == 0 {
		t.Fatal("no question was asked")
	}
	return f.asks[len(f.asks)-1]
}

// newTestServer creates a server whose default session is the fake.
func newTestServer(t *testing.T, fake *fakeSession) *Server {
	t.Helper()
	server, err := NewServer(Options{
		SessionsConfigPath: filepath.Join(t.TempDir(), "missing.json"),
		SessionFactory:     func(context.Context, SessionEntry) (Session, error) { return fake, nil },
	})
	if err != nil {
		t.Fatal(err)
	}
	return server
}

// newTestClient connects an in-memory MCP client to the server.
func newTestClient(t *testing.T, server *Server) *mcpsdk.ClientSession {
	t.Helper()
	ctx := context.Background()
	cTransport, sTransport := mcpsdk.NewInMemoryTransports()
	ss, err := server.MCP().Connect(ctx, sTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ss.Close() })
	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test", Version: "1"}, nil)
	cs, err := client.Connect(ctx, cTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

func callTool[T any](t *testing.T, cs *mcpsdk.ClientSession, name string, args map[string]any) T {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s): %v", name, err)
	}
	if res.IsError {
		var messages []string
		for _, content := range res.Content {
			if text, ok := content.(*mcpsdk.TextContent); ok {
				messages = append(messages, text.Text)
			}
		}
		t.Fatalf("tool %s returned an error: %s", name, strings.Join(messages, "; "))
	}
	data, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var out T
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

// --- helper unit tests ------------------------------------------------------

func TestParseInterfaces(t *testing.T) {
	if got, err := ParseInterfaces(""); err != nil || len(got) != 0 {
		t.Fatalf("empty: %v, %v", got, err)
	}
	got, err := ParseInterfaces("router1[GigabitEthernet0/0]")
	if err != nil || len(got) != 1 || got[0].Hostname != "router1" || got[0].Interface != "GigabitEthernet0/0" {
		t.Fatalf("single: %v, %v", got, err)
	}
	got, err = ParseInterfaces("r1[Gi0/0], r2[Gi0/1]")
	if err != nil || len(got) != 2 || got[0].Hostname != "r1" || got[1].Hostname != "r2" {
		t.Fatalf("multiple: %v, %v", got, err)
	}
	if _, err := ParseInterfaces("router1"); err == nil || !strings.Contains(err.Error(), "node[interface]") {
		t.Fatalf("bare node error = %v", err)
	}
	if _, err := ParseInterfaces("r1[Gi0/0], bare-node"); err == nil || !strings.Contains(err.Error(), "bare-node") {
		t.Fatalf("mixed error = %v", err)
	}
}

func TestBuildHeaderConstraints(t *testing.T) {
	empty := BuildHeaderConstraints("", "", "", "", "", "")
	if empty.DstIps != nil || empty.SrcIps != nil {
		t.Fatalf("empty = %+v", empty)
	}
	dst := BuildHeaderConstraints("10.0.0.1", "", "", "", "", "")
	if dst.DstIps != "10.0.0.1" {
		t.Fatalf("dst = %v", dst.DstIps)
	}
	all := BuildHeaderConstraints("10.0.0.1", "192.168.1.0/24", "SSH", "TCP", "1024-65535", "22")
	normalized, err := all.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if normalized.SrcIps != "192.168.1.0/24" || normalized.SrcPorts != "1024-65535" || normalized.DstPorts != "22" {
		t.Fatalf("normalized = %+v", normalized)
	}
	if !reflect.DeepEqual(normalized.Applications, []any{"SSH"}) || !reflect.DeepEqual(normalized.IpProtocols, []any{"TCP"}) {
		t.Fatalf("lists = %v, %v", normalized.Applications, normalized.IpProtocols)
	}
}

func TestDropLegacyNextHop(t *testing.T) {
	df := dataframe.FromRecords([]map[string]any{
		{"Network": "1.1.1.0/24", "Next_Hop_IP": "2.2.2.2", "Next_Hop": "ip 2.2.2.2"},
	})
	got := DropLegacyNextHop(df)
	if got.HasColumn("Next_Hop_IP") {
		t.Fatal("legacy column not dropped")
	}
	if !got.HasColumn("Next_Hop") || !got.HasColumn("Network") {
		t.Fatalf("columns = %v", got.Columns())
	}
}

func TestTableResult(t *testing.T) {
	df := dataframe.FromRecords([]map[string]any{{"a": 1}, {"a": 2}})
	result := NewTableResult(df)
	if result.Count != 2 || len(result.Rows) != 2 || !reflect.DeepEqual(result.Columns, []string{"a"}) {
		t.Fatalf("result = %+v", result)
	}
}

// --- registry tests ---------------------------------------------------------

func TestRegistryLoadConfigEnvHost(t *testing.T) {
	t.Setenv("BATFISH_HOST", "env-host")
	r := NewRegistry(nil)
	if err := r.LoadConfig(filepath.Join(t.TempDir(), "missing.json")); err != nil {
		t.Fatal(err)
	}
	entry := r.configs["default"]
	if entry.Type != "bf" || entry.Params["host"] != "env-host" {
		t.Fatalf("default = %+v", entry)
	}
}

func TestRegistryLoadConfigFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.json")
	if err := os.WriteFile(path, []byte(`{"prod": {"type": "bf", "params": {"host": "prod-host"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	r := NewRegistry(nil)
	if err := r.LoadConfig(path); err != nil {
		t.Fatal(err)
	}
	if r.configs["prod"].Params["host"] != "prod-host" {
		t.Fatalf("prod = %+v", r.configs["prod"])
	}
	if _, ok := r.configs["default"]; !ok {
		t.Fatal("default session was not added")
	}
}

func TestRegistryConfigOverridesDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sessions.json")
	if err := os.WriteFile(path, []byte(`{"default": {"type": "bf", "params": {"host": "custom-host"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	r := NewRegistry(nil)
	if err := r.LoadConfig(path); err != nil {
		t.Fatal(err)
	}
	if r.configs["default"].Params["host"] != "custom-host" {
		t.Fatalf("default = %+v", r.configs["default"])
	}
}

func TestRegistryCaches(t *testing.T) {
	var created int
	r := NewRegistry(func(context.Context, SessionEntry) (Session, error) {
		created++
		return newFakeSession(), nil
	})
	if err := r.LoadConfig(filepath.Join(t.TempDir(), "missing.json")); err != nil {
		t.Fatal(err)
	}
	s1, err := r.Get(context.Background(), "default")
	if err != nil {
		t.Fatal(err)
	}
	s2, _ := r.Get(context.Background(), "default")
	if created != 1 || s1 != s2 {
		t.Fatalf("created = %d, s1 == s2 = %v", created, s1 == s2)
	}
}

func TestRegistryUnknownRaises(t *testing.T) {
	r := NewRegistry(nil)
	if err := r.LoadConfig(filepath.Join(t.TempDir(), "missing.json")); err != nil {
		t.Fatal(err)
	}
	_, err := r.Get(context.Background(), "nonexistent")
	if err == nil || !strings.Contains(err.Error(), "No session named") {
		t.Fatalf("err = %v", err)
	}
}

func TestRegistryRegister(t *testing.T) {
	r := NewRegistry(func(context.Context, SessionEntry) (Session, error) { return newFakeSession(), nil })
	if err := r.LoadConfig(filepath.Join(t.TempDir(), "missing.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Register(context.Background(), "test", "bf", map[string]any{"host": "test-host"}); err != nil {
		t.Fatal(err)
	}
	if r.configs["test"].Params["host"] != "test-host" {
		t.Fatalf("config = %+v", r.configs["test"])
	}
}

// --- tool tests -------------------------------------------------------------

func TestAllToolsRegistered(t *testing.T) {
	server := newTestServer(t, newFakeSession())
	cs := newTestClient(t, server)
	result, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Tools) < 36 {
		names := make([]string, 0, len(result.Tools))
		for _, tool := range result.Tools {
			names = append(names, tool.Name)
		}
		t.Fatalf("registered %d tools: %v", len(result.Tools), names)
	}
	for _, want := range []string{"check_reachability", "get_routes", "search_filters", "get_bgp_rib", "detect_loops", "init_snapshot_from_text"} {
		found := false
		for _, tool := range result.Tools {
			if tool.Name == want {
				found = true
			}
		}
		if !found {
			t.Fatalf("tool %s not registered", want)
		}
	}
}

func TestListNetworksTool(t *testing.T) {
	fake := newFakeSession()
	fake.networks = []string{"n1", "n2"}
	server := newTestServer(t, fake)
	cs := newTestClient(t, server)
	out := callTool[NetworksResult](t, cs, "list_networks", map[string]any{})
	if !reflect.DeepEqual(out.Networks, []string{"n1", "n2"}) {
		t.Fatalf("networks = %v", out.Networks)
	}
}

func TestSetNetworkTool(t *testing.T) {
	fake := newFakeSession()
	server := newTestServer(t, fake)
	cs := newTestClient(t, server)
	out := callTool[NetworkResult](t, cs, "set_network", map[string]any{"network": "mynet"})
	if out.Network != "mynet" || !reflect.DeepEqual(fake.setNetworks, []string{"mynet"}) {
		t.Fatalf("out = %+v, set = %v", out, fake.setNetworks)
	}
}

func TestListSnapshotsTool(t *testing.T) {
	fake := newFakeSession()
	fake.snapshots = []string{"s1"}
	server := newTestServer(t, fake)
	cs := newTestClient(t, server)
	out := callTool[SnapshotListResult](t, cs, "list_snapshots", map[string]any{"network": "n1"})
	if !reflect.DeepEqual(out.Snapshots, []string{"s1"}) {
		t.Fatalf("snapshots = %v", out.Snapshots)
	}
	if !reflect.DeepEqual(fake.setNetworks, []string{"n1"}) {
		t.Fatalf("network not set: %v", fake.setNetworks)
	}
}

func TestInitSnapshotFromTextTool(t *testing.T) {
	fake := newFakeSession()
	server := newTestServer(t, fake)
	cs := newTestClient(t, server)
	out := callTool[SnapshotResult](t, cs, "init_snapshot_from_text", map[string]any{
		"network": "n1", "config_text": "hostname r1", "filename": "r1.cfg", "platform": "cisco",
	})
	if out.Snapshot != "snap-text" || fake.fromText.Filename != "r1.cfg" || fake.fromText.Platform != "cisco" {
		t.Fatalf("out = %+v, opts = %+v", out, fake.fromText)
	}
}

func TestForkSnapshotTool(t *testing.T) {
	fake := newFakeSession()
	server := newTestServer(t, fake)
	cs := newTestClient(t, server)
	out := callTool[SnapshotResult](t, cs, "fork_snapshot", map[string]any{
		"network": "n1", "base_snapshot": "base",
		"deactivate_nodes":      "r1,r2",
		"deactivate_interfaces": "r1[Gi0/0], r2[Gi0/1]",
	})
	if out.Snapshot != "snap-fork" {
		t.Fatalf("out = %+v", out)
	}
	if !reflect.DeepEqual(fake.forkOptions.DeactivateNodes, []string{"r1", "r2"}) {
		t.Fatalf("nodes = %v", fake.forkOptions.DeactivateNodes)
	}
	want := []datamodel.Interface{{Hostname: "r1", Interface: "Gi0/0"}, {Hostname: "r2", Interface: "Gi0/1"}}
	if !reflect.DeepEqual(fake.forkOptions.DeactivateInterfaces, want) {
		t.Fatalf("interfaces = %v", fake.forkOptions.DeactivateInterfaces)
	}
}

func TestCheckReachabilityTool(t *testing.T) {
	fake := newFakeSession()
	server := newTestServer(t, fake)
	cs := newTestClient(t, server)
	callTool[TableResult](t, cs, "check_reachability", map[string]any{
		"network": "n1", "snapshot": "s1",
		"src_locations": "r1", "dst_ips": "8.8.8.8", "actions": "DENIED_IN",
	})
	call := fake.lastAsk(t)
	if call.question != "reachability" {
		t.Fatalf("question = %q", call.question)
	}
	if call.vars["actions"] != "DENIED_IN" {
		t.Fatalf("actions = %v", call.vars["actions"])
	}
	pc, ok := call.vars["pathConstraints"].(datamodel.PathConstraints)
	if !ok || pc.StartLocation == nil || *pc.StartLocation != "r1" {
		t.Fatalf("pathConstraints = %v", call.vars["pathConstraints"])
	}
	headers, ok := call.vars["headers"].(datamodel.HeaderConstraints)
	if !ok || headers.DstIps != "8.8.8.8" {
		t.Fatalf("headers = %v", call.vars["headers"])
	}
}

func TestGetRoutesToolDropsLegacyColumns(t *testing.T) {
	fake := newFakeSession()
	fake.frames["routes"] = dataframe.FromRecords([]map[string]any{
		{"Network": "1.1.1.0/24", "Next_Hop_IP": "2.2.2.2", "Next_Hop": "ip 2.2.2.2"},
	})
	server := newTestServer(t, fake)
	cs := newTestClient(t, server)
	out := callTool[TableResult](t, cs, "get_routes", map[string]any{
		"network": "n1", "snapshot": "s1", "nodes": "r1", "protocols": "bgp",
	})
	for _, column := range out.Columns {
		if column == "Next_Hop_IP" {
			t.Fatalf("legacy column present: %v", out.Columns)
		}
	}
	call := fake.lastAsk(t)
	if call.question != "routes" || call.vars["nodes"] != "r1" || call.vars["protocols"] != "bgp" {
		t.Fatalf("call = %+v", call)
	}
}

func TestGetBgpSessionStatusTool(t *testing.T) {
	fake := newFakeSession()
	server := newTestServer(t, fake)
	cs := newTestClient(t, server)
	callTool[TableResult](t, cs, "get_bgp_session_status", map[string]any{
		"network": "n1", "snapshot": "s1", "nodes": "r1", "remote_nodes": "r2", "status": "ESTABLISHED",
	})
	call := fake.lastAsk(t)
	if call.question != "bgpSessionStatus" || call.vars["remoteNodes"] != "r2" || call.vars["status"] != "ESTABLISHED" {
		t.Fatalf("call = %+v", call)
	}
}

func TestCompareRoutesUsesReferenceSnapshot(t *testing.T) {
	fake := newFakeSession()
	server := newTestServer(t, fake)
	cs := newTestClient(t, server)
	callTool[TableResult](t, cs, "compare_routes", map[string]any{
		"network": "n1", "snapshot": "new", "reference_snapshot": "old",
	})
	call := fake.lastAsk(t)
	if call.ref == nil || *call.ref != "old" {
		t.Fatalf("ref = %v", call.ref)
	}
}

func TestUnknownSessionToolError(t *testing.T) {
	server := newTestServer(t, newFakeSession())
	cs := newTestClient(t, server)
	res, err := cs.CallTool(context.Background(), &mcpsdk.CallToolParams{
		Name: "list_networks", Arguments: map[string]any{"session": "missing"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.IsError {
		t.Fatal("expected tool error for unknown session")
	}
}
