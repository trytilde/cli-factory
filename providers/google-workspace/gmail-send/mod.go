package gmailsend

import (
	"context"

	"cli-factory/internal/provider"
	"cli-factory/internal/schema"
	"cli-factory/providers/google-workspace/internal/googleapi"
)

type Tool struct{}

func (Tool) ID() string               { return "gmail-send" }
func (Tool) Name() string             { return "Gmail Send" }
func (Tool) ShortDescription() string { return "Send a Gmail message from a Google Workspace account." }
func (Tool) LongDescription() string {
	return "Send a plain-text Gmail message through the Gmail API using OAuth-backed Google Workspace credentials."
}
func (Tool) Categories() []string { return []string{"email", "gmail"} }
func (Tool) Aliases() []string    { return []string{"send-email", "email"} }
func (Tool) InputSchema() schema.JSONSchema {
	return schema.JSONSchema{
		"type":     "object",
		"required": []any{"to", "subject", "body_text"},
		"properties": map[string]any{
			"to":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "minItems": 1, "description": "Recipient email addresses."},
			"cc":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Optional CC recipients."},
			"bcc":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Optional BCC recipients."},
			"subject":   map[string]any{"type": "string", "minLength": 1},
			"body_text": map[string]any{"type": "string", "minLength": 1},
		},
	}
}
func (Tool) OutputSchema() schema.JSONSchema {
	return schema.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"message_id": map[string]any{"type": "string"},
			"thread_id":  map[string]any{"type": "string"},
			"label_ids":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
		},
	}
}

func (Tool) Invoke(ctx context.Context, req provider.InvokeRequest, events provider.EventSink) (provider.InvokeResult, error) {
	client, err := googleapi.New(req.ProviderParams)
	if err != nil {
		return provider.InvokeResult{}, err
	}
	events.Emit(provider.Event{Type: "status", Message: "sending Gmail message"})
	msg, err := client.SendGmail(ctx, googleapi.EmailMessage{
		To:       googleapi.StringSlice(req.Params["to"]),
		CC:       googleapi.StringSlice(req.Params["cc"]),
		BCC:      googleapi.StringSlice(req.Params["bcc"]),
		Subject:  googleapi.StringValue(req.Params, "subject"),
		BodyText: googleapi.StringValue(req.Params, "body_text"),
	})
	if err != nil {
		return provider.InvokeResult{}, err
	}
	return provider.InvokeResult{Data: map[string]any{"message_id": msg.ID, "thread_id": msg.ThreadID, "label_ids": msg.LabelIDs}}, nil
}
