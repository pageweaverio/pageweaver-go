package pageweaver

import (
	"context"
	"net/http"
	"net/url"
)

// CommentsService operates on anchored comment threads on rendered documents: create, list, reply,
// and lifecycle (resolve / reopen / close). Writes require a review-scoped API key.
type CommentsService struct {
	client *Client
}

// CommentListParams filters a document's comment-thread listing.
type CommentListParams struct {
	PageNumber int
	Status     string
	Severity   string
	Cursor     string
	Limit      int
}

// Create creates an anchored thread (point/area/text/page) with its first message. Returns 201.
func (s *CommentsService) Create(ctx context.Context, body map[string]any) (Document, error) {
	return s.client.do(ctx, http.MethodPost, "/v1/comments", nil, body, nil, false)
}

// List returns a document's threads, newest first. Filter by page/status/severity; page with cursor.
func (s *CommentsService) List(ctx context.Context, documentID string, params CommentListParams) (Document, error) {
	q := url.Values{}
	setQueryInt(q, "pageNumber", params.PageNumber)
	setQuery(q, "status", params.Status)
	setQuery(q, "severity", params.Severity)
	setQuery(q, "cursor", params.Cursor)
	setQueryInt(q, "limit", params.Limit)
	return s.client.do(ctx, http.MethodGet, "/v1/documents/"+url.PathEscape(documentID)+"/comments", q, nil, nil, false)
}

// Get fetches one thread with its full message list.
func (s *CommentsService) Get(ctx context.Context, id string) (Document, error) {
	return s.client.do(ctx, http.MethodGet, "/v1/comments/"+url.PathEscape(id), nil, nil, nil, false)
}

// Update edits severity, assignment, due date, or relocates the anchor coordinates.
func (s *CommentsService) Update(ctx context.Context, id string, body map[string]any) (Document, error) {
	return s.client.do(ctx, http.MethodPatch, "/v1/comments/"+url.PathEscape(id), nil, body, nil, false)
}

// Reply posts a reply on a thread. body may set "parentMessageId" to nest this reply under an
// existing message in the thread (omit for a top-level reply; the message must belong to this
// thread). The returned CommentMessage carries "parentMessageId" back (null at the top level).
// Returns 201.
func (s *CommentsService) Reply(ctx context.Context, id string, body map[string]any) (Document, error) {
	return s.client.do(ctx, http.MethodPost, "/v1/comments/"+url.PathEscape(id)+"/messages", nil, body, nil, false)
}

// Resolve resolves a thread (open -> resolved).
func (s *CommentsService) Resolve(ctx context.Context, id string) (Document, error) {
	return s.client.do(ctx, http.MethodPost, "/v1/comments/"+url.PathEscape(id)+"/resolve", nil, nil, nil, false)
}

// Reopen reopens a resolved thread (resolved -> open).
func (s *CommentsService) Reopen(ctx context.Context, id string) (Document, error) {
	return s.client.do(ctx, http.MethodPost, "/v1/comments/"+url.PathEscape(id)+"/reopen", nil, nil, nil, false)
}

// Close closes a thread permanently (-> closed, final).
func (s *CommentsService) Close(ctx context.Context, id string) (Document, error) {
	return s.client.do(ctx, http.MethodPost, "/v1/comments/"+url.PathEscape(id)+"/close", nil, nil, nil, false)
}
