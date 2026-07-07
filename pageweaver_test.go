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

func TestOptions(t *testing.T) {
	hc := &http.Client{Timeout: time.Second}
	c := NewClient("k", WithBaseURL("https://example.test"), WithHTTPClient(hc))
	if c.BaseURL != "https://example.test" {
		t.Fatalf("base url override failed")
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
}
