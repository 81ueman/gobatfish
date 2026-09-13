// Package answer mirrors pybatfish.datamodel.answer. It provides Answer, a
// generic Batfish answer, and TableAnswer, which exposes the answer rows as a
// gobatfish dataframe.
package answer

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/81ueman/gobatfish/datamodel"
)

// Result is the common interface of Answer and TableAnswer.
type Result interface {
	// Dict returns a dictionary representation of the full answer.
	Dict() map[string]any
	// QuestionName returns the name of the question that produced the answer.
	QuestionName() *string
	// String renders the answer.
	String() string
	// Table returns the answer as a TableAnswer when it is one.
	Table() (*TableAnswer, bool)
}

var iterableSchemaPattern = regexp.MustCompile(`(?i)^(List|Set)<(.+)>$`)

// Answer represents a generic Batfish answer. It is a map, mirroring the Python
// dict subclass.
type Answer map[string]any

// NewAnswer creates an Answer from a dictionary.
func NewAnswer(dictionary map[string]any) Answer {
	if dictionary == nil {
		dictionary = map[string]any{}
	}
	return Answer(dictionary)
}

// Dict returns a dictionary representation of the full answer.
func (a Answer) Dict() map[string]any {
	out := make(map[string]any, len(a))
	for k, v := range a {
		out[k] = v
	}
	return out
}

// QuestionName returns the name of the question that produced this answer, or
// nil when unavailable.
func (a Answer) QuestionName() *string {
	q, ok := a["question"].(map[string]any)
	if !ok {
		return nil
	}
	inst, ok := q["instance"].(map[string]any)
	if !ok {
		return nil
	}
	name, ok := inst["instanceName"]
	if !ok || name == nil {
		return nil
	}
	s := fmt.Sprint(name)
	return &s
}

// String renders the answer as indented JSON.
func (a Answer) String() string {
	b, err := json.MarshalIndent(map[string]any(a), "", "  ")
	if err != nil {
		return fmt.Sprint(map[string]any(a))
	}
	return string(b)
}

// Table returns the answer as a TableAnswer when it is one.
func (a Answer) Table() (*TableAnswer, bool) { return nil, false }

// GetBaseSchema returns the underlying base schema for an iterable (list or
// set) schema.
func GetBaseSchema(schema string) string {
	if m := iterableSchemaPattern.FindStringSubmatch(schema); m != nil {
		return m[2]
	}
	return schema
}

// IsIterableSchema reports whether schema is an iterable/container schema.
func IsIterableSchema(schema string) bool {
	return iterableSchemaPattern.MatchString(schema)
}

// ParseJSONWithSchema processes a JSON object according to its schema, mirroring
// pybatfish.datamodel.answer.base._parse_json_with_schema.
func ParseJSONWithSchema(schema string, jsonObject any) (any, error) {
	if jsonObject == nil {
		return nil, nil
	}
	if IsIterableSchema(schema) {
		list, ok := jsonObject.([]any)
		if !ok {
			return nil, fmt.Errorf("got non-list value for list/set schema %s. Value: %v", schema, jsonObject)
		}
		baseSchema := GetBaseSchema(schema)
		if baseSchema == "TraceTree" {
			trees := make(datamodel.TraceTreeList, 0, len(list))
			for _, element := range list {
				parsed, err := ParseJSONWithSchema(baseSchema, element)
				if err != nil {
					return nil, err
				}
				trees = append(trees, parsed.(datamodel.TraceTree))
			}
			return trees, nil
		}
		out := make([]any, 0, len(list))
		for _, element := range list {
			parsed, err := ParseJSONWithSchema(baseSchema, element)
			if err != nil {
				return nil, err
			}
			out = append(out, parsed)
		}
		return datamodel.NewListWrapper(out), nil
	}

	switch schema {
	case "AclTrace":
		return datamodel.AclTraceFromDict(asMap(jsonObject)), nil
	case "FileLines":
		return datamodel.FileLinesFromDict(asMap(jsonObject)), nil
	case "Flow":
		return datamodel.FlowFromDict(asMap(jsonObject)), nil
	case "FlowTrace":
		return datamodel.FlowTraceFromDict(asMap(jsonObject)), nil
	case "Integer", "Long":
		return toInt(jsonObject), nil
	case "Interface":
		return datamodel.InterfaceFromDict(asMap(jsonObject)), nil
	case "Ip":
		return fmt.Sprint(jsonObject), nil
	case "NextHop":
		nh, err := datamodel.NextHopFromDict(asMap(jsonObject))
		if err != nil {
			return nil, err
		}
		return nh, nil
	case "Node":
		return asMap(jsonObject)["name"], nil
	case "BgpRoute":
		return datamodel.BgpRouteFromDict(asMap(jsonObject)), nil
	case "BgpRouteDiffs":
		return datamodel.BgpRouteDiffsFromDict(asMap(jsonObject)), nil
	case "Prefix":
		return fmt.Sprint(jsonObject), nil
	case "SelfDescribing":
		m := asMap(jsonObject)
		return ParseJSONWithSchema(fmt.Sprint(m["schema"]), m["value"])
	case "String":
		return fmt.Sprint(jsonObject), nil
	case "Trace":
		return datamodel.TraceFromDict(asMap(jsonObject)), nil
	case "TraceTree":
		return datamodel.TraceTreeFromDict(asMap(jsonObject)), nil
	default:
		return jsonObject, nil
	}
}

func asMap(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	if dm, ok := v.(datamodel.DataModelElement); ok {
		return dm.Dict()
	}
	return nil
}

func toInt(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	case string:
		var i int
		fmt.Sscan(n, &i)
		return i
	default:
		return 0
	}
}
