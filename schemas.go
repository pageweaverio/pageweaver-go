package pageweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// SchemasService offers read-only discovery of the JSON Schemas your payloads validate against.
type SchemasService struct {
	client *Client
}

// List returns all schemas owned by the key's account, newest-updated first.
func (s *SchemasService) List(ctx context.Context) ([]Document, error) {
	return s.client.doList(ctx, http.MethodGet, "/v1/schemas", nil, nil, nil)
}

// Get returns a schema's published JSON Schema plus a derived sample, for the latest published
// version or a specific version (pass 0 for latest).
func (s *SchemasService) Get(ctx context.Context, id string, version int) (Document, error) {
	q := url.Values{}
	setQueryInt(q, "version", version)
	return s.client.do(ctx, http.MethodGet, "/v1/schemas/"+url.PathEscape(id), q, nil, nil, false)
}

// Versions returns a schema's published version history (newest first).
func (s *SchemasService) Versions(ctx context.Context, id string) ([]Document, error) {
	return s.client.doList(ctx, http.MethodGet, "/v1/schemas/"+url.PathEscape(id)+"/versions", nil, nil, nil)
}

// Version returns one published version's metadata, plus its frozen FieldNode tree when
// include is "nodes".
func (s *SchemasService) Version(ctx context.Context, id string, version int, include string) (Document, error) {
	q := url.Values{}
	setQuery(q, "include", include)
	path := fmt.Sprintf("/v1/schemas/%s/versions/%d", url.PathEscape(id), version)
	return s.client.do(ctx, http.MethodGet, path, q, nil, nil, false)
}
