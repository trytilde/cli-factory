package sendemail

import (
	"context"
	"encoding/base64"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"cli-factory/internal/provider"
	"cli-factory/internal/schema"
	"cli-factory/providers/google-workspace/internal/googleapi"
)

type Tool struct{}

func (Tool) ID() string               { return "send-email" }
func (Tool) Name() string             { return "Send Email" }
func (Tool) ShortDescription() string { return "Send a Gmail message to one or more recipients." }
func (Tool) LongDescription() string {
	return "Sends an email through the Gmail API for the authenticated user. Supports To, Cc, Bcc, optional From, plain text or HTML bodies, and dry-run mode."
}
func (Tool) Categories() []string { return []string{"email", "communication"} }
func (Tool) Aliases() []string    { return []string{"send gmail", "email someone", "compose email"} }
func (Tool) InputSchema() schema.JSONSchema {
	return schema.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"user_id":   map[string]any{"type": "string", "description": "Gmail user id or email address. Use \"me\" for the authenticated user."},
			"to":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Email recipients for the To header. CLI accepts comma-separated values."},
			"cc":        map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Optional Cc recipients. CLI accepts comma-separated values."},
			"bcc":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Optional Bcc recipients. CLI accepts comma-separated values."},
			"from":      map[string]any{"type": "string", "description": "Optional From header. Must be allowed by the authenticated Gmail account."},
			"subject":   map[string]any{"type": "string", "description": "Email subject."},
			"body_text": map[string]any{"type": "string", "description": "Plain text message body."},
			"body_html": map[string]any{"type": "string", "description": "HTML message body. When provided, the message is sent as text/html."},
			"dry_run":   map[string]any{"type": "boolean", "description": "Build and validate the message without sending it."},
		},
		"required": []any{"to", "subject"},
	}
}
func (Tool) OutputSchema() schema.JSONSchema {
	return schema.JSONSchema{
		"type": "object",
		"properties": map[string]any{
			"id":              map[string]any{"type": "string", "description": "Gmail message id returned after sending."},
			"thread_id":       map[string]any{"type": "string", "description": "Gmail thread id returned after sending."},
			"label_ids":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Gmail label ids returned after sending."},
			"dry_run":         map[string]any{"type": "boolean", "description": "Whether the message was only built and not sent."},
			"recipient_count": map[string]any{"type": "integer", "description": "Number of To, Cc, and Bcc recipients in the built message."},
		},
	}
}

func (Tool) Invoke(ctx context.Context, req provider.InvokeRequest, events provider.EventSink) (provider.InvokeResult, error) {
	to := googleapi.StringSliceParam(req.Params, "to")
	if len(to) == 0 {
		return provider.InvokeResult{}, &provider.Error{Code: "validation_failed", Message: "to must include at least one recipient", Retryable: false}
	}
	subject := googleapi.StringParam(req.Params, "subject", "")
	bodyText := googleapi.StringParam(req.Params, "body_text", "")
	bodyHTML := googleapi.StringParam(req.Params, "body_html", "")
	if bodyText == "" && bodyHTML == "" {
		return provider.InvokeResult{}, &provider.Error{Code: "validation_failed", Message: "body_text or body_html is required", Retryable: false}
	}

	raw, recipientCount, err := buildRawMessage(req.Params, to)
	if err != nil {
		return provider.InvokeResult{}, err
	}
	if googleapi.BoolParam(req.Params, "dry_run", false) {
		events.Emit(provider.Event{Type: "status", Message: "built Gmail message without sending"})
		return provider.InvokeResult{Data: map[string]any{
			"dry_run":         true,
			"recipient_count": recipientCount,
			"subject":         subject,
		}}, nil
	}

	client, err := googleapi.New(req.ProviderParams, "gmail_base_url", googleapi.DefaultGmailBaseURL)
	if err != nil {
		return provider.InvokeResult{}, err
	}
	userID := url.PathEscape(googleapi.StringParam(req.Params, "user_id", "me"))
	var response struct {
		ID       string   `json:"id"`
		ThreadID string   `json:"threadId"`
		LabelIDs []string `json:"labelIds"`
	}
	events.Emit(provider.Event{Type: "status", Message: "sending Gmail message"})
	err = client.DoJSON(ctx, http.MethodPost, "/gmail/v1/users/"+userID+"/messages/send", nil, map[string]any{"raw": raw}, &response)
	if err != nil {
		return provider.InvokeResult{}, err
	}
	return provider.InvokeResult{Data: map[string]any{
		"id":              response.ID,
		"thread_id":       response.ThreadID,
		"label_ids":       response.LabelIDs,
		"dry_run":         false,
		"recipient_count": recipientCount,
	}}, nil
}

func buildRawMessage(params map[string]any, to []string) (string, int, error) {
	cc := googleapi.StringSliceParam(params, "cc")
	bcc := googleapi.StringSliceParam(params, "bcc")
	subject := googleapi.StringParam(params, "subject", "")
	bodyText := googleapi.StringParam(params, "body_text", "")
	bodyHTML := googleapi.StringParam(params, "body_html", "")
	headers := []string{
		"To: " + strings.Join(to, ", "),
		"Subject: " + mime.QEncoding.Encode("utf-8", sanitizeHeader(subject)),
		"MIME-Version: 1.0",
		"Date: " + time.Now().UTC().Format(time.RFC1123Z),
	}
	if from := googleapi.StringParam(params, "from", ""); from != "" {
		headers = append([]string{"From: " + sanitizeHeader(from)}, headers...)
	}
	if len(cc) > 0 {
		headers = append(headers, "Cc: "+strings.Join(cc, ", "))
	}
	if len(bcc) > 0 {
		headers = append(headers, "Bcc: "+strings.Join(bcc, ", "))
	}
	contentType := "text/plain; charset=UTF-8"
	body := bodyText
	if bodyHTML != "" {
		contentType = "text/html; charset=UTF-8"
		body = bodyHTML
	}
	headers = append(headers, "Content-Type: "+contentType)
	if strings.TrimSpace(body) == "" {
		return "", 0, &provider.Error{Code: "validation_failed", Message: "message body cannot be empty", Retryable: false}
	}
	message := strings.Join(headers, "\r\n") + "\r\n\r\n" + body
	return base64.RawURLEncoding.EncodeToString([]byte(message)), len(to) + len(cc) + len(bcc), nil
}

func sanitizeHeader(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	return strings.TrimSpace(value)
}
