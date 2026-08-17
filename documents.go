package pageweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// terminalStatuses are the statuses at which a document stops changing.
var terminalStatuses = map[string]bool{"done": true, "failed": true}

// DocumentsService operates on documents: the core of the API.
type DocumentsService struct {
	client *Client
}

// ListParams filters a document history listing.
type ListParams struct {
	Status     string
	TemplateID string
	Cursor     string
	Limit      int
}

// WaitOptions controls polling in WaitFor / CreateAndWait.
type WaitOptions struct {
	// Interval is the initial delay between polls. Default 1s.
	Interval time.Duration
	// MaxInterval caps the backing-off poll delay. Default 5s.
	MaxInterval time.Duration
	// Backoff multiplies the delay after each poll. Default 1.5.
	Backoff float64
	// Timeout gives up after this long. Default 60s.
	Timeout time.Duration
	// ThrowOnFailure returns an error if the document ends in "failed". Default true.
	ThrowOnFailure *bool
}

func (o WaitOptions) resolved() (interval, maxInterval, timeout time.Duration, backoff float64, throwOnFailure bool) {
	interval = o.Interval
	if interval <= 0 {
		interval = time.Second
	}
	maxInterval = o.MaxInterval
	if maxInterval <= 0 {
		maxInterval = 5 * time.Second
	}
	timeout = o.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	backoff = o.Backoff
	if backoff <= 0 {
		backoff = 1.5
	}
	throwOnFailure = true
	if o.ThrowOnFailure != nil {
		throwOnFailure = *o.ThrowOnFailure
	}
	return
}

// SyncResult is the content-negotiated result of CreateSync. Inspect Kind to pick the branch:
// "pdf" (raw bytes), "pending" (a 202 fallback whose ID you then poll), or "document" (a finished
// document as JSON).
type SyncResult struct {
	Kind     string // "pdf" | "pending" | "document"
	ID       string
	Version  *int
	Status   string
	PDF      []byte
	Document Document
}

// Create makes a document from a template (with a validated payload) or from inline HTML. Pass an
// idempotencyKey (or "") to safely retry. body may set "name" (a human-readable label) and
// "publicAlias" (true to publish a public /d/:alias link — requires the publicAlias plan capability;
// the minted link comes back as the result's alias.token). body["localization"]["direction"] sets
// the base text direction: "auto" (default) follows the locale, so an Arabic or Hebrew locale
// produces a right-to-left document with nothing else set; "ltr"/"rtl" override that derivation.
// Returns 202 immediately with the id and status.
func (s *DocumentsService) Create(ctx context.Context, body map[string]any, idempotencyKey string) (Document, error) {
	if err := requireObjectBody(body, "body"); err != nil {
		return nil, err
	}
	var headers map[string]string
	if idempotencyKey != "" {
		headers = map[string]string{"idempotency-key": idempotencyKey}
	}
	return s.client.do(ctx, http.MethodPost, "/v1/documents", nil, body, headers, false)
}

// Get fetches the current state of a document. When status is "done" it carries a download block.
func (s *DocumentsService) Get(ctx context.Context, id string) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodGet, "/v1/documents/"+url.PathEscape(id), nil, nil, nil, false)
}

// Verify fetches a document's integrity proof (content hash, hash-chain position, chainVerified).
func (s *DocumentsService) Verify(ctx context.Context, id string) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodGet, "/v1/documents/"+url.PathEscape(id)+"/verify", nil, nil, nil, false)
}

// Trust fetches a deterministic, agent-facing trust manifest for one document: the schema/template
// pin, the compiled artifact hash, the content hash + hash-chain position, and the signature status,
// in a single shape meant to be read (or diffed) programmatically rather than assembled from Get and
// Verify.
func (s *DocumentsService) Trust(ctx context.Context, id string) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodGet, "/v1/documents/"+url.PathEscape(id)+"/trust", nil, nil, nil, false)
}

// Diff computes a causal diff between this document and against: whether the payload, the pinned
// template version, or the per-render options changed, plus a page-count delta and whether the two
// content hashes are identical. Never renders or meters. Returns a "not_comparable_*" classification
// (not an error) when the two documents don't share a common template lineage.
func (s *DocumentsService) Diff(ctx context.Context, id, against string) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	if _, err := requireID(against, "against"); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("against", against)
	return s.client.do(ctx, http.MethodGet, "/v1/documents/"+url.PathEscape(id)+"/diff", q, nil, nil, false)
}

// AppendVersion issues a new version under this document's lineage (append, not replace): validates
// payload against the pinned template/schema, renders it as a new document, and fires a
// document.superseded webhook on the prior head. Only valid for a template-pinned document (an
// inline or url render has no lineage to append to — use Regenerate instead). Requires a plan with
// document versioning. Returns 202.
func (s *DocumentsService) AppendVersion(ctx context.Context, id string, body map[string]any) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	if err := requireObjectBody(body, "body"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodPost, "/v1/documents/"+url.PathEscape(id)+"/versions", nil, body, nil, false)
}

// Versions lists this document's own version sequence (newest first): the lineage AppendVersion
// builds.
func (s *DocumentsService) Versions(ctx context.Context, id string) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodGet, "/v1/documents/"+url.PathEscape(id)+"/versions", nil, nil, nil, false)
}

// Version fetches one immutable pinned version (by 1-based seq) from this document's lineage.
func (s *DocumentsService) Version(ctx context.Context, id string, seq int) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	if err := requirePositiveInt(seq, "seq"); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/v1/documents/%s/versions/%d", url.PathEscape(id), seq)
	return s.client.do(ctx, http.MethodGet, path, nil, nil, nil, false)
}

// Representations lists every artifact (PDF, e-invoice XML sidecar, JSON data twin, ...) produced
// for one version of this document. Defaults to the version id names; pass a nonzero version to
// inspect an older one.
func (s *DocumentsService) Representations(ctx context.Context, id string, version int) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	q := url.Values{}
	setQueryInt(q, "version", version)
	return s.client.do(ctx, http.MethodGet, "/v1/documents/"+url.PathEscape(id)+"/representations", q, nil, nil, false)
}

// Accessibility fetches a document's PDF/UA-1 conformance report: the validator's verdict, every
// failed rule with its ISO 14289-1 clause, and what the pipeline adjusted to make it conformant.
func (s *DocumentsService) Accessibility(ctx context.Context, id string) (Document, error) {
	return s.client.do(ctx, http.MethodGet, "/v1/documents/"+url.PathEscape(id)+"/accessibility", nil, nil, nil, false)
}

// List returns one page of the document history, newest first. Use the response nextCursor to page.
func (s *DocumentsService) List(ctx context.Context, params ListParams) (Document, error) {
	q := url.Values{}
	setQuery(q, "status", params.Status)
	setQuery(q, "templateId", params.TemplateID)
	setQuery(q, "cursor", params.Cursor)
	setQueryInt(q, "limit", params.Limit)
	return s.client.do(ctx, http.MethodGet, "/v1/documents", q, nil, nil, false)
}

// ListAll follows nextCursor across every page and returns all items as a flat []Document.
func (s *DocumentsService) ListAll(ctx context.Context, params ListParams) ([]Document, error) {
	var all []Document
	cursor := params.Cursor
	for {
		p := params
		p.Cursor = cursor
		page, err := s.List(ctx, p)
		if err != nil {
			return nil, err
		}
		if items, ok := page["items"].([]any); ok {
			for _, it := range items {
				if m, ok := it.(map[string]any); ok {
					all = append(all, Document(m))
				}
			}
		}
		next, _ := page["nextCursor"].(string)
		if next == "" {
			return all, nil
		}
		cursor = next
	}
}

// Regenerate faithfully replays a prior document (same version/source, payload, and options),
// returning a new document id (202). It counts as a new render.
func (s *DocumentsService) Regenerate(ctx context.Context, id string) (Document, error) {
	return s.client.do(ctx, http.MethodPost, "/v1/documents/"+url.PathEscape(id)+"/regenerate", nil, nil, nil, false)
}

// WaitFor polls a document until it reaches a terminal state (or the timeout elapses), with a
// backing-off delay. On timeout it returns an error; on "failed" it returns an error unless
// ThrowOnFailure is set false.
func (s *DocumentsService) WaitFor(ctx context.Context, id string, opts WaitOptions) (Document, error) {
	interval, maxInterval, timeout, backoff, throwOnFailure := opts.resolved()
	deadline := time.Now().Add(timeout)

	delay := interval
	last, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	for {
		status, _ := last["status"].(string)
		if terminalStatuses[status] {
			if status == "failed" && throwOnFailure {
				return last, fmt.Errorf("pageweaver: document %s failed: %s", id, docError(last))
			}
			return last, nil
		}
		if time.Now().After(deadline) {
			return last, fmt.Errorf("pageweaver: timed out after %s waiting for document %s (last status: %s)", timeout, id, status)
		}
		remaining := time.Until(deadline)
		wait := delay
		if wait > remaining {
			wait = remaining
		}
		select {
		case <-ctx.Done():
			return last, ctx.Err()
		case <-time.After(wait):
		}
		delay = time.Duration(float64(delay) * backoff)
		if delay > maxInterval {
			delay = maxInterval
		}
		last, err = s.Get(ctx, id)
		if err != nil {
			return nil, err
		}
	}
}

// CreateAndWait creates a document then WaitFor-s it, resolving with the finished document.
func (s *DocumentsService) CreateAndWait(ctx context.Context, body map[string]any, opts WaitOptions) (Document, error) {
	created, err := s.Create(ctx, body, "")
	if err != nil {
		return nil, err
	}
	id, _ := created["id"].(string)
	if id == "" {
		return created, nil
	}
	return s.WaitFor(ctx, id, opts)
}

// CreateSync creates a document synchronously by sending Prefer: wait so the server holds the
// response open until the render finishes. When pdf is true it also sends Accept: application/pdf
// to receive raw bytes for an unprotected document. Inspect the SyncResult.Kind to pick the branch.
func (s *DocumentsService) CreateSync(ctx context.Context, body map[string]any, pdf bool) (*SyncResult, error) {
	headers := map[string]string{"prefer": "wait"}
	if pdf {
		headers["accept"] = "application/pdf"
	}
	resp, err := s.client.send(ctx, http.MethodPost, "/v1/documents", nil, body, headers, false)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if isPDFContentType(resp.Header.Get("Content-Type")) {
		data, err := readAll(resp)
		if err != nil {
			return nil, err
		}
		return &SyncResult{
			Kind:    "pdf",
			ID:      resp.Header.Get("x-document-id"),
			Version: numberOrNil(resp.Header.Get("x-document-version")),
			PDF:     data,
		}, nil
	}

	doc, err := decodeDocument(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusAccepted {
		id, _ := doc["id"].(string)
		status, _ := doc["status"].(string)
		return &SyncResult{Kind: "pending", ID: id, Version: docVersion(doc), Status: status}, nil
	}
	return &SyncResult{Kind: "document", Document: doc}, nil
}

// Pages returns a document's per-page geometry plus whether extracted text and a thumbnail exist.
func (s *DocumentsService) Pages(ctx context.Context, id string) ([]Document, error) {
	return s.client.doList(ctx, http.MethodGet, "/v1/documents/"+url.PathEscape(id)+"/pages", nil, nil, nil)
}

// MigrateComments carries open comment threads forward from a previous same-template document onto
// this one. Returns 202; observe progress via CommentMigration.
func (s *DocumentsService) MigrateComments(ctx context.Context, id string, body map[string]any) (Document, error) {
	return s.client.do(ctx, http.MethodPost, "/v1/documents/"+url.PathEscape(id)+"/migrate-comments", nil, body, nil, false)
}

// CommentMigration returns the comment-migration rollup for a document, grouped by status.
func (s *DocumentsService) CommentMigration(ctx context.Context, id string) (Document, error) {
	return s.client.do(ctx, http.MethodGet, "/v1/documents/"+url.PathEscape(id)+"/comment-migration", nil, nil, nil, false)
}

// Download returns the finished PDF bytes. If password is non-empty, it streams from the
// password-gated content endpoint (no API key). Otherwise it resolves the document's short-lived
// signed URL and fetches it; a not-done or download-protected document returns an error.
func (s *DocumentsService) Download(ctx context.Context, id, password string) ([]byte, error) {
	if password != "" {
		headers := map[string]string{"x-document-password": password}
		return s.client.doBytes(ctx, http.MethodGet, "/v1/documents/"+url.PathEscape(id)+"/content", nil, headers, true)
	}
	doc, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	status, _ := doc["status"].(string)
	download, _ := doc["download"].(map[string]any)
	dlURL, _ := download["url"].(string)
	if status != "done" || dlURL == "" {
		return nil, fmt.Errorf("pageweaver: document %s is not downloadable (status: %s)", id, status)
	}
	if protected, _ := download["protected"].(bool); protected {
		return nil, fmt.Errorf("pageweaver: document %s is download-protected; supply a password", id)
	}
	return s.client.fetchURLBytes(ctx, dlURL)
}

// docError extracts the "error" string from a failed document, or a fallback.
func docError(doc Document) string {
	if s, ok := doc["error"].(string); ok && s != "" {
		return s
	}
	return "unknown error"
}

// docVersion reads the "version" field of a document as an *int.
func docVersion(doc Document) *int {
	if v, ok := doc["version"].(float64); ok {
		n := int(v)
		return &n
	}
	return nil
}
