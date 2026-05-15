package checkemails

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"cli-factory/internal/provider"
	"cli-factory/internal/schema"
	"cli-factory/providers/google-workspace/internal/googleapi"
)

type Tool struct{}

func (Tool) ID() string   { return "check-emails" }
func (Tool) Name() string { return "Check Emails" }
func (Tool) ShortDescription() string {
	return "Search recent Gmail messages and return readable summaries."
}
func (Tool) LongDescription() string {
	return "Lists Gmail messages matching an optional Gmail search query and fetches metadata for each result so agents can inspect sender, recipients, subject, date, snippet, message id, and thread id."
}
func (Tool) Categories() []string { return []string{"email", "communication"} }
func (Tool) Aliases() []string    { return []string{"search gmail", "read inbox", "check mail"} }
func (Tool) InputSchema() schema.JSONSchema {
	return schema.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"user_id":            map[string]any{"type": "string", "description": "Gmail user id or email address. Use \"me\" for the authenticated user."},
			"query":              map[string]any{"type": "string", "description": "Gmail search query, such as \"from:alice@example.com is:unread\"."},
			"label_ids":          map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Gmail label IDs to filter by. CLI accepts comma-separated values."},
			"max_results":        map[string]any{"type": "integer", "description": "Maximum messages to return, capped at 50."},
			"include_spam_trash": map[string]any{"type": "boolean", "description": "Include messages from Spam and Trash."},
		},
	}
}
func (Tool) OutputSchema() schema.JSONSchema {
	return schema.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"messages":             map[string]any{"type": "array", "items": map[string]any{"type": "object"}, "description": "Matching Gmail message summaries."},
			"result_size_estimate": map[string]any{"type": "integer", "description": "Gmail estimate of total matching results."},
			"next_page_token":      map[string]any{"type": "string", "description": "Token for the next Gmail result page when available."},
		},
	}
}

func (Tool) Invoke(ctx context.Context, req provider.InvokeRequest, events provider.EventSink) (provider.InvokeResult, error) {
	client, err := googleapi.New(req.ProviderParams, "gmail_base_url", googleapi.DefaultGmailBaseURL)
	if err != nil {
		return provider.InvokeResult{}, err
	}
	userID := url.PathEscape(googleapi.StringParam(req.Params, "user_id", "me"))
	maxResults := googleapi.IntParam(req.Params, "max_results", 10, 1, 50)
	query := url.Values{}
	query.Set("maxResults", strconv.Itoa(maxResults))
	if q := googleapi.StringParam(req.Params, "query", ""); q != "" {
		query.Set("q", q)
	}
	for _, labelID := range googleapi.StringSliceParam(req.Params, "label_ids") {
		query.Add("labelIds", labelID)
	}
	if googleapi.BoolParam(req.Params, "include_spam_trash", false) {
		query.Set("includeSpamTrash", "true")
	}
	var listed listResponse
	events.Emit(provider.Event{Type: "status", Message: "listing Gmail messages"})
	if err := client.DoJSON(ctx, http.MethodGet, "/gmail/v1/users/"+userID+"/messages", query, nil, &listed); err != nil {
		return provider.InvokeResult{}, err
	}
	messages := make([]map[string]any, 0, len(listed.Messages))
	for _, item := range listed.Messages {
		var detail messageResponse
		getQuery := url.Values{}
		getQuery.Set("format", "metadata")
		for _, header := range []string{"From", "To", "Cc", "Subject", "Date"} {
			getQuery.Add("metadataHeaders", header)
		}
		path := "/gmail/v1/users/" + userID + "/messages/" + url.PathEscape(item.ID)
		if err := client.DoJSON(ctx, http.MethodGet, path, getQuery, nil, &detail); err != nil {
			return provider.InvokeResult{}, err
		}
		messages = append(messages, summarize(detail))
	}
	return provider.InvokeResult{Data: map[string]any{
		"messages":             messages,
		"result_size_estimate": listed.ResultSizeEstimate,
		"next_page_token":      listed.NextPageToken,
	}}, nil
}

type listResponse struct {
	Messages           []messageRef `json:"messages"`
	NextPageToken      string       `json:"nextPageToken"`
	ResultSizeEstimate int          `json:"resultSizeEstimate"`
}

type messageRef struct {
	ID       string `json:"id"`
	ThreadID string `json:"threadId"`
}

type messageResponse struct {
	ID           string `json:"id"`
	ThreadID     string `json:"threadId"`
	Snippet      string `json:"snippet"`
	InternalDate string `json:"internalDate"`
	Payload      struct {
		Headers []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"headers"`
	} `json:"payload"`
}

func summarize(message messageResponse) map[string]any {
	headers := map[string]string{}
	for _, header := range message.Payload.Headers {
		headers[strings.ToLower(header.Name)] = header.Value
	}
	return map[string]any{
		"id":            message.ID,
		"thread_id":     message.ThreadID,
		"snippet":       message.Snippet,
		"internal_date": message.InternalDate,
		"from":          headers["from"],
		"to":            headers["to"],
		"cc":            headers["cc"],
		"subject":       headers["subject"],
		"date":          headers["date"],
	}
}
