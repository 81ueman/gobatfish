package client

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/81ueman/gobatfish/util"
)

func TestFormatElapsedTime(t *testing.T) {
	cases := []struct {
		delta RelativeDelta
		want  string
	}{
		{RelativeDelta{Years: 7, Months: 6, Days: 5, Hours: 4, Minutes: 3, Seconds: 2}, "7y6m5d04:03:02"},
		{RelativeDelta{Months: 6, Days: 5, Hours: 4, Minutes: 3, Seconds: 2}, "6m5d04:03:02"},
		{RelativeDelta{Days: 5, Hours: 4, Minutes: 3, Seconds: 2}, "5d04:03:02"},
		{RelativeDelta{Hours: 4, Minutes: 3, Seconds: 2}, "04:03:02"},
		{RelativeDelta{Minutes: 3, Seconds: 2}, "00:03:02"},
		{RelativeDelta{Seconds: 2}, "00:00:02"},
		{RelativeDelta{}, "00:00:00"},
	}
	for _, c := range cases {
		if got := formatElapsedTime(c.delta); got != c.want {
			t.Fatalf("formatElapsedTime(%+v) = %q, want %q", c.delta, got, c.want)
		}
	}
}

func TestParseNumericTimestamp(t *testing.T) {
	got, err := parseTimestamp("1511981483456")
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2017, 11, 29, 18, 51, 23, 456000000, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseRFC3339Timestamp(t *testing.T) {
	got, err := parseTimestamp("2017-11-29T18:51:23.456+0000")
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2017, 11, 29, 18, 51, 23, 456000000, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestRelativeDelta(t *testing.T) {
	start := time.Date(2016, 11, 22, 10, 43, 21, 0, time.UTC)
	end := time.Date(2017, 12, 20, 0, 0, 0, 0, time.UTC)
	got := relativeDelta(start, end)
	want := RelativeDelta{Years: 1, Days: 27, Hours: 13, Minutes: 16, Seconds: 39}
	if got != want {
		t.Fatalf("relativeDelta = %+v, want %+v", got, want)
	}
	if s := formatElapsedTime(got); s != "1y27d13:16:39" {
		t.Fatalf("formatted = %q", s)
	}
}

func TestPrintWorkStatusFreshTask(t *testing.T) {
	session := NewSession(SessionConfig{})
	taskDetails := `{"obtained":"2017-12-20 00:00:00 UTC","batches":[{"completed":0,"description":"Fooing the bar","size":0,"startDate":"2017-12-20 00:00:00 UTC"}]}`
	now := func(time.Time) time.Time { return time.Date(2017, 12, 20, 0, 0, 0, 0, time.UTC) }
	var buf bytes.Buffer
	printWorkStatusHelper(&buf, session, "TEST", taskDetails, now)
	out := buf.String()
	if !strings.Contains(out, "status: TEST") {
		t.Fatalf("missing status: %q", out)
	}
	obtained, _ := parseTimestamp("2017-12-20 00:00:00 UTC")
	want := ".... " + printTimestamp(obtained) + " Fooing the bar"
	if !strings.Contains(out, want) {
		t.Fatalf("output %q does not contain %q", out, want)
	}
	if strings.Contains(out, "elapsed") {
		t.Fatalf("unexpected elapsed time: %q", out)
	}
}

func TestPrintWorkStatusFreshTaskSubtasks(t *testing.T) {
	session := NewSession(SessionConfig{})
	taskDetails := `{"obtained":"2017-12-20 00:00:00 UTC","batches":[{"completed":1,"description":"Fooing the bar","size":2,"startDate":"2017-12-20 00:00:00 UTC"}]}`
	now := func(time.Time) time.Time { return time.Date(2017, 12, 20, 0, 0, 0, 0, time.UTC) }
	var buf bytes.Buffer
	printWorkStatusHelper(&buf, session, "TEST", taskDetails, now)
	obtained, _ := parseTimestamp("2017-12-20 00:00:00 UTC")
	want := ".... " + printTimestamp(obtained) + " Fooing the bar 1 / 2."
	if !strings.Contains(buf.String(), want) {
		t.Fatalf("output %q does not contain %q", buf.String(), want)
	}
}

func TestPrintWorkStatusOldTask(t *testing.T) {
	session := NewSession(SessionConfig{})
	taskDetails := `{"obtained":"2016-11-22 10:43:21 UTC","batches":[{"completed":0,"description":"Fooing the bar.","size":0,"startDate":"2016-11-22 10:43:22 UTC"}]}`
	now := func(time.Time) time.Time { return time.Date(2017, 12, 20, 0, 0, 0, 0, time.UTC) }
	var buf bytes.Buffer
	printWorkStatusHelper(&buf, session, "TEST", taskDetails, now)
	obtained, _ := parseTimestamp("2016-11-22 10:43:21 UTC")
	want := ".... " + printTimestamp(obtained) + " Fooing the bar. (1y27d13:16:39 elapsed)"
	if !strings.Contains(buf.String(), want) {
		t.Fatalf("output %q does not contain %q", buf.String(), want)
	}
}

func TestPrintWorkStatusOldTaskSubtasks(t *testing.T) {
	session := NewSession(SessionConfig{})
	taskDetails := `{"obtained":"2016-11-22 10:43:21 UTC","batches":[{"completed":1,"description":"Fooing the bar","size":2,"startDate":"2016-11-22 10:43:22 UTC"}]}`
	now := func(time.Time) time.Time { return time.Date(2017, 12, 20, 0, 0, 0, 0, time.UTC) }
	var buf bytes.Buffer
	printWorkStatusHelper(&buf, session, "TEST", taskDetails, now)
	obtained, _ := parseTimestamp("2016-11-22 10:43:21 UTC")
	want := ".... " + printTimestamp(obtained) + " Fooing the bar 1 / 2. (1y27d13:16:39 elapsed)"
	if !strings.Contains(buf.String(), want) {
		t.Fatalf("output %q does not contain %q", buf.String(), want)
	}
}

func TestExecuteRequestParams(t *testing.T) {
	var bodies [][]byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		bodies = append(bodies, body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	executeAndCapture := func(s *Session, extra map[string]any) map[string]any {
		item := NewWorkItem(s)
		item.RequestParams[ArgTestrig] = "snapshot"
		if _, err := execute(context.Background(), item, s, true, extra); err != nil {
			t.Fatal(err)
		}
		var posted map[string]any
		if err := json.Unmarshal(bodies[len(bodies)-1], &posted); err != nil {
			t.Fatal(err)
		}
		params, _ := posted["requestParams"].(map[string]any)
		return params
	}

	s := sessionForServer(t, srv.URL, nil)
	s.Network = util.StringPtr("net")

	// Unmodified work item.
	if params := executeAndCapture(s, nil); params["TESTARG"] != nil {
		t.Fatalf("unexpected TESTARG: %v", params)
	}

	s.AdditionalArgs["TESTARG"] = "addl"

	// Work item picks up additional args.
	if params := executeAndCapture(s, nil); params["TESTARG"] != "addl" {
		t.Fatalf("TESTARG = %v", params["TESTARG"])
	}

	// Extra args override additional args.
	if params := executeAndCapture(s, map[string]any{"TESTARG": "extra"}); params["TESTARG"] != "extra" {
		t.Fatalf("TESTARG = %v", params["TESTARG"])
	}

	// Additional args are not mutated.
	if params := executeAndCapture(s, nil); params["TESTARG"] != "addl" {
		t.Fatalf("TESTARG = %v", params["TESTARG"])
	}
}
