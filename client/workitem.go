package client

import (
	"encoding/json"

	"github.com/81ueman/gobatfish/util"
)

// WorkItem represents a Batfish task. It mirrors WorkItem.java in the
// batfish-common-protocol module and pybatfish.client.workitem.WorkItem.
type WorkItem struct {
	// ID is the unique work item id.
	ID string
	// Network is the network the work item belongs to.
	Network *string
	// RequestParams are the parameters of the work item.
	RequestParams map[string]any
}

// NewWorkItem creates a new work item for the session.
func NewWorkItem(session *Session) *WorkItem {
	params := make(map[string]any, len(session.AdditionalArgs))
	for k, v := range session.AdditionalArgs {
		params[k] = v
	}
	return &WorkItem{ID: util.GetUUID(), Network: session.Network, RequestParams: params}
}

// ToDict returns the dictionary representation of the work item.
func (w *WorkItem) ToDict() map[string]any {
	params := map[string]any{
		"containerName": nilIfEmpty(deref(w.Network)),
		"id":            w.ID,
		"requestParams": w.RequestParams,
	}
	if trName := w.RequestParams["testrig"]; trName != nil {
		params["testrigName"] = trName
	}
	return params
}

// ToJSON returns the JSON representation of the work item.
func (w *WorkItem) ToJSON() string {
	b, err := json.Marshal(w.ToDict())
	if err != nil {
		return ""
	}
	return string(b)
}
