// Package question mirrors pybatfish.question: Batfish questions and the logic
// for loading them from disk or from the Batfish service.
//
// pybatfish generates a Python class per question template, exposing dynamic
// attributes such as session.q.bgpSessionStatus. Go has no dynamic attributes,
// so questions are held in a registry: session.Q.Get("bgpSessionStatus")
// returns a fresh *Question that can be configured with Set and answered.
package question

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/81ueman/gobatfish/datamodel"
	"github.com/81ueman/gobatfish/datamodel/answer"
	"github.com/81ueman/gobatfish/exception"
	"github.com/81ueman/gobatfish/util"
)

// Session is the subset of the client Session that questions need. It is
// declared here to avoid an import cycle between the question and client
// packages.
type Session interface {
	// GetSnapshot returns the snapshot to answer against.
	GetSnapshot(snapshot *string) (string, error)
	// AnswerQuestion uploads, executes and returns the answer for a question.
	AnswerQuestion(ctx context.Context, questionStr, questionName string, background bool, snapshot string, referenceSnapshot *string, extraArgs map[string]any) (answer.Result, error)
	// FetchQuestionTemplates fetches question templates from the backend.
	FetchQuestionTemplates(ctx context.Context, verbose bool) (map[string]string, error)
}

// AllowedValue describes a whitelisted value for a question parameter.
type AllowedValue struct {
	Name        string
	Description *string
}

// AllowedValueFromDict builds an AllowedValue from a dictionary.
func AllowedValueFromDict(d map[string]any) AllowedValue {
	return AllowedValue{Name: fmt.Sprint(d["name"]), Description: optString(d, "description")}
}

// String renders the allowed value.
func (a AllowedValue) String() string {
	if a.Description != nil {
		return fmt.Sprintf("%s: %s", a.Name, *a.Description)
	}
	return a.Name
}

// QuestionTemplate is the template from which Question instances are created.
type QuestionTemplate struct {
	session         Session
	name            string
	description     string
	longDescription string
	tags            []string
	variables       []string
	template        map[string]any
	docstring       string
}

// Name returns the question name.
func (t *QuestionTemplate) Name() string { return t.name }

// Description returns the question description.
func (t *QuestionTemplate) Description() string { return t.description }

// LongDescription returns the question long description.
func (t *QuestionTemplate) LongDescription() string { return t.longDescription }

// Tags returns the question tags.
func (t *QuestionTemplate) Tags() []string { return append([]string(nil), t.tags...) }

// Variables returns the ordered variable names.
func (t *QuestionTemplate) Variables() []string { return append([]string(nil), t.variables...) }

// Docstring returns the generated docstring.
func (t *QuestionTemplate) Docstring() string { return t.docstring }

// Template returns a copy of the raw question template.
func (t *QuestionTemplate) Template() map[string]any { return deepCopyMap(t.template) }

// New creates a fresh Question instance from the template.
func (t *QuestionTemplate) New() *Question {
	q := &Question{
		session:     t.session,
		name:        t.name,
		description: t.description,
		variables:   append([]string(nil), t.variables...),
		template:    deepCopyMap(t.template),
	}
	instance := ensureMap(q.template, "instance")
	existing := fmt.Sprint(instance["instanceName"])
	if existing == "" {
		existing = t.name
	}
	instance["instanceName"] = fmt.Sprintf("__%s_%s", existing, util.GetUUID())
	return q
}

// Question is a concrete, answerable Batfish question.
type Question struct {
	session     Session
	name        string
	description string
	variables   []string
	template    map[string]any
	err         error
}

// Err returns the first validation error recorded by Set.
func (q *Question) Err() error { return q.err }

// Set sets a question variable (or the special question_name) and returns the
// question for chaining. Unsupported variables are recorded and returned by
// Err and Answer.
func (q *Question) Set(name string, value any) *Question {
	if q.err != nil {
		return q
	}
	instance := ensureMap(q.template, "instance")
	instanceVars := ensureMap(instance, "variables")

	switch name {
	case "exclusions":
		q.template["exclusions"] = value
		return q
	case "question_name":
		instance["instanceName"] = value
		return q
	}

	if _, ok := instanceVars[name]; !ok {
		q.err = exception.NewQuestionValidationErrorf("Received unsupported parameters/variables: {%s}", name)
		return q
	}
	varData := ensureMap(instanceVars, name)
	varData["value"] = value
	return q
}

// SetVar is an alias for Set.
func (q *Question) SetVar(name string, value any) *Question { return q.Set(name, value) }

// Name returns the question name.
func (q *Question) Name() string {
	return fmt.Sprint(ensureMap(q.template, "instance")["instanceName"])
}

// Description returns the question description.
func (q *Question) Description() string {
	if v, ok := ensureMap(q.template, "instance")["description"]; ok {
		return fmt.Sprint(v)
	}
	return q.description
}

// LongDescription returns the question long description.
func (q *Question) LongDescription() string {
	if v, ok := ensureMap(q.template, "instance")["longDescription"]; ok {
		return fmt.Sprint(v)
	}
	return ""
}

// Differential reports whether the question is a differential question.
func (q *Question) Differential() bool {
	return boolValue(q.template["differential"])
}

// IncludeOneTableKeys reports whether keys present in only one table are included.
func (q *Question) IncludeOneTableKeys() bool {
	return boolValue(q.template["includeOneTableKeys"])
}

// SetIncludeOneTableKeys sets includeOneTableKeys.
func (q *Question) SetIncludeOneTableKeys(v bool) { q.template["includeOneTableKeys"] = v }

// Dict returns the dictionary representing the question.
func (q *Question) Dict() map[string]any { return q.template }

// JSON returns the JSON string representing the question.
func (q *Question) JSON() (string, error) {
	b, err := json.MarshalIndent(datamodel.ToJSONValue(q.template), "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// SetAssertion sets an assertion for the question, overwriting any previous one.
func (q *Question) SetAssertion(assertion datamodel.Assertion) *Question {
	q.template["assertion"] = assertion.Dict()
	return q
}

// MakeCheck makes the question a check asserting that there are no results.
func (q *Question) MakeCheck() *Question {
	return q.SetAssertion(datamodel.Assertion{Type: datamodel.AssertionCountEquals, Expect: 0})
}

// Validate validates the question, mirroring pybatfish's _validate.
func (q *Question) Validate() error { return validate(q.template) }

// AnswerOptions configures Question.Answer.
type AnswerOptions struct {
	// Snapshot is the snapshot on which to answer the question. When nil the
	// active snapshot is used.
	Snapshot *string
	// ReferenceSnapshot is required for differential questions.
	ReferenceSnapshot *string
	// IncludeOneTableKeys, when non-nil, sets includeOneTableKeys.
	IncludeOneTableKeys *bool
	// Background returns the work item id instead of the answer.
	Background bool
	// ExtraArgs are extra arguments to pass with the question.
	ExtraArgs map[string]any
}

// Answer asks and returns the answer for this question.
func (q *Question) Answer(ctx context.Context, opts AnswerOptions) (answer.Result, error) {
	if q.err != nil {
		return nil, q.err
	}
	if q.session == nil {
		return nil, exception.NewBatfishError("question has no session")
	}
	realSnapshot, err := q.session.GetSnapshot(opts.Snapshot)
	if err != nil {
		return nil, err
	}
	if opts.ReferenceSnapshot == nil && q.Differential() {
		return nil, fmt.Errorf("reference_snapshot argument is required to answer a differential question")
	}
	if err := q.Validate(); err != nil {
		return nil, err
	}
	if opts.IncludeOneTableKeys != nil {
		q.SetIncludeOneTableKeys(*opts.IncludeOneTableKeys)
	}
	questionStr, err := q.JSON()
	if err != nil {
		return nil, err
	}
	return q.session.AnswerQuestion(ctx, questionStr, q.Name(), opts.Background, realSnapshot, opts.ReferenceSnapshot, opts.ExtraArgs)
}

// Questions holds and manages Batfish questions.
type Questions struct {
	session   Session
	templates map[string]*QuestionTemplate
	order     []string
}

// NewQuestions creates an empty Questions registry for the session.
func NewQuestions(session Session) *Questions {
	return &Questions{session: session, templates: make(map[string]*QuestionTemplate)}
}

// Get returns a fresh Question for the named template.
func (q *Questions) Get(name string) (*Question, error) {
	t, ok := q.templates[name]
	if !ok {
		return nil, exception.NewBatfishErrorf("%s question was not found", name)
	}
	return t.New(), nil
}

// MustGet returns a fresh Question or panics if the question is unknown.
func (q *Questions) MustGet(name string) *Question {
	question, err := q.Get(name)
	if err != nil {
		panic(err)
	}
	return question
}

// Has reports whether a question template is registered.
func (q *Questions) Has(name string) bool {
	_, ok := q.templates[name]
	return ok
}

// Names returns the registered question names.
func (q *Questions) Names() []string {
	return append([]string(nil), q.order...)
}

// List lists available questions, optionally filtered by tags.
func (q *Questions) List(tags ...string) []map[string]any {
	desired := make(map[string]bool, len(tags))
	for _, t := range tags {
		desired[strings.ToLower(t)] = true
	}
	var out []map[string]any
	for _, name := range q.order {
		t := q.templates[name]
		if len(desired) > 0 {
			matched := false
			for _, tag := range t.tags {
				if desired[strings.ToLower(tag)] {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}
		}
		out = append(out, map[string]any{"name": name, "description": t.description, "tags": append([]string(nil), t.tags...)})
	}
	return out
}

// ListTags returns the set of tags across all available questions.
func (q *Questions) ListTags() map[string]struct{} {
	out := make(map[string]struct{})
	for _, t := range q.templates {
		for _, tag := range t.tags {
			out[tag] = struct{}{}
		}
	}
	return out
}

// Install registers the given templates.
func (q *Questions) Install(templates ...*QuestionTemplate) {
	for _, t := range templates {
		if _, ok := q.templates[t.name]; !ok {
			q.order = append(q.order, t.name)
		}
		q.templates[t.name] = t
	}
}

// Load loads questions from a local directory, or from the Batfish service
// when directory is empty.
func (q *Questions) Load(ctx context.Context, directory string) error {
	if directory != "" {
		loaded := LoadQuestionsFromDir(directory, q.session)
		q.Install(mapTemplates(loaded)...)
		return nil
	}
	remote, err := LoadRemoteQuestionsTemplates(ctx, q.session)
	if err != nil {
		return err
	}
	q.Install(mapTemplates(remote)...)
	return nil
}

func mapTemplates(m map[string]*QuestionTemplate) []*QuestionTemplate {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]*QuestionTemplate, 0, len(names))
	for _, name := range names {
		out = append(out, m[name])
	}
	return out
}

// LoadQuestionsFromDir loads question templates from .json files in a directory.
func LoadQuestionsFromDir(questionDir string, session Session) map[string]*QuestionTemplate {
	out := make(map[string]*QuestionTemplate)
	var files []string
	_ = filepath.Walk(questionDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".json") {
			files = append(files, path)
		}
		return nil
	})
	if len(files) == 0 {
		return out
	}
	for _, path := range files {
		questionDict, err := readJSONFile(path)
		if err != nil {
			continue
		}
		name, tmpl, err := LoadQuestionDict(questionDict, session)
		if err != nil {
			continue
		}
		out[name] = tmpl
	}
	return out
}

// LoadRemoteQuestionsTemplates fetches and parses question templates from the
// Batfish service.
func LoadRemoteQuestionsTemplates(ctx context.Context, session Session) (map[string]*QuestionTemplate, error) {
	templates, err := session.FetchQuestionTemplates(ctx, false)
	if err != nil {
		return nil, err
	}
	out := make(map[string]*QuestionTemplate)
	for _, value := range templates {
		var questionDict map[string]any
		if err := json.Unmarshal([]byte(value), &questionDict); err != nil {
			continue
		}
		name, tmpl, err := LoadQuestionDict(questionDict, session)
		if err != nil {
			continue
		}
		out[name] = tmpl
	}
	return out, nil
}

// LoadQuestionDict creates a question template from a dictionary.
func LoadQuestionDict(question map[string]any, session Session) (string, *QuestionTemplate, error) {
	instanceData := mapField(question, "instance")
	if instanceData == nil {
		return "", nil, exception.NewQuestionValidationError("Missing instance data")
	}
	givenName := fmt.Sprint(instanceData["instanceName"])
	if instanceData["instanceName"] == nil {
		givenName = ""
	}
	if givenName == "" || util.ValidateQuestionName(givenName) != nil {
		return "", nil, exception.NewQuestionValidationErrorf("Invalid question name: %s", givenName)
	}
	questionName := givenName

	description := strings.TrimSpace(fmt.Sprint(instanceData["description"]))
	if description == "" {
		return "", nil, exception.NewQuestionValidationErrorf("Missing description for question '%s'", questionName)
	}
	if !strings.HasSuffix(description, ".") {
		description += "."
	}
	longDescription := strings.TrimSpace(fmt.Sprint(instanceData["longDescription"]))
	if longDescription != "" {
		if !strings.HasSuffix(longDescription, ".") {
			longDescription += "."
		}
		description = strings.Join([]string{description, longDescription}, "\n\n")
	}

	var tags []string
	if rawTags, ok := instanceData["tags"].([]any); ok {
		for _, t := range rawTags {
			tags = append(tags, fmt.Sprint(t))
		}
	}
	sort.Strings(tags)

	ivars := mapField(instanceData, "variables")
	orderedVariableNames := stringSlice(instanceData["orderedVariableNames"])
	variables, err := processVariables(questionName, ivars, orderedVariableNames)
	if err != nil {
		return "", nil, err
	}

	docstring := computeDocstring(description, variables, ivars)

	tmpl := &QuestionTemplate{
		session:         session,
		name:            questionName,
		description:     description,
		longDescription: longDescription,
		tags:            tags,
		variables:       variables,
		template:        deepCopyMap(question),
		docstring:       docstring,
	}
	return questionName, tmpl, nil
}

// --- helpers ----------------------------------------------------------------

func optString(d map[string]any, key string) *string {
	if d == nil {
		return nil
	}
	v, ok := d[key]
	if !ok || v == nil {
		return nil
	}
	s := fmt.Sprint(v)
	return &s
}

func boolValue(v any) bool {
	b, ok := v.(bool)
	return ok && b
}

func ensureMap(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	if existing, ok := m[key].(map[string]any); ok {
		return existing
	}
	created := make(map[string]any)
	m[key] = created
	return created
}

func mapField(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}
	if v, ok := m[key].(map[string]any); ok {
		return v
	}
	return nil
}

func stringSlice(v any) []string {
	vals, ok := v.([]any)
	if !ok {
		return nil
	}
	out := make([]string, len(vals))
	for i, item := range vals {
		out[i] = fmt.Sprint(item)
	}
	return out
}

func deepCopyMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = deepCopyValue(v)
	}
	return out
}

func deepCopyValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return deepCopyMap(t)
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = deepCopyValue(item)
		}
		return out
	default:
		return v
	}
}

func readJSONFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
