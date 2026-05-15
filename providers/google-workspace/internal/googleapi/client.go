package googleapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/quotedprintable"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"cli-factory/internal/provider"
)

const (
	gmailBaseURL    = "https://gmail.googleapis.com/gmail/v1"
	calendarBaseURL = "https://www.googleapis.com/calendar/v3"
	tokenURL        = "https://oauth2.googleapis.com/token"
)

type Config struct {
	ClientID     string
	ClientSecret string
	RefreshToken string
	UserID       string
	CalendarID   string
}

type Client struct {
	Config Config
	HTTP   *http.Client
	token  string
}

func New(params map[string]any) (*Client, error) {
	cfg := Config{
		ClientID:     stringParam(params, "client_id"),
		ClientSecret: stringParam(params, "client_secret"),
		RefreshToken: stringParam(params, "refresh_token"),
		UserID:       defaultString(stringParam(params, "user_id"), "me"),
		CalendarID:   defaultString(stringParam(params, "calendar_id"), "primary"),
	}
	var missing []string
	if cfg.ClientID == "" {
		missing = append(missing, "client_id")
	}
	if cfg.ClientSecret == "" {
		missing = append(missing, "client_secret")
	}
	if cfg.RefreshToken == "" {
		missing = append(missing, "refresh_token")
	}
	if len(missing) > 0 {
		return nil, &provider.Error{Code: "validation_failed", Message: "missing provider params: " + strings.Join(missing, ", "), Retryable: false}
	}
	return &Client{Config: cfg, HTTP: http.DefaultClient}, nil
}

func (c *Client) SendGmail(ctx context.Context, msg EmailMessage) (GmailMessage, error) {
	raw, err := BuildRawEmail(msg)
	if err != nil {
		return GmailMessage{}, err
	}
	var out GmailMessage
	err = c.do(ctx, http.MethodPost, gmailBaseURL+"/users/"+url.PathEscape(c.Config.UserID)+"/messages/send", nil, map[string]any{"raw": raw}, &out)
	return out, err
}

func (c *Client) ListGmail(ctx context.Context, query string, maxResults int, includeSnippet bool) ([]GmailMessage, error) {
	if maxResults <= 0 {
		maxResults = 10
	}
	if maxResults > 25 {
		maxResults = 25
	}
	values := url.Values{}
	values.Set("maxResults", strconv.Itoa(maxResults))
	if query != "" {
		values.Set("q", query)
	}
	var listed struct {
		Messages []GmailMessage `json:"messages"`
	}
	err := c.do(ctx, http.MethodGet, gmailBaseURL+"/users/"+url.PathEscape(c.Config.UserID)+"/messages?"+values.Encode(), nil, nil, &listed)
	if err != nil {
		return nil, err
	}
	if !includeSnippet {
		return listed.Messages, nil
	}
	out := make([]GmailMessage, 0, len(listed.Messages))
	for _, msg := range listed.Messages {
		detail, err := c.GetGmail(ctx, msg.ID)
		if err != nil {
			return nil, err
		}
		out = append(out, detail)
	}
	return out, nil
}

func (c *Client) GetGmail(ctx context.Context, id string) (GmailMessage, error) {
	var out GmailMessage
	values := url.Values{}
	values.Set("format", "metadata")
	err := c.do(ctx, http.MethodGet, gmailBaseURL+"/users/"+url.PathEscape(c.Config.UserID)+"/messages/"+url.PathEscape(id)+"?"+values.Encode(), nil, nil, &out)
	return out, err
}

func (c *Client) TrashGmail(ctx context.Context, id string) error {
	var out map[string]any
	return c.do(ctx, http.MethodPost, gmailBaseURL+"/users/"+url.PathEscape(c.Config.UserID)+"/messages/"+url.PathEscape(id)+"/trash", nil, map[string]any{}, &out)
}

func (c *Client) CreateEvent(ctx context.Context, input CalendarEventInput) (CalendarEvent, error) {
	payload := map[string]any{
		"summary": input.Summary,
		"start":   eventTime(input.Start, input.TimeZone),
		"end":     eventTime(input.End, input.TimeZone),
	}
	if input.Description != "" {
		payload["description"] = input.Description
	}
	if input.Location != "" {
		payload["location"] = input.Location
	}
	if len(input.Attendees) > 0 {
		payload["attendees"] = input.Attendees
	}
	var out CalendarEvent
	err := c.do(ctx, http.MethodPost, calendarBaseURL+"/calendars/"+url.PathEscape(c.Config.CalendarID)+"/events", nil, payload, &out)
	return out, err
}

func (c *Client) ListEvents(ctx context.Context, input CalendarListInput) ([]CalendarEvent, string, error) {
	maxResults := input.MaxResults
	if maxResults <= 0 {
		maxResults = 10
	}
	if maxResults > 50 {
		maxResults = 50
	}
	values := url.Values{}
	values.Set("timeMin", input.TimeMin)
	values.Set("timeMax", input.TimeMax)
	values.Set("maxResults", strconv.Itoa(maxResults))
	if input.Query != "" {
		values.Set("q", input.Query)
	}
	if input.SingleEvents {
		values.Set("singleEvents", "true")
		values.Set("orderBy", "startTime")
	}
	var out struct {
		Items         []CalendarEvent `json:"items"`
		NextPageToken string          `json:"nextPageToken"`
	}
	err := c.do(ctx, http.MethodGet, calendarBaseURL+"/calendars/"+url.PathEscape(c.Config.CalendarID)+"/events?"+values.Encode(), nil, nil, &out)
	return out.Items, out.NextPageToken, err
}

func (c *Client) DeleteEvent(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, calendarBaseURL+"/calendars/"+url.PathEscape(c.Config.CalendarID)+"/events/"+url.PathEscape(id), nil, nil, nil)
}

func (c *Client) do(ctx context.Context, method, endpoint string, headers map[string]string, payload any, out any) error {
	token, err := c.accessToken(ctx)
	if err != nil {
		return err
	}
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return &provider.Error{Code: "provider_request_failed", Message: err.Error(), Retryable: true}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return parseGoogleError(resp)
	}
	if out == nil || resp.StatusCode == http.StatusNoContent {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) accessToken(ctx context.Context) (string, error) {
	if c.token != "" {
		return c.token, nil
	}
	form := url.Values{}
	form.Set("client_id", c.Config.ClientID)
	form.Set("client_secret", c.Config.ClientSecret)
	form.Set("refresh_token", c.Config.RefreshToken)
	form.Set("grant_type", "refresh_token")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", &provider.Error{Code: "auth_failed", Message: err.Error(), Retryable: true}
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", parseGoogleError(resp)
	}
	var parsed struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", err
	}
	if parsed.AccessToken == "" {
		return "", &provider.Error{Code: "auth_failed", Message: "Google token response did not include access_token", Retryable: false}
	}
	c.token = parsed.AccessToken
	return c.token, nil
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

type EmailMessage struct {
	To       []string
	CC       []string
	BCC      []string
	Subject  string
	BodyText string
}

type GmailMessage struct {
	ID       string   `json:"id"`
	ThreadID string   `json:"threadId"`
	Snippet  string   `json:"snippet"`
	LabelIDs []string `json:"labelIds"`
}

func BuildRawEmail(msg EmailMessage) (string, error) {
	if len(msg.To) == 0 {
		return "", &provider.Error{Code: "validation_failed", Message: "to is required", Retryable: false}
	}
	if strings.TrimSpace(msg.Subject) == "" {
		return "", &provider.Error{Code: "validation_failed", Message: "subject is required", Retryable: false}
	}
	if strings.TrimSpace(msg.BodyText) == "" {
		return "", &provider.Error{Code: "validation_failed", Message: "body_text is required", Retryable: false}
	}
	headers := map[string]string{
		"To":           strings.Join(msg.To, ", "),
		"Subject":      mimeHeader(msg.Subject),
		"MIME-Version": "1.0",
		"Content-Type": `text/plain; charset="UTF-8"`,
	}
	if len(msg.CC) > 0 {
		headers["Cc"] = strings.Join(msg.CC, ", ")
	}
	if len(msg.BCC) > 0 {
		headers["Bcc"] = strings.Join(msg.BCC, ", ")
	}
	keys := make([]string, 0, len(headers))
	for k := range headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf bytes.Buffer
	for _, k := range keys {
		fmt.Fprintf(&buf, "%s: %s\r\n", k, headers[k])
	}
	buf.WriteString("\r\n")
	qp := quotedprintable.NewWriter(&buf)
	_, _ = qp.Write([]byte(msg.BodyText))
	_ = qp.Close()
	return base64.RawURLEncoding.EncodeToString(buf.Bytes()), nil
}

func mimeHeader(value string) string {
	return mime.QEncoding.Encode("UTF-8", value)
}

type CalendarEventInput struct {
	Summary     string
	Start       string
	End         string
	TimeZone    string
	Description string
	Location    string
	Attendees   []map[string]string
}

type CalendarListInput struct {
	TimeMin      string
	TimeMax      string
	Query        string
	MaxResults   int
	SingleEvents bool
}

type CalendarEvent struct {
	ID        string             `json:"id"`
	Summary   string             `json:"summary"`
	Status    string             `json:"status"`
	HTMLLink  string             `json:"htmlLink"`
	Start     CalendarEventTime  `json:"start"`
	End       CalendarEventTime  `json:"end"`
	Location  string             `json:"location"`
	Attendees []CalendarAttendee `json:"attendees"`
	Raw       map[string]any     `json:"-"`
}

type CalendarEventTime struct {
	DateTime string `json:"dateTime"`
	Date     string `json:"date"`
	TimeZone string `json:"timeZone"`
}

type CalendarAttendee struct {
	Email          string `json:"email"`
	ResponseStatus string `json:"responseStatus"`
}

func eventTime(value, tz string) map[string]string {
	out := map[string]string{"dateTime": value}
	if tz != "" {
		out["timeZone"] = tz
	}
	return out
}

func parseGoogleError(resp *http.Response) error {
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	var parsed struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}
	message := strings.TrimSpace(string(data))
	code := strings.ToLower(strings.ReplaceAll(resp.Status, " ", "_"))
	if json.Unmarshal(data, &parsed) == nil && parsed.Error.Message != "" {
		message = parsed.Error.Message
		if parsed.Error.Status != "" {
			code = strings.ToLower(parsed.Error.Status)
		}
	}
	if message == "" {
		message = resp.Status
	}
	return &provider.Error{Code: code, Message: message, ProviderStatus: resp.Status, Retryable: resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500}
}

func stringParam(params map[string]any, name string) string {
	value, _ := params[name].(string)
	return strings.TrimSpace(value)
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func StringSlice(value any) []string {
	switch v := value.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if v == "" {
			return nil
		}
		return []string{v}
	default:
		return nil
	}
}

func StringValue(params map[string]any, name string) string {
	value, _ := params[name].(string)
	return strings.TrimSpace(value)
}

func IntValue(params map[string]any, name string, fallback int) int {
	switch v := params[name].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case string:
		if parsed, err := strconv.Atoi(v); err == nil {
			return parsed
		}
	}
	return fallback
}

func BoolValue(params map[string]any, name string, fallback bool) bool {
	switch v := params[name].(type) {
	case bool:
		return v
	case string:
		switch strings.ToLower(v) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		}
	}
	return fallback
}

func RFC3339(value, name string) error {
	if value == "" {
		return &provider.Error{Code: "validation_failed", Message: name + " is required", Retryable: false}
	}
	if _, err := time.Parse(time.RFC3339, value); err != nil {
		return &provider.Error{Code: "validation_failed", Message: name + " must be RFC3339 date-time", Retryable: false}
	}
	return nil
}
