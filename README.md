# MCP Hub

A lightweight hub that connects multiple MCP servers into one. Think of it as a reverse proxy for MCP - your AI assistant talks to the hub, and the hub routes requests to the right server.

## Why?

If you're using multiple MCP servers (GitHub, filesystem, databases, etc.), managing them separately gets messy. MCP Hub gives you:

- **One connection** instead of many - your client connects to the hub, done
- **Tool namespacing** - no more conflicts when two servers have a `search` tool (`githubSearch` vs `filesSearch`)
- **Built-in scripting** - chain tool calls together with JavaScript
- **Auto-reconnection** - servers crash sometimes, the hub handles it

## Three Ways to Use It

**Server mode** - Run `mh serve -c config.json` to start a hub that aggregates multiple MCP servers. Your AI client connects to the hub, and the hub routes tool calls to the right backend.

**CLI mode** - Use `mh list`, `mh inspect`, `mh invoke` to interact with MCP servers directly from your terminal. Great for debugging, testing, or scripting.

**Skill mode** - Discover skills from [skills.sh](https://skills.sh) or generate lightweight "skill" files from MCP servers. Skills teach AI agents how to use tools on-demand without loading all schemas into context. Use `mh skills find/add` to browse the ecosystem, or follow the `mcp-skill-gen` workflow to create skills from any MCP server.

All modes support the same connection types: local servers via config, remote HTTP/SSE endpoints via URL, or stdio subprocesses.

## Quick Start

```bash
# Install
curl -fsSL https://raw.githubusercontent.com/vaayne/mcphub/main/scripts/install.sh | sh

# Create a config file
cat > config.json << 'EOF'
{
  "mcpServers": {
    "filesystem": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-filesystem", "/tmp"]
    }
  }
}
EOF

# Run as hub server
mh serve -c config.json

# Or explore tools from CLI
mh list -c config.json
mh invoke -c config.json filesystemReadFile '{"path": "/tmp/test.txt"}'
```

## Configuration

The config file is straightforward JSON. Each server gets a name and connection details:

```json
{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["-y", "@modelcontextprotocol/server-github"],
      "env": { "GITHUB_TOKEN": "ghp_xxx" }
    },
    "remote-api": {
      "transport": "http",
      "url": "https://api.example.com/mcp",
      "headers": { "Authorization": "Bearer ${API_TOKEN}" }
    },
    "streaming": {
      "transport": "sse",
      "url": "http://localhost:8080/sse"
    }
  }
}
```

**Transport types:**

- `stdio` (default) - runs a local command
- `http` - connects to a remote HTTP endpoint
- `sse` - Server-Sent Events for streaming servers

**Other options:**

- `enable: false` - disable a server without removing it
- `required: true` - fail startup if this server can't connect
- `timeout` - connection timeout in seconds (http/sse only)
- `tlsSkipVerify` - skip TLS verification (don't use in production)

## CLI Usage

### Server Mode

```bash
# Start hub on stdio (default, for AI clients)
mh serve -c config.json

# Start hub on HTTP (for web clients)
mh serve -c config.json -t http -p 8080
```

### CLI Mode

```bash
# From config file (connects to all servers defined in config)
mh list -c config.json
mh inspect -c config.json githubSearchRepos
mh invoke -c config.json githubSearchRepos '{"query": "mcp"}'

# From remote URL
mh list -u https://mcp.example.com
mh invoke -u http://localhost:3000 -t sse some_tool '{"arg": "value"}'

# From stdio subprocess
mh list --stdio -- npx @modelcontextprotocol/server-everything

# Enable debug logging
mh list -c config.json --verbose
```

## Built-in Tools

The hub includes a few tools of its own:

**`search`** - Find tools across all connected servers by name or description.

**`execute`** - Run JavaScript that can call any tool. Useful for chaining operations:

```javascript
const repos = mcp.callTool("githubSearchRepos", { query: "mcp" });
const readme = mcp.callTool("githubGetFile", {
  repo: repos[0].name,
  path: "README.md",
});
readme;
```

The JS runtime is intentionally limited - sync only, no network access, 15-second timeout. It's for glue code, not application logic.

**`refreshTools`** - Reload tool lists from servers (useful after server restarts).

## Security Notes

The hub takes a paranoid approach:

- Commands are validated - no shell injection, no path traversal
- Shell interpreters (bash, sh) are blocked as commands
- Environment variables are sanitized (no `LD_PRELOAD` tricks)
- JavaScript is sandboxed - no `eval`, no `Promise`, no `fetch`

That said, you're still running arbitrary MCP servers. Only configure servers you trust.

## Building from Source

```bash
git clone https://github.com/vaayne/mcphub.git
cd mcphub
go build -o mh .

# Or with mise
mise run build
```

Requires Go 1.23+.

## Skills

Skills are modular packages that extend AI agent capabilities. MCP Hub supports two workflows:

### Discover & Install Skills

Browse and install skills from [skills.sh](https://skills.sh), the open agent skills ecosystem:

```bash
# Search for skills by keyword
mh skills find react
mh skills find "code review"

# Install a skill
mh skills add anthropics/skills@mcp-builder
mh skills add vercel-labs/agent-skills@react-best-practices
```

Skills are installed to `.agents/skills/<skill-name>/` in the current directory.

### Generate Skills from MCP Servers

If you're using an AI coding agent, loading all MCP tools into context can be expensive. Instead, generate a "skill" file that teaches the agent how to discover and use tools on-demand.

```bash
# Preview tools from a remote MCP server
mh list -u https://mcp.exa.ai

# Preview tools from config
mh list -c config.json
```

Then use the `mcp-skill-gen` workflow (in `skills/mcp-skill-gen/`) to generate a SKILL.md that contains:

- Service URL/config and transport type
- List of available tools with descriptions
- Commands for `mh list`, `mh inspect`, `mh invoke`

When the AI needs a tool, it reads the skill, runs `mh inspect` to get the schema, then `mh invoke` to call it. No need to load all tool schemas upfront.

## Troubleshooting

**"context deadline exceeded"** - Server took too long to start. Check that the command exists and works standalone.

**"async functions are not allowed"** - The `execute` tool only supports sync code. Remove any `async`/`await` or `Promise` usage.

**Tool not found** - Remember tools are namespaced with server prefix in camelCase: `serverNameToolName`. Use `mh list` to see available tools.

## License

MIT

## Links

- [GitHub](https://github.com/vaayne/mcphub)
- [Issues](https://github.com/vaayne/mcphub/issues)
- [Testing Guide](docs/testing.md)
