package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/81ueman/gobatfish"
	"github.com/81ueman/gobatfish/datamodel"
)

// HTTPError is returned for non-2xx HTTP responses. It mirrors the
// requests.HTTPError surfaced by pybatfish.
type HTTPError struct {
	StatusCode int
	Message    string
}

// Error implements the error interface.
func (e *HTTPError) Error() string {
	return fmt.Sprintf("%d %s: %s", e.StatusCode, http.StatusText(e.StatusCode), e.Message)
}

var retryStatuses = map[int]bool{429: true, 500: true, 502: true, 503: true, 504: true}

// GetBaseURL2 generates the base URL for v2 of the coordinator APIs.
func (s *Session) GetBaseURL2() string {
	protocol := "http"
	if s.SSL {
		protocol = "https"
	}
	return fmt.Sprintf("%s://%s:%d%s", protocol, s.Host, s.PortV2, SvcCfgWorkMgr2)
}

func (s *Session) getHeaders() map[string]string {
	return map[string]string{
		HTTPHeaderBatfishAPIKey:  s.APIKey,
		HTTPHeaderBatfishVersion: gobatfish.Version,
	}
}

// request performs an HTTP request with pybatfish's retry behavior.
func (s *Session) request(ctx context.Context, method, urlTail string, body []byte, contentType string, params url.Values) (*http.Response, error) {
	endpoint := s.GetBaseURL2() + urlTail
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	maxRetries := MaxInitialTriesToConnect
	if s.maxRetries != nil {
		maxRetries = *s.maxRetries
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		var reader io.Reader
		if body != nil {
			reader = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
		if err != nil {
			return nil, err
		}
		for k, v := range s.getHeaders() {
			req.Header.Set(k, v)
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}

		resp, err := s.httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < maxRetries {
				sleepBackoff(ctx, attempt)
				continue
			}
			return nil, err
		}
		if retryStatuses[resp.StatusCode] && attempt < maxRetries {
			resp.Body.Close()
			sleepBackoff(ctx, attempt)
			continue
		}
		return resp, nil
	}
	return nil, lastErr
}

func sleepBackoff(ctx context.Context, attempt int) {
	delay := time.Duration(float64(time.Second) * RequestBackoffFactor * float64(int(1)<<uint(attempt)))
	if delay <= 0 {
		return
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

func (s *Session) checkStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	return &HTTPError{StatusCode: resp.StatusCode, Message: strings.TrimSpace(string(data))}
}

func (s *Session) get(ctx context.Context, urlTail string, params url.Values) (*http.Response, error) {
	resp, err := s.request(ctx, http.MethodGet, urlTail, nil, "", params)
	if err != nil {
		return nil, err
	}
	if err := s.checkStatus(resp); err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *Session) getDict(ctx context.Context, urlTail string, params url.Values) (map[string]any, error) {
	resp, err := s.get(ctx, urlTail, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Session) getList(ctx context.Context, urlTail string, params url.Values) ([]any, error) {
	resp, err := s.get(ctx, urlTail, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var out []any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Session) getStream(ctx context.Context, urlTail string, params url.Values) (io.ReadCloser, error) {
	resp, err := s.get(ctx, urlTail, params)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func (s *Session) getText(ctx context.Context, urlTail string, params url.Values) (string, error) {
	resp, err := s.get(ctx, urlTail, params)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *Session) post(ctx context.Context, urlTail string, body []byte, contentType string, params url.Values) error {
	resp, err := s.request(ctx, http.MethodPost, urlTail, body, contentType, params)
	if err != nil {
		return err
	}
	return s.checkStatus(resp)
}

func (s *Session) put(ctx context.Context, urlTail string, body []byte, contentType string, params url.Values) error {
	resp, err := s.request(ctx, http.MethodPut, urlTail, body, contentType, params)
	if err != nil {
		return err
	}
	return s.checkStatus(resp)
}

func (s *Session) delete(ctx context.Context, urlTail string, params url.Values) error {
	resp, err := s.request(ctx, http.MethodDelete, urlTail, nil, "", params)
	if err != nil {
		return err
	}
	return s.checkStatus(resp)
}

func mustJSON(v any) []byte {
	data, err := json.Marshal(datamodel.ToJSONValue(v))
	if err != nil {
		return nil
	}
	return data
}

// --- endpoints --------------------------------------------------------------

func (s *Session) listNetworks(ctx context.Context) ([]map[string]any, error) {
	raw, err := s.getList(ctx, "/"+RSCNetworks, nil)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out, nil
}

func (s *Session) listSnapshots(ctx context.Context, verbose bool) ([]any, error) {
	if s.Network == nil {
		return nil, fmt.Errorf("network must be set to list snapshots")
	}
	tail := fmt.Sprintf("/%s/%s/%s", RSCNetworks, *s.Network, RSCSnapshots)
	return s.getList(ctx, tail, url.Values{QPVerbose: []string{strconv.FormatBool(verbose)}})
}

func (s *Session) initNetwork(ctx context.Context, name string) error {
	return s.post(ctx, "/"+RSCNetworks, nil, "", url.Values{QPName: []string{name}})
}

func (s *Session) deleteNetwork(ctx context.Context, name string) error {
	return s.delete(ctx, fmt.Sprintf("/%s/%s", RSCNetworks, name), nil)
}

func (s *Session) getNetwork(ctx context.Context, name string) (map[string]any, error) {
	return s.getDict(ctx, fmt.Sprintf("/%s/%s", RSCNetworks, name), nil)
}

func (s *Session) uploadSnapshot(ctx context.Context, snapshotName string, data []byte) error {
	if s.Network == nil {
		return fmt.Errorf("network must be set to upload a snapshot")
	}
	tail := fmt.Sprintf("/%s/%s/%s/%s", RSCNetworks, *s.Network, RSCSnapshots, snapshotName)
	return s.post(ctx, tail, data, "application/octet-stream", nil)
}

func (s *Session) deleteSnapshot(ctx context.Context, snapshot, network string) error {
	tail := fmt.Sprintf("/%s/%s/%s/%s", RSCNetworks, network, RSCSnapshots, snapshot)
	return s.delete(ctx, tail, nil)
}

func (s *Session) forkSnapshot(ctx context.Context, obj map[string]any) error {
	if s.Network == nil {
		return fmt.Errorf("network must be set to fork a snapshot")
	}
	tail := fmt.Sprintf("/%s/%s/%s:%s", RSCNetworks, *s.Network, RSCSnapshots, RSCFork)
	return s.post(ctx, tail, mustJSON(obj), "application/json", nil)
}

func (s *Session) getAnswer(ctx context.Context, question string, params url.Values) (map[string]any, error) {
	if s.Network == nil {
		return nil, fmt.Errorf("network must be set to get an answer")
	}
	tail := fmt.Sprintf("/%s/%s/%s/%s/%s", RSCNetworks, *s.Network, RSCQuestions, question, RSCAnswer)
	return s.getDict(ctx, tail, params)
}

func (s *Session) getNetworkObject(ctx context.Context, key string) (io.ReadCloser, error) {
	tail := fmt.Sprintf("/%s/%s/%s", RSCNetworks, deref(s.Network), RSCObjects)
	return s.getStream(ctx, tail, url.Values{QPKey: []string{key}})
}

func (s *Session) getSnapshotInputObject(ctx context.Context, key string, snapshot *string) (io.ReadCloser, error) {
	snapshotName, err := s.GetSnapshot(snapshot)
	if err != nil {
		return nil, err
	}
	tail := fmt.Sprintf("/%s/%s/%s/%s/%s", RSCNetworks, deref(s.Network), RSCSnapshots, snapshotName, RSCInput)
	return s.getStream(ctx, tail, url.Values{QPKey: []string{key}})
}

func (s *Session) getSnapshotObject(ctx context.Context, key string, snapshot *string) (io.ReadCloser, error) {
	snapshotName, err := s.GetSnapshot(snapshot)
	if err != nil {
		return nil, err
	}
	tail := fmt.Sprintf("/%s/%s/%s/%s/%s", RSCNetworks, deref(s.Network), RSCSnapshots, snapshotName, RSCObjects)
	return s.getStream(ctx, tail, url.Values{QPKey: []string{key}})
}

func (s *Session) getNodeRoles(ctx context.Context) (map[string]any, error) {
	if s.Network == nil {
		return nil, fmt.Errorf("network must be set to get node roles")
	}
	tail := fmt.Sprintf("/%s/%s/%s", RSCNetworks, *s.Network, RSCNodeRoles)
	return s.getDict(ctx, tail, nil)
}

func (s *Session) getSnapshotInferredNodeRoles(ctx context.Context, snapshot *string) (map[string]any, error) {
	snapshotName, err := s.GetSnapshot(snapshot)
	if err != nil {
		return nil, err
	}
	tail := fmt.Sprintf("/%s/%s/%s/%s/%s", RSCNetworks, deref(s.Network), RSCSnapshots, snapshotName, RSCInferredNodeRoles)
	return s.getDict(ctx, tail, nil)
}

func (s *Session) getReferenceBook(ctx context.Context, bookName string) (map[string]any, error) {
	if s.Network == nil {
		return nil, fmt.Errorf("network must be set to get a reference book")
	}
	tail := fmt.Sprintf("/%s/%s/%s/%s", RSCNetworks, *s.Network, RSCReferenceLibrary, bookName)
	return s.getDict(ctx, tail, nil)
}

func (s *Session) getReferenceLibrary(ctx context.Context) (map[string]any, error) {
	if s.Network == nil {
		return nil, fmt.Errorf("network must be set to get the reference library")
	}
	tail := fmt.Sprintf("/%s/%s/%s", RSCNetworks, *s.Network, RSCReferenceLibrary)
	return s.getDict(ctx, tail, nil)
}

func (s *Session) putReferenceBook(ctx context.Context, book datamodel.ReferenceBook) error {
	if s.Network == nil {
		return fmt.Errorf("network must be set to add reference book")
	}
	tail := fmt.Sprintf("/%s/%s/%s/%s", RSCNetworks, *s.Network, RSCReferenceLibrary, book.Name)
	return s.put(ctx, tail, mustJSON(book), "application/json", nil)
}

func (s *Session) putNodeRoles(ctx context.Context, data datamodel.NodeRolesData) error {
	if s.Network == nil {
		return fmt.Errorf("network must be set to get node roles")
	}
	tail := fmt.Sprintf("/%s/%s/%s", RSCNetworks, *s.Network, RSCNodeRoles)
	return s.put(ctx, tail, mustJSON(data), "application/json", nil)
}

func (s *Session) putNetworkObject(ctx context.Context, key string, data []byte) error {
	tail := fmt.Sprintf("/%s/%s/%s", RSCNetworks, deref(s.Network), RSCObjects)
	return s.put(ctx, tail, data, "application/octet-stream", url.Values{QPKey: []string{key}})
}

func (s *Session) putSnapshotObject(ctx context.Context, key string, data []byte, snapshot *string) error {
	snapshotName, err := s.GetSnapshot(snapshot)
	if err != nil {
		return err
	}
	tail := fmt.Sprintf("/%s/%s/%s/%s/%s", RSCNetworks, deref(s.Network), RSCSnapshots, snapshotName, RSCObjects)
	return s.put(ctx, tail, data, "application/octet-stream", url.Values{QPKey: []string{key}})
}

func (s *Session) deleteNetworkObject(ctx context.Context, key string) error {
	tail := fmt.Sprintf("/%s/%s/%s", RSCNetworks, deref(s.Network), RSCObjects)
	return s.delete(ctx, tail, url.Values{QPKey: []string{key}})
}

func (s *Session) deleteNodeRoleDimension(ctx context.Context, dimension string) error {
	if s.Network == nil {
		return fmt.Errorf("network must be set to delete a node role dimension")
	}
	tail := fmt.Sprintf("/%s/%s/%s/%s", RSCNetworks, *s.Network, RSCNodeRoles, dimension)
	return s.delete(ctx, tail, nil)
}

func (s *Session) deleteReferenceBook(ctx context.Context, bookName string) error {
	if s.Network == nil {
		return fmt.Errorf("network must be set to delete a reference book")
	}
	tail := fmt.Sprintf("/%s/%s/%s/%s", RSCNetworks, *s.Network, RSCReferenceLibrary, bookName)
	return s.delete(ctx, tail, nil)
}

func (s *Session) deleteSnapshotObject(ctx context.Context, key string, snapshot *string) error {
	snapshotName, err := s.GetSnapshot(snapshot)
	if err != nil {
		return err
	}
	tail := fmt.Sprintf("/%s/%s/%s/%s/%s", RSCNetworks, deref(s.Network), RSCSnapshots, snapshotName, RSCObjects)
	return s.delete(ctx, tail, url.Values{QPKey: []string{key}})
}

func (s *Session) getWorkLog(ctx context.Context, snapshot *string, workID string) (string, error) {
	snapshotName, err := s.GetSnapshot(snapshot)
	if err != nil {
		return "", err
	}
	tail := fmt.Sprintf("/%s/%s/%s/%s/%s/%s", RSCNetworks, deref(s.Network), RSCSnapshots, snapshotName, RSCWorkLog, workID)
	return s.getText(ctx, tail, nil)
}

func (s *Session) getComponentVersions(ctx context.Context) (map[string]any, error) {
	return s.getDict(ctx, "/version", nil)
}

func (s *Session) getQuestionTemplates(ctx context.Context, verbose bool) (map[string]any, error) {
	return s.getDict(ctx, "/"+RSCQuestionTemplates, url.Values{QPVerbose: []string{strconv.FormatBool(verbose)}})
}

func (s *Session) queueWork(ctx context.Context, item *WorkItem) error {
	if s.Network == nil {
		return fmt.Errorf("network must be set to queue work")
	}
	tail := fmt.Sprintf("/%s/%s/%s", RSCNetworks, *s.Network, RSCWork)
	return s.post(ctx, tail, mustJSON(item.ToDict()), "application/json", nil)
}

func (s *Session) getWorkStatus(ctx context.Context, workItemID string) (map[string]any, error) {
	tail := fmt.Sprintf("/%s/%s/%s/%s", RSCNetworks, deref(s.Network), RSCWork, workItemID)
	return s.getDict(ctx, tail, nil)
}

func (s *Session) listIncompleteWork(ctx context.Context) ([]any, error) {
	tail := fmt.Sprintf("/%s/%s/%s", RSCNetworks, deref(s.Network), RSCWork)
	return s.getList(ctx, tail, nil)
}

func (s *Session) autoComplete(ctx context.Context, completionType string, query string, maxSuggestions *int) (map[string]any, error) {
	snapshotPart := ""
	if s.Snapshot != nil {
		snapshotPart = "/" + RSCSnapshots + "/" + *s.Snapshot
	}
	tail := fmt.Sprintf("/%s/%s%s/%s/%s", RSCNetworks, deref(s.Network), snapshotPart, RSCAutoComplete, completionType)
	params := url.Values{}
	if query != "" {
		params.Set(QPQuery, query)
	}
	if maxSuggestions != nil && *maxSuggestions != 0 {
		params.Set(QPMaxSuggestions, strconv.Itoa(*maxSuggestions))
	}
	return s.getDict(ctx, tail, params)
}

func (s *Session) uploadQuestion(ctx context.Context, questionName, questionStr string) error {
	if s.Network == nil {
		return fmt.Errorf("network must be set to upload a question")
	}
	tail := fmt.Sprintf("/%s/%s/%s/%s", RSCNetworks, *s.Network, RSCQuestions, questionName)
	return s.put(ctx, tail, []byte(questionStr), "application/octet-stream", nil)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
