package client

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/81ueman/gobatfish/dataframe"
	"github.com/81ueman/gobatfish/datamodel"
	"github.com/81ueman/gobatfish/util"
)

func durPtr(d time.Duration) *time.Duration { return &d }

func TestSessionAPIKey(t *testing.T) {
	s := NewSession(SessionConfig{APIKey: "foo"})
	if s.APIKey != "foo" {
		t.Fatalf("api key = %q", s.APIKey)
	}
}

func TestDefaultPort(t *testing.T) {
	if s := NewSession(SessionConfig{}); s.PortV2 != 9996 {
		t.Fatalf("port = %d", s.PortV2)
	}
}

func TestPortSet(t *testing.T) {
	s := NewSession(SessionConfig{Port: util.IntPtr(8888), PortV2: util.IntPtr(1111)})
	if s.PortV2 != 8888 {
		t.Fatalf("port = %d", s.PortV2)
	}
}

func TestPortV2Set(t *testing.T) {
	s := NewSession(SessionConfig{PortV2: util.IntPtr(8888)})
	if s.PortV2 != 8888 {
		t.Fatalf("port = %d", s.PortV2)
	}
}

func TestRequestKwargsDefaults(t *testing.T) {
	s := NewSession(SessionConfig{})
	if s.Proxies != nil {
		t.Fatal("expected nil proxies")
	}
	if s.Timeout == nil || *s.Timeout != 30*time.Second {
		t.Fatalf("timeout = %v", s.Timeout)
	}
	if got := s.GetRequestKwargs(); !reflect.DeepEqual(got, map[string]any{"timeout": float64(30)}) {
		t.Fatalf("kwargs = %v", got)
	}
}

func TestRequestKwargsCustomTimeout(t *testing.T) {
	s := NewSession(SessionConfig{Timeout: durPtr(60 * time.Second)})
	if got := s.GetRequestKwargs(); !reflect.DeepEqual(got, map[string]any{"timeout": float64(60)}) {
		t.Fatalf("kwargs = %v", got)
	}
}

func TestRequestKwargsTimeoutNone(t *testing.T) {
	s := NewSession(SessionConfig{Timeout: durPtr(0)})
	if _, ok := s.GetRequestKwargs()["timeout"]; ok {
		t.Fatalf("timeout should be omitted: %v", s.GetRequestKwargs())
	}
}

func TestRequestKwargsProxies(t *testing.T) {
	proxies := map[string]string{"http": "http://proxy:8080", "https": "http://proxy:8080"}
	s := NewSession(SessionConfig{Proxies: proxies})
	if got := s.GetRequestKwargs()["proxies"]; !reflect.DeepEqual(got, proxies) {
		t.Fatalf("proxies = %v", got)
	}
}

func TestRequestKwargsGeneric(t *testing.T) {
	s := NewSession(SessionConfig{Timeout: durPtr(0), RequestKwargs: map[string]any{"verify": false}})
	if got := s.GetRequestKwargs(); !reflect.DeepEqual(got, map[string]any{"verify": false}) {
		t.Fatalf("kwargs = %v", got)
	}
}

func TestRequestKwargsOverride(t *testing.T) {
	s := NewSession(SessionConfig{
		Timeout:       durPtr(10 * time.Second),
		Proxies:       map[string]string{"http": "http://proxy:8080"},
		RequestKwargs: map[string]any{"timeout": 999, "proxies": map[string]any{"http": "http://other:9090"}},
	})
	got := s.GetRequestKwargs()
	if got["timeout"] != float64(10) {
		t.Fatalf("timeout = %v", got["timeout"])
	}
	if got["proxies"].(map[string]string)["http"] != "http://proxy:8080" {
		t.Fatalf("proxies = %v", got["proxies"])
	}
}

func TestGetSessionTypes(t *testing.T) {
	types := SessionTypes()
	if _, ok := types["bf"]; !ok {
		t.Fatalf("bf session type missing: %v", types)
	}
}

func TestGetSession(t *testing.T) {
	session, err := GetSession("bf", SessionConfig{Host: "foobar"})
	if err != nil || session.Host != "foobar" {
		t.Fatalf("session = %+v, err = %v", session, err)
	}
}

func TestGetSessionBad(t *testing.T) {
	_, err := GetSession("bogus_session_type", SessionConfig{})
	if err == nil || !contains(err.Error(), "invalid session type") {
		t.Fatalf("err = %v", err)
	}
}

func TestAutoCompleteInvalidMaxSuggestions(t *testing.T) {
	s := NewSession(SessionConfig{})
	neg := -1
	if _, err := s.AutoComplete(context.Background(), datamodel.VariableBGPRouteStatusSpec, "foo", &neg); err == nil {
		t.Fatal("expected error for negative max suggestions")
	}
}

func TestTextWithPlatform(t *testing.T) {
	text := "abcdefgh\n"
	if got := textWithPlatform(text, ""); got != text {
		t.Fatalf("got %q", got)
	}
	want := "!RANCID-CONTENT-TYPE: arista\n" + text
	if got := textWithPlatform(text, "arista"); got != want {
		t.Fatalf("got %q", got)
	}
	if got := textWithPlatform(text, " aRiStA \t"); got != want {
		t.Fatalf("got %q", got)
	}
}

func TestCreateSingleFileZip(t *testing.T) {
	data, err := createInMemoryZip("abcdefgh\n", "myfile", "arista")
	if err != nil {
		t.Fatal(err)
	}
	names, contents := readZip(t, data)
	if len(names) != 1 || names[0] != "snapshot/configs/myfile" {
		t.Fatalf("names = %v", names)
	}
	if contents["snapshot/configs/myfile"] != "!RANCID-CONTENT-TYPE: arista\nabcdefgh\n" {
		t.Fatalf("contents = %q", contents["snapshot/configs/myfile"])
	}
}

func TestWorkItemToDict(t *testing.T) {
	s := NewSession(SessionConfig{})
	s.Network = util.StringPtr("net")
	item := NewWorkItem(s)
	item.RequestParams["testrig"] = "snap"
	d := item.ToDict()
	if d["containerName"] != "net" || d["testrigName"] != "snap" {
		t.Fatalf("dict = %v", d)
	}
}

func TestIsDictMatch(t *testing.T) {
	if !isDictMatch(map[string]any{"k1": "v1", "k2": "v2"}, map[string]any{"k1": "v1", "k2": "v2"}) {
		t.Fatal("expected match")
	}
	if !isDictMatch(map[string]any{"k1": "v1", "k2": "v2"}, map[string]any{"k1": "v1"}) {
		t.Fatal("expected subset match")
	}
	if isDictMatch(map[string]any{"k1": "v1", "k2": "v2"}, map[string]any{"k3": "v3"}) {
		t.Fatal("expected no match for extra key")
	}
	if isDictMatch(map[string]any{"k1": "v1", "k2": "v2"}, map[string]any{"k1": "v3"}) {
		t.Fatal("expected no match for wrong value")
	}
}

func TestFormatDFIllegalFormat(t *testing.T) {
	df := dataframe.Empty("a")
	if _, err := formatDF(df, "nothing"); err == nil {
		t.Fatal("expected error")
	}
}

func TestCapircaCreateReferenceBook(t *testing.T) {
	defs := NewDefinitions()
	defs.AddNetwork("NET1", "1.1.1.1", "10.0.0.0/8")
	defs.AddNetwork("NET2", "NET1", "2.2.2.2")
	book, err := CreateReferenceBookFromDefinitions(defs, "capirca")
	if err != nil {
		t.Fatal(err)
	}
	if book.Name != "capirca" || len(book.AddressGroups) != 2 {
		t.Fatalf("book = %+v", book)
	}
	byName := map[string]datamodel.AddressGroup{}
	for _, g := range book.AddressGroups {
		byName[g.Name] = g
	}
	if !reflect.DeepEqual(byName["NET1"].Addresses, []string{"1.1.1.1", "10.0.0.0/8"}) {
		t.Fatalf("NET1 addresses = %v", byName["NET1"].Addresses)
	}
	if !reflect.DeepEqual(byName["NET2"].ChildGroupNames, []string{"NET1"}) {
		t.Fatalf("NET2 children = %v", byName["NET2"].ChildGroupNames)
	}
}

func TestHTTPHeaders(t *testing.T) {
	var gotAPIKey, gotVersion string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAPIKey = r.Header.Get(HTTPHeaderBatfishAPIKey)
		gotVersion = r.Header.Get(HTTPHeaderBatfishVersion)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{"name": "net1"}})
	}))
	defer srv.Close()

	s := sessionForServer(t, srv.URL, nil)
	networks, err := s.ListNetworks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(networks, []string{"net1"}) {
		t.Fatalf("networks = %v", networks)
	}
	if gotAPIKey != DefaultAPIKey || gotVersion == "" {
		t.Fatalf("headers = %q, %q", gotAPIKey, gotVersion)
	}
}

func TestHTTPRetriesOnServerError(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode([]map[string]any{{"name": "net1"}})
	}))
	defer srv.Close()

	retries := 1
	s := sessionForServer(t, srv.URL, &retries)
	networks, err := s.ListNetworks(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(networks) != 1 || calls < 2 {
		t.Fatalf("networks = %v, calls = %d", networks, calls)
	}
}

func sessionForServer(t *testing.T, rawURL string, maxRetries *int) *Session {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil {
		t.Fatal(err)
	}
	return NewSession(SessionConfig{
		Host:       u.Hostname(),
		Port:       &port,
		SSL:        util.BoolPtr(false),
		MaxRetries: maxRetries,
	})
}

func readZip(t *testing.T, data []byte) ([]string, map[string]string) {
	t.Helper()
	names, contents := []string{}, map[string]string{}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range zr.File {
		names = append(names, f.Name)
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		buf, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatal(err)
		}
		contents[f.Name] = string(buf)
	}
	return names, contents
}

func contains(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}
