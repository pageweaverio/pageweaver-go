package pageweaver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestClientSideValidationNeverHitsNetwork verifies a blank id / missing required field is caught
// before any HTTP request is made — the test server flags itself if hit, and the test fails if so.
func TestClientSideValidationNeverHitsNetwork(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := NewClient("k", WithBaseURL(srv.URL))
	ctx := context.Background()

	if _, err := c.Documents.Get(ctx, ""); !isInvalidRequest(err) {
		t.Fatalf("expected InvalidRequestError for blank id, got %v", err)
	}
	if _, err := c.Documents.Create(ctx, nil, ""); !isInvalidRequest(err) {
		t.Fatalf("expected InvalidRequestError for nil body, got %v", err)
	}
	if _, err := c.Objects.Create(ctx, map[string]any{"data": map[string]any{"a": 1}}, ""); !isInvalidRequest(err) {
		t.Fatalf("expected InvalidRequestError for missing objectTypeKey/Id, got %v", err)
	}
	if _, err := c.Objects.Create(ctx, map[string]any{
		"objectTypeKey": "invoice", "objectTypeId": "ot_1", "data": map[string]any{"a": 1},
	}, ""); !isInvalidRequest(err) {
		t.Fatalf("expected InvalidRequestError when BOTH objectTypeKey and objectTypeId are set, got %v", err)
	}
	if _, err := c.WorkflowDefinitions.Version(ctx, "wf_1", 0); !isInvalidRequest(err) {
		t.Fatalf("expected InvalidRequestError for non-positive version, got %v", err)
	}
	if _, err := c.Intake.Sessions.UploadChunk(ctx, "sess_1", -1, []byte("x")); !isInvalidRequest(err) {
		t.Fatalf("expected InvalidRequestError for a negative chunk index, got %v", err)
	}
	if called {
		t.Fatal("client-side validation failure should never reach the network")
	}
}

func isInvalidRequest(err error) bool {
	var e *InvalidRequestError
	return errors.As(err, &e)
}
