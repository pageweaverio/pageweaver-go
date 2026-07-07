package pageweaver

import (
	"context"
	"net/http"
	"net/url"
)

// ReviewsService operates on review requests on documents: create, list, add participants, and
// collect approvals against a completion policy. Writes require a review-scoped API key.
type ReviewsService struct {
	client *Client
}

// Create opens a review on a document with an optional policy + participants. Returns 201.
func (s *ReviewsService) Create(ctx context.Context, body map[string]any) (Document, error) {
	return s.client.do(ctx, http.MethodPost, "/v1/reviews", nil, body, nil, false)
}

// List returns reviews, newest first. Filter by status/documentID; page with cursor.
func (s *ReviewsService) List(ctx context.Context, status, documentID, cursor string, limit int) (Document, error) {
	q := url.Values{}
	setQuery(q, "status", status)
	setQuery(q, "documentId", documentID)
	setQuery(q, "cursor", cursor)
	setQueryInt(q, "limit", limit)
	return s.client.do(ctx, http.MethodGet, "/v1/reviews", q, nil, nil, false)
}

// Get fetches one review with its participants, approvals, and computed policy state.
func (s *ReviewsService) Get(ctx context.Context, id string) (Document, error) {
	return s.client.do(ctx, http.MethodGet, "/v1/reviews/"+url.PathEscape(id), nil, nil, nil, false)
}

// AddParticipant adds a participant (member userId, or externalEmail + externalName) with a role.
func (s *ReviewsService) AddParticipant(ctx context.Context, id string, body map[string]any) (Document, error) {
	return s.client.do(ctx, http.MethodPost, "/v1/reviews/"+url.PathEscape(id)+"/participants", nil, body, nil, false)
}

// Approve records an approval decision (201); the review auto-completes when its policy is satisfied.
func (s *ReviewsService) Approve(ctx context.Context, id string, body map[string]any) (Document, error) {
	return s.client.do(ctx, http.MethodPost, "/v1/reviews/"+url.PathEscape(id)+"/approvals", nil, body, nil, false)
}

// Complete manually completes a review (policy-satisfied, or forced by an admin).
func (s *ReviewsService) Complete(ctx context.Context, id string) (Document, error) {
	return s.client.do(ctx, http.MethodPost, "/v1/reviews/"+url.PathEscape(id)+"/complete", nil, nil, nil, false)
}

// Cancel withdraws a review (open -> canceled).
func (s *ReviewsService) Cancel(ctx context.Context, id string) (Document, error) {
	return s.client.do(ctx, http.MethodPost, "/v1/reviews/"+url.PathEscape(id)+"/cancel", nil, nil, nil, false)
}
