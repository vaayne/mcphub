package tools

import (
	"fmt"
	"sort"
	"strings"
)

// jsonSchemaTypeToJS converts JSON Schema types to JavaScript types for JSDoc
func jsonSchemaTypeToJS(schemaType any) string {
	switch t := schemaType.(type) {
	case string:
		switch t {
		case "string":
			return "string"
		case "number", "integer":
			return "number"
		case "boolean":
			return "boolean"
		case "array":
			return "Array"
		case "object":
			return "Object"
		default:
			return "*"
		}
	default:
		return "*"
	}
}

// schemaToJSDoc generates a JSDoc comment and function stub from a tool's schema.
// The output includes required markers, enum values, and default values.
func schemaToJSDoc(toolName, description string, inputSchema map[string]any) string {
	var sb strings.Builder

	sb.WriteString("/**\n")

	// Use tool name as fallback if description is empty
	if description == "" {
		description = toolName
	}
	sb.WriteString(fmt.Sprintf(" * %s\n", description))

	// Extract required fields
	requiredFields := make(map[string]bool)
	if inputSchema != nil {
		if required, ok := inputSchema["required"].([]any); ok {
			for _, r := range required {
				if name, ok := r.(string); ok {
					requiredFields[name] = true
				}
			}
		}
	}

	// Extract properties from schema
	if inputSchema != nil {
		if props, ok := inputSchema["properties"].(map[string]any); ok && len(props) > 0 {
			sb.WriteString(" * @param {Object} params - Parameters\n")

			// Sort property names for consistent output
			propNames := make([]string, 0, len(props))
			for name := range props {
				propNames = append(propNames, name)
			}
			sort.Strings(propNames)

			for _, propName := range propNames {
				propDef := props[propName]
				propMap, ok := propDef.(map[string]any)
				if !ok {
					continue
				}

				jsType := jsonSchemaTypeToJS(propMap["type"])
				isRequired := requiredFields[propName]

				// Build param name with optional bracket notation
				paramName := "params." + propName
				if !isRequired {
					// Check for default value
					if defaultVal, hasDefault := propMap["default"]; hasDefault {
						paramName = fmt.Sprintf("[params.%s=%v]", propName, defaultVal)
					} else {
						paramName = fmt.Sprintf("[params.%s]", propName)
					}
				}

				// Handle enum values - use union type syntax
				if enum, ok := propMap["enum"].([]any); ok && len(enum) > 0 {
					enumStrs := make([]string, 0, len(enum))
					for _, e := range enum {
						enumStrs = append(enumStrs, fmt.Sprintf("%q", e))
					}
					jsType = strings.Join(enumStrs, "|")
				}

				// Build description
				propDesc := ""
				if d, ok := propMap["description"].(string); ok {
					propDesc = d
				}
				if isRequired && propDesc != "" {
					propDesc += " (required)"
				} else if isRequired {
					propDesc = "(required)"
				}

				if propDesc == "" {
					sb.WriteString(fmt.Sprintf(" * @param {%s} %s\n", jsType, paramName))
				} else {
					sb.WriteString(fmt.Sprintf(" * @param {%s} %s - %s\n", jsType, paramName, propDesc))
				}
			}
		}
	}

	sb.WriteString(" */\n")
	sb.WriteString(fmt.Sprintf("function %s(params) {}\n", toolName))

	return sb.String()
}
