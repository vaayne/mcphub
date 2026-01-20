Call a single tool with parameters. Use `inspect` first to get the tool signature.

## Parameters

- `name` - Tool name in camelCase, e.g. `githubSearchRepos` (required)
- `params` - Tool parameters as object (optional)

## Examples

```json
{"name": "githubSearchRepos", "params": {"query": "mcp", "perPage": 10}}
{"name": "webSearchExa", "params": {"query": "MCP protocol"}}
```

## Output

Returns the tool's result directly (text or JSON).
