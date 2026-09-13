package client

import (
	"context"
	"testing"

	"github.com/81ueman/gobatfish/util"
)

func TestNetworkValidation(t *testing.T) {
	s := NewSession(SessionConfig{})
	if _, err := s.SetNetwork(context.Background(), "foo/bar"); err == nil {
		t.Fatal("expected error for invalid network name")
	}
}

func TestSnapshotValidation(t *testing.T) {
	s := NewSession(SessionConfig{})
	s.Network = util.StringPtr("testnet")
	if _, err := s.InitSnapshot(context.Background(), "x", InitSnapshotOptions{Name: "foo/bar"}); err == nil {
		t.Fatal("expected error for invalid snapshot name")
	}
}

func TestForkSnapshotValidation(t *testing.T) {
	s := NewSession(SessionConfig{})
	if _, err := s.ForkSnapshot(context.Background(), "x", ForkSnapshotOptions{Name: "foo/bar"}); err == nil {
		t.Fatal("expected error when network is not set")
	}
}
