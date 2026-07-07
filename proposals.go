package pageweaver

import (
	"context"
	"net/http"
	"net/url"
)

// ProposalsService operates on template proposals — the PR analog for template changes (Pillar 2).
// A proposal freezes a candidate change; it is reviewed and approved, then promoted into a
// published version. Writes require an API key with the deploy scope. Reached as
// client.Templates.Proposals, scoped to a template id passed on each call.
type ProposalsService struct {
	client *Client
}

// Open opens a proposal on a template: freeze a candidate (inline html/css/payloadSchema, or
// fromDraft true to use the saved draft). Returns 202 with the proposal.
func (s *ProposalsService) Open(ctx context.Context, templateID string, body map[string]any) (Document, error) {
	return s.client.do(ctx, http.MethodPost, s.base(templateID), nil, body, nil, false)
}

// List returns a template's proposals, newest first. Filter by status; page with cursor.
func (s *ProposalsService) List(ctx context.Context, templateID, status, cursor string, limit int) (Document, error) {
	q := url.Values{}
	setQuery(q, "status", status)
	setQuery(q, "cursor", cursor)
	setQueryInt(q, "limit", limit)
	return s.client.do(ctx, http.MethodGet, s.base(templateID), q, nil, nil, false)
}

// Get fetches one proposal with its check summary, approvals, and promote-gate state.
func (s *ProposalsService) Get(ctx context.Context, templateID, proposalID string) (Document, error) {
	return s.client.do(ctx, http.MethodGet, s.item(templateID, proposalID), nil, nil, nil, false)
}

// RerunChecks re-runs the render-diff regression (candidate vs. the live version). Returns 202.
func (s *ProposalsService) RerunChecks(ctx context.Context, templateID, proposalID string) (Document, error) {
	return s.client.do(ctx, http.MethodPost, s.item(templateID, proposalID)+"/checks", nil, nil, nil, false)
}

// Approve appends an approval decision. The author can never approve. Returns 201.
func (s *ProposalsService) Approve(ctx context.Context, templateID, proposalID string, body map[string]any) (Document, error) {
	return s.client.do(ctx, http.MethodPost, s.item(templateID, proposalID)+"/approve", nil, body, nil, false)
}

// Reject appends a rejection decision and moves the proposal to the rejected terminal state.
func (s *ProposalsService) Reject(ctx context.Context, templateID, proposalID string, body map[string]any) (Document, error) {
	return s.client.do(ctx, http.MethodPost, s.item(templateID, proposalID)+"/reject", nil, body, nil, false)
}

// Promote publishes the candidate as the next version through the gate-checked publish. Fails 409
// when the approval gate is unmet, blocking comments are open, or the base version moved.
func (s *ProposalsService) Promote(ctx context.Context, templateID, proposalID string) (Document, error) {
	return s.client.do(ctx, http.MethodPost, s.item(templateID, proposalID)+"/promote", nil, nil, nil, false)
}

// Retract withdraws an open proposal (only while open). The live version is untouched.
func (s *ProposalsService) Retract(ctx context.Context, templateID, proposalID string) (Document, error) {
	return s.client.do(ctx, http.MethodDelete, s.item(templateID, proposalID), nil, nil, nil, false)
}

func (s *ProposalsService) base(templateID string) string {
	return "/v1/templates/" + url.PathEscape(templateID) + "/proposals"
}

func (s *ProposalsService) item(templateID, proposalID string) string {
	return s.base(templateID) + "/" + url.PathEscape(proposalID)
}
