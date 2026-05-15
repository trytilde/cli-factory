package calendarcheck

import (
	"context"

	"cli-factory/internal/provider"
	"cli-factory/internal/schema"
	"cli-factory/providers/google-workspace/internal/googleapi"
)

type Tool struct{}

func (Tool) ID() string               { return "calendar-check" }
func (Tool) Name() string             { return "Calendar Check" }
func (Tool) ShortDescription() string { return "Search Google Calendar events." }
func (Tool) LongDescription() string {
	return "List or search Google Calendar events in a time window on the configured calendar."
}
func (Tool) Categories() []string { return []string{"calendar", "scheduling", "search"} }
func (Tool) Aliases() []string    { return []string{"check-events", "list-events"} }
func (Tool) InputSchema() schema.JSONSchema {
	return schema.JSONSchema{
		"type":     "object",
		"required": []any{"time_min", "time_max"},
		"properties": map[string]any{
			"time_min":      map[string]any{"type": "string", "format": "date-time"},
			"time_max":      map[string]any{"type": "string", "format": "date-time"},
			"query":         map[string]any{"type": "string"},
			"max_results":   map[string]any{"type": "integer", "minimum": 1, "maximum": 50, "default": 10},
			"single_events": map[string]any{"type": "boolean", "default": true},
		},
	}
}
func (Tool) OutputSchema() schema.JSONSchema {
	return schema.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"result_count":    map[string]any{"type": "integer"},
			"next_page_token": map[string]any{"type": "string"},
			"events":          map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{"event_id": map[string]any{"type": "string"}, "summary": map[string]any{"type": "string"}, "status": map[string]any{"type": "string"}, "html_link": map[string]any{"type": "string"}, "start": map[string]any{"type": "string"}, "end": map[string]any{"type": "string"}, "location": map[string]any{"type": "string"}}}},
		},
	}
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
