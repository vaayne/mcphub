# CLI Flag Refactoring Plan

## Problem

Global flags defined on the root command in `main.go` appear in help output for all subcommands, including `skills` which doesn't use MCP-related flags:

```
mh skills find -h
...
GLOBAL OPTIONS:
   --config, --url, --transport, --stdio, --timeout, --header, --json, --verbose, --log-file
```

## Current Architecture Analysis

### Flag Usage Matrix

| Flag | serve | list | inspect | invoke | exec | update | skills |
|------|-------|------|---------|--------|------|--------|--------|
| `--config, -c` | ✓ | ✓ | ✓ | ✓ | ✓ | ✗ | ✗ |
| `--url, -u` | ✗ | ✓ | ✓ | ✓ | ✓ | ✗ | ✗ |
| `--transport, -t` | ✓ | ✓ | ✓ | ✓ | ✓ | ✗ | ✗ |
| `--stdio` | ✗ | ✓ | ✓ | ✓ | ✓ | ✗ | ✗ |
| `--timeout` | ✗ | ✓ | ✓ | ✓ | ✓ | ✗ | ✗ |
| `--header` | ✗ | ✓ | ✓ | ✓ | ✓ | ✗ | ✗ |
| `--json` | ✗ | ✓ | ✓ | ✓ | ✓ | ✗ | ✗ |
| `--verbose` | ✓ | ✓ | ✓ | ✓ | ✓ | ✗ | ✗ |
| `--log-file` | ✓ | ✓ | ✓ | ✓ | ✓ | ✗ | ✗ |

### Command Groups

1. **MCP Server Commands**: `serve` - starts the hub server
2. **MCP Client Commands**: `list`, `inspect`, `invoke`, `exec` - interact with MCP servers
3. **Standalone Commands**: `update`, `skills` - no MCP connection needed

## Proposed Architecture

### Option A: Command-Level Flags (Recommended)

Move flags to the commands that use them, using shared flag definitions to avoid duplication.

```
main.go
├── (no global flags except --version, --help)
└── Commands
    ├── serve (--config, --transport, --verbose, --log-file, --port, --host)
    ├── list (--url, --config, --transport, --stdio, --timeout, --header, --json, --verbose, --log-file)
    ├── inspect (same as list)
    ├── invoke (same as list)
    ├── exec (same as list)
    ├── update (--check)
    └── skills
        ├── find (--limit)
        └── add (--skill, --list)
```

### New File Structure

```
internal/cli/
├── flags.go          # NEW: Shared flag definitions
├── serve.go          # Add serve-specific flags
├── list.go           # Add client flags
├── inspect.go        # Add client flags
├── invoke.go         # Add client flags
├── exec.go           # Add client flags
├── update.go         # Already has own flags
├── skills.go         # Already has own flags
└── helpers.go        # Keep as-is
```

## Implementation Plan

### Phase 1: Create Shared Flag Definitions

Create `internal/cli/flags.go` with reusable flag sets:

```go
package cli

import ucli "github.com/urfave/cli/v3"

// MCPClientFlags are flags shared by commands that connect to MCP servers
var MCPClientFlags = []ucli.Flag{
    &ucli.StringFlag{
        Name:    "config",
        Aliases: []string{"c"},
        Usage:   "path to configuration file",
    },
    &ucli.StringFlag{
        Name:    "url",
        Aliases: []string{"u"},
        Usage:   "remote MCP service URL",
    },
    &ucli.StringFlag{
        Name:    "transport",
        Aliases: []string{"t"},
        Usage:   "transport type (http/sse)",
    },
    &ucli.BoolFlag{
        Name:  "stdio",
        Usage: "use stdio transport (spawn subprocess); command follows -- separator",
    },
    &ucli.IntFlag{
        Name:  "timeout",
        Usage: "connection timeout in seconds",
        Value: 30,
    },
    &ucli.StringSliceFlag{
        Name:  "header",
        Usage: "HTTP headers (repeatable, format: \"Key: Value\")",
    },
    &ucli.BoolFlag{
        Name:  "json",
        Usage: "output as JSON",
    },
    &ucli.BoolFlag{
        Name:  "verbose",
        Usage: "verbose logging",
    },
    &ucli.StringFlag{
        Name:  "log-file",
        Usage: "log file path (empty disables file logging)",
    },
}

// MCPServeFlags are flags for the serve command
var MCPServeFlags = []ucli.Flag{
    &ucli.StringFlag{
        Name:    "config",
        Aliases: []string{"c"},
        Usage:   "path to configuration file",
        Required: true,
    },
    &ucli.StringFlag{
        Name:    "transport",
        Aliases: []string{"t"},
        Usage:   "transport type (stdio/http/sse)",
        Value:   "stdio",
    },
    &ucli.BoolFlag{
        Name:  "verbose",
        Usage: "verbose logging",
    },
    &ucli.StringFlag{
        Name:  "log-file",
        Usage: "log file path (empty disables file logging)",
    },
    // serve-specific flags
    &ucli.IntFlag{
        Name:    "port",
        Aliases: []string{"p"},
        Usage:   "port for HTTP/SSE transport",
        Value:   3000,
    },
    &ucli.StringFlag{
        Name:  "host",
        Usage: "host for HTTP/SSE transport",
        Value: "localhost",
    },
}

// ValidateMCPClientFlags validates the mutual exclusivity of connection flags
func ValidateMCPClientFlags(cmd *ucli.Command) error {
    url := cmd.String("url")
    config := cmd.String("config")
    stdio := cmd.Bool("stdio")
    
    count := 0
    if url != "" { count++ }
    if config != "" { count++ }
    if stdio { count++ }
    
    if count == 0 {
        return fmt.Errorf("one of --url, --config, or --stdio is required")
    }
    if count > 1 {
        return fmt.Errorf("--url, --config, and --stdio are mutually exclusive")
    }
    return nil
}
```

### Phase 2: Update Each Command

#### 2.1 Update `serve.go`

```go
var ServeCmd = &ucli.Command{
    Name:   "serve",
    Usage:  "Start the MCP hub server",
    Flags:  MCPServeFlags,  // Use shared flags
    Action: runServe,
}
```

#### 2.2 Update Client Commands (`list.go`, `inspect.go`, `invoke.go`, `exec.go`)

```go
var ListCmd = &ucli.Command{
    Name:   "list",
    Usage:  "List tools from an MCP service",
    Flags:  append(MCPClientFlags, 
        &ucli.StringFlag{Name: "server", Usage: "filter by server"},
        &ucli.StringFlag{Name: "query", Usage: "filter by keywords"},
    ),
    Before: func(ctx context.Context, cmd *ucli.Command) (context.Context, error) {
        return ctx, ValidateMCPClientFlags(cmd)
    },
    Action: runList,
}
```

### Phase 3: Simplify `main.go`

Remove all global flags from root command:

```go
app := &ucli.Command{
    Name:    "mh",
    Usage:   "MCP Hub - Go implementation of Model Context Protocol hub",
    Version: version,
    Commands: []*ucli.Command{
        cli.ServeCmd,
        cli.ListCmd,
        cli.InspectCmd,
        cli.InvokeCmd,
        cli.ExecCmd,
        cli.UpdateCmd,
        cli.SkillsCmd,
    },
}
```

### Phase 4: Handle Legacy `mh -c config.json` Shortcut

Currently `mh -c config.json` runs serve. Options:
1. **Remove shortcut**: Require explicit `mh serve -c config.json`
2. **Keep shortcut**: Add a special case in root command (less clean)

Recommendation: Remove shortcut for cleaner architecture. Users can use `mh serve -c`.

## Migration Checklist

- [x] Create `internal/cli/flags.go` with shared flag definitions
- [x] Update `serve.go` to use `MCPServeFlags`
- [x] Update `list.go` to use `MCPClientFlags` + command-specific flags
- [x] Update `inspect.go` to use `MCPClientFlags`
- [x] Update `invoke.go` to use `MCPClientFlags`
- [x] Update `exec.go` to use `MCPClientFlags`
- [x] Update `main.go` to remove global flags
- [x] Update `helpers.go` if needed (should work as-is since flags are still on cmd)
- [x] Remove `RunServeFromRoot` and `validateGlobalFlags` from main.go
- [ ] Update README/docs for new CLI patterns
- [x] Run tests and verify all commands work

## Verification

After refactoring, `mh skills find -h` should show:

```
NAME:
   mh skills find - Search for skills

USAGE:
   mh skills find [options] [query]

OPTIONS:
   --limit int  maximum number of results (default: 10)
   --help, -h   show help
```

No more "GLOBAL OPTIONS" section with MCP flags.

## Risk Assessment

- **Low Risk**: Flag behavior stays the same, just moved to command level
- **Breaking Change**: `mh -c config.json` shortcut will no longer work
- **Testing**: All existing CLI tests should pass after updating flag locations
