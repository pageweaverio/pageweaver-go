package pageweaver

import (
	"context"
	"net/http"
)

// ErrorCodesService offers the public, unauthenticated catalog of every coded API failure: the HTTP
// status each code always answers with, plus a cause/resolution pair. Build typed handling around
// Error.Code against this instead of hardcoding strings, since status codes are shared across many
// failure kinds but Code is unique per cause. Requires no API key (like /openapi.json).
type ErrorCodesService struct {
	client *Client
}

// List returns the full error-code catalog: {"domains": [...], "codes": [...]}.
func (s *ErrorCodesService) List(ctx context.Context) (Document, error) {
	return s.client.do(ctx, http.MethodGet, "/v1/errors", nil, nil, nil, true)
}
