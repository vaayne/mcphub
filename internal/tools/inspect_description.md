Get full tool signature as JSDoc stub. Use before `invoke` or `exec` to see parameters.

## Parameters

- `name` - Tool name in camelCase, e.g. `githubSearchRepos` (required)

## Example

```json
{"name": "githubSearchRepos"}
```

## Output

JSDoc function stub (copy-paste ready for `exec`):

```javascript
/**
 * Search repositories on GitHub
 * @param {Object} params - Parameters
 * @param {string} params.query - Search query (required)
 * @param {"asc"|"desc"} [params.order="desc"] - Sort order
 * @param {number} [params.perPage=30] - Results per page
 */
function githubSearchRepos(params) {}
```
