package datamodel

import "fmt"

// AddressGroup contains information about an address group.
type AddressGroup struct {
	Name            string   `json:"name"`
	Addresses       []string `json:"addresses"`
	ChildGroupNames []string `json:"childGroupNames"`
}

// NewAddressGroup builds an address group, normalizing its list fields the way
// pybatfish's converters do. It returns an error if an element has the wrong
// type.
func NewAddressGroup(name string, addresses, childGroupNames any) (AddressGroup, error) {
	addrs, err := makeStringList(addresses)
	if err != nil {
		return AddressGroup{}, err
	}
	children, err := makeStringList(childGroupNames)
	if err != nil {
		return AddressGroup{}, err
	}
	return AddressGroup{Name: name, Addresses: addrs, ChildGroupNames: children}, nil
}

// AddressGroupFromDict builds an AddressGroup from a dictionary.
func AddressGroupFromDict(d map[string]any) AddressGroup {
	return AddressGroup{
		Name:            strField(d, "name"),
		Addresses:       stringSliceField(d, "addresses"),
		ChildGroupNames: stringSliceField(d, "childGroupNames"),
	}
}

// Dict returns the dictionary representation of the group.
func (a AddressGroup) Dict() map[string]any { return dictOf(a) }

// InterfaceGroup contains information about an interface group.
type InterfaceGroup struct {
	Name       string      `json:"name"`
	Interfaces []Interface `json:"interfaces"`
}

// NewInterfaceGroup builds an interface group, normalizing its list field.
func NewInterfaceGroup(name string, interfaces any) (InterfaceGroup, error) {
	items, err := makeTypedList(interfaces, func(v any) (any, bool) {
		if i, ok := v.(Interface); ok {
			return i, true
		}
		return nil, false
	})
	if err != nil {
		return InterfaceGroup{}, err
	}
	out := make([]Interface, len(items))
	for i, it := range items {
		out[i] = it.(Interface)
	}
	return InterfaceGroup{Name: name, Interfaces: out}, nil
}

// InterfaceGroupFromDict builds an InterfaceGroup from a dictionary.
func InterfaceGroupFromDict(d map[string]any) InterfaceGroup {
	vals := sliceField(d, "interfaces")
	out := make([]Interface, 0, len(vals))
	for _, v := range vals {
		out = append(out, InterfaceFromDict(toMap(v)))
	}
	return InterfaceGroup{Name: strField(d, "name"), Interfaces: out}
}

// Dict returns the dictionary representation of the group.
func (g InterfaceGroup) Dict() map[string]any { return dictOf(g) }

// RoleMapping is a mapping from node name to role dimensions.
type RoleMapping struct {
	Name                *string                      `json:"name"`
	Regex               string                       `json:"regex"`
	RoleDimensionGroups map[string][]int             `json:"roleDimensionGroups"`
	CanonicalRoleNames  map[string]map[string]string `json:"canonicalRoleNames"`
}

// RoleMappingFromDict builds a RoleMapping from a dictionary.
func RoleMappingFromDict(d map[string]any) RoleMapping {
	return RoleMapping{
		Name:                optStrField(d, "name"),
		Regex:               strField(d, "regex"),
		RoleDimensionGroups: intListMapField(d, "roleDimensionGroups"),
		CanonicalRoleNames:  stringMapMapField(d, "canonicalRoleNames"),
	}
}

// Dict returns the dictionary representation of the mapping.
func (r RoleMapping) Dict() map[string]any { return dictOf(r) }

// NodeRolesData contains node roles definitions.
type NodeRolesData struct {
	DefaultDimension   *string       `json:"defaultDimension"`
	RoleDimensionOrder []string      `json:"roleDimensionOrder"`
	RoleMappings       []RoleMapping `json:"roleMappings"`
}

// NewNodeRolesData builds node roles data, normalizing list fields.
func NewNodeRolesData(defaultDimension any, roleDimensionOrder, roleMappings any) (NodeRolesData, error) {
	var def *string
	if defaultDimension != nil {
		s, ok := defaultDimension.(string)
		if !ok {
			return NodeRolesData{}, fmt.Errorf("invalid default dimension %v", defaultDimension)
		}
		def = &s
	}
	order, err := makeStringList(roleDimensionOrder)
	if err != nil {
		return NodeRolesData{}, err
	}
	items, err := makeTypedList(roleMappings, func(v any) (any, bool) {
		if m, ok := v.(RoleMapping); ok {
			return m, true
		}
		return nil, false
	})
	if err != nil {
		return NodeRolesData{}, err
	}
	mappings := make([]RoleMapping, len(items))
	for i, it := range items {
		mappings[i] = it.(RoleMapping)
	}
	return NodeRolesData{DefaultDimension: def, RoleDimensionOrder: order, RoleMappings: mappings}, nil
}

// NodeRolesDataFromDict builds NodeRolesData from a dictionary.
func NodeRolesDataFromDict(d map[string]any) NodeRolesData {
	order := stringSliceField(d, "roleDimensionOrder")
	if order == nil {
		order = []string{}
	}
	vals := sliceField(d, "roleMappings")
	mappings := make([]RoleMapping, 0, len(vals))
	for _, v := range vals {
		mappings = append(mappings, RoleMappingFromDict(toMap(v)))
	}
	return NodeRolesData{
		DefaultDimension:   optStrField(d, "defaultDimension"),
		RoleDimensionOrder: order,
		RoleMappings:       mappings,
	}
}

// Dict returns the dictionary representation of the node roles data.
func (n NodeRolesData) Dict() map[string]any { return dictOf(n) }

// ReferenceBook contains a named collection of references.
type ReferenceBook struct {
	Name            string           `json:"name"`
	AddressGroups   []AddressGroup   `json:"addressGroups"`
	InterfaceGroups []InterfaceGroup `json:"interfaceGroups"`
}

// NewReferenceBook builds a reference book, normalizing list fields.
func NewReferenceBook(name string, addressGroups, interfaceGroups any) (ReferenceBook, error) {
	ags, err := makeTypedList(addressGroups, func(v any) (any, bool) {
		if g, ok := v.(AddressGroup); ok {
			return g, true
		}
		return nil, false
	})
	if err != nil {
		return ReferenceBook{}, err
	}
	igs, err := makeTypedList(interfaceGroups, func(v any) (any, bool) {
		if g, ok := v.(InterfaceGroup); ok {
			return g, true
		}
		return nil, false
	})
	if err != nil {
		return ReferenceBook{}, err
	}
	book := ReferenceBook{Name: name}
	for _, v := range ags {
		book.AddressGroups = append(book.AddressGroups, v.(AddressGroup))
	}
	for _, v := range igs {
		book.InterfaceGroups = append(book.InterfaceGroups, v.(InterfaceGroup))
	}
	return book, nil
}

// ReferenceBookFromDict builds a ReferenceBook from a dictionary.
func ReferenceBookFromDict(d map[string]any) ReferenceBook {
	ags := sliceField(d, "addressGroups")
	addressGroups := make([]AddressGroup, 0, len(ags))
	for _, v := range ags {
		addressGroups = append(addressGroups, AddressGroupFromDict(toMap(v)))
	}
	igs := sliceField(d, "interfaceGroups")
	interfaceGroups := make([]InterfaceGroup, 0, len(igs))
	for _, v := range igs {
		interfaceGroups = append(interfaceGroups, InterfaceGroupFromDict(toMap(v)))
	}
	return ReferenceBook{Name: strField(d, "name"), AddressGroups: addressGroups, InterfaceGroups: interfaceGroups}
}

// Dict returns the dictionary representation of the book.
func (b ReferenceBook) Dict() map[string]any { return dictOf(b) }

// ReferenceLibrary contains a list of reference books.
type ReferenceLibrary struct {
	Books []ReferenceBook `json:"books"`
}

// NewReferenceLibrary builds a reference library, normalizing the books field.
func NewReferenceLibrary(books any) (ReferenceLibrary, error) {
	items, err := makeTypedList(books, func(v any) (any, bool) {
		if b, ok := v.(ReferenceBook); ok {
			return b, true
		}
		return nil, false
	})
	if err != nil {
		return ReferenceLibrary{}, err
	}
	out := make([]ReferenceBook, len(items))
	for i, it := range items {
		out[i] = it.(ReferenceBook)
	}
	return ReferenceLibrary{Books: out}, nil
}

// ReferenceLibraryFromDict builds a ReferenceLibrary from a dictionary.
func ReferenceLibraryFromDict(d map[string]any) ReferenceLibrary {
	vals := sliceField(d, "books")
	books := make([]ReferenceBook, 0, len(vals))
	for _, v := range vals {
		books = append(books, ReferenceBookFromDict(toMap(v)))
	}
	return ReferenceLibrary{Books: books}
}

// Dict returns the dictionary representation of the library.
func (l ReferenceLibrary) Dict() map[string]any { return dictOf(l) }

// --- list normalization helpers ---------------------------------------------

func makeStringList(value any) ([]string, error) {
	if value == nil {
		return []string{}, nil
	}
	if s, ok := value.(string); ok {
		return []string{s}, nil
	}
	list, err := makeTypedList(value, func(v any) (any, bool) {
		if s, ok := v.(string); ok {
			return s, true
		}
		return nil, false
	})
	if err != nil {
		return nil, err
	}
	out := make([]string, len(list))
	for i, v := range list {
		out[i] = v.(string)
	}
	return out, nil
}

// makeTypedList normalizes value: nil becomes empty, a single item is wrapped,
// and list/array elements are checked against the predicate.
func makeTypedList(value any, ok func(any) (any, bool)) ([]any, error) {
	if value == nil {
		return []any{}, nil
	}
	if list, isList := asList(value); isList {
		out := make([]any, len(list))
		for i, item := range list {
			converted, valid := ok(item)
			if !valid {
				return nil, fmt.Errorf("invalid value '%v' of type %T", item, item)
			}
			out[i] = converted
		}
		return out, nil
	}
	converted, valid := ok(value)
	if !valid {
		return nil, fmt.Errorf("invalid value '%v' of type %T", value, value)
	}
	return []any{converted}, nil
}

func intListMapField(m map[string]any, key string) map[string][]int {
	raw := mapField(m, key)
	if raw == nil {
		return map[string][]int{}
	}
	out := make(map[string][]int, len(raw))
	for k, v := range raw {
		vals := sliceField(map[string]any{"v": v}, "v")
		ints := make([]int, len(vals))
		for i, item := range vals {
			ints[i] = toInt(item)
		}
		out[k] = ints
	}
	return out
}

func stringMapMapField(m map[string]any, key string) map[string]map[string]string {
	raw := mapField(m, key)
	if raw == nil {
		return map[string]map[string]string{}
	}
	out := make(map[string]map[string]string, len(raw))
	for k, v := range raw {
		inner := mapField(map[string]any{"v": v}, "v")
		converted := make(map[string]string, len(inner))
		for ik, iv := range inner {
			converted[ik] = fmt.Sprint(iv)
		}
		out[k] = converted
	}
	return out
}
