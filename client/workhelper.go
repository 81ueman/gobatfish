package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/81ueman/gobatfish/exception"
)

// maxLogLength bounds how much of a work log is shown on failure.
const maxLogLength = 64 * 1024

// execute submits a work item to Batfish and waits for it to finish.
//
// When background is true it returns a map with a "result" key; otherwise it
// returns a map with a "status" key.
func execute(ctx context.Context, workItem *WorkItem, session *Session, background bool, extraArgs map[string]any) (map[string]any, error) {
	for k, v := range extraArgs {
		workItem.RequestParams[k] = v
	}
	snapshot, _ := workItem.RequestParams[ArgTestrig].(string)
	if snapshot == "" {
		return nil, fmt.Errorf("work item %s does not include a snapshot name", workItem.ToJSON())
	}
	if err := session.queueWork(ctx, workItem); err != nil {
		return nil, err
	}
	if background {
		return map[string]any{"result": "True"}, nil
	}

	answer, status, taskDetails, err := getWorkStatus(ctx, workItem.ID, session)
	if err != nil {
		return nil, err
	}
	curSleep := 100 * time.Millisecond
	for !status.IsTerminated() {
		time.Sleep(curSleep)
		curSleep = time.Duration(float64(curSleep) * 1.5)
		if curSleep > time.Second {
			curSleep = time.Second
		}
		answer, status, taskDetails, err = getWorkStatus(ctx, workItem.ID, session)
		if err != nil {
			return nil, err
		}
		_ = answer
	}

	if status == WorkAssignmentError || status == WorkRequeueFailure {
		return nil, exception.NewBatfishErrorf("Work finished with status %s\nwork_item: %s\ntask_details: %s",
			status, workItem.ToJSON(), taskDetails)
	}
	if status == WorkTerminatedAbnormally {
		log, err := session.getWorkLog(ctx, &snapshot, workItem.ID)
		if err != nil {
			log = ""
		}
		prefix := ""
		logMsg := ""
		if len(log) > maxLogLength {
			logMsg = fmt.Sprintf("Full log written to %s\n", "batfish-work.log")
			prefix = "..."
			log = log[len(log)-maxLogLength:]
		}
		return nil, exception.NewBatfishErrorf("Work terminated abnormally\nwork_item: %s\n\n%slog: %s%s",
			workItem.ToJSON(), logMsg, prefix, log)
	}
	return map[string]any{"status": status}, nil
}

func getWorkStatus(ctx context.Context, workItemID string, session *Session) (map[string]any, WorkStatusCode, string, error) {
	answer, err := session.getWorkStatus(ctx, workItemID)
	if err != nil {
		return nil, "", "", err
	}
	status := WorkStatusCode(fmt.Sprint(answer[PropWorkStatusCode]))
	taskDetails, _ := json.Marshal(answer[PropTask])
	return map[string]any{
		SvcKeyWorkStatus: status,
		SvcKeyTaskStatus: string(taskDetails),
	}, status, string(taskDetails), nil
}

func getWorkitemAnswer(session *Session, questionName, snapshot string, referenceSnapshot *string) *WorkItem {
	item := NewWorkItem(session)
	item.RequestParams[CommandAnswer] = ""
	item.RequestParams[ArgQuestionName] = questionName
	item.RequestParams[ArgTestrig] = snapshot
	if referenceSnapshot != nil {
		item.RequestParams[ArgDeltaTestrig] = *referenceSnapshot
		item.RequestParams[ArgDifferential] = ""
	}
	return item
}

func getWorkitemGenerateDataplane(session *Session, snapshot string) *WorkItem {
	item := NewWorkItem(session)
	item.RequestParams[CommandDumpDP] = ""
	item.RequestParams[ArgTestrig] = snapshot
	return item
}

func getWorkitemParse(session *Session, snapshot string) *WorkItem {
	item := NewWorkItem(session)
	item.RequestParams[ArgTestrig] = snapshot
	item.RequestParams[CommandParseVendorIndependent] = ""
	item.RequestParams[CommandParseVendorSpecific] = ""
	item.RequestParams[CommandInitInfo] = ""
	return item
}

// batchDesc returns a string representation of a Batfish job batch.
func batchDesc(batch map[string]any) string {
	description := strings.TrimSpace(fmt.Sprint(batch["description"]))
	size := toIntValue(batch["size"])
	if size > 0 {
		description = fmt.Sprintf("%s %v / %v", description, batch["completed"], batch["size"])
	}
	if description == "" || !strings.ContainsRune("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~", rune(description[len(description)-1])) {
		description += "."
	}
	return description
}

// parseTimestamp converts a Batfish date string into a time.Time.
func parseTimestamp(timestamp string) (time.Time, error) {
	if millis, err := strconv.ParseInt(strings.TrimSpace(timestamp), 10, 64); err == nil {
		return time.UnixMilli(millis).UTC(), nil
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.999999999-0700",
		"2006-01-02T15:04:05-0700",
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05.999999999 MST",
		"2006-01-02 15:04:05 MST",
		"2006-01-02 15:04:05",
	} {
		if t, err := time.Parse(layout, timestamp); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("could not parse timestamp %q", timestamp)
}

// printTimestamp renders a timestamp in the local timezone.
func printTimestamp(t time.Time) string {
	return t.Local().Format("2006-01-02 15:04:05.000000")
}

// RelativeDelta is a calendar-aware difference between two timestamps,
// mirroring the subset of dateutil.relativedelta used by pybatfish.
type RelativeDelta struct {
	Years   int
	Months  int
	Days    int
	Hours   int
	Minutes int
	Seconds int
}

// relativeDelta computes the calendar-aware difference from start to end.
func relativeDelta(start, end time.Time) RelativeDelta {
	var d RelativeDelta
	years := end.Year() - start.Year()
	for years > 0 && start.AddDate(years, 0, 0).After(end) {
		years--
	}
	cur := start.AddDate(years, 0, 0)
	d.Years = years

	months := (end.Year()-cur.Year())*12 + int(end.Month()) - int(cur.Month())
	for months > 0 && cur.AddDate(0, months, 0).After(end) {
		months--
	}
	cur = cur.AddDate(0, months, 0)
	d.Months = months

	remaining := end.Sub(cur)
	d.Days = int(remaining.Hours() / 24)
	remaining -= time.Duration(d.Days) * 24 * time.Hour
	d.Hours = int(remaining.Hours())
	remaining -= time.Duration(d.Hours) * time.Hour
	d.Minutes = int(remaining.Minutes())
	remaining -= time.Duration(d.Minutes) * time.Minute
	d.Seconds = int(remaining.Seconds())
	return d
}

// formatElapsedTime formats a RelativeDelta the way pybatfish does.
func formatElapsedTime(delta RelativeDelta) string {
	years := ""
	if delta.Years != 0 {
		years = fmt.Sprintf("%dy", delta.Years)
	}
	months := ""
	if delta.Months != 0 {
		months = fmt.Sprintf("%dm", delta.Months)
	}
	days := ""
	if delta.Days != 0 {
		days = fmt.Sprintf("%dd", delta.Days)
	}
	return fmt.Sprintf("%s%s%s%02d:%02d:%02d", years, months, days, delta.Hours, delta.Minutes, delta.Seconds)
}

// printWorkStatusHelper writes a human readable work status to w.
func printWorkStatusHelper(w io.Writer, session *Session, workStatus any, taskDetails string, now func(time.Time) time.Time) {
	fmt.Fprintf(w, "status: %v\n", workStatus)

	var task map[string]any
	if err := json.Unmarshal([]byte(taskDetails), &task); err != nil || task == nil {
		fmt.Fprintln(w, ".... no task information")
		return
	}
	batches, _ := task["batches"].([]any)
	if len(batches) == 0 {
		return
	}
	taskStartTime, err := parseTimestamp(fmt.Sprint(task["obtained"]))
	if err != nil {
		return
	}
	taskStartTimeStr := printTimestamp(taskStartTime)

	nowTime := now(taskStartTime)
	printElapsed := nowTime.Sub(taskStartTime).Seconds() > float64(session.ElapsedDelay)

	for _, batch := range batches[:len(batches)-1] {
		if b, ok := batch.(map[string]any); ok {
			fmt.Fprintf(w, ".... %s %s\n", taskStartTimeStr, batchDesc(b))
		}
	}
	lastBatch, _ := batches[len(batches)-1].(map[string]any)

	totalTimeStr := ""
	if printElapsed {
		totalTimeStr = fmt.Sprintf(" (%s elapsed)", formatElapsedTime(relativeDelta(taskStartTime, nowTime)))
	}
	fmt.Fprintf(w, ".... %s %s%s\n", taskStartTimeStr, batchDesc(lastBatch), totalTimeStr)
}

func printWorkStatus(session *Session, workStatus any, taskDetails string) {
	printWorkStatusHelper(log.Writer(), session, workStatus, taskDetails, func(t time.Time) time.Time {
		return time.Now()
	})
}
