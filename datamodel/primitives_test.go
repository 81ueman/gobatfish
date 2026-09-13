package datamodel

import (
	"encoding/json"
	"testing"
)

// Ports of tests/datamodel/test_interface.py.

func TestInterfaceStr(t *testing.T) {
	// no escaping
	assertStr(t, (Interface{Hostname: "node", Interface: "iface"}).String(), "node[iface]")

	// escape hostname
	assertStr(t, (Interface{Hostname: "0node", Interface: "iface"}).String(), `"0node"[iface]`)

	// escape interface
	assertStr(t, (Interface{Hostname: "node", Interface: "/iface"}).String(), `node["/iface"]`)
}

func TestInterfaceHTML(t *testing.T) {
	i := Interface{Hostname: "host", Interface: "special&"}
	assertStr(t, i.HTML(), "host[&quot;special&amp;&quot;]")
	i = Interface{Hostname: "host", Interface: "normal:0/0.0"}
	assertStr(t, i.HTML(), "host[normal:0/0.0]")
}

// Ports of tests/datamodel/test_filelines.py.

func TestFileLinesMultipleLines(t *testing.T) {
	filelines := FileLinesFromDict(map[string]any{"filename": "myfile", "lines": []any{2, 3}})
	assertStr(t, filelines.Filename, "myfile")
	assertDeepEqual(t, filelines.Lines, []int{2, 3})
}

func TestFileLinesNoLines(t *testing.T) {
	filelines := FileLinesFromDict(map[string]any{"filename": "myfile"})
	assertStr(t, filelines.Filename, "myfile")
	if len(filelines.Lines) != 0 {
		t.Fatalf("expected no lines, got %v", filelines.Lines)
	}
}

func TestFileLinesZeroLines(t *testing.T) {
	filelines := FileLinesFromDict(map[string]any{"filename": "myfile", "lines": []any{}})
	assertStr(t, filelines.Filename, "myfile")
	if len(filelines.Lines) != 0 {
		t.Fatalf("expected no lines, got %v", filelines.Lines)
	}
}

// Ports of tests/datamodel/test_auto_complete_suggestion.py.

func TestAutoCompleteSuggestionAllFields(t *testing.T) {
	suggestion := AutoCompleteSuggestionFromDict(map[string]any{
		"description": "desc", "hint": "hint", "insertionIndex": 16,
		"isPartial": true, "rank": 42, "text": "suggestion",
	})
	assertStr(t, *suggestion.Description, "desc")
	assertStr(t, *suggestion.Hint, "hint")
	if suggestion.InsertionIndex != 16 {
		t.Fatalf("insertion index = %d", suggestion.InsertionIndex)
	}
	assertTrue(t, suggestion.IsPartial, "is partial")
	if suggestion.Rank != 42 {
		t.Fatalf("rank = %d", suggestion.Rank)
	}
	assertStr(t, suggestion.Text, "suggestion")
}

func TestAutoCompleteSuggestionNoOptionals(t *testing.T) {
	suggestion := AutoCompleteSuggestionFromDict(map[string]any{"isPartial": true, "rank": 42, "text": "suggestion"})
	if suggestion.Description != nil {
		t.Fatalf("description = %v", suggestion.Description)
	}
	if suggestion.Hint != nil {
		t.Fatalf("hint = %v", suggestion.Hint)
	}
	if suggestion.InsertionIndex != 0 {
		t.Fatalf("insertion index = %d", suggestion.InsertionIndex)
	}
	assertTrue(t, suggestion.IsPartial, "is partial")
	if suggestion.Rank != 42 {
		t.Fatalf("rank = %d", suggestion.Rank)
	}
	assertStr(t, suggestion.Text, "suggestion")
}

// Ports of tests/datamodel/test_primitives.py.
//
// pybatfish's ListWrapper is hashable and immutable. Go slices can be neither,
// so the equivalent Go guarantees are tested instead: NewListWrapper copies its
// input, so mutating the source afterwards does not change the wrapper, and the
// wrapper renders correctly.

func TestListWrapperCopiesInput(t *testing.T) {
	src := []any{1, 2, 3}
	wrapper := NewListWrapper(src)
	// Mutating the source slice must not affect the wrapper.
	src[1] = 99
	assertDeepEqual(t, wrapper[1], 2)
	assertDeepEqual(t, wrapper, ListWrapper{1, 2, 3})
	assertStr(t, wrapper.String(), "[1 2 3]")
	assertStr(t, wrapper.HTML(), "1<br><br>2<br><br>3")
}

// Ports of tests/datamodel/test_datamodel_element.py.

func TestDataModelElementAsDict(t *testing.T) {
	assertDeepEqual(t, (Interface{Hostname: "host", Interface: "iface"}).Dict(), map[string]any{
		"hostname":  "host",
		"interface": "iface",
	})

	// Make sure Edge dict is right if either string or Interface is passed in
	assertDeepEqual(t, NewEdge(
		"r1",
		"iface1",
		"r2",
		Interface{Hostname: "r2", Interface: "iface2"},
	).Dict(), map[string]any{
		"node1":          "r1",
		"node1interface": "iface1",
		"node2":          "r2",
		"node2interface": "iface2",
	})
}

func TestDataModelElementJSONSerialization(t *testing.T) {
	i := Interface{Hostname: "host", Interface: "iface"}
	// Load into dict from json to ignore key ordering
	encoded, err := json.Marshal(i)
	if err != nil {
		t.Fatal(err)
	}
	dictJSON, err := json.Marshal(i.Dict())
	if err != nil {
		t.Fatal(err)
	}
	var encodedMap, dictMap map[string]any
	if err := json.Unmarshal(encoded, &encodedMap); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(dictJSON, &dictMap); err != nil {
		t.Fatal(err)
	}
	assertDeepEqual(t, encodedMap, dictMap)
}
