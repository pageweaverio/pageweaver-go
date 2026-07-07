package pageweaver

import (
	"context"
	"net/http"
	"net/url"
)

// EnvironmentsService operates on environments & pins (Pillar 2): a named per-account pointer set
// over immutable template versions. Writes require a deploy-scoped API key.
type EnvironmentsService struct {
	client *Client
}

// List returns every environment for the account, with pin counts.
func (s *EnvironmentsService) List(ctx context.Context) ([]Document, error) {
	return s.client.doList(ctx, http.MethodGet, "/v1/environments", nil, nil, nil)
}

// Create creates a named pointer set (e.g. staging / production). Returns 201.
func (s *EnvironmentsService) Create(ctx context.Context, body map[string]any) (Document, error) {
	return s.client.do(ctx, http.MethodPost, "/v1/environments", nil, body, nil, false)
}

// Get fetches one environment by slug.
func (s *EnvironmentsService) Get(ctx context.Context, slug string) (Document, error) {
	return s.client.do(ctx, http.MethodGet, "/v1/environments/"+url.PathEscape(slug), nil, nil, nil, false)
}

// Update renames an environment or flips its production flag. The slug is immutable.
func (s *EnvironmentsService) Update(ctx context.Context, slug string, body map[string]any) (Document, error) {
	return s.client.do(ctx, http.MethodPatch, "/v1/environments/"+url.PathEscape(slug), nil, body, nil, false)
}

// Delete deletes an environment and its pins (audited).
func (s *EnvironmentsService) Delete(ctx context.Context, slug string) (Document, error) {
	return s.client.do(ctx, http.MethodDelete, "/v1/environments/"+url.PathEscape(slug), nil, nil, nil, false)
}

// Pins returns the template -> version pointers in an environment.
func (s *EnvironmentsService) Pins(ctx context.Context, slug string) ([]Document, error) {
	return s.client.doList(ctx, http.MethodGet, "/v1/environments/"+url.PathEscape(slug)+"/pins", nil, nil, nil)
}

// SetPin points a template at one of its published versions (creates or moves the pin).
func (s *EnvironmentsService) SetPin(ctx context.Context, slug, templateID string, version int) (Document, error) {
	path := "/v1/environments/" + url.PathEscape(slug) + "/pins/" + url.PathEscape(templateID)
	return s.client.do(ctx, http.MethodPut, path, nil, map[string]any{"version": version}, nil, false)
}

// RemovePin unpins a template from an environment.
func (s *EnvironmentsService) RemovePin(ctx context.Context, slug, templateID string) (Document, error) {
	path := "/v1/environments/" + url.PathEscape(slug) + "/pins/" + url.PathEscape(templateID)
	return s.client.do(ctx, http.MethodDelete, path, nil, nil, nil, false)
}

// Promote copies another environment's pin set onto this one (e.g. staging -> production).
func (s *EnvironmentsService) Promote(ctx context.Context, slug string, body map[string]any) (Document, error) {
	return s.client.do(ctx, http.MethodPost, "/v1/environments/"+url.PathEscape(slug)+"/promote", nil, body, nil, false)
}

// Rollback rolls an environment back to a prior deployment's pin set (a new pin-only deployment).
// Pass a nil body to roll back to the last successful deployment before the current head.
func (s *EnvironmentsService) Rollback(ctx context.Context, slug string, body map[string]any) (Document, error) {
	return s.client.do(ctx, http.MethodPost, "/v1/environments/"+url.PathEscape(slug)+"/rollback", nil, body, nil, false)
}
