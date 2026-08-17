package pageweaver

import (
	"net/http"
	"testing"
	"time"
)

func TestNewClientDefaults(t *testing.T) {
	c := NewClient("pk_test_x")
	if c.APIKey != "pk_test_x" {
		t.Fatalf("api key not set")
	}
	if c.BaseURL != DefaultBaseURL {
		t.Fatalf("expected default base url, got %s", c.BaseURL)
	}
	if c.HTTPClient == nil {
		t.Fatalf("http client should default")
	}
}

func TestServicesWired(t *testing.T) {
	c := NewClient("k")
	if c.Documents == nil {
		t.Fatal("Documents service nil")
	}
	if c.Templates == nil {
		t.Fatal("Templates service nil")
	}
	if c.Templates.Proposals == nil {
		t.Fatal("Templates.Proposals service nil")
	}
	if c.Schemas == nil {
		t.Fatal("Schemas service nil")
	}
	if c.Usage == nil {
		t.Fatal("Usage service nil")
	}
	if c.Comments == nil {
		t.Fatal("Comments service nil")
	}
	if c.Reviews == nil {
		t.Fatal("Reviews service nil")
	}
	if c.ShareLinks == nil {
		t.Fatal("ShareLinks service nil")
	}
	if c.Environments == nil {
		t.Fatal("Environments service nil")
	}
	if c.Deployments == nil {
		t.Fatal("Deployments service nil")
	}
	if c.ObjectTypes == nil {
		t.Fatal("ObjectTypes service nil")
	}
	if c.Objects == nil {
		t.Fatal("Objects service nil")
	}
	if c.RelationshipTypes == nil {
		t.Fatal("RelationshipTypes service nil")
	}
	if c.Search == nil {
		t.Fatal("Search service nil")
	}
	if c.WorkflowDefinitions == nil {
		t.Fatal("WorkflowDefinitions service nil")
	}
	if c.FormTemplates == nil {
		t.Fatal("FormTemplates service nil")
	}
	if c.Intake == nil {
		t.Fatal("Intake service nil")
	}
	if c.Intake.Sessions == nil {
		t.Fatal("Intake.Sessions service nil")
	}
	if c.ErrorCodes == nil {
		t.Fatal("ErrorCodes service nil")
	}
	if c.Events == nil {
		t.Fatal("Events service nil")
	}
	// Every service should reference the same client.
	if c.Documents.client != c || c.Templates.Proposals.client != c || c.Objects.client != c {
		t.Fatal("services not backed by the constructing client")
	}
}

func TestNewClientRetryDefaults(t *testing.T) {
	c := NewClient("k")
	if c.MaxRetries != DefaultMaxRetries {
		t.Fatalf("expected default max retries %d, got %d", DefaultMaxRetries, c.MaxRetries)
	}
	if c.RetryBaseDelay != DefaultRetryBaseDelay {
		t.Fatalf("expected default retry base delay, got %v", c.RetryBaseDelay)
	}
	if c.RetryMaxDelay != DefaultRetryMaxDelay {
		t.Fatalf("expected default retry max delay, got %v", c.RetryMaxDelay)
	}

	c2 := NewClient("k", WithMaxRetries(0), WithRetryBaseDelay(10*time.Millisecond), WithRetryMaxDelay(50*time.Millisecond))
	if c2.MaxRetries != 0 {
		t.Fatalf("expected overridden max retries 0, got %d", c2.MaxRetries)
	}
}

func TestOptions(t *testing.T) {
	hc := &http.Client{Timeout: time.Second}
	c := NewClient("k", WithBaseURL("https://example.test/"), WithHTTPClient(hc))
	if c.BaseURL != "https://example.test" {
		t.Fatalf("base url override / trailing-slash trim failed: %q", c.BaseURL)
	}
	if c.HTTPClient != hc {
		t.Fatalf("http client override failed")
	}
}

func TestErrorMessage(t *testing.T) {
	e := &Error{StatusCode: 402, Body: "quota"}
	if e.Error() == "" {
		t.Fatalf("error message empty")
	}
	withMsg := &Error{StatusCode: 400, Message: "bad payload", Body: `{"message":"bad payload"}`}
	if got := withMsg.Error(); got == "" {
		t.Fatalf("error message empty")
	}
}

func TestNewAPIErrorParsesBody(t *testing.T) {
	e := newAPIError(400, []byte(`{"message":"nope","code":"E_BAD"}`))
	if e.Message != "nope" {
		t.Fatalf("expected parsed message, got %q", e.Message)
	}
	if e.Code != "E_BAD" {
		t.Fatalf("expected parsed code, got %q", e.Code)
	}
	// Array-form message joins.
	e2 := newAPIError(400, []byte(`{"message":["a","b"]}`))
	if e2.Message != "a, b" {
		t.Fatalf("expected joined message, got %q", e2.Message)
	}
	// Non-JSON body keeps the raw text and empty fields.
	e3 := newAPIError(500, []byte("boom"))
	if e3.Body != "boom" || e3.Message != "" {
		t.Fatalf("unexpected non-json handling: %+v", e3)
	}
}

func TestWebhookSignatureRoundTrip(t *testing.T) {
	secret := "whsec_test"
	body := []byte(`{"event":"document.completed","documentId":"doc_1"}`)

	sig := SignWebhookBody(secret, body)
	if !VerifySignature(secret, body, sig) {
		t.Fatal("valid signature failed to verify")
	}
	if VerifySignature("wrong_secret", body, sig) {
		t.Fatal("wrong secret should not verify")
	}
	if VerifySignature(secret, body, "") {
		t.Fatal("empty signature should not verify")
	}
	if VerifySignature(secret, []byte("tampered"), sig) {
		t.Fatal("tampered body should not verify")
	}
}

func TestVerifyWebhook(t *testing.T) {
	secret := "whsec_test"
	body := []byte(`{"event":"document.failed","documentId":"doc_2"}`)
	sig := SignWebhookBody(secret, body)

	payload, err := VerifyWebhook(secret, body, sig)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if payload["documentId"] != "doc_2" {
		t.Fatalf("unexpected payload: %+v", payload)
	}

	if _, err := VerifyWebhook(secret, body, "sha256=deadbeef"); err != ErrInvalidWebhookSignature {
		t.Fatalf("expected ErrInvalidWebhookSignature, got %v", err)
	}
}

func TestWaitOptionsDefaults(t *testing.T) {
	interval, maxInterval, timeout, backoff, throwOnFailure := WaitOptions{}.resolved()
	if interval != time.Second || maxInterval != 5*time.Second || timeout != 60*time.Second {
		t.Fatalf("unexpected default durations: %v %v %v", interval, maxInterval, timeout)
	}
	if backoff != 1.5 {
		t.Fatalf("expected default backoff 1.5, got %v", backoff)
	}
	if !throwOnFailure {
		t.Fatal("expected throwOnFailure to default true")
	}
	no := false
	_, _, _, _, tf := WaitOptions{ThrowOnFailure: &no}.resolved()
	if tf {
		t.Fatal("expected throwOnFailure false when explicitly set")
	}
}

func TestWebhookHeaderConstants(t *testing.T) {
	if WebhookSignatureHeader != "x-pageweaver-signature" {
		t.Fatalf("unexpected signature header: %s", WebhookSignatureHeader)
	}
	if WebhookEventHeader != "x-pageweaver-event" {
		t.Fatalf("unexpected event header: %s", WebhookEventHeader)
	}
	if WebhookTimestampHeader != "x-pageweaver-timestamp" {
		t.Fatalf("unexpected timestamp header: %s", WebhookTimestampHeader)
	}
}
