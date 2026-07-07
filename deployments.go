package pageweaver

import (
	"context"
	"net/http"
	"net/url"
)

// DeploymentsService operates on deployments — documents-as-code (Pillar 3). Plan a pageweaver.yml
// manifest against a target environment, then apply it. Plan and apply are separate, explicit
// calls — planning never changes anything. Writes require a deploy-scoped API key.
type DeploymentsService struct {
	client *Client
}

// Plan plans a deployment: send the manifest text + the contents of every file it names + the
// target environment. Returns 202 with the plan. Pass an idempotencyKey (or "") to dedupe.
func (s *DeploymentsService) Plan(ctx context.Context, body map[string]any, idempotencyKey string) (Document, error) {
	var headers map[string]string
	if idempotencyKey != "" {
		headers = map[string]string{"Idempotency-Key": idempotencyKey}
	}
	return s.client.do(ctx, http.MethodPost, "/v1/deployments/plan", nil, body, headers, false)
}

// List returns recent deployments for the account, newest first. Filter by environment slug.
func (s *DeploymentsService) List(ctx context.Context, environment string, limit int) ([]Document, error) {
	q := url.Values{}
	setQuery(q, "environment", environment)
	setQueryInt(q, "limit", limit)
	return s.client.doList(ctx, http.MethodGet, "/v1/deployments", q, nil, nil)
}

// Get returns one deployment with its per-resource plan lines and their apply outcomes.
func (s *DeploymentsService) Get(ctx context.Context, id string) (Document, error) {
	return s.client.do(ctx, http.MethodGet, "/v1/deployments/"+url.PathEscape(id), nil, nil, nil, false)
}

// Apply applies a planned deployment: publish the changed versions and write the environment's
// pins. Returns 202 with the deployment in "applying" — poll Get for the terminal state.
func (s *DeploymentsService) Apply(ctx context.Context, id string) (Document, error) {
	return s.client.do(ctx, http.MethodPost, "/v1/deployments/"+url.PathEscape(id)+"/apply", nil, nil, nil, false)
}
