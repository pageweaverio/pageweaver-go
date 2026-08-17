package pageweaver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// TestRetriesTransientGetFailures verifies a GET is retried on 503 and eventually succeeds.
func TestRetriesTransientGetFailures(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "doc_1", "status": "done"})
	}))
	defer srv.Close()

	c := NewClient("k", WithBaseURL(srv.URL), WithRetryBaseDelay(time.Millisecond), WithRetryMaxDelay(5*time.Millisecond))
	doc, err := c.Documents.Get(context.Background(), "doc_1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc["id"] != "doc_1" {
		t.Fatalf("unexpected document: %+v", doc)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempts)
	}
}

// TestPlainPOSTNeverRetried verifies a POST with no idempotency key is not retried on 503.
func TestPlainPOSTNeverRetried(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := NewClient("k", WithBaseURL(srv.URL), WithRetryBaseDelay(time.Millisecond), WithRetryMaxDelay(5*time.Millisecond))
	_, err := c.Documents.Create(context.Background(), map[string]any{"templateId": "t"}, "")
	if err == nil {
		t.Fatal("expected an error")
	}
	if atomic.LoadInt32(&attempts) != 1 {
		t.Fatalf("expected exactly 1 attempt (no retry), got %d", attempts)
	}
	var srvErr *ServerError
	if !errors.As(err, &srvErr) {
		t.Fatalf("expected *ServerError, got %T", err)
	}
}

// TestIdempotentPOSTIsRetried verifies a POST WITH an idempotency key is retried on 503.
func TestIdempotentPOSTIsRetried(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "doc_2", "status": "queued"})
	}))
	defer srv.Close()

	c := NewClient("k", WithBaseURL(srv.URL), WithRetryBaseDelay(time.Millisecond), WithRetryMaxDelay(5*time.Millisecond))
	doc, err := c.Documents.Create(context.Background(), map[string]any{"templateId": "t"}, "idem-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc["id"] != "doc_2" {
		t.Fatalf("unexpected document: %+v", doc)
	}
	if atomic.LoadInt32(&attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

// TestMaxRetriesZeroDisablesRetry verifies WithMaxRetries(0) disables retry entirely.
func TestMaxRetriesZeroDisablesRetry(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	c := NewClient("k", WithBaseURL(srv.URL), WithMaxRetries(0))
	_, err := c.Documents.Get(context.Background(), "doc_1")
	if err == nil {
		t.Fatal("expected an error")
	}
	if atomic.LoadInt32(&attempts) != 1 {
		t.Fatalf("expected exactly 1 attempt, got %d", attempts)
	}
}

// TestRetryAfterHeaderHonored verifies a 429 with Retry-After: 0 still succeeds quickly (we don't
// assert on timing, only that the header is parsed without error and the retry proceeds).
func TestRetryAfterHeaderHonored(t *testing.T) {
	var attempts int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		if n < 2 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "doc_3"})
	}))
	defer srv.Close()

	c := NewClient("k", WithBaseURL(srv.URL))
	doc, err := c.Documents.Get(context.Background(), "doc_3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc["id"] != "doc_3" {
		t.Fatalf("unexpected document: %+v", doc)
	}
}

// TestRateLimitErrorCarriesRetryAfter verifies a non-retried (retries exhausted) 429 surfaces
// RetryAfterSeconds on the typed error.
func TestRateLimitErrorCarriesRetryAfter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	c := NewClient("k", WithBaseURL(srv.URL), WithMaxRetries(0))
	_, err := c.Documents.Get(context.Background(), "doc_1")
	var rle *RateLimitError
	if !errors.As(err, &rle) {
		t.Fatalf("expected *RateLimitError, got %T (%v)", err, err)
	}
	if rle.RetryAfterSeconds == nil || *rle.RetryAfterSeconds != 7 {
		t.Fatalf("expected RetryAfterSeconds 7, got %v", rle.RetryAfterSeconds)
	}
}

// TestMultipartUploadSendsFields verifies form-template creation sends the file + fields as
// multipart/form-data.
func TestMultipartUploadSendsFields(t *testing.T) {
	var gotContentType string
	var gotName string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotContentType = r.Header.Get("Content-Type")
		if err := r.ParseMultipartForm(10 << 20); err != nil {
			t.Fatalf("failed to parse multipart form: %v", err)
		}
		gotName = r.FormValue("name")
		file, header, err := r.FormFile("file")
		if err != nil {
			t.Fatalf("expected a file part: %v", err)
		}
		defer file.Close()
		if header.Filename != "template.pdf" {
			t.Fatalf("unexpected filename: %s", header.Filename)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "ftpl_1"})
	}))
	defer srv.Close()

	c := NewClient("k", WithBaseURL(srv.URL))
	doc, err := c.FormTemplates.Create(context.Background(), "Claim form", "", MultipartFile{
		Data: []byte("%PDF-1.4 fake"), Filename: "template.pdf", ContentType: "application/pdf",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if doc["id"] != "ftpl_1" {
		t.Fatalf("unexpected document: %+v", doc)
	}
	if !strings.HasPrefix(gotContentType, "multipart/form-data") {
		t.Fatalf("expected multipart content type, got %q", gotContentType)
	}
	if gotName != "Claim form" {
		t.Fatalf("expected name field 'Claim form', got %q", gotName)
	}
}
