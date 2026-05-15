package calendarcheck_test

import (
	"context"
	"testing"
	"time"

	"cli-factory/providers/google-workspace/internal/e2e"
	"cli-factory/providers/google-workspace/internal/googleapi"
)

func TestE2ECalendarCheckCommand(t *testing.T) {
	secrets := e2e.LoadSecrets(t)
	client := secrets.Client(t)
	summary := e2e.Unique("cli-factory-calendar-check")
	start := time.Now().UTC().Add(3 * time.Hour).Truncate(time.Second)
	end := start.Add(30 * time.Minute)
	event, err := client.CreateEvent(context.Background(), googleapi.CalendarEventInput{
		Summary:     summary,
		Start:       start.Format(time.RFC3339),
		End:         end.Format(time.RFC3339),
		Description: "Google Workspace calendar-check e2e.",
	})
	if err != nil {
		t.Fatalf("seed Calendar event: %v", err)
	}
	t.Cleanup(func() { _ = client.DeleteEvent(context.Background(), event.ID) })
	log := e2e.RunFactory(t,
		"google-workspace", "calendar-check",
		"--provider-params-json", secrets.ProviderParamsJSON(t),
		"--params-json", `{"time_min":"`+start.Add(-time.Hour).Format(time.RFC3339)+`","time_max":"`+end.Add(time.Hour).Format(time.RFC3339)+`","query":"`+summary+`","max_results":10,"single_events":true}`,
	)
	data := e2e.Data(t, log)
	if count, _ := data["result_count"].(float64); count < 1 {
		t.Fatalf("result_count = %v, want at least 1: %#v", data["result_count"], data)
	}
}
