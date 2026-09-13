package client

import (
	"bytes"
	"log"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func capircaDefinitions() *Definitions {
	d := NewDefinitions()
	add := func(name string, items ...string) {
		d.Networks[name] = &NetworkDefinition{Name: name, Items: items}
	}
	add("HOST_BITS", "1.2.3.4/8")
	add("RFC1918_10", "10.0.0.0/8")
	add("RFC1918_172", "172.16.0.0/12")
	add("RFC1918_192", "192.168.0.0/16")
	add("RFC1918", "RFC1918_10", "RFC1918_172", "RFC1918_192")
	add("LOOPBACK", "127.0.0.0/8", "::1/128")
	add("MULTICAST", "224.0.0.0/4", "FF00::/8")
	add("CLASS-E", "240.0.0.0/4")
	add("DENY-EXTERNAL-SRC", "RFC1918", "LOOPBACK", "RFC3330", "MULTICAST", "CLASS-E", "UNDEFINED")
	return d
}

func getCapircaGroup(t *testing.T, defs *Definitions, name string) (group interface {
	Dict() map[string]any
}, _ any) {
	t.Helper()
	_ = name
	return nil, nil
}

func TestEntryToGroupNaive(t *testing.T) {
	defs := capircaDefinitions()
	for name, want := range map[string]string{
		"RFC1918_10":  "10.0.0.0/8",
		"RFC1918_172": "172.16.0.0/12",
		"RFC1918_192": "192.168.0.0/16",
	} {
		g, err := entryToGroup(name, defs.Networks[name].Items, defs)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(g.Addresses, []string{want}) || len(g.ChildGroupNames) != 0 {
			t.Fatalf("%s = %+v", name, g)
		}
	}
}

func TestEntryToGroupHostBits(t *testing.T) {
	defs := capircaDefinitions()
	g, err := entryToGroup("HOST_BITS", defs.Networks["HOST_BITS"].Items, defs)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.Addresses, []string{"1.0.0.0/8"}) {
		t.Fatalf("addresses = %v", g.Addresses)
	}
}

func TestEntryToGroupRecursive(t *testing.T) {
	defs := capircaDefinitions()
	g, err := entryToGroup("RFC1918", defs.Networks["RFC1918"].Items, defs)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Addresses) != 0 {
		t.Fatalf("addresses = %v", g.Addresses)
	}
	for _, want := range []string{"RFC1918_10", "RFC1918_172", "RFC1918_192"} {
		if !slices.Contains(g.ChildGroupNames, want) {
			t.Fatalf("children = %v missing %s", g.ChildGroupNames, want)
		}
	}
}

func TestEntryToGroupMixed64(t *testing.T) {
	defs := capircaDefinitions()
	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(old)

	g, err := entryToGroup("LOOPBACK", defs.Networks["LOOPBACK"].Items, defs)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(g.Addresses, []string{"127.0.0.0/8"}) || len(g.ChildGroupNames) != 0 {
		t.Fatalf("group = %+v", g)
	}
	if !strings.Contains(buf.String(), "Skipping IPv6 addresses in LOOPBACK") {
		t.Fatalf("log = %q", buf.String())
	}
}

func TestEntryToGroupErrorUndefined(t *testing.T) {
	defs := capircaDefinitions()
	var buf bytes.Buffer
	old := log.Writer()
	log.SetOutput(&buf)
	defer log.SetOutput(old)

	g, err := entryToGroup("DENY-EXTERNAL-SRC", defs.Networks["DENY-EXTERNAL-SRC"].Items, defs)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Addresses) != 0 || len(g.ChildGroupNames) != 0 {
		t.Fatalf("group = %+v", g)
	}
	if !strings.Contains(buf.String(), "error converting DENY-EXTERNAL-SRC, creating empty group") {
		t.Fatalf("log = %q", buf.String())
	}
}

func TestCreateReferenceBook(t *testing.T) {
	defs := NewDefinitions()
	defs.AddNetwork("RFC1918_10", "10.0.0.0/8")
	defs.AddNetwork("RFC1918_172", "172.16.0.0/12")
	defs.AddNetwork("RFC1918_192", "192.168.0.0/16")
	defs.AddNetwork("RFC1918", "RFC1918_10", "RFC1918_172", "RFC1918_192")

	book, err := CreateReferenceBookFromDefinitions(defs, "capirca")
	if err != nil {
		t.Fatal(err)
	}
	if book.Name != "capirca" || len(book.AddressGroups) != 4 || len(book.InterfaceGroups) != 0 {
		t.Fatalf("book = %+v", book)
	}
	names := map[string]bool{}
	for _, g := range book.AddressGroups {
		names[g.Name] = true
	}
	for _, want := range []string{"RFC1918", "RFC1918_10", "RFC1918_172", "RFC1918_192"} {
		if !names[want] {
			t.Fatalf("missing group %s in %v", want, names)
		}
	}

	custom, err := CreateReferenceBookFromDefinitions(defs, "testbook")
	if err != nil {
		t.Fatal(err)
	}
	if custom.Name != "testbook" || !reflect.DeepEqual(custom.AddressGroups, book.AddressGroups) {
		t.Fatalf("custom book = %+v", custom)
	}
}
