# MCP Hub Agent Instructions

## Naming Convention

- **Product name**: MCP Hub (for docs and display)
- **CLI command**: `mh` (following the `gh` pattern)
- **Go module**: `mcphub`

## Commands

```bash
mise run build    # Build binary to bin/mh
mise run test     # Run all tests with coverage
mise run run      # Build and run with example config
mise run format   # Format code (go fmt, go mod tidy)
```

## Project Structure

```
main.go                    # CLI entry point (cobra)
internal/
├── cli/                   # Subcommands: serve, list, inspect, invoke, exec, update
├── server/                # MCP hub server - aggregates tools, handles requests
├── client/                # Remote MCP client manager - connections, reconnection logic
├── config/                # Configuration loading and validation
├── tools/                 # Built-in tools: search, execute, refreshTools
├── js/                    # Goja JavaScript runtime for execute tool
├── transport/             # Transport factory: stdio, http, sse
└── logging/               # Structured logging with zap
```

## Three Modes

**Server mode** (`mh serve` or `mh -c`): Starts an MCP hub server that aggregates multiple backend servers. AI clients connect to the hub.

**CLI mode** (`mh list/inspect/invoke/exec`): Interact with MCP servers directly from terminal. Supports config file (`-c`), remote URL (`-u`), or stdio subprocess (`--stdio`).

**Skill mode**: Generate lightweight skill files that describe MCP tools without loading them into context. AI agents read the skill, then use `mh` CLI to discover/invoke tools on-demand. Keeps context small while accessing hundreds of tools. See `mcp-skill-gen` workflow.

Server and CLI modes share connection logic in `internal/cli/` (config_client, remote, stdio_client).

## Architecture Overview

**Server mode flow:**
1. Load and validate config → `internal/config/`
2. Connect to each MCP server (stdio subprocess or http/sse client) → `internal/client/` + `internal/transport/`
3. Discover tools from each server, namespace them (`servername__toolname`)
4. Start MCP server on stdio/http/sse, expose aggregated tools → `internal/server/`
5. Route incoming tool calls to appropriate backend, return results

**CLI mode flow:**
1. Parse connection source (config/url/stdio) → `internal/cli/helpers.go`
2. Create appropriate client → `internal/cli/{config_client,remote,stdio_client}.go`
3. Execute command (list/inspect/invoke) using `internal/tools/` core functions
4. Format and print output

Built-in tools (`search`, `execute`, `refreshTools`) are registered in `internal/tools/` and don't go through backends.

## Key Design Decisions

- **Tool namespacing**: All backend tools are prefixed with server name to avoid conflicts (`github__search` not `search`)
- **Sync-only JS**: The `execute` tool blocks async to prevent complexity and security issues (no Promises, no setTimeout)
- **Transport abstraction**: `internal/transport/` provides unified interface for stdio/http/sse so client code doesn't care about transport type
- **Fail-open by default**: Servers with `required: false` (default) won't block hub startup if they fail to connect

## Code Style

- Go 1.23+, standard gofmt formatting
- All non-main packages live in `internal/` (not importable externally)
- Every package should have `*_test.go` files with meaningful coverage
- Use structured logging via `internal/logging` - no `fmt.Println` for runtime messages
- Error messages should be lowercase, no trailing punctuation
- Context should flow through all blocking operations

## Testing

```bash
go test ./...                    # Run all tests
go test -cover ./...             # With coverage
go test -v ./internal/js         # Verbose, specific package
```

For integration tests, see `docs/testing.md` which documents the built-in test server.

## Common Tasks

**Adding a new CLI command**: Create file in `internal/cli/`, register in `main.go`

**Adding a built-in tool**: Add to `internal/tools/builtin.go`, implement handler, add tests

**Supporting new transport**: Implement `Transport` interface in `internal/transport/`, register in factory

**Modifying config schema**: Update structs in `internal/config/config.go`, add validation, update tests
