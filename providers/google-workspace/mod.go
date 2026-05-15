package googleworkspace

import (
	"cli-factory/internal/provider"
	"cli-factory/providers/google-workspace/calendarcheck"
	"cli-factory/providers/google-workspace/calendarcreate"
	"cli-factory/providers/google-workspace/gmailcheck"
	"cli-factory/providers/google-workspace/gmailsend"
)

type Provider struct{}

func New() Provider { return Provider{} }

func (Provider) ID() string   { return "google-workspace" }
func (Provider) Name() string { return "Google Workspace" }
func (Provider) ShortDescription() string {
	return "Use Gmail and Google Calendar from a Google Workspace account."
}
func (Provider) LongDescription() string {
	return "Google Workspace exposes curated Gmail and Google Calendar workflows for agents: send and check email, and create and check calendar events with OAuth-backed API access."
}
func (Provider) Categories() []string { return []string{"email", "calendar", "productivity"} }
func (Provider) Aliases() []string    { return []string{"google", "gmail", "calendar"} }
func (Provider) Parameters() []provider.Parameter {
	return []provider.Parameter{
		{Name: "client_id", Description: "Google OAuth 2.0 client ID.", Required: true, Secret: true},
		{Name: "client_secret", Description: "Google OAuth 2.0 client secret.", Required: true, Secret: true},
		{Name: "refresh_token", Description: "OAuth refresh token for the Google Workspace user.", Required: true, Secret: true},
		{Name: "user_id", Description: "Gmail user identifier. Defaults to me.", Required: false},
		{Name: "calendar_id", Description: "Google Calendar ID. Defaults to primary.", Required: false},
	}
}
func (Provider) Tools() []provider.Tool {
	return []provider.Tool{
		gmailsend.Tool{},
		gmailcheck.Tool{},
		calendarcreate.Tool{},
		calendarcheck.Tool{},
	}
}
