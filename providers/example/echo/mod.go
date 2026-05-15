package echo

import (
	"context"

	"cli-factory/internal/provider"
	"cli-factory/internal/schema"
)

type Tool struct{}

func (Tool) ID() string               { return toolID }
func (Tool) Name() string             { return toolName }
func (Tool) ShortDescription() string { return toolShortDescription }
func (Tool) LongDescription() string  { return toolLongDescription }
func (Tool) Categories() []string {
	return append([]string(nil), toolCategories...)
}
func (Tool) Aliases() []string {
	return append([]string(nil), toolAliases...)
}
func (Tool) InputSchema() schema.JSONSchema {
	return toolInputSchema
}
func (Tool) OutputSchema() schema.JSONSchema {
	return toolOutputSchema
}
func (Tool) Invoke(_ context.Context, req provider.InvokeRequest, events provider.EventSink) (provider.InvokeResult, error) {
	message, _ := req.Params["message"].(string)
	events.Emit(provider.Event{Type: "status", Message: "echoing message"})
	return provider.InvokeResult{Data: map[string]any{"message": message}}, nil
}
