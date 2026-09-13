package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/81ueman/gobatfish/question"
)

func installQuestion(t *testing.T, bf *Session, name, description string, tags ...string) {
	t.Helper()
	tagValues := make([]any, len(tags))
	for i, tag := range tags {
		tagValues[i] = tag
	}
	_, tmpl, err := question.LoadQuestionDict(map[string]any{"instance": map[string]any{
		"instanceName": name, "description": description, "tags": tagValues,
	}}, &factStubSession{})
	if err != nil {
		t.Fatal(err)
	}
	bf.Q.Install(tmpl)
}

func TestListQuestions(t *testing.T) {
	bf := NewSession(SessionConfig{})
	if len(bf.Q.List()) != 0 {
		t.Fatal("expected no questions initially")
	}
	installQuestion(t, bf, "qName1", "description1.", "tag1")
	installQuestion(t, bf, "qName2", "description2.", "tags2")

	listed := bf.Q.List()
	if len(listed) != 2 {
		t.Fatalf("listed = %v", listed)
	}
	names := map[string]bool{}
	for _, q := range listed {
		names[q["name"].(string)] = true
	}
	if !names["qName1"] || !names["qName2"] {
		t.Fatalf("names = %v", names)
	}
}

func TestListTags(t *testing.T) {
	bf := NewSession(SessionConfig{})
	if len(bf.Q.ListTags()) != 0 {
		t.Fatal("expected no tags initially")
	}
	installQuestion(t, bf, "qName1", "description", "tag1", "tag2")
	installQuestion(t, bf, "qName2", "description")
	installQuestion(t, bf, "qName3", "description", "tag2")

	want := map[string]struct{}{"tag1": {}, "tag2": {}}
	if got := bf.Q.ListTags(); !reflect.DeepEqual(got, want) {
		t.Fatalf("tags = %v, want %v", got, want)
	}
}

func TestLoadQuestionsRemote(t *testing.T) {
	templates := map[string]string{}
	for _, name := range []string{"q1", "q2"} {
		data, _ := json.Marshal(map[string]any{"class": "class", "instance": map[string]any{
			"description": "description.", "instanceName": name, "tags": []any{},
		}})
		templates[name] = string(data)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(templates)
	}))
	defer srv.Close()

	bf := sessionForServer(t, srv.URL, nil)
	if err := bf.LoadQuestions(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !bf.Q.Has("q1") || !bf.Q.Has("q2") {
		t.Fatalf("questions not loaded: %v", bf.Q.Names())
	}
}
