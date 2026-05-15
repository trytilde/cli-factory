package calendarcreate

import (
	"context"

	"cli-factory/internal/provider"
	"cli-factory/internal/schema"
	"cli-factory/providers/google-workspace/internal/googleapi"
)

type Tool struct{}

func (Tool) ID() string               { return "calendar-create" }
func (Tool) Name() string             { return "Calendar Create" }
func (Tool) ShortDescription() string { return "Create a Google Calendar event." }
func (Tool) LongDescription() string {
	return "Create a Google Calendar event on the configured calendar and return compact event details."
}
func (Tool) Categories() []string { return []string{"calendar", "scheduling"} }
func (Tool) Aliases() []string    { return []string{"create-event", "schedule"} }
func (Tool) InputSchema() schema.JSONSchema {
	return schema.JSONSchema{
		"type":     "object",
		"required": []any{"summary", "start", "end"},
		"properties": map[string]any{
			"summary":     map[string]any{"type": "string", "minLength": 1},
			"start":       map[string]any{"type": "string", "format": "date-time", "description": "RFC3339 start timestamp."},
			"end":         map[string]any{"type": "string", "format": "date-time", "description": "RFC3339 end timestamp."},
			"time_zone":   map[string]any{"type": "string", "description": "Optional IANA time zone."},
			"description": map[string]any{"type": "string"},
			"location":    map[string]any{"type": "string"},
			"attendees":   map[string]any{"type": "array", "items": map[string]any{"type": "object", "required": []any{"email"}, "properties": map[string]any{"email": map[string]any{"type": "string"}, "display_name": map[string]any{"type": "string"}}}},
		},
	}
}
func (Tool) OutputSchema() schema.JSONSchema {
	return schema.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"event_id":  map[string]any{"type": "string"},
			"html_link": map[string]any{"type": "string"},
			"status":    map[string]any{"type": "string"},
			"summary":   map[string]any{"type": "string"},
			"start":     map[string]any{"type": "string"},
			"end":       map[string]any{"type": "string"},
			"attendees": map[string]any{"type": "array", "items": map[string]any{"type": "object", "properties": map[string]any{"email": map[string]any{"type": "string"}, "response_status": map[string]any{"type": "string"}}}},
		},
	}
}

func (Tool) Invoke(ctx context.Context, req provider.InvokeRequest, events provider.EventSink) (provider.InvokeResult, error) {
	if err := googleapi.RFC3339(googleapi.StringValue(req.Params, "start"), "start"); err != nil {
		return provider.InvokeResult{}, err
	}
	if err := googleapi.RFC3339(googleapi.StringValue(req.Params, "end"), "end"); err != nil {
		return provider.InvokeResult{}, err
	}
	client, err := googleapi.New(req.ProviderParams)
	if err != nil {
		return provider.InvokeResult{}, err
	}
	events.Emit(provider.Event{Type: "status", Message: "creating Calendar event"})
	event, err := client.CreateEvent(ctx, googleapi.CalendarEventInput{
		Summary:     googleapi.StringValue(req.Params, "summary"),
		Start:       googleapi.StringValue(req.Params, "start"),
		End:         googleapi.StringValue(req.Params, "end"),
		TimeZone:    googleapi.StringValue(req.Params, "time_zone"),
		Description: googleapi.StringValue(req.Params, "description"),
		Location:    googleapi.StringValue(req.Params, "location"),
		Attendees:   attendees(req.Params["attendees"]),
	})
	if err != nil {
		return provider.InvokeResult{}, err
	}
	return provider.InvokeResult{Data: eventData(event)}, nil
}

func attendees(value any) []map[string]string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]string, 0, len(items))
	for _, item := range items {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		email, _ := m["email"].(string)
		if email == "" {
			continue
		}
		next := map[string]string{"email": email}
		if display, _ := m["display_name"].(string); display != "" {
			next["displayName"] = display
		}
		out = append(out, next)
	}
	return out
}

func eventData(event googleapi.CalendarEvent) map[string]any {
	attendees := make([]map[string]any, 0, len(event.Attendees))
	for _, attendee := range event.Attendees {
		attendees = append(attendees, map[string]any{"email": attendee.Email, "response_status": attendee.ResponseStatus})
	}
	return map[string]any{
		"event_id":  event.ID,
		"html_link": event.HTMLLink,
		"status":    event.Status,
		"summary":   event.Summary,
		"start":     firstNonEmpty(event.Start.DateTime, event.Start.Date),
		"end":       firstNonEmpty(event.End.DateTime, event.End.Date),
		"attendees": attendees,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
