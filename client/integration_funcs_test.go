package client

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/81ueman/gobatfish/datamodel"
	"github.com/81ueman/gobatfish/util"
)

func newIntegrationNetwork(t *testing.T, s *Session) string {
	t.Helper()
	ctx := context.Background()
	name, err := s.SetNetwork(ctx, "")
	if err != nil {
		t.Fatalf("SetNetwork: %v", err)
	}
	t.Cleanup(func() { _ = s.DeleteNetwork(ctx, name) })
	return name
}

func TestIntegrationNetworkFuncs(t *testing.T) {
	integrationEnabled(t)
	ctx := context.Background()
	s := integrationSession(t)
	defer s.Close()

	name := newIntegrationNetwork(t, s)
	if !strings.HasPrefix(name, DefaultNetworkPrefix) {
		t.Fatalf("network name = %q", name)
	}
	networks, err := s.ListNetworks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, n := range networks {
		if n == name {
			found = true
		}
	}
	if !found {
		t.Fatalf("network %q not listed in %v", name, networks)
	}

	works, err := s.ListIncompleteWorks(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if works[SvcKeyWorkList] == nil {
		t.Fatalf("works = %v", works)
	}
}

func TestIntegrationNodeRolesAndReferenceBook(t *testing.T) {
	integrationEnabled(t)
	ctx := context.Background()
	s := integrationSession(t)
	defer s.Close()
	newIntegrationNetwork(t, s)

	mapping := datamodel.RoleMapping{
		Name:                util.StringPtr("mapping1"),
		Regex:               "(.*)-(.*)",
		RoleDimensionGroups: map[string][]int{"type": {1}, "index": {2}},
		CanonicalRoleNames:  map[string]map[string]string{},
	}
	roles := datamodel.NodeRolesData{
		RoleDimensionOrder: []string{"type", "index"},
		RoleMappings:       []datamodel.RoleMapping{mapping},
	}
	if err := s.PutNodeRoles(ctx, roles); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetNodeRoles(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.RoleDimensionOrder, []string{"type", "index"}) || len(got.RoleMappings) != 1 {
		t.Fatalf("node roles = %+v", got)
	}
	if got.RoleMappings[0].Name == nil || *got.RoleMappings[0].Name != "mapping1" {
		t.Fatalf("role mapping = %+v", got.RoleMappings[0])
	}

	book, err := datamodel.NewReferenceBook("b1", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.PutReferenceBook(ctx, book); err != nil {
		t.Fatal(err)
	}
	gotBook, err := s.GetReferenceBook(ctx, "b1")
	if err != nil {
		t.Fatal(err)
	}
	if gotBook.Name != "b1" {
		t.Fatalf("reference book = %+v", gotBook)
	}
	// Put again to check idempotence.
	if err := s.PutReferenceBook(ctx, book); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetReferenceBook(ctx, "b1"); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationSnapshotListAndObjects(t *testing.T) {
	integrationEnabled(t)
	ctx := context.Background()
	s := integrationSession(t)
	defer s.Close()
	newIntegrationNetwork(t, s)

	// Empty network.
	snapshots, err := s.ListSnapshots(ctx, false)
	if err != nil || len(snapshots) != 0 {
		t.Fatalf("snapshots = %v, err = %v", snapshots, err)
	}

	name, err := s.InitSnapshotFromText(ctx, integrationConfig, InitSnapshotFromTextOptions{Platform: "cisco"})
	if err != nil {
		t.Fatal(err)
	}
	snapshots, err = s.ListSnapshots(ctx, false)
	if err != nil || len(snapshots) != 1 {
		t.Fatalf("snapshots = %v, err = %v", snapshots, err)
	}
	verbose, err := s.ListSnapshots(ctx, true)
	if err != nil || len(verbose) != 1 {
		t.Fatalf("verbose snapshots = %v, err = %v", verbose, err)
	}
	meta, ok := verbose[0].(map[string]any)
	if !ok || meta["name"] != name {
		t.Fatalf("verbose snapshot = %v", verbose[0])
	}

	// Snapshot objects.
	if err := s.PutSnapshotObject(ctx, "new_object", []byte("goodbye")); err != nil {
		t.Fatal(err)
	}
	text, err := s.GetSnapshotObjectText(ctx, "new_object", "utf-8", nil)
	if err != nil || text != "goodbye" {
		t.Fatalf("object text = %q, err = %v", text, err)
	}
	if _, err := s.GetSnapshotObjectText(ctx, "missing_object", "utf-8", nil); !isHTTPStatus(err, 404) {
		t.Fatalf("expected 404, got %v", err)
	}
	if err := s.DeleteSnapshotObject(ctx, "new_object", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetSnapshotObjectText(ctx, "new_object", "utf-8", nil); !isHTTPStatus(err, 404) {
		t.Fatalf("expected 404, got %v", err)
	}

	// Network objects.
	if err := s.PutNetworkObject(ctx, "new_object", []byte("goodbye")); err != nil {
		t.Fatal(err)
	}
	text, err = s.GetNetworkObjectText(ctx, "new_object", "utf-8")
	if err != nil || text != "goodbye" {
		t.Fatalf("network object text = %q, err = %v", text, err)
	}
	if err := s.DeleteNetworkObject(ctx, "new_object"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.GetNetworkObjectText(ctx, "new_object", "utf-8"); !isHTTPStatus(err, 404) {
		t.Fatalf("expected 404, got %v", err)
	}
}

func TestIntegrationForkSnapshot(t *testing.T) {
	integrationEnabled(t)
	ctx := context.Background()
	s := integrationSession(t)
	defer s.Close()
	newIntegrationNetwork(t, s)

	base, err := s.InitSnapshotFromText(ctx, integrationConfig, InitSnapshotFromTextOptions{Platform: "cisco"})
	if err != nil {
		t.Fatal(err)
	}
	forkName := util.GetUUID()
	if _, err := s.ForkSnapshot(ctx, base, ForkSnapshotOptions{Name: forkName}); err != nil {
		t.Fatalf("fork: %v", err)
	}
	defer func() { _ = s.DeleteSnapshot(ctx, forkName) }()

	// Forking to an existing name without overwrite must fail.
	if _, err := s.ForkSnapshot(ctx, base, ForkSnapshotOptions{Name: forkName}); err == nil {
		t.Fatal("expected error for existing snapshot")
	}
	// Forking from a bogus base must fail.
	if _, err := s.ForkSnapshot(ctx, "bogus", ForkSnapshotOptions{Name: util.GetUUID()}); !isHTTPStatus(err, 404) {
		t.Fatalf("expected 404, got %v", err)
	}
}

func TestIntegrationGenerateDataplane(t *testing.T) {
	integrationEnabled(t)
	ctx := context.Background()
	s := integrationSession(t)
	defer s.Close()
	newIntegrationNetwork(t, s)

	if _, err := s.InitSnapshotFromText(ctx, integrationConfig, InitSnapshotFromTextOptions{Platform: "cisco"}); err != nil {
		t.Fatal(err)
	}
	status, err := s.GenerateDataplane(ctx, nil, nil)
	if err != nil || status == "" {
		t.Fatalf("dataplane status = %q, err = %v", status, err)
	}
}

func TestIntegrationAutoComplete(t *testing.T) {
	integrationEnabled(t)
	ctx := context.Background()
	s := integrationSession(t)
	defer s.Close()
	newIntegrationNetwork(t, s)

	if _, err := s.InitSnapshotFromText(ctx, integrationConfig, InitSnapshotFromTextOptions{Platform: "cisco"}); err != nil {
		t.Fatal(err)
	}
	for _, completionType := range []datamodel.VariableType{
		datamodel.VariableNodeName,
		datamodel.VariableInterfaceName,
	} {
		suggestions, err := s.AutoComplete(ctx, completionType, ".*", nil)
		if err != nil {
			t.Fatalf("%s: %v", completionType, err)
		}
		for _, suggestion := range suggestions {
			if suggestion.Text == "" {
				t.Fatalf("%s: empty suggestion", completionType)
			}
		}
	}

	max := 1
	suggestions, err := s.AutoComplete(ctx, datamodel.VariableNodeName, ".*", &max)
	if err != nil {
		t.Fatal(err)
	}
	if len(suggestions) > max {
		t.Fatalf("expected at most %d suggestions, got %d", max, len(suggestions))
	}
}

func isHTTPStatus(err error, status int) bool {
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) {
		return false
	}
	return httpErr.StatusCode == status
}
