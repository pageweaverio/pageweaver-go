package pageweaver

import (
	"context"
	"net/http"
	"net/url"
)

// RelationshipTypesService operates on edge-rule definitions between object types (key,
// label/inverseLabel, allowed source/target types, cardinality). Deliberately not versioned —
// nothing is ever validated against a frozen snapshot of one. Reads need objects:read; writes need
// relationships:manage.
type RelationshipTypesService struct {
	client *Client
}

// List returns relationship types. Filter by status ("active" | "deprecated"); page with cursor.
func (s *RelationshipTypesService) List(ctx context.Context, status, cursor string, limit int) (Document, error) {
	q := url.Values{}
	setQuery(q, "status", status)
	setQuery(q, "cursor", cursor)
	setQueryInt(q, "limit", limit)
	return s.client.do(ctx, http.MethodGet, "/v1/relationship-types", q, nil, nil, false)
}

// Get fetches one relationship type.
func (s *RelationshipTypesService) Get(ctx context.Context, id string) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodGet, "/v1/relationship-types/"+url.PathEscape(id), nil, nil, nil, false)
}

// Create creates a relationship type. body must set "key", "label", and "inverseLabel" —
// relationships read in both directions (source->target, target->source).
func (s *RelationshipTypesService) Create(ctx context.Context, body map[string]any) (Document, error) {
	if err := requireObjectBody(body, "body"); err != nil {
		return nil, err
	}
	key, _ := body["key"].(string)
	if err := requireString(key, "body.key"); err != nil {
		return nil, err
	}
	label, _ := body["label"].(string)
	if err := requireString(label, "body.label"); err != nil {
		return nil, err
	}
	inverseLabel, _ := body["inverseLabel"].(string)
	if err := requireString(inverseLabel, "body.inverseLabel"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodPost, "/v1/relationship-types", nil, body, nil, false)
}

// Update edits a relationship type. Changes govern only edges created AFTER the update; existing
// edges are never re-checked or removed.
func (s *RelationshipTypesService) Update(ctx context.Context, id string, body map[string]any) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	if err := requireObjectBody(body, "body"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodPatch, "/v1/relationship-types/"+url.PathEscape(id), nil, body, nil, false)
}

// Deprecate deprecates a relationship type. body must set "reason". No delete — existing edges of
// this type are untouched.
func (s *RelationshipTypesService) Deprecate(ctx context.Context, id string, body map[string]any) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	reason, _ := body["reason"].(string)
	if err := requireString(reason, "body.reason"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodPost, "/v1/relationship-types/"+url.PathEscape(id)+"/deprecate", nil, body, nil, false)
}
