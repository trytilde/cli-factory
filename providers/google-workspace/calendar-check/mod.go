package calendarcheck

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
	if err := googleapi.RFC3339(googleapi.StringValue(req.Params, "time_min"), "time_min"); err != nil {
		return provider.InvokeResult{}, err
	}
	if err := googleapi.RFC3339(googleapi.StringValue(req.Params, "time_max"), "time_max"); err != nil {
		return provider.InvokeResult{}, err
	}
	client, err := googleapi.New(req.ProviderParams)
	if err != nil {
		return provider.InvokeResult{}, err
	}
	events.Emit(provider.Event{Type: "status", Message: "checking Calendar events"})
	eventsOut, next, err := client.ListEvents(ctx, googleapi.CalendarListInput{
		TimeMin:      googleapi.StringValue(req.Params, "time_min"),
		TimeMax:      googleapi.StringValue(req.Params, "time_max"),
		Query:        googleapi.StringValue(req.Params, "query"),
		MaxResults:   googleapi.IntValue(req.Params, "max_results", 10),
		SingleEvents: googleapi.BoolValue(req.Params, "single_events", true),
	})
	if err != nil {
		return provider.InvokeResult{}, err
	}
	items := make([]map[string]any, 0, len(eventsOut))
	for _, event := range eventsOut {
		items = append(items, map[string]any{
			"event_id":  event.ID,
			"summary":   event.Summary,
			"status":    event.Status,
			"html_link": event.HTMLLink,
			"start":     firstNonEmpty(event.Start.DateTime, event.Start.Date),
			"end":       firstNonEmpty(event.End.DateTime, event.End.Date),
			"location":  event.Location,
		})
	}
	return provider.InvokeResult{Data: map[string]any{"result_count": len(items), "next_page_token": next, "events": items}}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
