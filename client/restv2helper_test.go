package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestCheckResponseStatusError(t *testing.T) {
	s := NewSession(SessionConfig{})
	resp := &http.Response{
		StatusCode: 400,
		Status:     "400 Bad Request",
		Body:       io.NopCloser(strings.NewReader("error detail")),
	}
	err := s.checkStatus(resp)
	if err == nil || !strings.Contains(err.Error(), "error detail") {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckResponseStatusOK(t *testing.T) {
	s := NewSession(SessionConfig{})
	resp := &http.Response{
		StatusCode: 200,
		Status:     "200 OK",
		Body:       io.NopCloser(strings.NewReader("no error")),
	}
	if err := s.checkStatus(resp); err != nil {
		t.Fatalf("err = %v", err)
	}
}

func TestGetHeaders(t *testing.T) {
	s := NewSession(SessionConfig{APIKey: "0000"})
	headers := s.getHeaders()
	if headers[HTTPHeaderBatfishAPIKey] != "0000" {
		t.Fatalf("apikey header = %q", headers[HTTPHeaderBatfishAPIKey])
	}
	if headers[HTTPHeaderBatfishVersion] == "" {
		t.Fatal("version header missing")
	}
}

func TestRetryConfiguration(t *testing.T) {
	for _, status := range []int{429, 500, 502, 503, 504} {
		if !retryStatuses[status] {
			t.Fatalf("expected %d to be retried", status)
		}
	}
	if retryStatuses[200] || retryStatuses[404] {
		t.Fatal("unexpected retry status")
	}
	// pybatfish retries every method, including POST.
	if MaxRetriesToConnect < MaxInitialTriesToConnect {
		t.Fatal("unexpected retry defaults")
	}
}

func TestHTTPMethodShapes(t *testing.T) {
	type captured struct {
		method      string
		path        string
		query       string
		contentType string
		apiKey      string
		body        string
	}
	var got []captured
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		got = append(got, captured{
			method:      r.Method,
			path:        r.URL.Path,
			query:       r.URL.RawQuery,
			contentType: r.Header.Get("Content-Type"),
			apiKey:      r.Header.Get(HTTPHeaderBatfishAPIKey),
			body:        string(body),
		})
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	s := sessionForServer(t, srv.URL, nil)
	ctx := context.Background()

	if _, err := s.get(ctx, "/test/url", url.Values{"a": []string{"b"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.post(ctx, "/test/url", []byte(`{"x":1}`), "application/json", nil); err != nil {
		t.Fatal(err)
	}
	if err := s.put(ctx, "/test/url", []byte("data"), "application/octet-stream", url.Values{"key": []string{"k"}}); err != nil {
		t.Fatal(err)
	}
	if err := s.delete(ctx, "/test/url", nil); err != nil {
		t.Fatal(err)
	}

	want := []captured{
		{method: "GET", path: "/v2/test/url", query: "a=b", apiKey: DefaultAPIKey},
		{method: "POST", path: "/v2/test/url", contentType: "application/json", apiKey: DefaultAPIKey, body: `{"x":1}`},
		{method: "PUT", path: "/v2/test/url", query: "key=k", contentType: "application/octet-stream", apiKey: DefaultAPIKey, body: "data"},
		{method: "DELETE", path: "/v2/test/url", apiKey: DefaultAPIKey},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("requests = %+v, want %+v", got, want)
	}
}

func TestGetAPIVersionOld(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{})
	}))
	defer srv.Close()
	s := sessionForServer(t, srv.URL, nil)
	version, err := s.GetAPIVersion(context.Background())
	if err != nil || version != "2.0.0" {
		t.Fatalf("version = %q, err = %v", version, err)
	}
}

func TestGetAPIVersionNew(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{KeyAPIVersion: "2.1.0"})
	}))
	defer srv.Close()
	s := sessionForServer(t, srv.URL, nil)
	version, err := s.GetAPIVersion(context.Background())
	if err != nil || version != "2.1.0" {
		t.Fatalf("version = %q, err = %v", version, err)
	}
}
