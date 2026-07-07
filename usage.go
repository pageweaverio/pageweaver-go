package pageweaver

import (
	"context"
	"net/http"
)

// UsageService reports page consumption against the plan quota for the current billing period.
type UsageService struct {
	client *Client
}

// Get returns current-period usage: billable document pages and editor preview pages, with limits.
func (s *UsageService) Get(ctx context.Context) (Document, error) {
	return s.client.do(ctx, http.MethodGet, "/v1/usage", nil, nil, nil, false)
}
