package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// LogsAPI defines the interface for Datadog Logs API operations.
// This interface enables testing with mocks or httptest servers.
type LogsAPI interface {
	SearchLogs(ctx context.Context, params SearchLogsParams) (*LogsListResponse, error)
	AggregateLogs(ctx context.Context, params AggregateLogsParams) (*LogsAggregateResponse, error)
}

// Client is the concrete Datadog API client.
type Client struct {
	BaseURL    string
	APIKey     string
	AppKey     string
	HTTPClient *http.Client
}

// NewClient creates a new Datadog API client.
// baseURL should be like "https://api.datadoghq.com".
func NewClient(baseURL, apiKey, appKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		AppKey:  appKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) do(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("DD-API-KEY", c.APIKey)
	req.Header.Set("DD-APPLICATION-KEY", c.AppKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("executing request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, parseAPIError(resp.StatusCode, respBody)
	}

	return respBody, nil
}

// APIError represents an error response from the Datadog API.
type APIError struct {
	StatusCode int
	Errors     []string `json:"errors"`
}

func (e *APIError) Error() string {
	if len(e.Errors) > 0 {
		return fmt.Sprintf("datadog api error (%d): %s", e.StatusCode, e.Errors[0])
	}
	return fmt.Sprintf("datadog api error (%d)", e.StatusCode)
}

func parseAPIError(statusCode int, body []byte) error {
	apiErr := &APIError{StatusCode: statusCode}
	// Try to parse error details; if that fails, return with just the status code
	_ = json.Unmarshal(body, apiErr)
	return apiErr
}
