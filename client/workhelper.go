package client

import (
	"context"
	"encoding/json"
	"fmt"
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
