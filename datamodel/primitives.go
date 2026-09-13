package datamodel

import (
	"fmt"
	"strings"

	"github.com/81ueman/gobatfish/util"
)

// AssertionType identifies how an assertion is evaluated.
type AssertionType string

// Assertion types, mirroring pybatfish.datamodel.primitives.AssertionType.
const (
	AssertionCountEquals   AssertionType = "countequals"
	AssertionCountLessThan AssertionType = "countlessthan"
	AssertionCountMoreThan AssertionType = "countmorethan"
	AssertionEquals        AssertionType = "equals"
)

// AssertionTypeFromName resolves an assertion type from its Python enum member
// name (e.g. "COUNT_EQUALS"), which is what Batfish returns.
func AssertionTypeFromName(name string) (AssertionType, error) {
	switch name {
	case "COUNT_EQUALS":
		return AssertionCountEquals, nil
	case "COUNT_LESSTHAN":
		return AssertionCountLessThan, nil
	case "COUNT_MORETHAN":
		return AssertionCountMoreThan, nil
	case "EQUALS":
		return AssertionEquals, nil
	default:
		return "", fmt.Errorf("unknown assertion type %q", name)
	}
}

// Name returns the Python enum member name for the assertion type.
func (t AssertionType) Name() string {
	switch t {
	case AssertionCountEquals:
		return "COUNT_EQUALS"
	case AssertionCountLessThan:
		return "COUNT_LESSTHAN"
	case AssertionCountMoreThan:
		return "COUNT_MORETHAN"
	case AssertionEquals:
		return "EQUALS"
	default:
		return string(t)
	}
}

// Assertion is a Batfish assertion, combined with a question to form a check.
type Assertion struct {
	Type   AssertionType
	Expect any
}

// AssertionFromDict builds an Assertion from its dictionary representation.
func AssertionFromDict(d map[string]any) (Assertion, error) {
	t, err := AssertionTypeFromName(strField(d, "type"))
	if err != nil {
		if av, ok := d["type"].(AssertionType); ok {
			return Assertion{Type: av, Expect: d["expect"]}, nil
		}
		if s, ok := d["type"].(string); ok {
			return Assertion{Type: AssertionType(s), Expect: d["expect"]}, nil
		}
		return Assertion{}, err
	}
	return Assertion{Type: t, Expect: d["expect"]}, nil
}

// Dict returns the dictionary representation of the assertion.
func (a Assertion) Dict() map[string]any {
	return map[string]any{"type": string(a.Type), "expect": jsonValue(a.Expect)}
}

// VariableType is the type of a question variable used for auto completion.
type VariableType string

// Auto completion types. Values must be in sync with
// org.batfish.datamodel.questions.Variable.Type.
const (
	VariableAddressGroupName           VariableType = "addressGroupName"
	VariableAnswerElement              VariableType = "answerElement"
	VariableApplicationSpec            VariableType = "applicationSpec"
	VariableBGPPeerPropertySpec        VariableType = "bgpPeerPropertySpec"
	VariableBGPProcessPropertySpec     VariableType = "bgpProcessPropertySpec"
	VariableBGPRoutes                  VariableType = "bgpRoutes"
	VariableBGPRouteConstraints        VariableType = "bgpRouteConstraints"
	VariableBGPRouteStatusSpec         VariableType = "bgpRouteStatusSpec"
	VariableBGPSessionCompatStatusSpec VariableType = "bgpSessionCompatStatusSpec"
	VariableBGPSessionProperties       VariableType = "bgpSessionProperties"
	VariableBGPSessionStatusSpec       VariableType = "bgpSessionStatusSpec"
	VariableBGPSessionTypeSpec         VariableType = "bgpSessionTypeSpec"
	VariableBoolean                    VariableType = "boolean"
	VariableComparator                 VariableType = "comparator"
	VariableDouble                     VariableType = "double"
	VariableDispositionSpec            VariableType = "dispositionSpec"
	VariableFilter                     VariableType = "filter"
	VariableFilterName                 VariableType = "filter"
	VariableFilterSpec                 VariableType = "filterSpec"
	VariableFloat                      VariableType = "float"
	VariableHeaderConstraint           VariableType = "headerConstraint"
	VariableInteger                    VariableType = "integer"
	VariableIntegerSpace               VariableType = "integerSpace"
	VariableInterface                  VariableType = "interface"
	VariableInterfaceGroupName         VariableType = "interfaceGroupName"
	VariableInterfaceName              VariableType = "interfaceName"
	VariableInterfacePropertySpec      VariableType = "interfacePropertySpec"
	VariableInterfacesSpec             VariableType = "interfacesSpec"
	VariableIP                         VariableType = "ip"
	VariableIPProtocol                 VariableType = "ipProtocol"
	VariableIPProtocolSpec             VariableType = "ipProtocolSpec"
	VariableIPSpaceSpec                VariableType = "ipSpaceSpec"
	VariableIPWildcard                 VariableType = "ipWildcard"
	VariableIPsecSessionStatusSpec     VariableType = "ipsecSessionStatusSpec"
	VariableJavaRegex                  VariableType = "javaRegex"
	VariableJSONPath                   VariableType = "jsonPath"
	VariableJSONPathRegex              VariableType = "jsonPathRegex"
	VariableLocationSpec               VariableType = "locationSpec"
	VariableLong                       VariableType = "long"
	VariableMlagID                     VariableType = "mlagId"
	VariableMlagIDSpec                 VariableType = "mlagIdSpec"
	VariableNamedStructureSpec         VariableType = "namedStructureSpec"
	VariableNodeName                   VariableType = "nodeName"
	VariableNodePropertySpec           VariableType = "nodePropertySpec"
	VariableNodeRoleDimensionName      VariableType = "nodeRoleDimensionName"
	VariableNodeRoleName               VariableType = "nodeRoleName"
	VariableNodeSpec                   VariableType = "nodeSpec"
	VariableOSPFInterfacePropertySpec  VariableType = "ospfInterfacePropertySpec"
	VariableOSPFProcessPropertySpec    VariableType = "ospfProcessPropertySpec"
	VariableOSPFSessionStatusSpec      VariableType = "ospfSessionStatusSpec"
	VariablePathConstraint             VariableType = "pathConstraint"
	VariablePrefix                     VariableType = "prefix"
	VariablePrefixRange                VariableType = "prefixRange"
	VariableProtocol                   VariableType = "protocol"
	VariableReferenceBookName          VariableType = "referenceBookName"
	VariableQuestion                   VariableType = "question"
	VariableRoutingPolicySpec          VariableType = "routingPolicySpec"
	VariableRoutingProtocolSpec        VariableType = "routingProtocolSpec"
	VariableString                     VariableType = "string"
	VariableStructureName              VariableType = "structureName"
	VariableSubrange                   VariableType = "subrange"
	VariableVrf                        VariableType = "vrf"
	VariableVxlanVniPropertySpec       VariableType = "vxlanVniPropertySpec"
	VariableZone                       VariableType = "zone"
)

// DefaultAutoCompleteRank is the rank assigned to suggestions without one.
const DefaultAutoCompleteRank = 0x7FFFFFFF

// AutoCompleteSuggestion represents one auto complete suggestion.
type AutoCompleteSuggestion struct {
	Description    *string `json:"description"`
	Hint           *string `json:"hint"`
	InsertionIndex int     `json:"insertion_index"`
	IsPartial      bool    `json:"is_partial"`
	Rank           int     `json:"rank"`
	Text           string  `json:"text"`
}

// AutoCompleteSuggestionFromDict builds a suggestion from a dictionary.
func AutoCompleteSuggestionFromDict(d map[string]any) AutoCompleteSuggestion {
	return AutoCompleteSuggestion{
		Description:    optStrField(d, "description"),
		Hint:           optStrField(d, "hint"),
		InsertionIndex: intField(d, "insertionIndex"),
		IsPartial:      boolField(d, "isPartial"),
		Rank:           intField(d, "rank"),
		Text:           strField(d, "text"),
	}
}

// Dict returns the dictionary representation of the suggestion.
func (a AutoCompleteSuggestion) Dict() map[string]any {
	return map[string]any{
		"description":     jsonValue(a.Description),
		"hint":            jsonValue(a.Hint),
		"insertion_index": a.InsertionIndex,
		"is_partial":      a.IsPartial,
		"rank":            a.Rank,
		"text":            a.Text,
	}
}

// Interface is a network interface: a combination of node and interface names.
type Interface struct {
	Hostname  string `json:"hostname"`
	Interface string `json:"interface"`
}

// InterfaceFromDict builds an Interface from a dictionary.
func InterfaceFromDict(d map[string]any) Interface {
	return Interface{Hostname: strField(d, "hostname"), Interface: strField(d, "interface")}
}

// Dict returns the dictionary representation of the interface.
func (i Interface) Dict() map[string]any { return dictOf(i) }

// String renders the interface, escaping names as needed.
func (i Interface) String() string {
	return fmt.Sprintf("%s[%s]", util.EscapeName(i.Hostname), util.EscapeName(i.Interface))
}

// HTML returns the HTML representation of the interface.
func (i Interface) HTML() string { return util.EscapeHTML(i.String()) }

func interfaceConverter(val any) string {
	if i, ok := val.(Interface); ok {
		return i.Interface
	}
	if p, ok := val.(*Interface); ok && p != nil {
		return p.Interface
	}
	return fmt.Sprint(val)
}

// Edge is a network edge (a link between two node/interface pairs).
type Edge struct {
	Node1          string `json:"node1"`
	Node1Interface string `json:"node1interface"`
	Node2          string `json:"node2"`
	Node2Interface string `json:"node2interface"`
}

// NewEdge builds an Edge, converting Interface values to their interface names.
func NewEdge(node1 string, node1Interface any, node2 string, node2Interface any) Edge {
	return Edge{
		Node1:          node1,
		Node1Interface: interfaceConverter(node1Interface),
		Node2:          node2,
		Node2Interface: interfaceConverter(node2Interface),
	}
}

// EdgeFromDict builds an Edge from a dictionary.
func EdgeFromDict(d map[string]any) Edge {
	return NewEdge(
		strField(d, "node1"),
		strField(d, "node1interface"),
		strField(d, "node2"),
		strField(d, "node2interface"),
	)
}

// Dict returns the dictionary representation of the edge.
func (e Edge) Dict() map[string]any { return dictOf(e) }

// String renders the edge.
func (e Edge) String() string {
	return fmt.Sprintf("%s:%s -> %s:%s", e.Node1, e.Node1Interface, e.Node2, e.Node2Interface)
}

// HTML returns the HTML representation of the edge.
func (e Edge) HTML() string {
	return fmt.Sprintf("%s:%s &rarr; %s:%s", e.Node1, e.Node1Interface, e.Node2, e.Node2Interface)
}

// FileLines represents a set of lines in a file.
type FileLines struct {
	Filename string `json:"filename"`
	Lines    []int  `json:"lines"`
}

// FileLinesFromDict builds FileLines from a dictionary.
func FileLinesFromDict(d map[string]any) FileLines {
	vals := sliceField(d, "lines")
	lines := make([]int, len(vals))
	for i, v := range vals {
		lines[i] = toInt(v)
	}
	return FileLines{Filename: strField(d, "filename"), Lines: lines}
}

// Dict returns the dictionary representation of the file lines.
func (f FileLines) Dict() map[string]any { return dictOf(f) }

// String renders the file lines.
func (f FileLines) String() string { return fmt.Sprintf("%s:%v", f.Filename, f.Lines) }

// ListWrapper is a list of values. pybatfish's ListWrapper is immutable; Go
// slices cannot enforce that, but NewListWrapper copies its input.
type ListWrapper []any

// NewListWrapper creates a ListWrapper holding a copy of values.
func NewListWrapper(values []any) ListWrapper {
	out := make(ListWrapper, len(values))
	copy(out, values)
	return out
}

// String renders the list.
func (l ListWrapper) String() string {
	parts := make([]string, len(l))
	for i, v := range l {
		parts[i] = fmt.Sprint(v)
	}
	return "[" + strings.Join(parts, " ") + "]"
}

// HTML returns the HTML representation of the list.
func (l ListWrapper) HTML() string {
	parts := make([]string, len(l))
	for i, v := range l {
		parts[i] = util.GetHTML(v)
	}
	return strings.Join(parts, "<br><br>")
}
