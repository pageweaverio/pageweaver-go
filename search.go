package pageweaver

import (
	"context"
	"net/http"
	"net/url"
)

// SearchService offers full-text search across objects and documents, permission-trimmed: a hit
// the caller may not view is silently dropped, never surfaced as a 403 (avoids confirming a hidden
// record exists). Requires the search:read scope; object hits are additionally gated by
// objects:read.
type SearchService struct {
	client *Client
}

// SearchParams filters a search query. Q is required and uses websearch syntax: quote a phrase,
// -exclude, OR.
type SearchParams struct {
	Q              string
	SubjectType    string // "object" | "document"
	ObjectTypeKey  string
	Classification string
	OwnerUserID    string
	UpdatedAfter   string
	UpdatedBefore  string
	Cursor         string
	Limit          int
}

// Query runs a search query.
func (s *SearchService) Query(ctx context.Context, params SearchParams) (Document, error) {
	if err := requireString(params.Q, "params.Q"); err != nil {
		return nil, err
	}
	q := url.Values{}
	q.Set("q", params.Q)
	setQuery(q, "subjectType", params.SubjectType)
	setQuery(q, "objectTypeKey", params.ObjectTypeKey)
	setQuery(q, "classification", params.Classification)
	setQuery(q, "ownerUserId", params.OwnerUserID)
	setQuery(q, "updatedAfter", params.UpdatedAfter)
	setQuery(q, "updatedBefore", params.UpdatedBefore)
	setQuery(q, "cursor", params.Cursor)
	setQueryInt(q, "limit", params.Limit)
	return s.client.do(ctx, http.MethodGet, "/v1/search", q, nil, nil, false)
}
