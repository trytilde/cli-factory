package createcalendarevent

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"cli-factory/internal/provider"
	"cli-factory/internal/schema"
	"cli-factory/providers/google-workspace/internal/googleapi"
)

type Tool struct{}

func (Tool) ID() string               { return "create-calendar-event" }
func (Tool) Name() string             { return "Create Calendar Event" }
func (Tool) ShortDescription() string { return "Create an event on a Google Calendar." }
func (Tool) LongDescription() string {
	return "Creates a Google Calendar event with summary, start and end times, optional attendees, location, description, timezone, and Google Meet conference data. A dry-run mode validates and returns the event payload without writing."
}
func (Tool) Categories() []string { return []string{"calendar", "scheduling"} }
func (Tool) Aliases() []string {
	return []string{"schedule meeting", "create google calendar event", "add calendar event"}
}
func (Tool) InputSchema() schema.JSONSchema {
	return schema.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"calendar_id": map[string]any{"type": "string", "description": "Calendar identifier. Use \"primary\" for the authenticated user's primary calendar."},
			"summary":     map[string]any{"type": "string", "description": "Event title."},
			"description": map[string]any{"type": "string", "description": "Optional event description."},
			"location":    map[string]any{"type": "string", "description": "Optional event location."},
			"start":       map[string]any{"type": "string", "description": "RFC3339 start datetime, or YYYY-MM-DD for all-day events."},
			"end":         map[string]any{"type": "string", "description": "RFC3339 end datetime, or YYYY-MM-DD for all-day events."},
			"timezone":    map[string]any{"type": "string", "description": "IANA timezone for date-time events, such as America/New_York."},
			"attendees":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Attendee email addresses. CLI accepts comma-separated values."},
			"create_meet": map[string]any{"type": "boolean", "description": "Request Google Meet conference data."},
			"dry_run":     map[string]any{"type": "boolean", "description": "Build and validate the event payload without creating it."},
		},
		"required": []any{"summary", "start", "end"},
	}
}
func (Tool) OutputSchema() schema.JSONSchema {
	return schema.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"id":        map[string]any{"type": "string", "description": "Created Google Calendar event id."},
			"html_link": map[string]any{"type": "string", "description": "Browser URL for the event."},
			"status":    map[string]any{"type": "string", "description": "Event status returned by Google Calendar."},
			"dry_run":   map[string]any{"type": "boolean", "description": "Whether the event was only built and not created."},
			"event":     map[string]any{"type": "object", "description": "Event payload for dry-run responses."},
		},
	}
}

func (Tool) Invoke(ctx context.Context, req provider.InvokeRequest, events provider.EventSink) (provider.InvokeResult, error) {
	event, query, err := buildEvent(req.Params)
	if err != nil {
		return provider.InvokeResult{}, err
	}
	if googleapi.BoolParam(req.Params, "dry_run", false) {
		events.Emit(provider.Event{Type: "status", Message: "built calendar event without creating it"})
		return provider.InvokeResult{Data: map[string]any{"dry_run": true, "event": event}}, nil
	}
	client, err := googleapi.New(req.ProviderParams, "calendar_base_url", googleapi.DefaultCalendarBaseURL)
	if err != nil {
		return provider.InvokeResult{}, err
	}
	calendarID := url.PathEscape(googleapi.StringParam(req.Params, "calendar_id", "primary"))
	var response struct {
		ID       string `json:"id"`
		HTMLLink string `json:"htmlLink"`
		Status   string `json:"status"`
	}
	events.Emit(provider.Event{Type: "status", Message: "creating Google Calendar event"})
	if err := client.DoJSON(ctx, http.MethodPost, "/calendar/v3/calendars/"+calendarID+"/events", query, event, &response); err != nil {
		return provider.InvokeResult{}, err
	}
	return provider.InvokeResult{Data: map[string]any{
		"id":        response.ID,
		"html_link": response.HTMLLink,
		"status":    response.Status,
		"dry_run":   false,
	}}, nil
}

func buildEvent(params map[string]any) (map[string]any, url.Values, error) {
	start := googleapi.StringParam(params, "start", "")
	end := googleapi.StringParam(params, "end", "")
	if start == "" || end == "" {
		return nil, nil, &provider.Error{Code: "validation_failed", Message: "start and end are required", Retryable: false}
	}
	event := map[string]any{
		"summary": googleapi.StringParam(params, "summary", ""),
		"start":   eventTime(start, googleapi.StringParam(params, "timezone", "")),
		"end":     eventTime(end, googleapi.StringParam(params, "timezone", "")),
	}
	if description := googleapi.StringParam(params, "description", ""); description != "" {
		event["description"] = description
	}
	if location := googleapi.StringParam(params, "location", ""); location != "" {
		event["location"] = location
	}
	if attendees := googleapi.StringSliceParam(params, "attendees"); len(attendees) > 0 {
		items := make([]map[string]any, 0, len(attendees))
		for _, attendee := range attendees {
			items = append(items, map[string]any{"email": attendee})
		}
		event["attendees"] = items
	}
	query := url.Values{}
	if googleapi.BoolParam(params, "create_meet", false) {
		query.Set("conferenceDataVersion", "1")
		event["conferenceData"] = map[string]any{
			"createRequest": map[string]any{
				"requestId": strings.ReplaceAll(googleapi.StringParam(params, "summary", "event"), " ", "-"),
			},
		}
	}
	return event, query, nil
}

func eventTime(value, timezone string) map[string]any {
	if len(value) == len("2006-01-02") && strings.Count(value, "-") == 2 {
		return map[string]any{"date": value}
	}
	out := map[string]any{"dateTime": value}
	if timezone != "" {
		out["timeZone"] = timezone
	}
	return out
}
