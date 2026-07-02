package appleads

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/crevas/Apple-Ads-CLI/internal/auth"
)

type Client struct {
	BaseURL       string
	ContextHeader string
	TokenSource   *auth.TokenSource
	HTTPClient    *http.Client
	LogWriter     io.Writer
	Verbose       bool
}

func NewClient(baseURL string, contextHeader string, timeout time.Duration, tokenSource *auth.TokenSource) *Client {
	return &Client{
		BaseURL:       strings.TrimRight(baseURL, "/"),
		ContextHeader: contextHeader,
		TokenSource:   tokenSource,
		HTTPClient:    &http.Client{Timeout: timeout},
		LogWriter:     io.Discard,
	}
}

func (c *Client) Do(method string, path string, body any) (RawResponse, error) {
	return c.DoWithContext(context.Background(), method, path, body)
}

func (c *Client) DoWithContext(ctx context.Context, method string, path string, body any) (RawResponse, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reader = bytes.NewReader(data)
	}

	token, err := c.TokenSource.Token(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if c.ContextHeader != "" {
		req.Header.Set("X-AP-Context", c.ContextHeader)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if c.Verbose {
		fmt.Fprintf(c.LogWriter, "%s %s\n", method, req.URL.String())
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request apple ads: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode == http.StatusNoContent {
		return RawResponse{}, nil
	}

	var parsed map[string]any
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, fmt.Errorf("parse response (%d): %w: %s", resp.StatusCode, err, string(raw))
		}
	}
	if resp.StatusCode >= 400 {
		return nil, APIError{StatusCode: resp.StatusCode, Body: parsed, Raw: string(raw)}
	}
	return RawResponse(parsed), nil
}

type APIError struct {
	StatusCode int
	Body       map[string]any
	Raw        string
}

func (e APIError) Error() string {
	if msg := platformErrorMessage(e.Body); msg != "" {
		return fmt.Sprintf("apple ads api error %d: %s", e.StatusCode, msg)
	}
	if e.Raw != "" {
		return fmt.Sprintf("apple ads api error %d: %s", e.StatusCode, e.Raw)
	}
	return fmt.Sprintf("apple ads api error %d", e.StatusCode)
}

func platformErrorMessage(body map[string]any) string {
	if body == nil {
		return ""
	}
	if errorObj, ok := body["error"].(map[string]any); ok {
		if msg, ok := errorObj["message"].(string); ok && msg != "" {
			return msg
		}
		if details, ok := errorObj["details"].([]any); ok && len(details) > 0 {
			if detail, ok := details[0].(map[string]any); ok {
				if msg, ok := detail["message"].(string); ok && msg != "" {
					return msg
				}
			}
		}
		if errors, ok := errorObj["errors"].([]any); ok && len(errors) > 0 {
			if detail, ok := errors[0].(map[string]any); ok {
				if msg, ok := detail["message"].(string); ok && msg != "" {
					return msg
				}
			}
		}
	}
	if msg, ok := body["message"].(string); ok && msg != "" {
		return msg
	}
	return ""
}
