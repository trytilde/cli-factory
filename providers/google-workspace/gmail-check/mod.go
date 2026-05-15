package gmailcheck

import (
	"context"

	"cli-factory/internal/provider"
	"cli-factory/internal/schema"
	"cli-factory/providers/google-workspace/internal/googleapi"
)

type Tool struct{}

func (Tool) ID() string               { return toolID }
func (Tool) Name() string             { return toolName }
func (Tool) ShortDescription() string { return toolShortDescription }
func (Tool) LongDescription() string  { return toolLongDescription }
func (Tool) Categories() []string {
	return append([]string(nil), toolCategories...)
}
func (Tool) Aliases() []string { return append([]string(nil), toolAliases...) }
func (Tool) InputSchema() schema.JSONSchema {
	return toolInputSchema
}
func (Tool) OutputSchema() schema.JSONSchema {
	return toolOutputSchema
}

func (Tool) Invoke(ctx context.Context, req provider.InvokeRequest, events provider.EventSink) (provider.InvokeResult, error) {
	client, err := googleapi.New(req.ProviderParams)
	if err != nil {
		return provider.InvokeResult{}, err
	}
	events.Emit(provider.Event{Type: "status", Message: "checking Gmail messages"})
	messages, err := client.ListGmail(ctx, googleapi.StringValue(req.Params, "query"), googleapi.IntValue(req.Params, "max_results", 10), googleapi.BoolValue(req.Params, "include_snippet", true))
	if err != nil {
		return provider.InvokeResult{}, err
	}
	out := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		out = append(out, map[string]any{"id": msg.ID, "thread_id": msg.ThreadID, "snippet": msg.Snippet, "label_ids": msg.LabelIDs})
	}
	return provider.InvokeResult{Data: map[string]any{"result_count": len(out), "messages": out}}, nil
}
