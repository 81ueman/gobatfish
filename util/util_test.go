package util

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConditionalStr(t *testing.T) {
	if got := ConditionalStr("pre", []any{}, "after"); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := ConditionalStr("pre", map[string]any{}, "after"); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := ConditionalStr("pre", nil, "after"); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := ConditionalStr("pre", []any{"hey"}, "after"); got != "pre [hey] after" {
		t.Fatalf("got %q", got)
	}
}

func TestValidateName(t *testing.T) {
	if err := ValidateName("goodname", "snapshot"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, name := range []string{strings.Repeat("x", 151), "/", "/etc", "seTTings", "settings", ".", "../../../../etc/"} {
		if err := ValidateName(name, "snapshot"); err == nil {
			t.Fatalf("expected error for %q", name)
		}
	}
	for _, name := range []string{"23", "___", "foo-bar"} {
		if err := ValidateName(name, "snapshot"); err != nil {
			t.Fatalf("unexpected error for %q: %v", name, err)
		}
	}
}

func TestValidateQuestionName(t *testing.T) {
	for _, name := range []string{"/", "/etc", strings.Repeat("x", 151)} {
		if err := ValidateQuestionName(name); err == nil {
			t.Fatalf("expected error for %q", name)
		}
	}
	if err := ValidateQuestionName("goodname"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestEscapeHTML(t *testing.T) {
	cases := map[string]string{
		"":             "",
		"a":            "a",
		`"a"`:          "&quot;a&quot;",
		"a&b":          "a&amp;b",
		"a & b":        "a &amp; b",
		"host[\"x&\"]": "host[&quot;x&amp;&quot;]",
	}
	for in, want := range cases {
		if got := EscapeHTML(in); got != want {
			t.Fatalf("EscapeHTML(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEscapeName(t *testing.T) {
	cases := map[string]string{
		"":       "",
		"a":      "a",
		"abc":    "abc",
		`"a`:     `""a"`,
		"/a":     `"/a"`,
		"0a":     `"0a"`,
		"a#":     `"a#"`,
		"normal": "normal",
	}
	for in, want := range cases {
		if got := EscapeName(in); got != want {
			t.Fatalf("EscapeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGetUUIDUnique(t *testing.T) {
	a := GetUUID()
	b := GetUUID()
	if a == b || len(a) != 36 {
		t.Fatalf("uuids = %q, %q", a, b)
	}
}

type htmlThing struct{ s string }

func (h htmlThing) HTML() string { return "<b>" + h.s + "</b>" }

func TestGetHTML(t *testing.T) {
	if got := GetHTML("astring"); got != "astring" {
		t.Fatalf("got %q", got)
	}
	if got := GetHTML(1.2); got != "1.2" {
		t.Fatalf("got %q", got)
	}
	if got := GetHTML(100); got != "100" {
		t.Fatalf("got %q", got)
	}
	if got := GetHTML(htmlThing{s: "x"}); got != "<b>x</b>" {
		t.Fatalf("got %q", got)
	}
	if got := GetHTML("a&b"); got != "a&amp;b" {
		t.Fatalf("got %q", got)
	}
}

func TestZipDir(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "dirname")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	contents := "file contents"
	if err := os.WriteFile(filepath.Join(sub, "filename"), []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := ZipDir(sub, &buf); err != nil {
		t.Fatal(err)
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range zr.File {
		if f.Name == "dirname/filename" {
			found = true
			rc, err := f.Open()
			if err != nil {
				t.Fatal(err)
			}
			data := make([]byte, f.UncompressedSize64)
			_, _ = rc.Read(data)
			rc.Close()
			if string(data) != contents {
				t.Fatalf("contents = %q", data)
			}
		}
	}
	if !found {
		t.Fatalf("dirname/filename not found in %v", zr.File)
	}
}
