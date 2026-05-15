package echo

import (
	"context"

	"cli-factory/internal/provider"
	"cli-factory/internal/schema"
)

type Tool struct{}

func (Tool) ID() string               { return "echo" }
func (Tool) Name() string             { return "Echo" }
func (Tool) ShortDescription() string { return "Echo a message for CLI and e2e validation." }
func (Tool) LongDescription() string {
	return "Echo returns the provided message and is intentionally local, deterministic, and safe for CI e2e tests."
}
func (Tool) Categories() []string { return []string{"example", "debug"} }
func (Tool) Aliases() []string    { return []string{"repeat", "say"} }
func (Tool) InputSchema() schema.JSONSchema {
	return schema.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{"type": "string", "description": "Message to echo."},
		},
		"required": []any{"message"},
	}
}
func (Tool) OutputSchema() schema.JSONSchema {
	return schema.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"message": map[string]any{"type": "string"},
		},
	}
}
func (Tool) Invoke(_ context.Context, req provider.InvokeRequest, events provider.EventSink) (provider.InvokeResult, error) {
	message, _ := req.Params["message"].(string)
	events.Emit(provider.Event{Type: "status", Message: "echoing message"})
	return provider.InvokeResult{Data: map[string]any{"message": message}}, nil
}
