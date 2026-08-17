package pageweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// ObjectTypesService operates on typed business-record definitions: a draft + immutable-published-
// version model, mirroring template versioning. Reads need the objects:read scope; writes need
// object-types:manage. See ObjectsService for the records themselves.
type ObjectTypesService struct {
	client *Client
}

// ListObjectTypesParams filters an object-type listing.
type ListObjectTypesParams struct {
	Status string // "draft" | "published" | "deprecated"
	Cursor string
	Limit  int
}

// List returns object types owned by the key's account.
func (s *ObjectTypesService) List(ctx context.Context, params ListObjectTypesParams) (Document, error) {
	q := url.Values{}
	setQuery(q, "status", params.Status)
	setQuery(q, "cursor", params.Cursor)
	setQueryInt(q, "limit", params.Limit)
	return s.client.do(ctx, http.MethodGet, "/v1/object-types", q, nil, nil, false)
}

// Get fetches one object type's current view plus its draft (unpublished working) artifact.
func (s *ObjectTypesService) Get(ctx context.Context, id string) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodGet, "/v1/object-types/"+url.PathEscape(id), nil, nil, nil, false)
}

// Create creates an object type draft. body's "key" is immutable once set; body must also set
// "nameSingular" and "namePlural". Publish it with Publish.
func (s *ObjectTypesService) Create(ctx context.Context, body map[string]any) (Document, error) {
	if err := requireObjectBody(body, "body"); err != nil {
		return nil, err
	}
	key, _ := body["key"].(string)
	if err := requireString(key, "body.key"); err != nil {
		return nil, err
	}
	nameSingular, _ := body["nameSingular"].(string)
	if err := requireString(nameSingular, "body.nameSingular"); err != nil {
		return nil, err
	}
	namePlural, _ := body["namePlural"].(string)
	if err := requireString(namePlural, "body.namePlural"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodPost, "/v1/object-types", nil, body, nil, false)
}

// Update edits the draft. Any field omitted is left unchanged; editing clears the prior
// hasUnpublishedChanges hash.
func (s *ObjectTypesService) Update(ctx context.Context, id string, body map[string]any) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	if err := requireObjectBody(body, "body"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodPatch, "/v1/object-types/"+url.PathEscape(id), nil, body, nil, false)
}

// Versions lists published (immutable) versions, newest first.
func (s *ObjectTypesService) Versions(ctx context.Context, id, cursor string, limit int) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	q := url.Values{}
	setQuery(q, "cursor", cursor)
	setQueryInt(q, "limit", limit)
	return s.client.do(ctx, http.MethodGet, "/v1/object-types/"+url.PathEscape(id)+"/versions", q, nil, nil, false)
}

// Version fetches one immutable published version, including its compiled field policies.
func (s *ObjectTypesService) Version(ctx context.Context, id string, version int) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	if err := requirePositiveInt(version, "version"); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/v1/object-types/%s/versions/%d", url.PathEscape(id), version)
	return s.client.do(ctx, http.MethodGet, path, nil, nil, nil, false)
}

// Publish publishes the draft, freezing its schema + policies into a new immutable version.
// Republishing an unchanged draft is a no-op: it returns the CURRENT version with unchanged: true
// (no new version minted).
func (s *ObjectTypesService) Publish(ctx context.Context, id string, body map[string]any) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodPost, "/v1/object-types/"+url.PathEscape(id)+"/publish", nil, body, nil, false)
}

// Deprecate deprecates a type (idempotent no-op if already deprecated). body must set "reason".
// Existing records are unaffected.
func (s *ObjectTypesService) Deprecate(ctx context.Context, id string, body map[string]any) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	reason, _ := body["reason"].(string)
	if err := requireString(reason, "body.reason"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodPost, "/v1/object-types/"+url.PathEscape(id)+"/deprecate", nil, body, nil, false)
}
