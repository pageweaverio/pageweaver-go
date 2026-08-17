package pageweaver

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
)

// IntakeService offers first-class document ingestion: bring in a PDF you already have (not a
// template render). Small files use Create directly; larger ones use the resumable chunked
// Sessions upload. Every route requires the documents:upload scope.
type IntakeService struct {
	client *Client

	// Sessions handles resumable chunked (and batch) uploads.
	Sessions *IntakeSessionsService
}

// Create synchronously ingests one PDF (multipart). objectID/objectRole/classification are
// optional ("" to omit). Returns 202.
func (s *IntakeService) Create(ctx context.Context, objectID, objectRole, classification string, file MultipartFile) (Document, error) {
	if file.Filename == "" {
		file.Filename = "document.pdf"
	}
	fields := map[string]string{
		"objectId":       objectID,
		"objectRole":     objectRole,
		"classification": classification,
	}
	files := map[string]MultipartFile{"file": file}
	return s.client.doMultipart(ctx, http.MethodPost, "/v1/documents/intake", fields, files, nil)
}

// IntakeSessionsService offers resumable chunked uploads: start a session, PUT each chunk, then
// finalize. Sessions expire 24h after creation.
type IntakeSessionsService struct {
	client *Client
}

// Create starts a resumable upload session. body should set "filename", "totalBytes", and
// "chunkSize" (capped at 10 MiB by the API). Returns 201.
func (s *IntakeSessionsService) Create(ctx context.Context, body map[string]any) (Document, error) {
	if err := requireObjectBody(body, "body"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodPost, "/v1/documents/intake/sessions", nil, body, nil, false)
}

// CreateBatch starts up to 200 resumable sessions at once (bulk import). body must set "files" to
// a non-empty array. Partial failure is expected: each file's outcome is reported individually.
func (s *IntakeSessionsService) CreateBatch(ctx context.Context, body map[string]any) (Document, error) {
	if err := requireObjectBody(body, "body"); err != nil {
		return nil, err
	}
	files, _ := body["files"].([]any)
	if err := requireNonEmptySlice(files, "body.files"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodPost, "/v1/documents/intake/sessions/batch", nil, body, nil, false)
}

// Get fetches one upload session's state.
func (s *IntakeSessionsService) Get(ctx context.Context, id string) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodGet, "/v1/documents/intake/sessions/"+url.PathEscape(id), nil, nil, nil, false)
}

// Abandon abandons a session and deletes its staged chunks. A "done" session cannot be abandoned.
func (s *IntakeSessionsService) Abandon(ctx context.Context, id string) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodDelete, "/v1/documents/intake/sessions/"+url.PathEscape(id), nil, nil, nil, false)
}

// UploadChunk uploads one 0-based chunk (multipart, "chunk" field). Idempotent: re-sending an
// already-received index is a no-op success.
func (s *IntakeSessionsService) UploadChunk(ctx context.Context, id string, index int, chunk []byte) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	if err := requireNonNegativeInt(index, "index"); err != nil {
		return nil, err
	}
	path := "/v1/documents/intake/sessions/" + url.PathEscape(id) + "/chunks/" + url.PathEscape(strconv.Itoa(index))
	files := map[string]MultipartFile{"chunk": {Data: chunk, Filename: "chunk"}}
	return s.client.doMultipart(ctx, http.MethodPut, path, nil, files, nil)
}

// Finalize finalizes a session once every chunk has arrived. A single-file/PDF session resolves to
// an ordinary intake result; a ZIP session expands into many documents. Returns 202.
func (s *IntakeSessionsService) Finalize(ctx context.Context, id string) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodPost, "/v1/documents/intake/sessions/"+url.PathEscape(id)+"/finalize", nil, nil, nil, false)
}
