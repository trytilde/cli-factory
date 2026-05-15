package checkcalendarevents

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"cli-factory/internal/provider"
	"cli-factory/internal/schema"
	"cli-factory/providers/google-workspace/internal/googleapi"
)

type Tool struct{}

func (Tool) ID() string               { return "check-calendar-events" }
func (Tool) Name() string             { return "Check Calendar Events" }
func (Tool) ShortDescription() string { return "List Google Calendar events in a time window." }
func (Tool) LongDescription() string {
	return "Lists events from a Google Calendar with optional time bounds, text query, attendee expansion, ordering, and single-event expansion for recurring events."
}
func (Tool) Categories() []string { return []string{"calendar", "scheduling"} }
func (Tool) Aliases() []string    { return []string{"list calendar", "check schedule", "upcoming events"} }
func (Tool) InputSchema() schema.JSONSchema {
	return schema.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"calendar_id":   map[string]any{"type": "string", "description": "Calendar identifier. Use \"primary\" for the authenticated user's primary calendar."},
			"time_min":      map[string]any{"type": "string", "description": "Lower bound RFC3339 datetime for event end time."},
			"time_max":      map[string]any{"type": "string", "description": "Upper bound RFC3339 datetime for event start time."},
			"query":         map[string]any{"type": "string", "description": "Free-text search terms for events."},
			"max_results":   map[string]any{"type": "integer", "description": "Maximum events to return, capped at 50."},
			"single_events": map[string]any{"type": "boolean", "description": "Expand recurring events into individual instances."},
			"show_deleted":  map[string]any{"type": "boolean", "description": "Include deleted events."},
		},
	}
}
func (Tool) OutputSchema() schema.JSONSchema {
	return schema.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"events":          map[string]any{"type": "array", "items": map[string]any{"type": "object"}, "description": "Matching calendar event summaries."},
			"next_page_token": map[string]any{"type": "string", "description": "Token for the next Calendar result page when available."},
			"time_zone":       map[string]any{"type": "string", "description": "Calendar timezone returned by Google Calendar."},
		},
	}
}

func (Tool) Invoke(ctx context.Context, req provider.InvokeRequest, events provider.EventSink) (provider.InvokeResult, error) {
	client, err := googleapi.New(req.ProviderParams, "calendar_base_url", googleapi.DefaultCalendarBaseURL)
	if err != nil {
		return provider.InvokeResult{}, err
	}
	calendarID := url.PathEscape(googleapi.StringParam(req.Params, "calendar_id", "primary"))
	query := url.Values{}
	query.Set("maxResults", strconv.Itoa(googleapi.IntParam(req.Params, "max_results", 10, 1, 50)))
	query.Set("singleEvents", boolString(googleapi.BoolParam(req.Params, "single_events", true)))
	query.Set("showDeleted", boolString(googleapi.BoolParam(req.Params, "show_deleted", false)))
	if googleapi.BoolParam(req.Params, "single_events", true) {
		query.Set("orderBy", "startTime")
	}
	if value := googleapi.StringParam(req.Params, "time_min", ""); value != "" {
		query.Set("timeMin", value)
	}
	if value := googleapi.StringParam(req.Params, "time_max", ""); value != "" {
		query.Set("timeMax", value)
	}
	if value := googleapi.StringParam(req.Params, "query", ""); value != "" {
		query.Set("q", value)
	}
	var response eventsResponse
	events.Emit(provider.Event{Type: "status", Message: "listing Google Calendar events"})
	if err := client.DoJSON(ctx, http.MethodGet, "/calendar/v3/calendars/"+calendarID+"/events", query, nil, &response); err != nil {
		return provider.InvokeResult{}, err
	}
	items := make([]map[string]any, 0, len(response.Items))
	for _, item := range response.Items {
		items = append(items, map[string]any{
			"id":          item.ID,
			"summary":     item.Summary,
			"description": item.Description,
			"location":    item.Location,
			"status":      item.Status,
			"html_link":   item.HTMLLink,
			"start":       item.Start,
			"end":         item.End,
			"attendees":   item.Attendees,
		})
	}
	return provider.InvokeResult{Data: map[string]any{
		"events":          items,
		"next_page_token": response.NextPageToken,
		"time_zone":       response.TimeZone,
	}}, nil
}

type eventsResponse struct {
	Items         []eventItem `json:"items"`
	NextPageToken string      `json:"nextPageToken"`
	TimeZone      string      `json:"timeZone"`
}

type eventItem struct {
	ID          string           `json:"id"`
	Summary     string           `json:"summary"`
	Description string           `json:"description"`
	Location    string           `json:"location"`
	Status      string           `json:"status"`
	HTMLLink    string           `json:"htmlLink"`
	Start       map[string]any   `json:"start"`
	End         map[string]any   `json:"end"`
	Attendees   []map[string]any `json:"attendees"`
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}
