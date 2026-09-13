package datamodel

import (
	"reflect"
	"testing"
)

// TestToJSONValue mirrors pybatfish's BfJsonEncoder tests: primitives pass
// through and datamodel elements are converted via Dict.
func TestToJSONValue(t *testing.T) {
	if got := ToJSONValue(1); got != 1 {
		t.Fatalf("got %v", got)
	}
	if got := ToJSONValue(3.14); got != 3.14 {
		t.Fatalf("got %v", got)
	}
	if got := ToJSONValue("some_string"); got != "some_string" {
		t.Fatalf("got %v", got)
	}
	if got := ToJSONValue(nil); got != nil {
		t.Fatalf("got %v", got)
	}
	if got := ToJSONValue([]any{1, 2, "some_string"}); !reflect.DeepEqual(got, []any{1, 2, "some_string"}) {
		t.Fatalf("got %v", got)
	}

	iface := Interface{Hostname: "node", Interface: "iface"}
	if got := ToJSONValue(iface); !reflect.DeepEqual(got, iface.Dict()) {
		t.Fatalf("got %v", got)
	}
	nested := ToJSONValue(map[string]any{"name": map[string]any{"nested": iface}})
	want := map[string]any{"name": map[string]any{"nested": map[string]any{"hostname": "node", "interface": "iface"}}}
	if !reflect.DeepEqual(nested, want) {
		t.Fatalf("got %v, want %v", nested, want)
	}
}
