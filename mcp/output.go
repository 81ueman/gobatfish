package mcp

import "github.com/81ueman/gobatfish/dataframe"

// TableResult is the structured output of a question-backed tool.
type TableResult struct {
	// Columns are the table column names in order.
	Columns []string `json:"columns" jsonschema:"table column names"`
	// Rows are the table rows as records.
	Rows []map[string]any `json:"rows" jsonschema:"result rows, one object per row"`
	// Count is the number of rows.
	Count int `json:"count" jsonschema:"number of rows"`
}

// NewTableResult converts a data frame into a structured table result.
func NewTableResult(df *dataframe.DataFrame) TableResult {
	rows := df.Records()
	if rows == nil {
		rows = []map[string]any{}
	}
	columns := df.Columns()
	if columns == nil {
		columns = []string{}
	}
	return TableResult{Columns: columns, Rows: rows, Count: df.Len()}
}

// NetworksResult is the output of list_networks.
type NetworksResult struct {
	Networks []string `json:"networks" jsonschema:"network names"`
}

// NetworkResult is the output of set_network.
type NetworkResult struct {
	Network string `json:"network" jsonschema:"the active network name"`
}

// SnapshotResult is the output of snapshot creation tools.
type SnapshotResult struct {
	Snapshot string `json:"snapshot" jsonschema:"the snapshot name"`
}

// SnapshotListResult is the output of list_snapshots.
type SnapshotListResult struct {
	Snapshots []string `json:"snapshots" jsonschema:"snapshot names"`
}

// DeletedResult is the output of delete tools.
type DeletedResult struct {
	Deleted string `json:"deleted" jsonschema:"the name of the deleted item"`
}

// SessionsResult is the output of list_sessions.
type SessionsResult struct {
	Sessions map[string]string `json:"sessions" jsonschema:"session name to session type"`
}

// RegisteredResult is the output of register_session.
type RegisteredResult struct {
	Registered string `json:"registered" jsonschema:"the registered session name"`
	Type       string `json:"type" jsonschema:"the session type"`
}
