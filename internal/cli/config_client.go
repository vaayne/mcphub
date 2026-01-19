package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/vaayne/mcphub/internal/config"
	"github.com/vaayne/mcphub/internal/transport"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type ConfigClient struct {
	logger   *slog.Logger
	sessions map[string]*mcp.ClientSession
	tools    map[string]*mcp.Tool
	refs     map[string]toolRef
}

type toolRef struct {
	serverID string
	toolName string
}

func NewConfigClient(ctx context.Context, configPath string, logger *slog.Logger, timeout time.Duration) (*ConfigClient, error) {
	if configPath == "" {
		return nil, fmt.Errorf("--config is required for config mode")
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	client := &ConfigClient{
		logger:   logger,
		sessions: make(map[string]*mcp.ClientSession),
		tools:    make(map[string]*mcp.Tool),
		refs:     make(map[string]toolRef),
	}

	factory := transport.NewDefaultFactory(logger)
	var optionalErrors []error

	for serverID, serverCfg := range cfg.MCPServers {
		if !serverCfg.IsEnabled() {
			logger.Info("Skipping disabled server", slog.String("serverID", serverID))
			continue
		}

		logger.Info("Connecting to server", slog.String("serverID", serverID))
		transportName := strings.ToLower(serverCfg.GetTransport())

		mcpTransport, err := factory.CreateTransport(serverCfg)
		if err != nil {
			if serverCfg.Required {
				return nil, fmt.Errorf("required server %s failed to create transport: %w", serverID, err)
			}
			optionalErrors = append(optionalErrors, fmt.Errorf("server %s: %w", serverID, err))
			continue
		}

		clientInstance := mcp.NewClient(&mcp.Implementation{
			Name:    "mh-cli",
			Version: "v1.0.0",
		}, nil)

		connectCtx, cancel := context.WithTimeout(ctx, timeout)
		session, err := clientInstance.Connect(connectCtx, mcpTransport, nil)
		// For non-SSE transports, cancel immediately after connect.
		// For SSE, the context is used by background goroutines, so we don't cancel it here.
		// The context will be canceled when the parent ctx is canceled.
		if transportName != "sse" {
			cancel()
		} else {
			// Acknowledge that we're intentionally not canceling for SSE.
			// The cancel func will be called when the parent context is done.
			_ = cancel
		}
		if err != nil {
			cancel() // Always cancel on error
			if serverCfg.Required {
				return nil, fmt.Errorf("required server %s failed to connect: %w", serverID, err)
			}
			optionalErrors = append(optionalErrors, fmt.Errorf("server %s: %w", serverID, err))
			continue
		}

		toolsCtx, toolsCancel := context.WithTimeout(ctx, timeout)
		toolsResult, err := session.ListTools(toolsCtx, nil)
		toolsCancel()
		if err != nil {
			session.Close()
			if serverCfg.Required {
				return nil, fmt.Errorf("required server %s failed to list tools: %w", serverID, err)
			}
			optionalErrors = append(optionalErrors, fmt.Errorf("server %s: %w", serverID, err))
			continue
		}

		client.sessions[serverID] = session

		for _, tool := range toolsResult.Tools {
			namespacedName := fmt.Sprintf("%s__%s", serverID, tool.Name)
			if _, exists := client.tools[namespacedName]; exists {
				return nil, fmt.Errorf("duplicate tool name detected: %s", namespacedName)
			}

			client.tools[namespacedName] = &mcp.Tool{
				Name:        namespacedName,
				Description: tool.Description,
				InputSchema: tool.InputSchema,
			}
			client.refs[namespacedName] = toolRef{serverID: serverID, toolName: tool.Name}
		}
	}

	if len(optionalErrors) > 0 {
		logger.Warn("Some optional servers failed to connect", slog.Int("count", len(optionalErrors)))
	}

	return client, nil
}

func (c *ConfigClient) ListTools(ctx context.Context) ([]*mcp.Tool, error) {
	tools := make([]*mcp.Tool, 0, len(c.tools))
	for _, tool := range c.tools {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		tools = append(tools, tool)
	}

	return tools, nil
}

func (c *ConfigClient) GetTool(ctx context.Context, namespacedName string) (*mcp.Tool, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	tool, ok := c.tools[namespacedName]
	if !ok {
		return nil, fmt.Errorf("tool '%s' not found", namespacedName)
	}
	return tool, nil
}

func (c *ConfigClient) CallTool(ctx context.Context, namespacedName string, params json.RawMessage) (*mcp.CallToolResult, error) {
	ref, ok := c.refs[namespacedName]
	if !ok {
		return nil, fmt.Errorf("tool '%s' not found", namespacedName)
	}

	session, ok := c.sessions[ref.serverID]
	if !ok {
		return nil, fmt.Errorf("server not connected: %s", ref.serverID)
	}

	var args map[string]any
	if len(params) > 0 {
		if err := json.Unmarshal(params, &args); err != nil {
			return nil, fmt.Errorf("invalid tool arguments: %w", err)
		}
	}

	callParams := &mcp.CallToolParams{
		Name:      ref.toolName,
		Arguments: args,
	}

	result, err := session.CallTool(ctx, callParams)
	if err != nil {
		return nil, fmt.Errorf("failed to call tool '%s': %w", namespacedName, err)
	}

	return result, nil
}

func (c *ConfigClient) Close() error {
	var errs []error
	for serverID, session := range c.sessions {
		if session == nil {
			continue
		}
		if err := session.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close session %s: %w", serverID, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing sessions: %v", errs)
	}

	return nil
}
