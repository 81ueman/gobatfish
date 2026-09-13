package answer

import (
	"fmt"

	"github.com/81ueman/gobatfish/dataframe"
)

// ColumnMetadata is metadata for a single table column.
type ColumnMetadata struct {
	Name        string `json:"name"`
	Schema      string `json:"schema"`
	Description string `json:"description"`
	IsKey       bool   `json:"isKey"`
	IsValue     bool   `json:"isValue"`
}

// NewColumnMetadata builds column metadata from a dictionary, validating the
// required fields.
func NewColumnMetadata(dictionary map[string]any) (ColumnMetadata, error) {
	if _, ok := dictionary["name"]; !ok {
		return ColumnMetadata{}, fmt.Errorf("bad column metadata: 'name' not found")
	}
	if _, ok := dictionary["schema"]; !ok {
		return ColumnMetadata{}, fmt.Errorf("bad column metadata: 'schema' not found")
	}
	description := "No description provided"
	if v, ok := dictionary["description"]; ok && v != nil {
		description = fmt.Sprint(v)
	}
	isKey := true
	if v, ok := dictionary["isKey"]; ok && v != nil {
		isKey = asBool(v)
	}
	isValue := true
	if v, ok := dictionary["isValue"]; ok && v != nil {
		isValue = asBool(v)
	}
	return ColumnMetadata{
		Name:        fmt.Sprint(dictionary["name"]),
		Schema:      fmt.Sprint(dictionary["schema"]),
		Description: description,
		IsKey:       isKey,
		IsValue:     isValue,
	}, nil
}

// TableMetadata is metadata for a Batfish table answer.
type TableMetadata struct {
	ColumnMetadata []ColumnMetadata `json:"columnMetadata"`
	Hints          any              `json:"displayHints"`
}

// NewTableMetadata builds table metadata from a dictionary.
func NewTableMetadata(dictionary map[string]any) (TableMetadata, error) {
	var columns []ColumnMetadata
	if raw, ok := dictionary["columnMetadata"].([]any); ok {
		for _, c := range raw {
			cm, err := NewColumnMetadata(asMap(c))
			if err != nil {
				return TableMetadata{}, err
			}
			columns = append(columns, cm)
		}
	}
	return TableMetadata{ColumnMetadata: columns, Hints: dictionary["displayHints"]}, nil
}

// GetColumnNames returns the ordered column names.
func (t TableMetadata) GetColumnNames() []string {
	out := make([]string, len(t.ColumnMetadata))
	for i, cm := range t.ColumnMetadata {
		out[i] = cm.Name
	}
	return out
}

// Row represents a table row.
type Row map[string]any

// TableAnswer is a Batfish answer in the form of a table.
type TableAnswer struct {
	Answer

	Metadata     TableMetadata
	Rows         []Row
	TableData    *dataframe.DataFrame
	ExcludedRows map[string][]Row
}

// NewTableAnswer builds a TableAnswer from a dictionary, validating its shape.
func NewTableAnswer(dictionary map[string]any) (*TableAnswer, error) {
	answerElements, ok := dictionary["answerElements"].([]any)
	if !ok {
		return nil, fmt.Errorf("answer elements not found in dictionary")
	}
	if len(answerElements) == 0 {
		return nil, fmt.Errorf("empty answer elements list in dictionary")
	}
	answerElement := asMap(answerElements[0])
	metadataRaw, ok := answerElement["metadata"]
	if !ok {
		return nil, fmt.Errorf("TableMetadata not found in dictionary")
	}
	metadata, err := NewTableMetadata(asMap(metadataRaw))
	if err != nil {
		return nil, err
	}

	rows := make([]Row, 0)
	if rawRows, ok := answerElement["rows"].([]any); ok {
		for _, r := range rawRows {
			rows = append(rows, Row(asMap(r)))
		}
	}

	excluded := make(map[string][]Row)
	if rawExcluded, ok := answerElement["excludedRows"].([]any); ok {
		for _, e := range rawExcluded {
			exclusion := asMap(e)
			name, ok := exclusion["exclusionName"]
			if !ok {
				return nil, fmt.Errorf("exclusion does not have 'exclusionName'")
			}
			var exclRows []Row
			if rawRows, ok := exclusion["rows"].([]any); ok {
				for _, r := range rawRows {
					exclRows = append(exclRows, Row(asMap(r)))
				}
			}
			excluded[fmt.Sprint(name)] = exclRows
		}
	}

	ta := &TableAnswer{
		Answer:       NewAnswer(dictionary),
		Metadata:     metadata,
		Rows:         rows,
		TableData:    rowsToFrame(metadata, rows),
		ExcludedRows: excluded,
	}
	return ta, nil
}

// Frame returns answer data as a dataframe.
func (t *TableAnswer) Frame() *dataframe.DataFrame { return t.TableData }

// ExcludedFrame returns the excluded data for exclusionName as a dataframe.
func (t *TableAnswer) ExcludedFrame(exclusionName string) (*dataframe.DataFrame, error) {
	rows, ok := t.ExcludedRows[exclusionName]
	if !ok {
		return nil, fmt.Errorf("exclusion name %s does not exist", exclusionName)
	}
	return rowsToFrame(t.Metadata, rows), nil
}

// Len returns the number of rows.
func (t *TableAnswer) Len() int { return t.TableData.Len() }

// String renders the table.
func (t *TableAnswer) String() string { return t.TableData.String() }

// HTML returns the HTML representation of the table.
func (t *TableAnswer) HTML() string { return t.TableData.HTML() }

// Dict returns a dictionary representation of the full answer.
func (t *TableAnswer) Dict() map[string]any { return t.Answer.Dict() }

// Table returns the answer as a TableAnswer.
func (t *TableAnswer) Table() (*TableAnswer, bool) { return t, true }

func rowsToFrame(metadata TableMetadata, rows []Row) *dataframe.DataFrame {
	order := metadata.GetColumnNames()
	data := make(map[string][]any, len(order))
	for _, cm := range metadata.ColumnMetadata {
		values := make([]any, len(rows))
		for i, row := range rows {
			parsed, err := ParseJSONWithSchema(cm.Schema, row[cm.Name])
			if err != nil {
				parsed = nil
			}
			values[i] = parsed
		}
		data[cm.Name] = values
	}
	return dataframe.New(order, data)
}

// IsTableAns reports whether a dictionary represents a table answer.
func IsTableAns(d map[string]any) bool {
	answerElements, ok := d["answerElements"].([]any)
	if !ok || len(answerElements) == 0 {
		return false
	}
	element := asMap(answerElements[0])
	return element["class"] == "org.batfish.datamodel.table.TableAnswerElement"
}

func asBool(v any) bool {
	switch b := v.(type) {
	case bool:
		return b
	case string:
		return b == "true" || b == "True" || b == "1"
	default:
		return false
	}
}
