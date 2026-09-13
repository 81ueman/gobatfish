# MCP server

`gobatfish/mcp` is a [Model Context Protocol](https://modelcontextprotocol.io)
(MCP) server that exposes Batfish network analysis to AI agents, mirroring
`pybatfish.mcp`. It uses the official
[Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk) with typed,
**structured tool outputs**.

> **Beta.** Tool names and parameters mirror pybatfish's beta MCP server and may
> change.

## Build

```bash
go build -o batfish-mcp ./cmd/batfish-mcp
```

## Run

```bash
# stdio (default) — for local desktop clients that spawn the process
batfish-mcp

# Streamable HTTP
batfish-mcp -transport http -addr :8080

# both at once
batfish-mcp -transport both -addr :8080
```

| Flag | Meaning |
|---|---|
| `-transport` | `stdio`, `http`, or `both` (default `stdio`) |
| `-addr` | HTTP listen address (default `:8080`) |
| `-stateless` | use stateless Streamable HTTP |
| `-json-response` | return `application/json` instead of SSE |
| `-auth-token` | bearer token required for HTTP (or `BATFISH_MCP_AUTH_TOKEN`) |
| `-sessions-config` | path to the sessions config file |
| `-name` | MCP server name (default `Batfish`) |

### Transports

stdio and Streamable HTTP are **not mutually exclusive**: they are two
transports over the same server implementation. stdio is one client per process
and needs no authentication; Streamable HTTP serves multiple remote clients and
should be protected with a bearer token:

```bash
BATFISH_MCP_AUTH_TOKEN=secret batfish-mcp -transport http -addr :8080
# clients send: Authorization: Bearer secret
```

## Sessions

Named sessions are configured in `~/.batfish/sessions.json`:

```json
{
  "default": {"type": "bf", "params": {"host": "localhost"}},
  "prod":    {"type": "bf", "params": {"host": "batfish.example.com"}}
}
```

If no `default` session is configured, one is created using the `BATFISH_HOST`
environment variable, falling back to `localhost`. Sessions are created lazily
and cached. The `register_session` and `list_sessions` tools manage sessions at
runtime; all other tools accept an optional `session` parameter.

## Tools

Management:

- `register_session`, `list_sessions`
- `list_networks`, `set_network`, `delete_network`
- `list_snapshots`, `init_snapshot`, `init_snapshot_from_text`, `delete_snapshot`, `fork_snapshot`

Diagnostics:

- `get_parse_warnings`, `get_init_issues`, `get_file_parse_status`, `get_vi_conversion_status`, `get_vi_conversion_warnings`

Analysis:

- `run_traceroute`, `run_bidirectional_traceroute`, `check_reachability`
- `analyze_acl`, `search_filters`
- `get_routes`, `compare_routes`, `get_bgp_rib`, `get_evpn_rib`
- `get_bgp_session_status`, `get_bgp_session_compatibility`, `get_bgp_peer_configuration`, `get_bgp_process_configuration`
- `get_node_properties`, `get_interface_properties`, `get_ip_owners`
- `compare_filters`, `get_undefined_references`, `get_defined_structures`, `get_unused_structures`, `detect_loops`

## Structured output

Table tools return a structured result rather than a JSON string:

```json
{
  "columns": ["Node", "VRF", "Network", "Next_Hop"],
  "rows": [{"Node": "r1", "VRF": "default", "Network": "10.0.0.0/24", "Next_Hop": "ip 10.0.1.2"}],
  "count": 1
}
```

Route tools drop deprecated next-hop columns (`Next_Hop_IP`,
`Next_Hop_Interface`, `NextHopIp`, `NextHopInterface`) in favour of the
structured `Next_Hop` column.

## Client configuration

Claude Desktop (`claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "batfish": {
      "command": "/path/to/batfish-mcp"
    }
  }
}
```

For a remote Streamable HTTP server, point the client at
`http://host:8080` and configure the `Authorization: Bearer <token>` header.

## Testing

```bash
go test ./mcp/ -count=1
BATFISH_INTEGRATION=1 go test ./mcp/ -run Integration -v
```
