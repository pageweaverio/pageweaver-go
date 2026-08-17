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
	"math/rand"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"time"
)

// DefaultBaseURL is the production API endpoint.
const DefaultBaseURL = "https://api.pageweaver.io"

// Default automatic-retry policy. See WithMaxRetries / WithRetryBaseDelay / WithRetryMaxDelay.
const (
	DefaultMaxRetries     = 2
	DefaultRetryBaseDelay = 300 * time.Millisecond
	DefaultRetryMaxDelay  = 5 * time.Second
)

// Document is a JSON object returned by the API, e.g. a generated document, a template, or a
// comment thread. Fields are accessed by key; missing keys read as their zero value.
type Document map[string]any

// Client is a PageWeaver API client. Construct it with NewClient and reach the API through the
// resource-service fields.
type Client struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client

	// MaxRetries is the maximum retry attempts after the initial try, for transient failures (429,
	// 5xx, network errors) on requests safe to repeat: GET/HEAD/PUT/DELETE always, POST only when an
	// idempotency-key header is present. Default DefaultMaxRetries (2); 0 disables retries entirely.
	MaxRetries int
	// RetryBaseDelay is the base delay before the first retry; it doubles each attempt, capped by
	// RetryMaxDelay, then jittered. Default DefaultRetryBaseDelay.
	RetryBaseDelay time.Duration
	// RetryMaxDelay caps the backoff delay before jitter is applied. Default DefaultRetryMaxDelay.
	RetryMaxDelay time.Duration

	Documents           *DocumentsService
	Templates           *TemplatesService
	Schemas             *SchemasService
	Usage               *UsageService
	Comments            *CommentsService
	Reviews             *ReviewsService
	ShareLinks          *ShareLinksService
	Environments        *EnvironmentsService
	Deployments         *DeploymentsService
	ObjectTypes         *ObjectTypesService
	Objects             *ObjectsService
	RelationshipTypes   *RelationshipTypesService
	Search              *SearchService
	WorkflowDefinitions *WorkflowDefinitionsService
	FormTemplates       *FormTemplatesService
	Intake              *IntakeService
	ErrorCodes          *ErrorCodesService
	Events              *EventsService
}

// Option configures a Client.
type Option func(*Client)

// WithBaseURL overrides the API base URL (e.g. http://localhost:4000 in dev).
func WithBaseURL(u string) Option { return func(c *Client) { c.BaseURL = u } }

// WithHTTPClient overrides the underlying *http.Client.
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.HTTPClient = h } }

// WithMaxRetries overrides the maximum automatic-retry attempts (0 disables retries entirely).
func WithMaxRetries(n int) Option { return func(c *Client) { c.MaxRetries = n } }

// WithRetryBaseDelay overrides the base backoff delay before the first retry.
func WithRetryBaseDelay(d time.Duration) Option { return func(c *Client) { c.RetryBaseDelay = d } }

// WithRetryMaxDelay overrides the backoff delay cap (before jitter).
func WithRetryMaxDelay(d time.Duration) Option { return func(c *Client) { c.RetryMaxDelay = d } }

// NewClient creates a client with the given API key. The base URL defaults to DefaultBaseURL, the
// HTTP client to a 30s-timeout *http.Client, and the retry policy to DefaultMaxRetries attempts;
// all can be overridden with options.
func NewClient(apiKey string, opts ...Option) *Client {
	c := &Client{
		APIKey:         apiKey,
		BaseURL:        DefaultBaseURL,
		HTTPClient:     &http.Client{Timeout: 30 * time.Second},
		MaxRetries:     DefaultMaxRetries,
		RetryBaseDelay: DefaultRetryBaseDelay,
		RetryMaxDelay:  DefaultRetryMaxDelay,
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
	c.ObjectTypes = &ObjectTypesService{client: c}
	c.Objects = &ObjectsService{client: c}
	c.RelationshipTypes = &RelationshipTypesService{client: c}
	c.Search = &SearchService{client: c}
	c.WorkflowDefinitions = &WorkflowDefinitionsService{client: c}
	c.FormTemplates = &FormTemplatesService{client: c}
	c.Intake = &IntakeService{client: c}
	c.Intake.Sessions = &IntakeSessionsService{client: c}
	c.ErrorCodes = &ErrorCodesService{client: c}
	c.Events = &EventsService{client: c}
	return c
}

// APIError is the base API error type, returned when the API responds with a non-2xx status.
// Message and Code are parsed from the JSON body when present; Body is always the raw response
// text.
//
// send/do/doList/doBytes actually return one of the typed subtypes (*ValidationError,
// *AuthenticationError, *PlanRequiredError, *PermissionError, *NotFoundError, *ConflictError,
// *RateLimitError, *ServerError), selected by status code, each of which embeds *APIError — so
// `errors.As(err, &apiErr)` with `var apiErr *pageweaver.APIError` still catches all of them (as
// does `var apiErr *pageweaver.Error`, an alias kept for backward compatibility), or match a
// specific subtype to branch on the failure kind. Every subtype also promotes APIError's fields
// (StatusCode, Message, Code, ...), so e.g. `ve.StatusCode` works directly on a *ValidationError.
type APIError struct {
	StatusCode int
	Message    string
	Code       string
	Body       string
	// FieldErrors carries field-level validation detail from the JSON body's "errors" key, when
	// present (typically on a 400).
	FieldErrors any
	// RetryAfterSeconds is parsed from the Retry-After response header, when present (429/503).
	RetryAfterSeconds *int
	// RequestID is the account-scoped X-Request-Id/X-Correlation-Id the API sent, when present, for
	// support tickets.
	RequestID string
}

// Error is a backward-compatible alias for APIError (the pre-0.2.0 name).
type Error = APIError

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("pageweaver: request failed with status %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("pageweaver: request failed with status %d: %s", e.StatusCode, e.Body)
}

// newAPIError builds an *APIError, extracting a message/code/field-errors from a JSON body when it
// has them.
func newAPIError(status int, raw []byte) *APIError {
	e := &APIError{StatusCode: status, Body: string(raw)}
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
		if errs, ok := parsed["errors"]; ok {
			e.FieldErrors = errs
		}
	}
	return e
}

// safeRetryMethods are always safe to retry: they never double-render or double-charge.
var safeRetryMethods = map[string]bool{
	http.MethodGet:    true,
	http.MethodHead:   true,
	http.MethodPut:    true,
	http.MethodDelete: true,
}

// send performs an HTTP request and returns the raw *http.Response (non-2xx maps to a typed
// *ValidationError/*AuthenticationError/etc). It is the single transport used by every
// higher-level helper. Transient failures (429, 5xx, and network errors) are retried automatically
// with exponential backoff + jitter, honoring Retry-After, restricted to methods safe to repeat:
// GET/HEAD/PUT/DELETE always, POST only when an idempotency-key header is present.
func (c *Client) send(ctx context.Context, method, path string, query url.Values, body map[string]any, headers map[string]string, noAuth bool) (*http.Response, error) {
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyBytes = b
	}
	return c.sendBytes(ctx, method, path, query, bodyBytes, body != nil, headers, noAuth, true)
}

// sendBytes is the shared retry loop underlying send and sendMultipart. jsonBody indicates whether
// bodyBytes should be sent with Content-Type: application/json (false for a pre-built multipart
// body, whose caller sets its own Content-Type header). retryable additionally gates whether this
// call is eligible for retry at all (multipart uploads never are: the body can't be safely replayed).
func (c *Client) sendBytes(ctx context.Context, method, path string, query url.Values, bodyBytes []byte, jsonBody bool, headers map[string]string, noAuth, retryable bool) (*http.Response, error) {
	u := c.BaseURL + path
	if q := query.Encode(); q != "" {
		u += "?" + q
	}

	hasIdempotencyKey := false
	for k := range headers {
		if strings.EqualFold(k, "idempotency-key") {
			hasIdempotencyKey = true
			break
		}
	}
	upperMethod := strings.ToUpper(method)
	retryable = retryable && c.MaxRetries > 0 &&
		(safeRetryMethods[upperMethod] || (upperMethod == http.MethodPost && hasIdempotencyKey))

	for attempt := 0; ; attempt++ {
		var reqBody io.Reader
		if bodyBytes != nil {
			reqBody = bytes.NewReader(bodyBytes)
		}
		req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Accept", "application/json")
		if !noAuth {
			req.Header.Set("x-api-key", c.APIKey)
		}
		if bodyBytes != nil && jsonBody {
			req.Header.Set("Content-Type", "application/json")
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			if retryable && attempt < c.MaxRetries {
				if !c.sleepBeforeRetry(ctx, attempt, nil) {
					return nil, ctx.Err()
				}
				continue
			}
			return nil, err
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			data, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
			if retryable && attempt < c.MaxRetries && retriableStatus(resp.StatusCode) {
				if !c.sleepBeforeRetry(ctx, attempt, retryAfter) {
					return nil, ctx.Err()
				}
				continue
			}
			apiErr := newAPIError(resp.StatusCode, data)
			apiErr.RetryAfterSeconds = retryAfter
			apiErr.RequestID = resp.Header.Get("X-Request-Id")
			if apiErr.RequestID == "" {
				apiErr.RequestID = resp.Header.Get("X-Correlation-Id")
			}
			return nil, apiErrorForStatus(apiErr)
		}
		return resp, nil
	}
}

// retriableStatus reports whether a response status is worth retrying: 429 or a 5xx.
func retriableStatus(status int) bool {
	switch status {
	case 429, 500, 502, 503, 504:
		return true
	default:
		return false
	}
}

// sleepBeforeRetry waits the backed-off + jittered delay for this attempt (or the server-requested
// Retry-After, when given), returning false if ctx is done first.
func (c *Client) sleepBeforeRetry(ctx context.Context, attempt int, retryAfterSeconds *int) bool {
	var delay time.Duration
	if retryAfterSeconds != nil {
		delay = time.Duration(*retryAfterSeconds) * time.Second
	} else {
		backoff := c.RetryBaseDelay * time.Duration(1<<uint(attempt))
		if backoff > c.RetryMaxDelay {
			backoff = c.RetryMaxDelay
		}
		jittered := backoff/2 + time.Duration(rand.Float64()*float64(backoff/2))
		delay = jittered
	}
	if delay <= 0 {
		return true
	}
	select {
	case <-ctx.Done():
		return false
	case <-time.After(delay):
		return true
	}
}

// parseRetryAfter parses a Retry-After header (seconds, or an HTTP-date) into whole seconds.
func parseRetryAfter(value string) *int {
	if value == "" {
		return nil
	}
	if secs, err := parseInt(value); err == nil {
		if secs < 0 {
			secs = 0
		}
		return &secs
	}
	if when, err := http.ParseTime(value); err == nil {
		secs := int(time.Until(when).Seconds())
		if secs < 0 {
			secs = 0
		}
		return &secs
	}
	return nil
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
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

// MultipartFile is one file part of a multipart/form-data upload (see doMultipart).
type MultipartFile struct {
	Data        []byte
	Filename    string
	ContentType string // optional; defaults to application/octet-stream
}

// quoteEscaper mirrors the escaping mime/multipart uses internally for Content-Disposition values.
var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

// doMultipart performs a multipart/form-data request and decodes a JSON object response into a
// Document. Used by the file-upload endpoints (form templates, intake). Never retried: a file body
// can't be safely replayed.
func (c *Client) doMultipart(ctx context.Context, method, path string, fields map[string]string, files map[string]MultipartFile, headers map[string]string) (Document, error) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	for k, v := range fields {
		if v == "" {
			continue
		}
		if err := w.WriteField(k, v); err != nil {
			return nil, err
		}
	}
	for field, f := range files {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			quoteEscaper.Replace(field), quoteEscaper.Replace(f.Filename)))
		ct := f.ContentType
		if ct == "" {
			ct = "application/octet-stream"
		}
		h.Set("Content-Type", ct)
		fw, err := w.CreatePart(h)
		if err != nil {
			return nil, err
		}
		if _, err := fw.Write(f.Data); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}

	allHeaders := map[string]string{"Content-Type": w.FormDataContentType()}
	for k, v := range headers {
		allHeaders[k] = v
	}

	resp, err := c.sendBytes(ctx, method, path, nil, buf.Bytes(), false, allHeaders, false, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return decodeDocument(resp)
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
		return nil, apiErrorForStatus(newAPIError(resp.StatusCode, data))
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

// setQueryBool adds "true" to v when value is true; omitted (never "false") when false, since these
// query flags all default to false server-side.
func setQueryBool(v url.Values, key string, value bool) {
	if value {
		v.Set(key, "true")
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
