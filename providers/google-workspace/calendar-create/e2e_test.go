package calendarcreate_test

import (
	"context"
	"testing"
	"time"

	"cli-factory/providers/google-workspace/internal/e2e"
	"cli-factory/providers/google-workspace/internal/googleapi"
)

func TestE2ECalendarCreateCommand(t *testing.T) {
	secrets := e2e.LoadSecrets(t)
	summary := e2e.Unique("cli-factory-calendar-create")
	start := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second)
	end := start.Add(30 * time.Minute)
	log := e2e.RunFactory(t,
		"google-workspace", "calendar-create",
		"--provider-params-json", secrets.ProviderParamsJSON(t),
		"--params-json", `{"summary":"`+summary+`","start":"`+start.Format(time.RFC3339)+`","end":"`+end.Format(time.RFC3339)+`","description":"Google Workspace calendar-create e2e."}`,
	)
	data := e2e.Data(t, log)
	id, _ := data["event_id"].(string)
	if id == "" {
		t.Fatalf("event_id missing from result: %#v", data)
	}
	client := secrets.Client(t)
	t.Cleanup(func() { _ = client.DeleteEvent(context.Background(), id) })
	events, _, err := client.ListEvents(context.Background(), googleapi.CalendarListInput{
		TimeMin:      start.Add(-time.Hour).Format(time.RFC3339),
		TimeMax:      end.Add(time.Hour).Format(time.RFC3339),
		Query:        summary,
		MaxResults:   10,
		SingleEvents: true,
	})
	if err != nil {
		t.Fatalf("verify created event: %v", err)
	}
	for _, event := range events {
		if event.ID == id {
			return
		}
	}
	t.Fatalf("created event %q was not found", id)
}
