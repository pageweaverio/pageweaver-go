// Package pageweaver is the official Go client for the PageWeaver PDF generation API.
//
// A Client exposes the API through resource services hung off exported fields, e.g.
//
//	pw := pageweaver.NewClient("pk_live_...")
//	doc, err := pw.Documents.Create(ctx, map[string]any{"templateId": "tmpl_invoice", "payload": p}, "")
//
// Object responses are returned as Document (a map[string]any) and arrays as []Document, so the
// SDK stays dependency-free and forward-compatible with new response fields.
package pageweaver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is the production API endpoint.
const DefaultBaseURL = "https://api.pageweaver.io"

// Document is a JSON object returned by the API, e.g. a generated document, a template, or a
// comment thread. Fields are accessed by key; missing keys read as their zero value.
type Document map[string]any

// Client is a PageWeaver API client. Construct it with NewClient and reach the API through the
// resource-service fields.
type Client struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client

	Documents    *DocumentsService
	Templates    *TemplatesService
	Schemas      *SchemasService
	Usage        *UsageService
	Comments     *CommentsService
	Reviews      *ReviewsService
	ShareLinks   *ShareLinksService
	Environments *EnvironmentsService
	Deployments  *DeploymentsService
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the API base URL (e.g. http://localhost:4000 in dev).
func WithBaseURL(u string) Option { return func(c *Client) { c.BaseURL = u } }

// WithHTTPClient overrides the underlying *http.Client.
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.HTTPClient = h } }

// NewClient creates a client with the given API key. The base URL defaults to DefaultBaseURL and
// the HTTP client to a 30s-timeout *http.Client; both can be overridden with options.
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		APIKey:     apiKey,
		BaseURL:    DefaultBaseURL,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
	for _, opt := range opts {
		opt(c)
	}
	c.BaseURL = strings.TrimRight(c.BaseURL, "/")

	c.Documents = &DocumentsService{client: c}
	c.Templates = &TemplatesService{client: c}
	c.Templates.Proposals = &ProposalsService{client: c}
	c.Schemas = &SchemasService{client: c}
	c.Usage = &UsageService{client: c}
	c.Comments = &CommentsService{client: c}
	c.Reviews = &ReviewsService{client: c}
	c.ShareLinks = &ShareLinksService{client: c}
	c.Environments = &EnvironmentsService{client: c}
	c.Deployments = &DeploymentsService{client: c}
	return c
}

// Error is returned when the API responds with a non-2xx status. Message and Code are parsed from
// the JSON body when present; Body is always the raw response text.
type Error struct {
	StatusCode int
	Message    string
	Code       string
	Body       string
}

func (e *Error) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("pageweaver: request failed with status %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("pageweaver: request failed with status %d: %s", e.StatusCode, e.Body)
}

// newAPIError builds an *Error, extracting a message/code from a JSON body when it has them.
func newAPIError(status int, raw []byte) *Error {
	e := &Error{StatusCode: status, Body: string(raw)}
	var parsed map[string]any
	if len(raw) > 0 && json.Unmarshal(raw, &parsed) == nil {
		switch m := parsed["message"].(type) {
		case string:
			e.Message = m
		case []any:
			parts := make([]string, 0, len(m))
			for _, p := range m {
				if s, ok := p.(string); ok {
					parts = append(parts, s)
				}
			}
			e.Message = strings.Join(parts, ", ")
		}
		if code, ok := parsed["code"].(string); ok {
			e.Code = code
		}
	}
	return e
}

// send performs an HTTP request and returns the raw *http.Response (non-2xx maps to *Error). It is
// the single transport used by every higher-level helper.
func (c *Client) send(ctx context.Context, method, path string, query url.Values, body map[string]any, headers map[string]string, noAuth bool) (*http.Response, error) {
	u := c.BaseURL + path
	if q := query.Encode(); q != "" {
		u += "?" + q
	}

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if !noAuth {
		req.Header.Set("x-api-key", c.APIKey)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, newAPIError(resp.StatusCode, data)
	}
	return resp, nil
}

// do performs a request and decodes a JSON object response into a Document. An empty body (e.g. a
// 204) returns a nil Document and nil error.
func (c *Client) do(ctx context.Context, method, path string, query url.Values, body map[string]any, headers map[string]string, noAuth bool) (Document, error) {
	resp, err := c.send(ctx, method, path, query, body, headers, noAuth)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
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

// doList performs a request and decodes a JSON array response into a []Document.
func (c *Client) doList(ctx context.Context, method, path string, query url.Values, body map[string]any, headers map[string]string) ([]Document, error) {
	resp, err := c.send(ctx, method, path, query, body, headers, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}
	var list []Document
	if err := json.Unmarshal(data, &list); err != nil {
		return nil, err
	}
	return list, nil
}

// doBytes performs a request and returns the raw response body bytes (for PDF downloads).
func (c *Client) doBytes(ctx context.Context, method, path string, query url.Values, headers map[string]string, noAuth bool) ([]byte, error) {
	resp, err := c.send(ctx, method, path, query, nil, headers, noAuth)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

// fetchURLBytes GETs an absolute URL (e.g. a signed download URL) with no API key and returns its
// bytes.
func (c *Client) fetchURLBytes(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
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
		return nil, newAPIError(resp.StatusCode, data)
	}
	return data, nil
}

// setQuery adds a non-empty string value to v.
func setQuery(v url.Values, key, value string) {
	if value != "" {
		v.Set(key, value)
	}
}

// setQueryInt adds a non-zero int value to v.
func setQueryInt(v url.Values, key string, value int) {
	if value != 0 {
		v.Set(key, fmt.Sprintf("%d", value))
	}
}

// readAll reads and closes a response body.
func readAll(resp *http.Response) ([]byte, error) {
	return io.ReadAll(resp.Body)
}

// decodeDocument reads a response body and unmarshals it into a Document.
func decodeDocument(resp *http.Response) (Document, error) {
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
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

// isPDFContentType reports whether a Content-Type header names a PDF body.
func isPDFContentType(ct string) bool {
	return strings.Contains(strings.ToLower(ct), "application/pdf")
}

// numberOrNil parses a numeric response-header value into an *int, or nil when absent/non-numeric.
func numberOrNil(s string) *int {
	if s == "" {
		return nil
	}
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return nil
	}
	return &n
}
