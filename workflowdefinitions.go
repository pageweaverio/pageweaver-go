package pageweaver

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// WorkflowDefinitionsService offers read-only discovery of workflow definitions (the stage graph /
// transitions / task templates a workflow compiles to). No public write route yet — authoring is
// via deploy / documents-as-code or the portal designer. Requires the workflows:read scope.
type WorkflowDefinitionsService struct {
	client *Client
}

// List returns workflow definitions. Filter by status ("draft" | "published" | "deprecated"); page
// with cursor.
func (s *WorkflowDefinitionsService) List(ctx context.Context, status, cursor string, limit int) (Document, error) {
	q := url.Values{}
	setQuery(q, "status", status)
	setQuery(q, "cursor", cursor)
	setQueryInt(q, "limit", limit)
	return s.client.do(ctx, http.MethodGet, "/v1/workflow-definitions", q, nil, nil, false)
}

// Get fetches one workflow definition.
func (s *WorkflowDefinitionsService) Get(ctx context.Context, id string) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodGet, "/v1/workflow-definitions/"+url.PathEscape(id), nil, nil, nil, false)
}

// Versions lists a workflow definition's published version history.
func (s *WorkflowDefinitionsService) Versions(ctx context.Context, id, cursor string, limit int) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	q := url.Values{}
	setQuery(q, "cursor", cursor)
	setQueryInt(q, "limit", limit)
	return s.client.do(ctx, http.MethodGet, "/v1/workflow-definitions/"+url.PathEscape(id)+"/versions", q, nil, nil, false)
}

// Version fetches one published version.
func (s *WorkflowDefinitionsService) Version(ctx context.Context, id string, version int) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	if err := requirePositiveInt(version, "version"); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/v1/workflow-definitions/%s/versions/%d", url.PathEscape(id), version)
	return s.client.do(ctx, http.MethodGet, path, nil, nil, nil, false)
}
