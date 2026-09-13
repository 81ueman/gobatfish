package datamodel

import (
	"fmt"
	"strings"
	"testing"
)

// Ports of tests/datamodel/test_acltrace.py, test_traceelement.py,
// test_traceevent.py and test_flowtracehop.py.

const (
	textFragmentClass = "org.batfish.datamodel.TraceElement$TextFragment"
	linkFragmentClass = "org.batfish.datamodel.TraceElement$LinkFragment"
)

// --- test_acltrace.py -------------------------------------------------------

// test if an acl trace is deserialized properly
func TestAclTraceDeserialization(t *testing.T) {
	traceDict := map[string]any{"events": []any{
		map[string]any{"description": "aa"},
		map[string]any{"description": "bb"},
	}}

	// check deserialization
	aclTrace := AclTraceFromDict(traceDict)
	if len(aclTrace.Events) != 2 {
		t.Fatalf("events = %d", len(aclTrace.Events))
	}

	// check stringification works
	_ = aclTrace.String()
}

func TestTraceTreeNoChildren(t *testing.T) {
	traceTreeDict := map[string]any{"traceElement": map[string]any{
		"fragments": []any{map[string]any{"class": textFragmentClass, "text": "aaa"}},
	}}
	traceTree := TraceTreeFromDict(traceTreeDict)
	if len(traceTree.Children) != 0 {
		t.Fatalf("children = %d", len(traceTree.Children))
	}
	assertStr(t, traceTree.String(), "aaa")
	assertStr(t, traceTree.HTML(), "aaa")
}

func TestTraceTreeWithChildren(t *testing.T) {
	traceTreeDict := map[string]any{
		"traceElement": map[string]any{"fragments": []any{map[string]any{"class": textFragmentClass, "text": "aaa"}}},
		"children": []any{
			map[string]any{"traceElement": map[string]any{"fragments": []any{map[string]any{"class": textFragmentClass, "text": "bbb"}}}},
			map[string]any{"traceElement": map[string]any{"fragments": []any{map[string]any{"class": textFragmentClass, "text": "ccc"}}}},
		},
	}
	traceTree := TraceTreeFromDict(traceTreeDict)
	if len(traceTree.Children) != 2 {
		t.Fatalf("children = %d", len(traceTree.Children))
	}
	assertStr(t, traceTree.String(), "aaa\n  - bbb\n  - ccc")
	htmlText := traceTree.HTML()
	assertContains(t, htmlText, "aaa")
	assertContains(t, htmlText, "<li>bbb</li>")
	assertContains(t, htmlText, "<li>ccc</li>")
}

func TestTraceTreeNestedChildren(t *testing.T) {
	traceTreeDict := map[string]any{
		"traceElement": map[string]any{"fragments": []any{map[string]any{"class": textFragmentClass, "text": "aaa"}}},
		"children": []any{
			map[string]any{
				"traceElement": map[string]any{"fragments": []any{map[string]any{"class": textFragmentClass, "text": "bbb"}}},
				"children": []any{
					map[string]any{"traceElement": map[string]any{"fragments": []any{map[string]any{"class": textFragmentClass, "text": "ccc"}}}},
				},
			},
			map[string]any{"traceElement": map[string]any{"fragments": []any{map[string]any{"class": textFragmentClass, "text": "ddd"}}}},
		},
	}
	traceTree := TraceTreeFromDict(traceTreeDict)
	if len(traceTree.Children) != 2 {
		t.Fatalf("children = %d", len(traceTree.Children))
	}
	if len(traceTree.Children[0].Children) != 1 {
		t.Fatalf("nested children = %d", len(traceTree.Children[0].Children))
	}
	assertStr(t, traceTree.String(), "aaa\n  - bbb\n    - ccc\n  - ddd")
	htmlText := traceTree.HTML()
	assertContains(t, htmlText, "aaa")
	assertContains(t, htmlText, "<li>bbb <ul>")
	assertContains(t, htmlText, "<ul><li>ccc</li></ul>")
	assertContains(t, htmlText, "<li>ddd</li>")
}

func TestTraceTreeListDeserialization(t *testing.T) {
	rawTraceTreeList := []map[string]any{
		{"traceElement": map[string]any{"fragments": []any{map[string]any{"class": textFragmentClass, "text": "aaa"}}}},
		{"traceElement": map[string]any{"fragments": []any{map[string]any{"class": textFragmentClass, "text": "bbb"}}}},
	}
	traceTreeList := make(TraceTreeList, 0, len(rawTraceTreeList))
	for _, d := range rawTraceTreeList {
		traceTreeList = append(traceTreeList, TraceTreeFromDict(d))
	}
	if len(traceTreeList) != 2 {
		t.Fatalf("trace tree list length = %d", len(traceTreeList))
	}
	assertDeepEqual(t, traceTreeList[0], TraceTree{
		TraceElement: TraceElement{Fragments: []Fragment{TextFragment{Text: "aaa"}}},
		Children:     []TraceTree{},
	})
	assertDeepEqual(t, traceTreeList[1], TraceTree{
		TraceElement: TraceElement{Fragments: []Fragment{TextFragment{Text: "bbb"}}},
		Children:     []TraceTree{},
	})
}

func TestTraceTreeListRepresentation(t *testing.T) {
	rawTraceTreeList := []map[string]any{
		{
			"traceElement": map[string]any{"fragments": []any{map[string]any{"class": textFragmentClass, "text": "aaa"}}},
			"children": []any{
				map[string]any{
					"traceElement": map[string]any{"fragments": []any{map[string]any{"class": textFragmentClass, "text": "bbb"}}},
					"children": []any{
						map[string]any{"traceElement": map[string]any{"fragments": []any{map[string]any{"class": textFragmentClass, "text": "ccc"}}}},
					},
				},
				map[string]any{"traceElement": map[string]any{"fragments": []any{map[string]any{"class": textFragmentClass, "text": "ddd"}}}},
			},
		},
		{
			"traceElement": map[string]any{"fragments": []any{map[string]any{"class": textFragmentClass, "text": "eee"}}},
			"children": []any{
				map[string]any{"traceElement": map[string]any{"fragments": []any{map[string]any{"class": textFragmentClass, "text": "fff"}}}},
			},
		},
	}
	traceTreeList := make(TraceTreeList, 0, len(rawTraceTreeList))
	for _, d := range rawTraceTreeList {
		traceTreeList = append(traceTreeList, TraceTreeFromDict(d))
	}
	assertStr(t, traceTreeList.String(), strings.Join([]string{"- aaa", "  - bbb", "    - ccc", "  - ddd", "- eee", "  - fff"}, "\n"))
	htmlText := strings.ReplaceAll(strings.ReplaceAll(traceTreeList.HTML(), " ", ""), "\n", "")
	assertStr(t, htmlText, "<ul><li>aaa<ul><li>bbb<ul><li>ccc</li></ul></li><li>ddd</li></ul></li><li>eee<ul><li>fff</li></ul></li></ul>")
}

// --- test_traceelement.py ---------------------------------------------------

func TestTextFragmentDeserialization(t *testing.T) {
	fragmentDict := map[string]any{"class": textFragmentClass, "text": "aaa"}
	fragment, err := FragmentFromDict(fragmentDict)
	assertDeepEqual(t, err, nil)

	textFragment, ok := fragment.(TextFragment)
	if !ok {
		t.Fatalf("expected TextFragment, got %T", fragment)
	}
	assertStr(t, textFragment.Text, "aaa")
}

func TestLinkFragmentDeserialization(t *testing.T) {
	fragmentDict := map[string]any{
		"class": linkFragmentClass,
		"text":  "aaa",
		"vendorStructureId": map[string]any{
			"filename":      "some-config",
			"structureType": "some-type",
			"structureName": "some-name",
		},
	}
	fragment, err := FragmentFromDict(fragmentDict)
	assertDeepEqual(t, err, nil)

	linkFragment, ok := fragment.(LinkFragment)
	if !ok {
		t.Fatalf("expected LinkFragment, got %T", fragment)
	}
	assertStr(t, linkFragment.Text, "aaa")
	assertDeepEqual(t, linkFragment.VendorStructureID, VendorStructureId{Filename: "some-config", StructureType: "some-type", StructureName: "some-name"})
}

func TestTraceElementDeserialization(t *testing.T) {
	traceElementDict := map[string]any{
		"fragments": []any{
			map[string]any{"class": textFragmentClass, "text": "aaa"},
			map[string]any{
				"class": linkFragmentClass,
				"text":  "bbb",
				"vendorStructureId": map[string]any{
					"filename": "aa", "structureType": "bb", "structureName": "cc",
				},
			},
			map[string]any{"class": textFragmentClass, "text": "ccc"},
		},
	}
	traceElement := TraceElementFromDict(traceElementDict)
	if len(traceElement.Fragments) != 3 {
		t.Fatalf("fragments = %d", len(traceElement.Fragments))
	}
	assertDeepEqual(t, traceElement.Fragments[0], TextFragment{Text: "aaa"})
	assertDeepEqual(t, traceElement.Fragments[1], LinkFragment{Text: "bbb", VendorStructureID: VendorStructureId{Filename: "aa", StructureType: "bb", StructureName: "cc"}})
	assertDeepEqual(t, traceElement.Fragments[2], TextFragment{Text: "ccc"})
}

func TestTraceElementStr(t *testing.T) {
	traceElement := TraceElement{Fragments: []Fragment{
		TextFragment{Text: "aaa "},
		LinkFragment{Text: "bbb", VendorStructureID: VendorStructureId{Filename: "aa", StructureType: "bb", StructureName: "cc"}},
		TextFragment{Text: " ccc"},
	}}
	assertStr(t, traceElement.String(), "aaa bbb ccc")
}

// --- test_traceevent.py -----------------------------------------------------

// test if a trace event with description is deserialized and string-ified properly
func TestTraceEventWithDescription(t *testing.T) {
	dict := map[string]any{"description": "aa"}

	// check deserialization
	traceEvent := AclTraceEventFromDict(dict)
	assertStr(t, *traceEvent.Description, "aa")

	// check str
	eventStr := traceEvent.String()
	assertContains(t, eventStr, "aa")
}

func TestTraceStr(t *testing.T) {
	event1 := map[string]any{"description": "event1"}
	event2 := map[string]any{}
	event3 := map[string]any{"description": "event3"}
	events := []any{event1, event2, event3}
	trace := AclTraceFromDict(map[string]any{"events": events})

	assertStr(t, trace.String(), "event1\nevent3")
}

// --- test_flowtracehop.py ---------------------------------------------------

// test if a flowtracehop is deserialized properly and converted to string properly
func TestFlowTraceHopDeserialization(t *testing.T) {
	hopDict := map[string]any{
		"edge": map[string]any{
			"node1": "node1", "node1interface": "Ethernet9",
			"node2": "(none)", "node2interface": "null_interface",
		},
		"routes": []any{"BgpRoute<12.10.16.8/25,nhip:9.1.1.2,nhint:dynamic>_fnhip:9.1.1.2"},
		"transformedFlow": map[string]any{
			"dscp": 0, "dstIp": "2.1.1.1", "dstPort": 0, "ecn": 0, "fragmentOffset": 0,
			"icmpCode": 255, "icmpVar": 255, "ingressNode": "ingress", "ingressVrf": "default",
			"ipProtocol": "IP", "packetLength": 0, "srcIp": "5.5.1.1", "srcPort": 0,
			"tcpFlagsAck": 0, "tcpFlagsCwr": 0, "tcpFlagsEce": 0, "tcpFlagsFin": 0,
			"tcpFlagsPsh": 0, "tcpFlagsRst": 0, "tcpFlagsSyn": 0, "tcpFlagsUrg": 0,
		},
	}
	hop := FlowTraceHopFromDict(hopDict)
	hopStr := hop.String()

	// check deserialization
	assertTrue(t, hop.Edge.Node1 != "", "hop.edge")
	if len(hop.Routes) != 1 {
		t.Fatalf("routes = %d", len(hop.Routes))
	}
	if hop.TransformedFlow == nil {
		t.Fatal("expected transformed flow")
	}

	// check the string representation has the essential elements (without forcing a strict format)
	assertContains(t, hopStr, hop.Edge.String())
	assertContains(t, hopStr, fmt.Sprint(hop.Routes[0]))
	assertContains(t, hopStr, hop.TransformedFlow.String())
}

// test that deserialization can tolerate hops with no routes.
func TestFlowTraceHopDeserializationNoRoutes(t *testing.T) {
	hopDict := map[string]any{
		"edge": map[string]any{
			"node1": "node1", "node1interface": "Ethernet9",
			"node2": "(none)", "node2interface": "null_interface",
		},
	}
	hop := FlowTraceHopFromDict(hopDict)

	// check deserialization
	if len(hop.Routes) != 0 {
		t.Fatalf("routes = %v", hop.Routes)
	}
}
