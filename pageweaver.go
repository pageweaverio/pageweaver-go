// Package pageweaver is the official Go client for the PageWeaver PDF generation API.
package pageweaver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DefaultBaseURL is the production API endpoint.
const DefaultBaseURL = "https://api.pageweaver.io"

var terminalStatuses = map[string]bool{"done": true, "failed": true, "error": true}

// Client is a PageWeaver API client.
type Client struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the API base URL.
func WithBaseURL(u string) Option { return func(c *Client) { c.BaseURL = u } }

// WithHTTPClient overrides the underlying *http.Client.
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.HTTPClient = h } }

// NewClient creates a client with the given API key.
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		APIKey:     apiKey,
		BaseURL:    DefaultBaseURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Error is returned when the API responds with a non-2xx status.
type Error struct {
	StatusCode int
	Body       string
}

func (e *Error) Error() string {
	return fmt.Sprintf("pageweaver: request failed with status %d: %s", e.StatusCode, e.Body)
}

// Document is a generated document as returned by the API.
type Document map[string]any

func (c *Client) do(ctx context.Context, method, path string, body map[string]any) (Document, error) {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.BaseURL+path, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("x-api-key", c.APIKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &Error{StatusCode: resp.StatusCode, Body: string(data)}
	}
	if len(data) == 0 {
		return nil, nil
	}
	var doc Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	return doc, nil
}

// CreateDocument creates a document. The body is sent as-is, so use the API's
// field names, e.g. map[string]any{"templateId": "...", "payload": ...}.
func (c *Client) CreateDocument(ctx context.Context, body map[string]any) (Document, error) {
	return c.do(ctx, http.MethodPost, "/v1/documents", body)
}

// GetDocument fetches a document by id.
func (c *Client) GetDocument(ctx context.Context, id string) (Document, error) {
	return c.do(ctx, http.MethodGet, "/v1/documents/"+id, nil)
}

// CreateAndWait creates a document and polls until it reaches a terminal state.
func (c *Client) CreateAndWait(ctx context.Context, body map[string]any, pollInterval, timeout time.Duration) (Document, error) {
	created, err := c.CreateDocument(ctx, body)
	if err != nil {
		return nil, err
	}
	id, _ := created["id"].(string)
	if id == "" {
		return created, nil
	}
	deadline := time.Now().Add(timeout)
	for {
		doc, err := c.GetDocument(ctx, id)
		if err != nil {
			return nil, err
		}
		if status, _ := doc["status"].(string); terminalStatuses[status] {
			return doc, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("pageweaver: timed out waiting for document %s", id)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}
