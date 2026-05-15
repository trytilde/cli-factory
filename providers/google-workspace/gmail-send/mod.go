package gmailsend

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
