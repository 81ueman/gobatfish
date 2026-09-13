package client

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/81ueman/gobatfish/datamodel"
	"github.com/81ueman/gobatfish/question"
)

// integrationEnabled reports whether integration tests should run. They require
// a Batfish allinone server, for example:
//
//	docker run --rm -p 9996:9996 -p 9997:9997 batfish/allinone
//	BATFISH_INTEGRATION=1 go test ./client/... -run Integration
func integrationEnabled(t *testing.T) {
	t.Helper()
	if os.Getenv("BATFISH_INTEGRATION") == "" {
		t.Skip("set BATFISH_INTEGRATION=1 to run integration tests")
	}
}

func integrationSession(t *testing.T) *Session {
	t.Helper()
	host := os.Getenv("BATFISH_HOST")
	if host == "" {
		host = "localhost"
	}
	port := 9996
	cfg := SessionConfig{Host: host, Port: &port}
	timeout := 60 * time.Second
	cfg.Timeout = &timeout
	return NewSession(cfg)
}

const integrationConfig = `hostname r1
!
interface GigabitEthernet0/0
 ip address 10.0.0.1 255.255.255.0
 no shutdown
!
interface GigabitEthernet0/1
 ip address 10.0.1.1 255.255.255.0
 no shutdown
!
`

func TestIntegrationSmoke(t *testing.T) {
	integrationEnabled(t)
	ctx := context.Background()
	s := integrationSession(t)
	defer s.Close()

	if _, err := s.SetNetwork(ctx, ""); err != nil {
		t.Fatalf("SetNetwork: %v", err)
	}
	defer func() { _ = s.DeleteNetwork(ctx, deref(s.Network)) }()

	if _, err := s.InitSnapshotFromText(ctx, integrationConfig, InitSnapshotFromTextOptions{
		Platform: "cisco",
		Filename: "r1.cfg",
	}); err != nil {
		t.Fatalf("InitSnapshotFromText: %v", err)
	}
	if s.Snapshot == nil {
		t.Fatal("snapshot not set after init")
	}

	versions, err := s.GetComponentVersions(ctx)
	if err != nil {
		t.Fatalf("GetComponentVersions: %v", err)
	}
	fmt.Printf("versions: %v\n", versions)

	if err := s.LoadQuestions(ctx); err != nil {
		t.Fatalf("LoadQuestions: %v", err)
	}
	if !s.Q.Has("nodeProperties") {
		t.Fatalf("nodeProperties question not loaded; have %d questions", len(s.Q.Names()))
	}

	q, err := s.Q.Get("nodeProperties")
	if err != nil {
		t.Fatal(err)
	}
	q.Set("nodes", ".*")
	result, err := q.Answer(ctx, question.AnswerOptions{Snapshot: s.Snapshot})
	if err != nil {
		t.Fatalf("answer nodeProperties: %v", err)
	}
	table, ok := result.Table()
	if !ok {
		t.Fatalf("nodeProperties did not return a table: %T", result)
	}
	t.Logf("nodeProperties rows: %d, columns: %v", table.Len(), table.Frame().Columns())
	if table.Len() != 1 {
		t.Fatalf("expected 1 node, got %d", table.Len())
	}
	if got := fmt.Sprint(table.Frame().At(0, "Node")); got != "r1" {
		t.Fatalf("node = %q, want r1", got)
	}

	if _, err := s.Asserts.AssertNoUndefinedReferences(ctx, false, nil, "table"); err != nil {
		t.Fatalf("AssertNoUndefinedReferences: %v", err)
	}
}

const integrationACLConfig = `hostname r1
!
ip access-list extended acl_in
 permit ip 10.0.0.0 0.0.0.255 any
 deny ip any any
!
interface GigabitEthernet0/0
 ip address 10.0.0.1 255.255.255.0
 no shutdown
!
`

func TestIntegrationHeaderConstraints(t *testing.T) {
	integrationEnabled(t)
	ctx := context.Background()
	s := integrationSession(t)
	defer s.Close()

	if _, err := s.SetNetwork(ctx, ""); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.DeleteNetwork(ctx, deref(s.Network)) }()

	if _, err := s.InitSnapshotFromText(ctx, integrationACLConfig, InitSnapshotFromTextOptions{
		Platform: "cisco",
		Filename: "r1.cfg",
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.LoadQuestions(ctx); err != nil {
		t.Fatal(err)
	}

	q, err := s.Q.Get("searchFilters")
	if err != nil {
		t.Fatal(err)
	}
	q.Set("headers", datamodel.HeaderConstraints{
		SrcIps:      "10.0.0.5",
		DstIps:      "8.8.8.8",
		IpProtocols: []any{"TCP"},
	})
	q.Set("action", "permit")
	q.Set("filters", "acl_in")

	ans, err := q.Answer(ctx, question.AnswerOptions{Snapshot: s.Snapshot})
	if err != nil {
		t.Fatalf("searchFilters answer: %v", err)
	}
	table, ok := ans.Table()
	if !ok {
		t.Fatalf("searchFilters did not return a table: %T", ans)
	}
	t.Logf("searchFilters rows: %d", table.Len())

	pass, err := s.Asserts.AssertFilterHasNoUnreachableLines(ctx, "acl_in", false, nil, "table")
	if err != nil || !pass {
		t.Fatalf("AssertFilterHasNoUnreachableLines: pass=%v err=%v", pass, err)
	}
}
