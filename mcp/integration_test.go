package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/81ueman/gobatfish/client"
	"github.com/81ueman/gobatfish/util"
)

const integrationConfig = `hostname r1
!
interface GigabitEthernet0/0
 ip address 10.0.0.1 255.255.255.0
 no shutdown
!
`

func TestIntegrationServer(t *testing.T) {
	if os.Getenv("BATFISH_INTEGRATION") == "" {
		t.Skip("set BATFISH_INTEGRATION=1 to run integration tests")
	}
	ctx := context.Background()
	host := os.Getenv("BATFISH_HOST")
	if host == "" {
		host = "localhost"
	}
	port := 9996
	session := client.NewSession(client.SessionConfig{Host: host, Port: &port})
	defer session.Close()
	if err := session.LoadQuestions(ctx); err != nil {
		t.Fatalf("LoadQuestions: %v", err)
	}

	server, err := NewServer(Options{
		DefaultSession:     session,
		SessionsConfigPath: filepath.Join(t.TempDir(), "missing.json"),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer server.Close()
	cs := newTestClient(t, server)

	network := "mcp_" + util.GetUUID()
	defer func() { _ = session.DeleteNetwork(ctx, network) }()

	initResult := callTool[SnapshotResult](t, cs, "init_snapshot_from_text", map[string]any{
		"network":     network,
		"config_text": integrationConfig,
		"platform":    "cisco",
	})
	if initResult.Snapshot == "" {
		t.Fatal("empty snapshot name")
	}

	props := callTool[TableResult](t, cs, "get_node_properties", map[string]any{
		"network": network, "snapshot": initResult.Snapshot,
	})
	if props.Count < 1 {
		t.Fatalf("expected at least one node, got %+v", props)
	}

	loops := callTool[TableResult](t, cs, "detect_loops", map[string]any{
		"network": network, "snapshot": initResult.Snapshot,
	})
	if loops.Count != 0 {
		t.Fatalf("unexpected forwarding loops: %+v", loops)
	}

	snapshots := callTool[SnapshotListResult](t, cs, "list_snapshots", map[string]any{"network": network})
	if len(snapshots.Snapshots) != 1 {
		t.Fatalf("snapshots = %v", snapshots.Snapshots)
	}
}
