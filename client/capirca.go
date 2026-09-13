package client

// This file provides the parts of pybatfish.client.capirca that can be
// implemented without the Python-only Capirca library.
//
// pybatfish's capirca module converts Capirca policy files into Batfish
// snapshots and reference books. Capirca itself is a Python library, so
// gobatfish accepts an already-rendered ACL (for InitSnapshotFromACL) or a
// parsed definitions model (for CreateReferenceBookFromDefinitions).

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/81ueman/gobatfish/datamodel"
)

// NetworkDefinition is one Capirca network definition: a name and its items.
type NetworkDefinition struct {
	Name  string
	Items []string
}

// Definitions holds Capirca network definitions.
type Definitions struct {
	Networks map[string]*NetworkDefinition
}

// NewDefinitions creates an empty Definitions.
func NewDefinitions() *Definitions {
	return &Definitions{Networks: make(map[string]*NetworkDefinition)}
}

// AddNetwork adds a network definition.
func (d *Definitions) AddNetwork(name string, items ...string) {
	d.Networks[name] = &NetworkDefinition{Name: name, Items: items}
}

// CreateReferenceBookFromDefinitions creates a ReferenceBook from Capirca
// network definitions, mirroring pybatfish.client.capirca.create_reference_book.
func CreateReferenceBookFromDefinitions(definitions *Definitions, bookName string) (datamodel.ReferenceBook, error) {
	if definitions == nil {
		return datamodel.ReferenceBook{}, fmt.Errorf("definitions must not be nil")
	}
	var groups []datamodel.AddressGroup
	for name, network := range definitions.Networks {
		group, err := entryToGroup(name, network.Items, definitions)
		if err != nil {
			return datamodel.ReferenceBook{}, err
		}
		groups = append(groups, group)
	}
	book, err := datamodel.NewReferenceBook(bookName, groups, nil)
	if err != nil {
		return datamodel.ReferenceBook{}, err
	}
	return book, nil
}

func entryToGroup(name string, items []string, definitions *Definitions) (datamodel.AddressGroup, error) {
	var convertedV4 []string
	var childGroups []string
	for _, item := range items {
		repr, kind, err := itemToPythonRepr(item, definitions)
		if err != nil {
			log.Printf("error converting %s, creating empty group", name)
			return datamodel.AddressGroup{Name: name}, nil
		}
		switch kind {
		case "v4":
			convertedV4 = append(convertedV4, repr)
		case "v6":
			log.Printf("Skipping IPv6 addresses in %s", name)
		case "group":
			childGroups = append(childGroups, repr)
		}
	}
	return datamodel.NewAddressGroup(name, convertedV4, childGroups)
}

// itemToPythonRepr converts a Capirca item into a typed value, returning the
// rendered representation and its kind ("v4", "v6" or "group").
func itemToPythonRepr(item string, definitions *Definitions) (string, string, error) {
	s := strings.TrimSpace(strings.SplitN(item, "#", 2)[0])
	if _, ok := definitions.Networks[s]; ok {
		return s, "group", nil
	}
	if ip := net.ParseIP(s); ip != nil {
		if ip.To4() != nil {
			return ip.String(), "v4", nil
		}
		return ip.String(), "v6", nil
	}
	if _, ipNet, err := net.ParseCIDR(s); err == nil {
		if ipNet.IP.To4() != nil {
			return ipNet.String(), "v4", nil
		}
		return ipNet.String(), "v6", nil
	}
	return "", "", fmt.Errorf("unknown how to convert %s", s)
}

// InitSnapshotFromACL initializes a snapshot containing a single host with the
// given, already-rendered ACL text.
func InitSnapshotFromACL(ctx context.Context, session *Session, aclText, platform string, filename string, snapshotName string, overwrite bool, extraArgs map[string]any) (string, error) {
	return session.InitSnapshotFromText(ctx, aclText, InitSnapshotFromTextOptions{
		Filename:  filename,
		Name:      snapshotName,
		Platform:  platform,
		Overwrite: overwrite,
		ExtraArgs: extraArgs,
	})
}
