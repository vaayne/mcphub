# Changelog

All notable changes to MCP Hub will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.2] - 2026-01-20

### Fixed

- Update command now uses correct repository (`vaayne/mcphub`) and asset names (`mh_*`)

## [0.1.1] - 2026-01-20

### Fixed

- Filter builtin tools from list output and handle string params in invoke

### Changed

- Simplified MCP tool descriptions for better token efficiency
- Unified tool name handling across MCP server and CLI
- Aligned CLI and MCP server output formats for list/inspect commands

## [0.1.0] - 2026-01-19

Initial release of MCP Hub.

### Added

- **Multi-server aggregation**: Connect multiple MCP servers through a single hub
- **Transport support**:
  - Stdio for local MCP servers
  - HTTP with custom headers, timeout, and TLS configuration
  - SSE (Server-Sent Events) for real-time streaming
- **Tool namespacing**: Automatic camelCase naming (`serverToolName`) to prevent conflicts
- **CLI subcommands**:
  - `mh serve` - Start the MCP hub server
  - `mh list` - List tools from MCP services
  - `mh inspect` - Inspect tool details and input schema
  - `mh invoke` - Invoke tools with JSON params
  - `mh exec` - Execute JavaScript code with MCP tool access
  - `mh update` - Self-update to latest version
  - `mh version` - Show version information
- **Built-in tools**:
  - `list` - Search and list available tools across all connected servers
  - `invoke` - Invoke tools by name with JSON parameters
  - `inspect` - Get detailed tool schema information
  - `exec` - Execute JavaScript code with access to MCP tools
- **JavaScript runtime** (Goja):
  - Synchronous execution with `mcp.callTool("serverToolName", params)`
  - `console.log/info/warn/error/debug` for logging
  - 15-second execution timeout (configurable)
- **Security features**:
  - Input validation for commands, arguments, and environment variables
  - Shell metacharacter and interpreter blocking
  - Clean environment isolation
  - AST-based async detection
  - Dangerous globals blocked: `eval`, `Function`, `Reflect`, `Proxy`, `WebAssembly`
  - Frozen prototype chains
  - Script size limits (100KB)
- **Structured logging**: JSON format with configurable levels and file output
- **Automatic reconnection**: Exponential backoff for failed server connections
- **Multi-platform releases**: linux/darwin/windows × amd64/arm64

[Unreleased]: https://github.com/vaayne/mcphub/compare/v0.1.2...HEAD
[0.1.2]: https://github.com/vaayne/mcphub/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/vaayne/mcphub/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/vaayne/mcphub/releases/tag/v0.1.0
