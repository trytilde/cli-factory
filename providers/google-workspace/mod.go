package googleworkspace

import (
	"cli-factory/internal/provider"
	checkcalendarevents "cli-factory/providers/google-workspace/check-calendar-events"
	checkemails "cli-factory/providers/google-workspace/check-emails"
	createcalendarevent "cli-factory/providers/google-workspace/create-calendar-event"
	sendemail "cli-factory/providers/google-workspace/send-email"
)

type Provider struct{}

func New() Provider { return Provider{} }

func (Provider) ID() string   { return "google-workspace" }
func (Provider) Name() string { return "Google Workspace" }
func (Provider) ShortDescription() string {
	return "Send Gmail messages and manage Google Calendar events."
}
func (Provider) LongDescription() string {
	return "Google Workspace provides curated Gmail and Google Calendar tools for sending email, checking email, creating calendar events, and checking calendar events."
}
func (Provider) Categories() []string {
	return []string{"communication", "calendar", "email", "productivity"}
}
func (Provider) Aliases() []string {
	return []string{"google", "gmail", "google calendar", "workspace"}
}
func (Provider) Parameters() []provider.Parameter {
	return []provider.Parameter{
		{Name: "bearer_token", Description: "OAuth 2.0 access token with the Gmail or Calendar scopes required by the selected tool.", Required: true, Secret: true},
		{Name: "gmail_base_url", Description: "Optional Gmail API base URL. Defaults to https://gmail.googleapis.com.", Required: false},
		{Name: "calendar_base_url", Description: "Optional Calendar API base URL. Defaults to https://www.googleapis.com.", Required: false},
	}
}
func (Provider) Tools() []provider.Tool {
	return []provider.Tool{
		sendemail.Tool{},
		checkemails.Tool{},
		createcalendarevent.Tool{},
		checkcalendarevents.Tool{},
	}
}
