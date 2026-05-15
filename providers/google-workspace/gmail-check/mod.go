package gmailcheck

import (
	"context"

	"cli-factory/internal/provider"
	"cli-factory/internal/schema"
	"cli-factory/providers/google-workspace/internal/googleapi"
)

type Tool struct{}

func (Tool) ID() string               { return "gmail-check" }
func (Tool) Name() string             { return "Gmail Check" }
func (Tool) ShortDescription() string { return "Search recent Gmail messages." }
func (Tool) LongDescription() string {
	return "Search or list Gmail messages for the authenticated Google Workspace user and return compact message details."
}
func (Tool) Categories() []string { return []string{"email", "gmail", "search"} }
func (Tool) Aliases() []string    { return []string{"check-email", "search-email"} }
func (Tool) InputSchema() schema.JSONSchema {
	return schema.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"query":           map[string]any{"type": "string", "description": "Gmail search query."},
			"max_results":     map[string]any{"type": "integer", "minimum": 1, "maximum": 25, "default": 10},
			"include_snippet": map[string]any{"type": "boolean", "default": true},
		},
	}
}
func (Tool) OutputSchema() schema.JSONSchema {
	return schema.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"result_count": map[string]any{"type": "integer"},
			"messages": map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{
				"id":        map[string]any{"type": "string"},
				"thread_id": map[string]any{"type": "string"},
				"snippet":   map[string]any{"type": "string"},
				"label_ids": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			}}},
		},
	}
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
