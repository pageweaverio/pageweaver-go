package pageweaver

import (
	"context"
	"net/http"
	"net/url"
)

// ShareLinksService operates on capability-scoped links that let people without an account view,
// comment on, or approve a document. Requires a review-scoped API key.
type ShareLinksService struct {
	client *Client
}

// Create creates a share link. The response includes the raw url and token exactly once — only the
// hash is stored server-side, so capture it now.
func (s *ShareLinksService) Create(ctx context.Context, body map[string]any) (Document, error) {
	return s.client.do(ctx, http.MethodPost, "/v1/share-links", nil, body, nil, false)
}

// List returns active + disabled links (never the tokens). Filter by document or review.
func (s *ShareLinksService) List(ctx context.Context, documentID, reviewRequestID string) (Document, error) {
	q := url.Values{}
	setQuery(q, "documentId", documentID)
	setQuery(q, "reviewRequestId", reviewRequestID)
	return s.client.do(ctx, http.MethodGet, "/v1/share-links", q, nil, nil, false)
}

// Disable disables a link immediately (the kill switch).
func (s *ShareLinksService) Disable(ctx context.Context, id string) (Document, error) {
	return s.client.do(ctx, http.MethodPost, "/v1/share-links/"+url.PathEscape(id)+"/disable", nil, nil, nil, false)
}
