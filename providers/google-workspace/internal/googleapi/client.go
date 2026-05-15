package googleapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"cli-factory/internal/provider"
)

const (
	DefaultGmailBaseURL    = "https://gmail.googleapis.com"
	DefaultCalendarBaseURL = "https://www.googleapis.com"
)

type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

func New(providerParams map[string]any, baseURLParam, defaultBaseURL string) (Client, error) {
	token := StringParam(providerParams, "bearer_token", "")
	if token == "" {
		return Client{}, &provider.Error{
			Code:      "missing_auth",
			Message:   "provider parameter bearer_token is required",
			Retryable: false,
		}
	}
	baseURL := StringParam(providerParams, baseURLParam, defaultBaseURL)
	baseURL = strings.TrimRight(baseURL, "/")
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return Client{}, &provider.Error{
			Code:      "invalid_provider_param",
			Message:   fmt.Sprintf("%s must be an absolute URL", baseURLParam),
			Retryable: false,
		}
	}
	return Client{BaseURL: baseURL, Token: token, HTTP: http.DefaultClient}, nil
}

func (c Client) DoJSON(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	endpoint, err := url.Parse(c.BaseURL + path)
	if err != nil {
		return err
	}
	if query != nil {
		endpoint.RawQuery = query.Encode()
	}
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), reader)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return &provider.Error{Code: "request_failed", Message: err.Error(), Retryable: true}
	}
	defer resp.Body.Close()
	data, readErr := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if readErr != nil {
		return readErr
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return GoogleError(resp.StatusCode, data)
	}
	if out == nil || len(data) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return err
	}
	return nil
}

func GoogleError(status int, data []byte) *provider.Error {
	var parsed struct {
		Error struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
			Status  string `json:"status"`
		} `json:"error"`
	}
	_ = json.Unmarshal(data, &parsed)
	message := strings.TrimSpace(parsed.Error.Message)
	if message == "" {
		message = strings.TrimSpace(string(data))
	}
	if message == "" {
		message = http.StatusText(status)
	}
	code := "google_api_failed"
	if status == http.StatusUnauthorized || status == http.StatusForbidden {
		code = "google_auth_failed"
	}
	if status == http.StatusTooManyRequests || status >= 500 {
		code = "google_api_retryable"
	}
	return &provider.Error{
		Code:           code,
		Message:        message,
		Retryable:      status == http.StatusTooManyRequests || status >= 500,
		ProviderStatus: fmt.Sprintf("%d", status),
		Details: map[string]any{
			"google_status": parsed.Error.Status,
		},
	}
}

func StringParam(params map[string]any, name, fallback string) string {
	value, ok := params[name]
	if !ok || value == nil {
		return fallback
	}
	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return fallback
		}
		return strings.TrimSpace(v)
	default:
		return fmt.Sprint(v)
	}
}

func BoolParam(params map[string]any, name string, fallback bool) bool {
	value, ok := params[name]
	if !ok || value == nil {
		return fallback
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		default:
			return fallback
		}
	default:
		return fallback
	}
}

func IntParam(params map[string]any, name string, fallback, min, max int) int {
	value, ok := params[name]
	if !ok || value == nil {
		return fallback
	}
	var parsed int
	switch v := value.(type) {
	case float64:
		parsed = int(v)
	case int:
		parsed = v
	case string:
		if _, err := fmt.Sscanf(strings.TrimSpace(v), "%d", &parsed); err != nil {
			return fallback
		}
	default:
		return fallback
	}
	if parsed < min {
		return min
	}
	if parsed > max {
		return max
	}
	return parsed
}

func StringSliceParam(params map[string]any, name string) []string {
	value, ok := params[name]
	if !ok || value == nil {
		return nil
	}
	var out []string
	switch v := value.(type) {
	case []any:
		for _, item := range v {
			addCSV(&out, fmt.Sprint(item))
		}
	case []string:
		for _, item := range v {
			addCSV(&out, item)
		}
	case string:
		addCSV(&out, v)
	default:
		addCSV(&out, fmt.Sprint(v))
	}
	return out
}

func addCSV(out *[]string, value string) {
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			*out = append(*out, part)
		}
	}
}
