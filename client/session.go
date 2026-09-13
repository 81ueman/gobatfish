package client

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/81ueman/gobatfish/dataframe"
	"github.com/81ueman/gobatfish/datamodel"
	"github.com/81ueman/gobatfish/datamodel/answer"
	"github.com/81ueman/gobatfish/exception"
	"github.com/81ueman/gobatfish/question"
	"github.com/81ueman/gobatfish/util"
)

// SessionConfig configures a Session. Zero values fall back to the same
// defaults as pybatfish.
type SessionConfig struct {
	Host           string
	Port           *int
	PortV1         int
	PortV2         *int
	SSL            *bool
	VerifySSLCerts *bool
	APIKey         string
	Proxies        map[string]string
	Timeout        *time.Duration
	RequestKwargs  map[string]any
	AdditionalArgs map[string]any
	MaxRetries     *int
}

// Session keeps the configuration needed to connect to a Batfish server.
type Session struct {
	Host           string
	PortV1         int
	PortV2         int
	SSL            bool
	VerifySSLCerts bool
	APIKey         string
	Network        *string
	Snapshot       *string
	AdditionalArgs map[string]any
	ElapsedDelay   int
	StaleTimeout   int
	Proxies        map[string]string
	Timeout        *time.Duration
	RequestKwargs  map[string]any

	Q       *question.Questions
	Asserts *Asserts

	httpClient *http.Client
	maxRetries *int
}

// NewSession creates a Session. Unlike pybatfish it does not load questions
// implicitly; call LoadQuestions to fetch them from the backend.
func NewSession(cfg SessionConfig) *Session {
	host := cfg.Host
	if host == "" {
		host = CoordinatorHost
	}
	portV2 := CoordinatorWorkV2Port
	if cfg.Port != nil {
		portV2 = *cfg.Port
	} else if cfg.PortV2 != nil {
		portV2 = *cfg.PortV2
	}
	ssl := UseSSL
	if cfg.SSL != nil {
		ssl = *cfg.SSL
	}
	verify := VerifySSLCerts
	if cfg.VerifySSLCerts != nil {
		verify = *cfg.VerifySSLCerts
	}
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = DefaultAPIKey
	}
	timeout := DefaultTimeout
	if cfg.Timeout != nil {
		if *cfg.Timeout > 0 {
			timeout = *cfg.Timeout
		} else {
			timeout = 0
		}
	}
	var sessionTimeout *time.Duration
	if timeout > 0 {
		sessionTimeout = &timeout
	}

	transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	if len(cfg.Proxies) > 0 {
		transport.Proxy = proxyFunc(cfg.Proxies)
	}
	if !verify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	client := &http.Client{Transport: transport}
	if timeout > 0 {
		client.Timeout = timeout
	}

	s := &Session{
		Host:           host,
		PortV1:         cfg.PortV1,
		PortV2:         portV2,
		SSL:            ssl,
		VerifySSLCerts: verify,
		APIKey:         apiKey,
		AdditionalArgs: cfg.AdditionalArgs,
		ElapsedDelay:   5,
		StaleTimeout:   5,
		Proxies:        cfg.Proxies,
		Timeout:        sessionTimeout,
		RequestKwargs:  cfg.RequestKwargs,
		httpClient:     client,
		maxRetries:     cfg.MaxRetries,
	}
	if s.AdditionalArgs == nil {
		s.AdditionalArgs = map[string]any{}
	}
	if s.RequestKwargs == nil {
		s.RequestKwargs = map[string]any{}
	}
	s.Q = question.NewQuestions(s)
	s.Asserts = &Asserts{session: s}
	return s
}

func proxyFunc(proxies map[string]string) func(*http.Request) (*url.URL, error) {
	return func(req *http.Request) (*url.URL, error) {
		if p, ok := proxies[req.URL.Scheme]; ok {
			return url.Parse(p)
		}
		return nil, nil
	}
}

// GetRequestKwargs returns merged request options, mirroring
// Session._get_request_kwargs.
func (s *Session) GetRequestKwargs() map[string]any {
	merged := make(map[string]any, len(s.RequestKwargs)+2)
	for k, v := range s.RequestKwargs {
		merged[k] = v
	}
	if s.Proxies != nil {
		merged["proxies"] = s.Proxies
	}
	if s.Timeout != nil {
		merged["timeout"] = s.Timeout.Seconds()
	}
	return merged
}

// LoadQuestions loads question templates from the backend.
func (s *Session) LoadQuestions(ctx context.Context) error {
	return s.Q.Load(ctx, "")
}

// LoadQuestionsFromDir loads question templates from a local directory.
func (s *Session) LoadQuestionsFromDir(ctx context.Context, directory string) error {
	return s.Q.Load(ctx, directory)
}

// FetchQuestionTemplates fetches raw question templates from the backend.
func (s *Session) FetchQuestionTemplates(ctx context.Context, verbose bool) (map[string]string, error) {
	raw, err := s.getQuestionTemplates(ctx, verbose)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(raw))
	for k, v := range raw {
		out[k] = fmt.Sprint(v)
	}
	return out, nil
}

// GetBFVersion returns the Batfish backend version.
func (s *Session) GetBFVersion(ctx context.Context) (string, error) {
	versions, err := s.GetComponentVersions(ctx)
	if err != nil {
		return "", err
	}
	v, ok := versions["Batfish"]
	if !ok || v == nil || fmt.Sprint(v) == "" {
		return "", exception.NewBatfishError("backend did not return a version for 'Batfish'")
	}
	return fmt.Sprint(v), nil
}

// GetComponentVersions returns backend component versions.
func (s *Session) GetComponentVersions(ctx context.Context) (map[string]any, error) {
	return s.getComponentVersions(ctx)
}

// GetSnapshot returns the specified or active snapshot name.
func (s *Session) GetSnapshot(snapshot *string) (string, error) {
	if snapshot != nil {
		return *snapshot, nil
	}
	if s.Snapshot != nil {
		return *s.Snapshot, nil
	}
	return "", fmt.Errorf("snapshot must be either provided or set using set_snapshot (e.g. bf.set_snapshot('NAME')")
}

// SetNetwork configures the network used for analysis and returns its name.
func (s *Session) SetNetwork(ctx context.Context, name string) (string, error) {
	if name == "" {
		name = DefaultNetworkPrefix + util.GetUUID()
	}
	if err := util.ValidateName(name, "network"); err != nil {
		return "", err
	}
	net, err := s.getNetwork(ctx, name)
	if err == nil {
		s.Network = util.StringPtr(fmt.Sprint(net["name"]))
		return *s.Network, nil
	}
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != http.StatusNotFound {
		return "", exception.WrapBatfishError("unknown error accessing network", err)
	}
	if err := s.initNetwork(ctx, name); err != nil {
		return "", err
	}
	s.Network = util.StringPtr(name)
	return name, nil
}

// DeleteNetwork deletes a network by name.
func (s *Session) DeleteNetwork(ctx context.Context, name string) error {
	if name == "" {
		return fmt.Errorf("network to be deleted must be supplied")
	}
	return s.deleteNetwork(ctx, name)
}

// ListNetworks lists networks the API key can access.
func (s *Session) ListNetworks(ctx context.Context) ([]string, error) {
	networks, err := s.listNetworks(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(networks))
	for _, n := range networks {
		out = append(out, fmt.Sprint(n["name"]))
	}
	return out, nil
}

// ListSnapshots lists snapshots for the current network.
func (s *Session) ListSnapshots(ctx context.Context, verbose bool) ([]any, error) {
	return s.listSnapshots(ctx, verbose)
}

// snapshotNames returns snapshot names from a list that may contain metadata.
func snapshotNames(snapshots []any) []string {
	out := make([]string, 0, len(snapshots))
	for _, snap := range snapshots {
		if m, ok := snap.(map[string]any); ok {
			out = append(out, fmt.Sprint(m["name"]))
		} else {
			out = append(out, fmt.Sprint(snap))
		}
	}
	return out
}

// SetSnapshot sets the current snapshot by name or index.
func (s *Session) SetSnapshot(ctx context.Context, name string, index *int) (string, error) {
	if name == "" && index == nil {
		return "", fmt.Errorf("one of name and index must be set")
	}
	if name != "" && index != nil {
		return "", fmt.Errorf("only one of name and index can be set")
	}
	snapshots, err := s.ListSnapshots(ctx, false)
	if err != nil {
		return "", err
	}
	names := snapshotNames(snapshots)
	if index != nil {
		i := *index
		if i < -len(names) || i >= len(names) {
			return "", fmt.Errorf("server has only %d snapshots: %v", len(names), names)
		}
		if i < 0 {
			i += len(names)
		}
		s.Snapshot = util.StringPtr(names[i])
	} else {
		found := false
		for _, n := range names {
			if n == name {
				found = true
				break
			}
		}
		if !found {
			return "", fmt.Errorf("no snapshot named %s was found in network %s: %v", name, deref(s.Network), names)
		}
		s.Snapshot = util.StringPtr(name)
	}
	return *s.Snapshot, nil
}

// InitSnapshotOptions configures InitSnapshot.
type InitSnapshotOptions struct {
	Name       string
	Overwrite  bool
	ExtraArgs  map[string]any
	Background bool
}

// InitSnapshot initializes a new snapshot from a zip file or directory.
func (s *Session) InitSnapshot(ctx context.Context, upload string, opts InitSnapshotOptions) (string, error) {
	name, err := s.initSnapshot(ctx, upload, opts)
	if err != nil {
		return "", err
	}
	return name, nil
}

func (s *Session) initSnapshot(ctx context.Context, upload string, opts InitSnapshotOptions) (string, error) {
	name, err := s.prepareSnapshot(ctx, opts)
	if err != nil {
		return "", err
	}
	data, err := snapshotBytes(upload)
	if err != nil {
		return "", err
	}
	if err := s.uploadSnapshot(ctx, name, data); err != nil {
		return "", err
	}
	return s.parseSnapshot(ctx, name, opts.Background, opts.ExtraArgs)
}

func (s *Session) initSnapshotData(ctx context.Context, data []byte, opts InitSnapshotOptions) (string, error) {
	name, err := s.prepareSnapshot(ctx, opts)
	if err != nil {
		return "", err
	}
	if err := s.uploadSnapshot(ctx, name, data); err != nil {
		return "", err
	}
	return s.parseSnapshot(ctx, name, opts.Background, opts.ExtraArgs)
}

// prepareSnapshot ensures a network is set, generates a name when needed,
// validates it, and checks for an existing snapshot.
func (s *Session) prepareSnapshot(ctx context.Context, opts InitSnapshotOptions) (string, error) {
	if s.Network == nil {
		if _, err := s.SetNetwork(ctx, ""); err != nil {
			return "", err
		}
	}
	name := opts.Name
	if name == "" {
		name = DefaultSnapshotPrefix + util.GetUUID()
	}
	if err := util.ValidateName(name, "snapshot"); err != nil {
		return "", err
	}
	if err := s.checkSnapshotOverwrite(ctx, name, opts.Overwrite); err != nil {
		return "", err
	}
	return name, nil
}

// InitSnapshotFromTextOptions configures InitSnapshotFromText.
type InitSnapshotFromTextOptions struct {
	Filename   string
	Name       string
	Platform   string
	Overwrite  bool
	ExtraArgs  map[string]any
	Background bool
}

// InitSnapshotFromText initializes a snapshot of a single configuration file
// with the given text.
func (s *Session) InitSnapshotFromText(ctx context.Context, text string, opts InitSnapshotFromTextOptions) (string, error) {
	filename := opts.Filename
	if filename == "" {
		filename = "config"
	}
	data, err := createInMemoryZip(text, filename, opts.Platform)
	if err != nil {
		return "", err
	}
	return s.initSnapshotData(ctx, data, InitSnapshotOptions{
		Name:       opts.Name,
		Overwrite:  opts.Overwrite,
		ExtraArgs:  opts.ExtraArgs,
		Background: opts.Background,
	})
}

func (s *Session) checkSnapshotOverwrite(ctx context.Context, name string, overwrite bool) error {
	snapshots, err := s.ListSnapshots(ctx, false)
	if err != nil {
		return err
	}
	for _, existing := range snapshotNames(snapshots) {
		if existing != name {
			continue
		}
		if overwrite {
			return s.deleteSnapshot(ctx, name, deref(s.Network))
		}
		return fmt.Errorf("a snapshot named %s already exists in network %s. Use overwrite = True if you want to overwrite the existing snapshot", name, deref(s.Network))
	}
	return nil
}

// snapshotBytes reads snapshot data from a directory, file or already-created
// zip bytes.
func snapshotBytes(upload string) ([]byte, error) {
	info, err := os.Stat(upload)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		var buf bytes.Buffer
		if err := util.ZipDir(upload, &buf); err != nil {
			return nil, err
		}
		return buf.Bytes(), nil
	}
	data, err := os.ReadFile(upload)
	if err != nil {
		return nil, err
	}
	if _, err := zip.NewReader(bytes.NewReader(data), int64(len(data))); err != nil {
		return nil, fmt.Errorf("%s is not a valid zip file", upload)
	}
	return data, nil
}

// DeleteSnapshot deletes the specified snapshot from the current network.
func (s *Session) DeleteSnapshot(ctx context.Context, name string) error {
	if s.Network == nil {
		return fmt.Errorf("network is not set")
	}
	if name == "" {
		return fmt.Errorf("snapshot to be deleted must be supplied")
	}
	return s.deleteSnapshot(ctx, name, *s.Network)
}

// DeleteNetworkObject deletes the network object with the specified key.
func (s *Session) DeleteNetworkObject(ctx context.Context, key string) error {
	return s.deleteNetworkObject(ctx, key)
}

// DeleteNodeRoleDimension deletes a node role dimension.
func (s *Session) DeleteNodeRoleDimension(ctx context.Context, dimension string) error {
	return s.deleteNodeRoleDimension(ctx, dimension)
}

// DeleteReferenceBook deletes a reference book.
func (s *Session) DeleteReferenceBook(ctx context.Context, name string) error {
	return s.deleteReferenceBook(ctx, name)
}

// DeleteSnapshotObject deletes a snapshot object.
func (s *Session) DeleteSnapshotObject(ctx context.Context, key string, snapshot *string) error {
	return s.deleteSnapshotObject(ctx, key, snapshot)
}

// GenerateDataplane generates the data plane for the supplied snapshot.
func (s *Session) GenerateDataplane(ctx context.Context, snapshot *string, extraArgs map[string]any) (string, error) {
	snapshotName, err := s.GetSnapshot(snapshot)
	if err != nil {
		return "", err
	}
	item := getWorkitemGenerateDataplane(s, snapshotName)
	res, err := execute(ctx, item, s, false, extraArgs)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(res["status"]), nil
}

// AnswerQuestion uploads, executes and returns the answer for a question.
func (s *Session) AnswerQuestion(ctx context.Context, questionStr, questionName string, background bool, snapshot string, referenceSnapshot *string, extraArgs map[string]any) (answer.Result, error) {
	if questionName == "" {
		questionName = DefaultQuestionPrefix + "_" + util.GetUUID()
	}
	if err := s.uploadQuestion(ctx, questionName, questionStr); err != nil {
		return nil, err
	}
	item := getWorkitemAnswer(s, questionName, snapshot, referenceSnapshot)
	if _, err := execute(ctx, item, s, background, extraArgs); err != nil {
		return nil, err
	}
	if background {
		return answer.NewAnswer(map[string]any{"workItemId": item.ID}), nil
	}
	return s.GetAnswer(ctx, questionName, snapshot, referenceSnapshot)
}

// GetAnswer gets the answer for a previously asked question.
func (s *Session) GetAnswer(ctx context.Context, questionName, snapshot string, referenceSnapshot *string) (answer.Result, error) {
	params := url.Values{"snapshot": []string{snapshot}}
	if referenceSnapshot != nil {
		params.Set("referenceSnapshot", *referenceSnapshot)
	}
	ans, err := s.getAnswer(ctx, questionName, params)
	if err != nil {
		return nil, err
	}
	if answer.IsTableAns(ans) {
		return answer.NewTableAnswer(ans)
	}
	return answer.NewAnswer(ans), nil
}

// Ask configures the named question with vars, answers it, and returns the
// answer table as a data frame. It is a convenience for programmatic callers
// (such as the MCP server) that do not want to manage question objects.
func (s *Session) Ask(ctx context.Context, questionName string, vars map[string]any, snapshot, referenceSnapshot *string) (*dataframe.DataFrame, error) {
	q, err := s.Q.Get(questionName)
	if err != nil {
		return nil, err
	}
	for name, value := range vars {
		q.Set(name, value)
	}
	result, err := q.Answer(ctx, question.AnswerOptions{Snapshot: snapshot, ReferenceSnapshot: referenceSnapshot})
	if err != nil {
		return nil, err
	}
	table, ok := result.Table()
	if !ok {
		return nil, fmt.Errorf("%s did not return a table answer", questionName)
	}
	return table.Frame(), nil
}

// GetNodeRoles returns node roles definitions for the active network or
// inferred roles for the active snapshot.
func (s *Session) GetNodeRoles(ctx context.Context, inferred bool) (datamodel.NodeRolesData, error) {
	if inferred {
		if s.Snapshot == nil {
			return datamodel.NodeRolesData{}, fmt.Errorf("snapshot is not set")
		}
		raw, err := s.getSnapshotInferredNodeRoles(ctx, nil)
		if err != nil {
			return datamodel.NodeRolesData{}, err
		}
		return datamodel.NodeRolesDataFromDict(raw), nil
	}
	raw, err := s.getNodeRoles(ctx)
	if err != nil {
		return datamodel.NodeRolesData{}, err
	}
	return datamodel.NodeRolesDataFromDict(raw), nil
}

// GetReferenceBook returns the specified reference book for the active network.
func (s *Session) GetReferenceBook(ctx context.Context, name string) (datamodel.ReferenceBook, error) {
	raw, err := s.getReferenceBook(ctx, name)
	if err != nil {
		return datamodel.ReferenceBook{}, err
	}
	return datamodel.ReferenceBookFromDict(raw), nil
}

// GetReferenceLibrary returns the reference library for the active network.
func (s *Session) GetReferenceLibrary(ctx context.Context) (datamodel.ReferenceLibrary, error) {
	raw, err := s.getReferenceLibrary(ctx)
	if err != nil {
		return datamodel.ReferenceLibrary{}, err
	}
	return datamodel.ReferenceLibraryFromDict(raw), nil
}

// PutReferenceBook puts a reference book in the active network.
func (s *Session) PutReferenceBook(ctx context.Context, book datamodel.ReferenceBook) error {
	return s.putReferenceBook(ctx, book)
}

// PutNodeRoles writes node roles definitions for the active network.
func (s *Session) PutNodeRoles(ctx context.Context, data datamodel.NodeRolesData) error {
	return s.putNodeRoles(ctx, data)
}

// PutNetworkObject puts data as the network object with the specified key.
func (s *Session) PutNetworkObject(ctx context.Context, key string, data []byte) error {
	return s.putNetworkObject(ctx, key, data)
}

// PutSnapshotObject puts data as the snapshot object with the specified key.
func (s *Session) PutSnapshotObject(ctx context.Context, key string, data []byte) error {
	return s.putSnapshotObject(ctx, key, data, nil)
}

// GetNetworkObjectStream returns a binary stream of a network object.
func (s *Session) GetNetworkObjectStream(ctx context.Context, key string) (io.ReadCloser, error) {
	return s.getNetworkObject(ctx, key)
}

// GetNetworkObjectText returns the text content of a network object.
func (s *Session) GetNetworkObjectText(ctx context.Context, key, encoding string) (string, error) {
	stream, err := s.GetNetworkObjectStream(ctx, key)
	if err != nil {
		return "", err
	}
	defer stream.Close()
	data, err := io.ReadAll(stream)
	if err != nil {
		return "", err
	}
	return decodeText(data, encoding), nil
}

// GetSnapshotInputObjectStream returns a binary stream of a snapshot input object.
func (s *Session) GetSnapshotInputObjectStream(ctx context.Context, key string, snapshot *string) (io.ReadCloser, error) {
	return s.getSnapshotInputObject(ctx, key, snapshot)
}

// GetSnapshotInputObjectText returns the text content of a snapshot input object.
func (s *Session) GetSnapshotInputObjectText(ctx context.Context, key, encoding string, snapshot *string) (string, error) {
	stream, err := s.GetSnapshotInputObjectStream(ctx, key, snapshot)
	if err != nil {
		return "", err
	}
	defer stream.Close()
	data, err := io.ReadAll(stream)
	if err != nil {
		return "", err
	}
	return decodeText(data, encoding), nil
}

// GetSnapshotObjectStream returns a binary stream of a snapshot object.
func (s *Session) GetSnapshotObjectStream(ctx context.Context, key string, snapshot *string) (io.ReadCloser, error) {
	return s.getSnapshotObject(ctx, key, snapshot)
}

// GetSnapshotObjectText returns the text content of a snapshot object.
func (s *Session) GetSnapshotObjectText(ctx context.Context, key, encoding string, snapshot *string) (string, error) {
	stream, err := s.GetSnapshotObjectStream(ctx, key, snapshot)
	if err != nil {
		return "", err
	}
	defer stream.Close()
	data, err := io.ReadAll(stream)
	if err != nil {
		return "", err
	}
	return decodeText(data, encoding), nil
}

// GetWorkStatus gets the status for the specified work item.
func (s *Session) GetWorkStatus(ctx context.Context, workItemID string) (map[string]any, error) {
	if s.Network == nil {
		return nil, fmt.Errorf("network is not set")
	}
	answerMap, status, taskDetails, err := getWorkStatus(ctx, workItemID, s)
	if err != nil {
		return nil, err
	}
	answerMap[SvcKeyWorkStatus] = status
	answerMap[SvcKeyTaskStatus] = taskDetails
	return answerMap, nil
}

// ListIncompleteWorks gets pending work that is incomplete.
func (s *Session) ListIncompleteWorks(ctx context.Context) (map[string]any, error) {
	if s.Network == nil {
		return nil, fmt.Errorf("network is not set")
	}
	statuses, err := s.listIncompleteWork(ctx)
	if err != nil {
		return nil, err
	}
	b, _ := jsonString(statuses)
	return map[string]any{SvcKeyWorkList: b}, nil
}

// ForkSnapshotOptions configures ForkSnapshot.
type ForkSnapshotOptions struct {
	Name                 string
	Overwrite            bool
	DeactivateInterfaces []datamodel.Interface
	DeactivateNodes      []string
	RestoreInterfaces    []datamodel.Interface
	RestoreNodes         []string
	AddFiles             string
	ExtraArgs            map[string]any
}

// ForkSnapshot copies an existing snapshot, optionally deactivating or
// reactivating nodes/interfaces on the copy.
func (s *Session) ForkSnapshot(ctx context.Context, baseName string, opts ForkSnapshotOptions) (string, error) {
	if s.Network == nil {
		return "", fmt.Errorf("network is not set")
	}
	name := opts.Name
	if name == "" {
		name = DefaultSnapshotPrefix + util.GetUUID()
	}
	if err := util.ValidateName(name, "snapshot"); err != nil {
		return "", err
	}
	if err := s.checkSnapshotOverwrite(ctx, name, opts.Overwrite); err != nil {
		return "", err
	}
	var encoded string
	if opts.AddFiles != "" {
		data, err := snapshotBytes(opts.AddFiles)
		if err != nil {
			return "", err
		}
		encoded = base64Encode(data)
	}
	obj := map[string]any{
		"snapshotBase":         baseName,
		"snapshotNew":          name,
		"deactivateInterfaces": opts.DeactivateInterfaces,
		"deactivateNodes":      opts.DeactivateNodes,
		"restoreInterfaces":    opts.RestoreInterfaces,
		"restoreNodes":         opts.RestoreNodes,
		"zipFile":              nilIfEmpty(encoded),
	}
	if err := s.forkSnapshot(ctx, obj); err != nil {
		return "", err
	}
	return s.parseSnapshot(ctx, name, false, opts.ExtraArgs)
}

// AutoComplete returns autocomplete suggestions for the provided query.
func (s *Session) AutoComplete(ctx context.Context, completionType datamodel.VariableType, query string, maxSuggestions *int) ([]datamodel.AutoCompleteSuggestion, error) {
	if maxSuggestions != nil && *maxSuggestions < 0 {
		return nil, fmt.Errorf("max_suggestions cannot be negative")
	}
	if s.Network == nil {
		return nil, fmt.Errorf("network is not set")
	}
	response, err := s.autoComplete(ctx, string(completionType), query, maxSuggestions)
	if err != nil {
		return nil, err
	}
	var out []datamodel.AutoCompleteSuggestion
	if suggestions, ok := response[SvcKeySuggestions].([]any); ok {
		for _, s := range suggestions {
			if m, ok := s.(map[string]any); ok {
				out = append(out, datamodel.AutoCompleteSuggestionFromDict(m))
			}
		}
	}
	return out, nil
}

func (s *Session) parseSnapshot(ctx context.Context, name string, background bool, extraArgs map[string]any) (string, error) {
	item := getWorkitemParse(s, name)
	result, err := execute(ctx, item, s, background, extraArgs)
	if err != nil {
		return "", err
	}
	if background {
		s.Snapshot = util.StringPtr(name)
		return name, nil
	}
	status, _ := result["status"].(WorkStatusCode)
	if status != WorkTerminatedNormally {
		log, _ := s.getWorkLog(ctx, &name, item.ID)
		return "", exception.NewBatfishErrorf("Initializing snapshot %s failed with status %s\n%s", name, status, log)
	}
	s.Snapshot = util.StringPtr(name)
	return name, nil
}

// sessionTypes holds the registered session types.
var sessionTypes = map[string]func(SessionConfig) *Session{
	"bf": NewSession,
}

// SessionTypes returns the available session types.
func SessionTypes() map[string]func(SessionConfig) *Session {
	out := make(map[string]func(SessionConfig) *Session, len(sessionTypes))
	for k, v := range sessionTypes {
		out[k] = v
	}
	return out
}

// GetSession creates a session of the specified type.
func GetSession(type_ string, cfg SessionConfig) (*Session, error) {
	factory, ok := sessionTypes[type_]
	if !ok {
		return nil, fmt.Errorf("invalid session type. Specified type '%s' does not match any registered session type", type_)
	}
	return factory(cfg), nil
}

// Close releases resources held by the session.
func (s *Session) Close() {
	s.httpClient.CloseIdleConnections()
}

func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func jsonString(v any) (string, error) {
	b, err := jsonMarshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func decodeText(data []byte, encoding string) string {
	return string(data)
}

func base64Encode(data []byte) string {
	return base64StdEncode(data)
}

// textWithPlatform returns the text with platform prepended if needed.
func textWithPlatform(text, platform string) string {
	if platform == "" {
		return text
	}
	return fmt.Sprintf("!RANCID-CONTENT-TYPE: %s\n%s", strings.ToLower(strings.TrimSpace(platform)), text)
}

// createInMemoryZip creates a zip file for a single file snapshot.
func createInMemoryZip(text, filename, platform string) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("snapshot/configs/" + filename)
	if err != nil {
		return nil, err
	}
	if _, err := w.Write([]byte(textWithPlatform(text, platform))); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
