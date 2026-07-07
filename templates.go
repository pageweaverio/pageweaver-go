package pageweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// TemplatesService offers read-only discovery of published templates and their pinnable versions.
// Template change proposals are reached through the nested Proposals service.
type TemplatesService struct {
	client *Client

	// Proposals — the PR analog for template changes (requires a deploy-scoped key).
	Proposals *ProposalsService
}

// List returns all templates owned by the key's account, newest-updated first.
func (s *TemplatesService) List(ctx context.Context) ([]Document, error) {
	return s.client.doList(ctx, http.MethodGet, "/v1/templates", nil, nil, nil)
}

// Get returns one template's metadata (name, current version, associated schema, authoring mode).
func (s *TemplatesService) Get(ctx context.Context, id string) (Document, error) {
	return s.client.do(ctx, http.MethodGet, "/v1/templates/"+url.PathEscape(id), nil, nil, nil, false)
}

// Versions returns a template's published version history (newest first).
func (s *TemplatesService) Versions(ctx context.Context, id string) ([]Document, error) {
	return s.client.doList(ctx, http.MethodGet, "/v1/templates/"+url.PathEscape(id)+"/versions", nil, nil, nil)
}

// Version returns one published version's metadata, plus its frozen editor source when
// include is "source".
func (s *TemplatesService) Version(ctx context.Context, id string, version int, include string) (Document, error) {
	q := url.Values{}
	setQuery(q, "include", include)
	path := fmt.Sprintf("/v1/templates/%s/versions/%d", url.PathEscape(id), version)
	return s.client.do(ctx, http.MethodGet, path, q, nil, nil, false)
}
