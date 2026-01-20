Execute JavaScript code to orchestrate multiple tool calls with logic.

Use `inspect` first to get tool signatures, then write JS code using `mcp.callTool(name, params)`.

## Parameters

- `code` - JavaScript code to execute (required)

## API

- `mcp.callTool(name, params)` - Call a tool, returns result or throws on error
- `console.log/info/warn/error` - Logging (captured in output)
- `require("node:buffer/url/util")` - Node.js modules

## Supported

- Variables, loops, conditionals
- `async/await`, Promises
- `try/catch` for error handling
- Last expression is returned (no top-level `return`)

## Not Available

- `fetch`, `window`, `document` - Use MCP tools for external data
- `fs`, `child_process` - No filesystem/process access

## Examples

Single call:
```javascript
mcp.callTool("webSearchExa", {query: "MCP protocol"})
```

Chain calls:
```javascript
const user = mcp.callTool("dbGetUser", {id: 123});
mcp.callTool("emailSend", {to: user.email, subject: "Hello"});
```

Batch with error handling:
```javascript
const ids = [1, 2, 3];
ids.map(id => {
  try {
    return {id, ok: true, data: mcp.callTool("dbGetUser", {id})};
  } catch (e) {
    return {id, ok: false, error: e.message};
  }
});
```

Async pattern:
```javascript
(async () => {
  const users = await mcp.callTool("dbListUsers", {limit: 10});
  return users.filter(u => u.active);
})();
```

## Constraints

- Timeout: 15 seconds
- Max code size: 100KB
- Max log entries: 1000

## Output

- `result` - Last expression value
- `logs` - Array of console/mcp.log entries
- `error` - Error details if execution fails
