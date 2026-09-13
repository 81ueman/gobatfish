// Command batfish-mcp runs the Batfish Model Context Protocol (MCP) server.
//
// It mirrors the pybatfish `batfish-mcp` entry point but supports both the
// stdio and Streamable HTTP transports.
//
// Usage:
//
//	batfish-mcp                                  # stdio transport
//	batfish-mcp -transport http -addr :8080      # Streamable HTTP
//	batfish-mcp -transport both -addr :8080      # serve both
//
// Sessions are configured in ~/.batfish/sessions.json:
//
//	{
//	  "default": {"type": "bf", "params": {"host": "localhost"}},
//	  "prod":    {"type": "bf", "params": {"host": "batfish.example.com"}}
//	}
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	gobatfish "github.com/81ueman/gobatfish"
	"github.com/81ueman/gobatfish/mcp"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "batfish-mcp:", err)
		os.Exit(1)
	}
}

func run() error {
	var (
		sessionsConfig = flag.String("sessions-config", "", "path to sessions JSON config (default ~/.batfish/sessions.json)")
		transport      = flag.String("transport", "stdio", "transport: stdio, http, or both")
		addr           = flag.String("addr", ":8080", "HTTP listen address (http/both transports)")
		stateless      = flag.Bool("stateless", false, "use stateless Streamable HTTP")
		jsonResponse   = flag.Bool("json-response", false, "return application/json responses instead of SSE")
		authToken      = flag.String("auth-token", "", "bearer token required for HTTP requests (or BATFISH_MCP_AUTH_TOKEN)")
		name           = flag.String("name", "Batfish", "MCP server name")
	)
	flag.Parse()

	token := *authToken
	if token == "" {
		token = os.Getenv("BATFISH_MCP_AUTH_TOKEN")
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))

	server, err := mcp.NewServer(mcp.Options{
		Name:               *name,
		Version:            gobatfish.Version,
		SessionsConfigPath: *sessionsConfig,
		Logger:             logger,
	})
	if err != nil {
		return err
	}
	defer server.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	httpOptions := &mcpsdk.StreamableHTTPOptions{Stateless: *stateless, JSONResponse: *jsonResponse, Logger: logger}

	switch *transport {
	case "stdio":
		return server.RunStdio(ctx)
	case "http":
		return serveHTTP(ctx, server, *addr, token, httpOptions)
	case "both":
		errCh := make(chan error, 1)
		go func() { errCh <- serveHTTP(ctx, server, *addr, token, httpOptions) }()
		if err := server.RunStdio(ctx); err != nil {
			return err
		}
		select {
		case err := <-errCh:
			return err
		default:
			return nil
		}
	default:
		return fmt.Errorf("invalid transport %q: must be stdio, http, or both", *transport)
	}
}

func serveHTTP(ctx context.Context, server *mcp.Server, addr, token string, opts *mcpsdk.StreamableHTTPOptions) error {
	handler := server.HTTPHandler(opts)
	if token != "" {
		handler = bearerAuth(token, handler)
	}
	httpServer := &http.Server{Addr: addr, Handler: handler, ReadHeaderTimeout: 10 * time.Second}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}()

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// bearerAuth requires an "Authorization: Bearer <token>" header.
func bearerAuth(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") || strings.TrimPrefix(header, "Bearer ") != token {
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
