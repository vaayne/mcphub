Invoke a single MCP tool by name.

Use this tool for simple, one-off tool calls. For chaining multiple tools or adding logic, use `exec` instead.

## When To Use

- Single tool call with known parameters
- Quick testing of a tool
- Simple operations without orchestration

## Usage

Provide the tool name in either format:
- **JS name (camelCase)**: `githubSearchRepos`
- **Original name**: `github__search_repos`

```json
{ "name": "githubSearchRepos", "params": { "query": "mcp", "limit": 10 } }
```

For tools with no parameters:

```json
{ "name": "serverListItems" }
```

## Output Format

Returns the raw tool result. For text content:

```
Search found 3 repositories:
- repo1: Description of repo1
- repo2: Description of repo2
- repo3: Description of repo3
```

With `--json` or structured output, returns the full CallToolResult:

```json
{
  "content": [
    { "type": "text", "text": "Search found 3 repositories..." }
  ],
  "isError": false
}
```

## See Also

- `list` - Find available tools
- `inspect` - Get tool schema before invoking
- `exec` - Chain multiple tool calls with JavaScript
