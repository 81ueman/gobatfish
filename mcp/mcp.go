// Package mcp implements the Batfish Model Context Protocol (MCP) server.
//
// It exposes Batfish network analysis capabilities as MCP tools, allowing AI
// agents to perform snapshot management, reachability analysis, traceroute
// simulation, ACL/filter inspection, and routing queries. It mirrors
// pybatfish.mcp.server, but uses the official Go MCP SDK with typed, structured
// tool outputs and supports both stdio and Streamable HTTP transports.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"github.com/81ueman/gobatfish/client"
	"github.com/81ueman/gobatfish/dataframe"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// Session is the subset of client.Session used by the MCP tools. It is an
// interface so tests can substitute a fake.
type Session interface {
	ListNetworks(ctx context.Context) ([]string, error)
	SetNetwork(ctx context.Context, name string) (string, error)
	DeleteNetwork(ctx context.Context, name string) error
	ListSnapshots(ctx context.Context, verbose bool) ([]any, error)
	SetSnapshot(ctx context.Context, name string, index *int) (string, error)
	InitSnapshot(ctx context.Context, upload string, opts client.InitSnapshotOptions) (string, error)
	InitSnapshotFromText(ctx context.Context, text string, opts client.InitSnapshotFromTextOptions) (string, error)
	DeleteSnapshot(ctx context.Context, name string) error
	ForkSnapshot(ctx context.Context, baseName string, opts client.ForkSnapshotOptions) (string, error)
	Ask(ctx context.Context, question string, vars map[string]any, snapshot, referenceSnapshot *string) (*dataframe.DataFrame, error)
	Close()
}

// SessionEntry is one configured session in the sessions config file.
type SessionEntry struct {
	Type   string         `json:"type"`
	Params map[string]any `json:"params"`
}

// SessionFactory creates a session from its configuration.
type SessionFactory func(ctx context.Context, entry SessionEntry) (Session, error)

// Registry holds named session configurations and lazily creates and caches
// sessions, mirroring pybatfish.mcp.server's session registry.
type Registry struct {
	mu          sync.Mutex
	configs     map[string]SessionEntry
	cache       map[string]Session
	factory     SessionFactory
	defaultHost string
}

// NewRegistry creates a registry. When factory is nil the default Batfish
// session factory is used.
func NewRegistry(factory SessionFactory) *Registry {
	if factory == nil {
		factory = DefaultSessionFactory
	}
	return &Registry{
		configs:     make(map[string]SessionEntry),
		cache:       make(map[string]Session),
		factory:     factory,
		defaultHost: "localhost",
	}
}

// DefaultSessionsConfigPath returns ~/.batfish/sessions.json.
func DefaultSessionsConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".batfish", "sessions.json")
	}
	return filepath.Join(home, ".batfish", "sessions.json")
}

// LoadConfig loads session configurations from a JSON file. If no "default"
// session is configured, one is added using the BATFISH_HOST environment
// variable (falling back to localhost). An empty path uses the default path.
func (r *Registry) LoadConfig(path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if path == "" {
		path = DefaultSessionsConfigPath()
	}
	if data, err := os.ReadFile(path); err == nil {
		entries := map[string]SessionEntry{}
		if err := json.Unmarshal(data, &entries); err != nil {
			return fmt.Errorf("invalid sessions config %s: %w", path, err)
		}
		for name, entry := range entries {
			if entry.Params == nil {
				entry.Params = map[string]any{}
			}
			r.configs[name] = entry
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if _, ok := r.configs["default"]; !ok {
		host := os.Getenv("BATFISH_HOST")
		if host == "" {
			host = r.defaultHost
		}
		r.configs["default"] = SessionEntry{Type: "bf", Params: map[string]any{"host": host}}
	}
	return nil
}

// Register registers and immediately creates a named session.
func (r *Registry) Register(ctx context.Context, name, type_ string, params map[string]any) (Session, error) {
	if params == nil {
		params = map[string]any{}
	}
	r.mu.Lock()
	r.configs[name] = SessionEntry{Type: type_, Params: params}
	delete(r.cache, name)
	r.mu.Unlock()
	return r.Get(ctx, name)
}

// SetPrecreated registers an already-created session under name, bypassing the
// factory.
func (r *Registry) SetPrecreated(name string, session Session) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.configs[name] = SessionEntry{Type: "precreated", Params: map[string]any{}}
	r.cache[name] = session
}

// Get returns the cached session for name, creating it lazily.
func (r *Registry) Get(ctx context.Context, name string) (Session, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if session, ok := r.cache[name]; ok {
		return session, nil
	}
	entry, ok := r.configs[name]
	if !ok {
		names := make([]string, 0, len(r.configs))
		for n := range r.configs {
			names = append(names, n)
		}
		sort.Strings(names)
		return nil, fmt.Errorf("No session named '%s'. Available sessions: %v. Use the register_session tool to create one", name, names)
	}
	session, err := r.factory(ctx, entry)
	if err != nil {
		return nil, err
	}
	r.cache[name] = session
	return session, nil
}

// Names returns the registered session names mapped to their types.
func (r *Registry) Names() map[string]string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make(map[string]string, len(r.configs))
	for name, entry := range r.configs {
		out[name] = entry.Type
	}
	return out
}

// Clear removes all configurations and cached sessions.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache = make(map[string]Session)
	r.configs = make(map[string]SessionEntry)
}

// Close closes all cached sessions.
func (r *Registry) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, session := range r.cache {
		session.Close()
	}
	r.cache = make(map[string]Session)
}

// mgmt returns the named session with an optional network set.
func (r *Registry) mgmt(ctx context.Context, session, network string) (Session, error) {
	s, err := r.Get(ctx, session)
	if err != nil {
		return nil, err
	}
	if network != "" {
		if _, err := s.SetNetwork(ctx, network); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// analysis returns the named session with network and snapshot set.
func (r *Registry) analysis(ctx context.Context, session, network, snapshot string) (Session, error) {
	s, err := r.Get(ctx, session)
	if err != nil {
		return nil, err
	}
	if network != "" {
		if _, err := s.SetNetwork(ctx, network); err != nil {
			return nil, err
		}
	}
	if snapshot != "" {
		if _, err := s.SetSnapshot(ctx, snapshot, nil); err != nil {
			return nil, err
		}
	}
	return s, nil
}

// DefaultSessionFactory creates a client.Session from a session entry and loads
// its question templates.
func DefaultSessionFactory(ctx context.Context, entry SessionEntry) (Session, error) {
	cfg := client.SessionConfig{}
	if host, ok := entry.Params["host"].(string); ok {
		cfg.Host = host
	}
	if port, ok := intParam(entry.Params["port"]); ok {
		cfg.Port = &port
	}
	if ssl, ok := entry.Params["ssl"].(bool); ok {
		cfg.SSL = &ssl
	}
	if apiKey, ok := entry.Params["api_key"].(string); ok {
		cfg.APIKey = apiKey
	}
	session := client.NewSession(cfg)
	if err := session.LoadQuestions(ctx); err != nil {
		session.Close()
		return nil, err
	}
	return session, nil
}

func intParam(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	default:
		return 0, false
	}
}

// Options configures the MCP server.
type Options struct {
	// Name is the MCP server name (default "Batfish").
	Name string
	// Version is the server version.
	Version string
	// Instructions are advertised to clients.
	Instructions string
	// DefaultSession, when non-nil, is registered as the "default" session.
	DefaultSession Session
	// SessionsConfigPath overrides the sessions config file path.
	SessionsConfigPath string
	// SessionFactory overrides session creation (used by tests).
	SessionFactory SessionFactory
	// Logger is used for server logging.
	Logger *slog.Logger
}

// DefaultInstructions is the instruction text advertised to MCP clients.
const DefaultInstructions = "[BETA] This server provides tools to interact with a Batfish network analysis service. " +
	"Use these tools to load network snapshots, run traceroutes, analyze reachability, " +
	"inspect ACLs/firewall rules, query routing tables, and compare snapshots. " +
	"Most tools require a 'network' parameter (the network name in Batfish) " +
	"and a 'snapshot' parameter (the snapshot name). " +
	"All tools accept an optional 'session' parameter to select a named session " +
	"(default: 'default'). Use register_session to configure additional sessions. " +
	"Start by listing networks or initializing a snapshot, then run analysis tools."

// Server is the Batfish MCP server.
type Server struct {
	registry *Registry
	mcp      *mcpsdk.Server
}

// NewServer creates and configures the Batfish MCP server.
func NewServer(opts Options) (*Server, error) {
	name := opts.Name
	if name == "" {
		name = "Batfish"
	}
	instructions := opts.Instructions
	if instructions == "" {
		instructions = DefaultInstructions
	}
	registry := NewRegistry(opts.SessionFactory)
	if err := registry.LoadConfig(opts.SessionsConfigPath); err != nil {
		return nil, err
	}
	if opts.DefaultSession != nil {
		registry.SetPrecreated("default", opts.DefaultSession)
	}
	server := &Server{registry: registry}
	server.mcp = mcpsdk.NewServer(
		&mcpsdk.Implementation{Name: name, Version: opts.Version},
		&mcpsdk.ServerOptions{Instructions: instructions, Logger: opts.Logger},
	)
	registerManagementTools(server)
	registerAnalysisTools(server)
	return server, nil
}

// MCP returns the underlying SDK server.
func (s *Server) MCP() *mcpsdk.Server { return s.mcp }

// Registry returns the session registry.
func (s *Server) Registry() *Registry { return s.registry }

// Close releases sessions held by the server.
func (s *Server) Close() { s.registry.Close() }

// RunStdio serves the server over stdin/stdout until the client disconnects.
func (s *Server) RunStdio(ctx context.Context) error {
	return s.mcp.Run(ctx, &mcpsdk.StdioTransport{})
}

// HTTPHandler returns a Streamable HTTP handler for the server.
func (s *Server) HTTPHandler(opts *mcpsdk.StreamableHTTPOptions) http.Handler {
	if opts == nil {
		opts = &mcpsdk.StreamableHTTPOptions{}
	}
	return mcpsdk.NewStreamableHTTPHandler(func(*http.Request) *mcpsdk.Server { return s.mcp }, opts)
}

// askFrame answers a table question and returns the raw data frame.
func (s *Server) askFrame(ctx context.Context, session, network, snapshot, question string, vars map[string]any, referenceSnapshot *string) (*dataframe.DataFrame, error) {
	sess, err := s.registry.analysis(ctx, sessionName(session), network, snapshot)
	if err != nil {
		return nil, err
	}
	return sess.Ask(ctx, question, vars, nil, referenceSnapshot)
}

// ask answers a table question and returns its result as a structured table.
func (s *Server) ask(ctx context.Context, session, network, snapshot, question string, vars map[string]any, referenceSnapshot *string) (TableResult, error) {
	df, err := s.askFrame(ctx, session, network, snapshot, question, vars, referenceSnapshot)
	if err != nil {
		return TableResult{}, err
	}
	return NewTableResult(df), nil
}
