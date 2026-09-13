package datamodel

import "testing"

// Ports of tests/datamodel/test_referencelibrary.py.

func TestAddressgroupConstructionEmpty(t *testing.T) {
	emptyGroup, err := NewAddressGroup("g1", []any{}, []any{})
	assertDeepEqual(t, err, nil)

	g, err := NewAddressGroup("g1", nil, nil)
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, g, emptyGroup)
}

func TestAddressgroupConstructionBadtype(t *testing.T) {
	if _, err := NewAddressGroup("g1", AddressGroup{Name: "g1"}, nil); err == nil {
		t.Fatal("expected ValueError for addresses")
	}
	if _, err := NewAddressGroup("g1", nil, AddressGroup{Name: "g1"}); err == nil {
		t.Fatal("expected ValueError for childGroupNames")
	}
	if _, err := NewAddressGroup("book1", []any{"ag", AddressGroup{Name: "ag1"}}, nil); err == nil {
		t.Fatal("expected ValueError for addresses list")
	}
	if _, err := NewAddressGroup("book1", nil, []any{"ag", AddressGroup{Name: "ag1"}}); err == nil {
		t.Fatal("expected ValueError for childGroupNames list")
	}
}

func TestAddressgroupConstructionItem(t *testing.T) {
	a, err := NewAddressGroup("g1", "ag", nil)
	assertDeepEqual(t, err, nil)
	b, err := NewAddressGroup("g1", []any{"ag"}, nil)
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, a, b)

	c, err := NewAddressGroup("g1", nil, "ag")
	assertDeepEqual(t, err, nil)
	d, err := NewAddressGroup("g1", nil, []any{"ag"})
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, c, d)
}

func TestAddressgroupConstructionList(t *testing.T) {
	group, err := NewAddressGroup("g1", []any{"ag"}, []any{"cg"})
	assertDeepEqual(t, err, nil)

	assertStr(t, group.Name, "g1")
	assertDeepEqual(t, group.Addresses, []string{"ag"})
	assertDeepEqual(t, group.ChildGroupNames, []string{"cg"})
}

func TestAddressgroupDeserBothSubfields(t *testing.T) {
	dict := map[string]any{
		"name":            "ag1",
		"addresses":       []any{"1.1.1.1/24", "2.2.2.2"},
		"childGroupNames": []any{"child1", "child2"},
	}

	addressGroup := AddressGroupFromDict(dict)

	if len(addressGroup.Addresses) != 2 {
		t.Fatalf("addresses = %v", addressGroup.Addresses)
	}
	if len(addressGroup.ChildGroupNames) != 2 {
		t.Fatalf("childGroupNames = %v", addressGroup.ChildGroupNames)
	}
}

func TestAddressgroupDeserNoneSubfields(t *testing.T) {
	dict := map[string]any{"name": "ag1"}

	addressGroup := AddressGroupFromDict(dict)

	if len(addressGroup.Addresses) != 0 {
		t.Fatalf("addresses = %v", addressGroup.Addresses)
	}
	if len(addressGroup.ChildGroupNames) != 0 {
		t.Fatalf("childGroupNames = %v", addressGroup.ChildGroupNames)
	}
}

func TestAddressgroupDeserOnlyAddresses(t *testing.T) {
	dict := map[string]any{"name": "ag1", "addresses": []any{"1.1.1.1/24", "2.2.2.2"}}

	addressGroup := AddressGroupFromDict(dict)

	if len(addressGroup.Addresses) != 2 {
		t.Fatalf("addresses = %v", addressGroup.Addresses)
	}
	if len(addressGroup.ChildGroupNames) != 0 {
		t.Fatalf("childGroupNames = %v", addressGroup.ChildGroupNames)
	}
}

func TestAddressgroupDeserOnlyChildGroups(t *testing.T) {
	dict := map[string]any{"name": "ag1", "childGroupNames": []any{"child1", "child2"}}

	addressGroup := AddressGroupFromDict(dict)

	if len(addressGroup.Addresses) != 0 {
		t.Fatalf("addresses = %v", addressGroup.Addresses)
	}
	if len(addressGroup.ChildGroupNames) != 2 {
		t.Fatalf("childGroupNames = %v", addressGroup.ChildGroupNames)
	}
}

func TestInterfacegroupConstructionEmpty(t *testing.T) {
	emptyGroup, err := NewInterfaceGroup("g1", []any{})
	assertDeepEqual(t, err, nil)

	g, err := NewInterfaceGroup("g1", nil)
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, g, emptyGroup)
}

func TestInterfacegroupConstructionBadtype(t *testing.T) {
	if _, err := NewInterfaceGroup("g1", "i1"); err == nil {
		t.Fatal("expected ValueError for string interfaces")
	}
	if _, err := NewInterfaceGroup("book1", []any{"ag", Interface{Hostname: "h1", Interface: "i1"}}); err == nil {
		t.Fatal("expected ValueError for mixed interfaces")
	}
}

func TestInterfacegroupConstructionItem(t *testing.T) {
	iface := Interface{Hostname: "h1", Interface: "i1"}
	interfaceGroup, err := NewInterfaceGroup("g1", []any{iface})
	assertDeepEqual(t, err, nil)
	actual, err := NewInterfaceGroup("g1", iface)
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, actual, interfaceGroup)
}

func TestInterfacegroupConstructionList(t *testing.T) {
	iface := Interface{Hostname: "h1", Interface: "i1"}
	group, err := NewInterfaceGroup("g1", []any{iface})
	assertDeepEqual(t, err, nil)

	assertStr(t, group.Name, "g1")
	assertDeepEqual(t, group.Interfaces, []Interface{iface})
}

func TestNoderolesdataConstructionEmpty(t *testing.T) {
	empty, err := NewNodeRolesData(nil, []any{}, []any{})
	assertDeepEqual(t, err, nil)

	a, err := NewNodeRolesData(nil, nil, nil)
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, a, empty)

	b, err := NewNodeRolesData(nil, []any{}, nil)
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, b, empty)

	c, err := NewNodeRolesData(nil, []any{}, []any{})
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, c, empty)
}

func TestNoderolesdataConstructionBadtype(t *testing.T) {
	if _, err := NewNodeRolesData("", []any{1}, []any{}); err == nil {
		t.Fatal("expected ValueError for roleDimensionOrder")
	}
	if _, err := NewNodeRolesData("", []any{}, []any{1}); err == nil {
		t.Fatal("expected ValueError for roleMappings")
	}
}

func TestNoderolesdataConstructionItem(t *testing.T) {
	dimension := "dim"
	expected, err := NewNodeRolesData(nil, []any{dimension}, []any{})
	assertDeepEqual(t, err, nil)
	actual, err := NewNodeRolesData(nil, dimension, []any{})
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, actual, expected)

	mapping := RoleMapping{}
	expected, err = NewNodeRolesData(nil, []any{}, []any{mapping})
	assertDeepEqual(t, err, nil)
	actual, err = NewNodeRolesData(nil, []any{}, mapping)
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, actual, expected)
}

func TestNoderolesdataConstructionList(t *testing.T) {
	dimension := "dim"
	data, err := NewNodeRolesData(nil, []any{dimension}, []any{})
	assertDeepEqual(t, err, nil)

	assertDeepEqual(t, data.RoleDimensionOrder, []string{dimension})

	mapping := RoleMapping{}
	data, err = NewNodeRolesData(nil, []any{}, []any{mapping})
	assertDeepEqual(t, err, nil)

	assertDeepEqual(t, data.RoleMappings, []RoleMapping{mapping})
}

func TestReferencebookConstructionEmpty(t *testing.T) {
	emptyBook, err := NewReferenceBook("b1", []any{}, []any{})
	assertDeepEqual(t, err, nil)

	book, err := NewReferenceBook("b1", nil, nil)
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, book, emptyBook)
}

func TestReferencebookConstructionAddressgroupBadtype(t *testing.T) {
	if _, err := NewReferenceBook("book1", "ag", nil); err == nil {
		t.Fatal("expected ValueError for string addressGroups")
	}
	if _, err := NewReferenceBook("book1", []any{"ag"}, nil); err == nil {
		t.Fatal("expected ValueError for string addressGroups list")
	}
	if _, err := NewReferenceBook("book1", []any{"ag", AddressGroup{Name: "ag1"}}, nil); err == nil {
		t.Fatal("expected ValueError for mixed addressGroups list")
	}
}

func TestReferencebookConstructionAddressgroupItem(t *testing.T) {
	refBook, err := NewReferenceBook("book1", AddressGroup{Name: "ag"}, nil)
	assertDeepEqual(t, err, nil)

	assertStr(t, refBook.Name, "book1")
	assertDeepEqual(t, refBook.AddressGroups, []AddressGroup{{Name: "ag"}})
}

func TestReferencebookConstructionAddressgroupList(t *testing.T) {
	refBook, err := NewReferenceBook("book1", []any{AddressGroup{Name: "ag"}}, nil)
	assertDeepEqual(t, err, nil)

	assertStr(t, refBook.Name, "book1")
	assertDeepEqual(t, refBook.AddressGroups, []AddressGroup{{Name: "ag"}})
}

func TestReferencebookConstructionInterfacegroupBadtype(t *testing.T) {
	if _, err := NewReferenceBook("book1", nil, "g"); err == nil {
		t.Fatal("expected ValueError for string interfaceGroups")
	}
	if _, err := NewReferenceBook("book1", nil, []any{"g"}); err == nil {
		t.Fatal("expected ValueError for string interfaceGroups list")
	}
	if _, err := NewReferenceBook("book1", nil, []any{"g", InterfaceGroup{Name: "g1"}}); err == nil {
		t.Fatal("expected ValueError for mixed interfaceGroups list")
	}
}

func TestReferencebookConstructionInterfacegroupItem(t *testing.T) {
	refBook, err := NewReferenceBook("book1", nil, InterfaceGroup{Name: "g"})
	assertDeepEqual(t, err, nil)

	assertStr(t, refBook.Name, "book1")
	assertDeepEqual(t, refBook.InterfaceGroups, []InterfaceGroup{{Name: "g"}})
}

func TestReferencebookConstructionInterfacegroupList(t *testing.T) {
	refBook, err := NewReferenceBook("book1", nil, []any{InterfaceGroup{Name: "g"}})
	assertDeepEqual(t, err, nil)

	assertStr(t, refBook.Name, "book1")
	assertDeepEqual(t, refBook.InterfaceGroups, []InterfaceGroup{{Name: "g"}})
}

func TestReferencelibraryConstructionEmpty(t *testing.T) {
	empty, err := NewReferenceLibrary([]any{})
	assertDeepEqual(t, err, nil)

	library, err := NewReferenceLibrary(nil)
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, library, empty)
}

func TestReferencelibraryConstructionBadtype(t *testing.T) {
	if _, err := NewReferenceLibrary("i1"); err == nil {
		t.Fatal("expected ValueError for string books")
	}
	if _, err := NewReferenceLibrary([]any{"ag", ReferenceBook{Name: "a"}}); err == nil {
		t.Fatal("expected ValueError for mixed books")
	}
}

func TestReferencelibraryConstructionItem(t *testing.T) {
	book := ReferenceBook{Name: "a"}
	library, err := NewReferenceLibrary([]any{book})
	assertDeepEqual(t, err, nil)
	actual, err := NewReferenceLibrary(book)
	assertDeepEqual(t, err, nil)
	assertDeepEqual(t, actual, library)
}

func TestReferencelibraryConstructionList(t *testing.T) {
	book := ReferenceBook{Name: "a"}
	library, err := NewReferenceLibrary([]any{book})
	assertDeepEqual(t, err, nil)

	assertDeepEqual(t, library.Books, []ReferenceBook{book})
}

func TestReferencelibraryDeserEmpty(t *testing.T) {
	referenceLibrary := ReferenceLibraryFromDict(map[string]any{})
	if len(referenceLibrary.Books) != 0 {
		t.Fatalf("books = %v", referenceLibrary.Books)
	}

	referenceLibrary = ReferenceLibraryFromDict(map[string]any{"books": []any{}})
	if len(referenceLibrary.Books) != 0 {
		t.Fatalf("books = %v", referenceLibrary.Books)
	}
}

func TestReferencelibraryDeserAddressgroups(t *testing.T) {
	dict := map[string]any{
		"books": []any{
			map[string]any{
				"name": "book1",
				"addressGroups": []any{
					map[string]any{
						"name":      "ag1",
						"addresses": []any{"1.1.1.1/24", "2.2.2.2", "3.3.3.3:0.0.0.8"},
					},
					map[string]any{"name": "ag2"},
				},
			},
			map[string]any{"name": "book2"},
		},
	}
	referenceLibrary := ReferenceLibraryFromDict(dict)

	if len(referenceLibrary.Books) != 2 {
		t.Fatalf("books = %d", len(referenceLibrary.Books))
	}
	assertStr(t, referenceLibrary.Books[0].Name, "book1")
	if len(referenceLibrary.Books[0].AddressGroups) != 2 {
		t.Fatalf("addressGroups = %d", len(referenceLibrary.Books[0].AddressGroups))
	}
	assertStr(t, referenceLibrary.Books[0].AddressGroups[0].Name, "ag1")
	if len(referenceLibrary.Books[0].AddressGroups[0].Addresses) != 3 {
		t.Fatalf("addresses = %d", len(referenceLibrary.Books[0].AddressGroups[0].Addresses))
	}
}

func TestReferencelibraryDeserInterfacegroups(t *testing.T) {
	dict := map[string]any{
		"books": []any{
			map[string]any{
				"name": "book1",
				"interfaceGroups": []any{
					map[string]any{
						"name": "g1",
						"interfaces": []any{
							map[string]any{"hostname": "h1", "interface": "i1"},
							map[string]any{"hostname": "h2", "interface": "i2"},
						},
					},
					map[string]any{"name": "g2"},
				},
			},
			map[string]any{"name": "book2"},
		},
	}
	referenceLibrary := ReferenceLibraryFromDict(dict)

	if len(referenceLibrary.Books) != 2 {
		t.Fatalf("books = %d", len(referenceLibrary.Books))
	}
	assertStr(t, referenceLibrary.Books[0].Name, "book1")
	if len(referenceLibrary.Books[0].InterfaceGroups) != 2 {
		t.Fatalf("interfaceGroups = %d", len(referenceLibrary.Books[0].InterfaceGroups))
	}
	assertStr(t, referenceLibrary.Books[0].InterfaceGroups[0].Name, "g1")
	if len(referenceLibrary.Books[0].InterfaceGroups[0].Interfaces) != 2 {
		t.Fatalf("interfaces = %d", len(referenceLibrary.Books[0].InterfaceGroups[0].Interfaces))
	}
}

func TestNoderolesdata(t *testing.T) {
	dict := map[string]any{"roleDimensionOrder": []any{"dim1", "dim2"}, "roleMappings": []any{}}
	nodeRoleData := NodeRolesDataFromDict(dict)

	if nodeRoleData.DefaultDimension != nil {
		t.Fatalf("defaultDimension = %v", nodeRoleData.DefaultDimension)
	}
	assertDeepEqual(t, nodeRoleData.RoleDimensionOrder, []string{"dim1", "dim2"})
	if len(nodeRoleData.RoleMappings) != 0 {
		t.Fatalf("roleMappings = %v", nodeRoleData.RoleMappings)
	}
}

func TestRolemapping(t *testing.T) {
	regex := "re"
	roleDimensionGroups := map[string]any{"dim1": []any{1}, "dim2": []any{1, 2}}
	canonicalRoleNames := map[string]any{"dim1": map[string]any{"a": "A"}, "dim2": map[string]any{"a-b": "AB"}}
	dict := map[string]any{
		"regex":               regex,
		"roleDimensionGroups": roleDimensionGroups,
		"canonicalRoleNames":  canonicalRoleNames,
	}
	roleMapping := RoleMappingFromDict(dict)
	if roleMapping.Name != nil {
		t.Fatalf("name = %v", roleMapping.Name)
	}
	assertDeepEqual(t, roleMapping.RoleDimensionGroups, map[string][]int{"dim1": {1}, "dim2": {1, 2}})
	assertDeepEqual(t, roleMapping.CanonicalRoleNames, map[string]map[string]string{"dim1": {"a": "A"}, "dim2": {"a-b": "AB"}})

	dict["name"] = "name"
	roleMapping = RoleMappingFromDict(dict)
	assertStr(t, *roleMapping.Name, "name")
}
