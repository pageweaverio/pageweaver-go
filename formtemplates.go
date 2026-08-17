package pageweaver

import (
	"context"
	"net/http"
	"net/url"
)

// FormTemplatesService operates on fillable AcroForm templates: upload a PDF that already has its
// own form fields, then fill and render it as a document. Distinct from the Liquid + JSON Schema
// template flow. Uploads need the documents:upload scope, reads need read, filling needs render.
type FormTemplatesService struct {
	client *Client
}

// Create uploads a PDF as a new fillable template. Scans, safety-checks, and enumerates its
// AcroForm fields. name is required; description is optional ("").
func (s *FormTemplatesService) Create(ctx context.Context, name, description string, file MultipartFile) (Document, error) {
	if err := requireString(name, "name"); err != nil {
		return nil, err
	}
	if file.Filename == "" {
		file.Filename = "template.pdf"
	}
	fields := map[string]string{"name": name, "description": description}
	files := map[string]MultipartFile{"file": file}
	return s.client.doMultipart(ctx, http.MethodPost, "/v1/form-templates", fields, files, nil)
}

// List returns the account's fillable form templates.
func (s *FormTemplatesService) List(ctx context.Context) ([]Document, error) {
	return s.client.doList(ctx, http.MethodGet, "/v1/form-templates", nil, nil, nil)
}

// Get fetches a template plus its current version's derived field-schema contract.
func (s *FormTemplatesService) Get(ctx context.Context, id string) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodGet, "/v1/form-templates/"+url.PathEscape(id), nil, nil, nil, false)
}

// Versions returns a template's version history.
func (s *FormTemplatesService) Versions(ctx context.Context, id string) ([]Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	return s.client.doList(ctx, http.MethodGet, "/v1/form-templates/"+url.PathEscape(id)+"/versions", nil, nil, nil)
}

// AddVersion uploads a new version of an existing template. Re-runs the full pipeline (scan,
// safety checks, field re-enumeration).
func (s *FormTemplatesService) AddVersion(ctx context.Context, id string, file MultipartFile) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	if file.Filename == "" {
		file.Filename = "template.pdf"
	}
	files := map[string]MultipartFile{"file": file}
	return s.client.doMultipart(ctx, http.MethodPost, "/v1/form-templates/"+url.PathEscape(id)+"/versions", nil, files, nil)
}

// Fill fills and renders the template with body["payload"] (keyed by the AcroForm's dotted field
// name). Stored as an ordinary document — hash chain, retention, and delivery are all inherited.
// Returns 202.
func (s *FormTemplatesService) Fill(ctx context.Context, id string, body map[string]any) (Document, error) {
	if _, err := requireID(id, "id"); err != nil {
		return nil, err
	}
	if err := requireObjectBody(body, "body"); err != nil {
		return nil, err
	}
	payload, _ := body["payload"].(map[string]any)
	if err := requireObjectBody(payload, "body.payload"); err != nil {
		return nil, err
	}
	return s.client.do(ctx, http.MethodPost, "/v1/form-templates/"+url.PathEscape(id)+"/fill", nil, body, nil, false)
}
