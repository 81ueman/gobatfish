package datamodel

import (
	"fmt"
	"strings"
)

// AclTraceEvent is one event in a packet's life through an ACL.
type AclTraceEvent struct {
	Description *string `json:"description"`
}

// AclTraceEventFromDict builds an event from a dictionary.
func AclTraceEventFromDict(d map[string]any) AclTraceEvent {
	return AclTraceEvent{Description: optStrField(d, "description")}
}

// Dict returns the dictionary representation of the event.
func (e AclTraceEvent) Dict() map[string]any { return dictOf(e) }

// String renders the event description.
func (e AclTraceEvent) String() string {
	if e.Description == nil {
		return "None"
	}
	return *e.Description
}

// AclTrace is the trace of a packet's life through an ACL.
type AclTrace struct {
	Events []AclTraceEvent `json:"events"`
}

// AclTraceFromDict builds an AclTrace from a dictionary.
func AclTraceFromDict(d map[string]any) AclTrace {
	events := sliceField(d, "events")
	out := make([]AclTraceEvent, 0, len(events))
	for _, e := range events {
		out = append(out, AclTraceEventFromDict(toMap(e)))
	}
	return AclTrace{Events: out}
}

// Dict returns the dictionary representation of the trace.
func (t AclTrace) Dict() map[string]any { return dictOf(t) }

// String renders the events with descriptions, one per line.
func (t AclTrace) String() string {
	var lines []string
	for _, e := range t.Events {
		if e.Description != nil {
			lines = append(lines, e.String())
		}
	}
	return strings.Join(lines, "\n")
}

// VendorStructureId identifies a vendor structure in a configuration file.
type VendorStructureId struct {
	Filename      string `json:"filename"`
	StructureType string `json:"structureType"`
	StructureName string `json:"structureName"`
}

// VendorStructureIdFromDict builds a VendorStructureId from a dictionary.
func VendorStructureIdFromDict(d map[string]any) VendorStructureId {
	return VendorStructureId{
		Filename:      strField(d, "filename"),
		StructureType: strField(d, "structureType"),
		StructureName: strField(d, "structureName"),
	}
}

// Dict returns the dictionary representation of the id.
func (v VendorStructureId) Dict() map[string]any { return dictOf(v) }

// Fragment is an element of a TraceElement: a TextFragment or LinkFragment.
type Fragment interface {
	fragment()
	String() string
}

// FragmentFromDict builds the appropriate Fragment from a dictionary.
func FragmentFromDict(d map[string]any) (Fragment, error) {
	switch strField(d, "class") {
	case "org.batfish.datamodel.TraceElement$TextFragment":
		return TextFragmentFromDict(d), nil
	case "org.batfish.datamodel.TraceElement$LinkFragment":
		return LinkFragmentFromDict(d), nil
	default:
		return nil, fmt.Errorf("unknown Fragment type %v", d["class"])
	}
}

// TextFragment is a plain-text fragment.
type TextFragment struct {
	Text string `json:"text"`
}

func (TextFragment) fragment() {}

// TextFragmentFromDict builds a TextFragment from a dictionary.
func TextFragmentFromDict(d map[string]any) TextFragment {
	return TextFragment{Text: strField(d, "text")}
}

// Dict returns the dictionary representation of the fragment.
func (f TextFragment) Dict() map[string]any { return dictOf(f) }

// String returns the fragment text.
func (f TextFragment) String() string { return f.Text }

// LinkFragment is a fragment that links to a vendor structure.
type LinkFragment struct {
	Text              string            `json:"text"`
	VendorStructureID VendorStructureId `json:"vendorStructureId"`
}

func (LinkFragment) fragment() {}

// LinkFragmentFromDict builds a LinkFragment from a dictionary.
func LinkFragmentFromDict(d map[string]any) LinkFragment {
	return LinkFragment{
		Text:              strField(d, "text"),
		VendorStructureID: VendorStructureIdFromDict(mapField(d, "vendorStructureId")),
	}
}

// Dict returns the dictionary representation of the fragment.
func (f LinkFragment) Dict() map[string]any { return dictOf(f) }

// String returns the fragment text.
func (f LinkFragment) String() string { return f.Text }

// TraceElement is metadata used to create human-readable traces.
type TraceElement struct {
	Fragments []Fragment `json:"fragments"`
}

// TraceElementFromDict builds a TraceElement from a dictionary.
func TraceElementFromDict(d map[string]any) TraceElement {
	frags := sliceField(d, "fragments")
	out := make([]Fragment, 0, len(frags))
	for _, f := range frags {
		frag, err := FragmentFromDict(toMap(f))
		if err == nil {
			out = append(out, frag)
		}
	}
	return TraceElement{Fragments: out}
}

// Dict returns the dictionary representation of the trace element.
func (t TraceElement) Dict() map[string]any { return dictOf(t) }

// String concatenates the fragment strings.
func (t TraceElement) String() string {
	var b strings.Builder
	for _, f := range t.Fragments {
		b.WriteString(f.String())
	}
	return b.String()
}

// TraceTree represents a filter trace tree.
type TraceTree struct {
	TraceElement TraceElement `json:"traceElement"`
	Children     []TraceTree  `json:"children"`
}

// TraceTreeFromDict builds a TraceTree from a dictionary.
func TraceTreeFromDict(d map[string]any) TraceTree {
	children := sliceField(d, "children")
	out := make([]TraceTree, 0, len(children))
	for _, c := range children {
		out = append(out, TraceTreeFromDict(toMap(c)))
	}
	return TraceTree{
		TraceElement: TraceElementFromDict(mapField(d, "traceElement")),
		Children:     out,
	}
}

// Dict returns the dictionary representation of the tree.
func (t TraceTree) Dict() map[string]any { return dictOf(t) }

// String renders the tree indented.
func (t TraceTree) String() string {
	lines := []string{t.TraceElement.String()}
	var walk func(nodes []TraceTree, depth int)
	walk = func(nodes []TraceTree, depth int) {
		for _, child := range nodes {
			lines = append(lines, strings.Repeat("  ", depth)+"- "+child.TraceElement.String())
			walk(child.Children, depth+1)
		}
	}
	walk(t.Children, 1)
	return strings.Join(lines, "\n")
}

// HTML returns the HTML representation of the tree.
func (t TraceTree) HTML() string {
	if len(t.Children) > 0 {
		var b strings.Builder
		for _, child := range t.Children {
			b.WriteString("<li>")
			b.WriteString(child.HTML())
			b.WriteString("</li>")
		}
		return fmt.Sprintf("%s <ul>%s</ul>", t.TraceElement.String(), b.String())
	}
	return t.TraceElement.String()
}

// TraceTreeList is a list of TraceTree with custom string and HTML rendering.
type TraceTreeList []TraceTree

// String renders each tree prefixed with "- ".
func (l TraceTreeList) String() string {
	parts := make([]string, len(l))
	for i, tree := range l {
		parts[i] = "- " + tree.String()
	}
	return strings.Join(parts, "\n")
}

// HTML returns the HTML representation of the list.
func (l TraceTreeList) HTML() string {
	var b strings.Builder
	b.WriteString("<ul>")
	for _, tree := range l {
		b.WriteString("<li>")
		b.WriteString(tree.HTML())
		b.WriteString("</li>")
	}
	b.WriteString("</ul>")
	return b.String()
}

func toMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}
