package pageweaver

import (
	"errors"
	"testing"
)

func TestApiErrorForStatusTypes(t *testing.T) {
	cases := []struct {
		status int
		check  func(error) bool
	}{
		{400, func(err error) bool { var e *ValidationError; return errors.As(err, &e) }},
		{422, func(err error) bool { var e *ValidationError; return errors.As(err, &e) }},
		{401, func(err error) bool { var e *AuthenticationError; return errors.As(err, &e) }},
		{402, func(err error) bool { var e *PlanRequiredError; return errors.As(err, &e) }},
		{403, func(err error) bool { var e *PermissionError; return errors.As(err, &e) }},
		{404, func(err error) bool { var e *NotFoundError; return errors.As(err, &e) }},
		{409, func(err error) bool { var e *ConflictError; return errors.As(err, &e) }},
		{429, func(err error) bool { var e *RateLimitError; return errors.As(err, &e) }},
		{500, func(err error) bool { var e *ServerError; return errors.As(err, &e) }},
		{503, func(err error) bool { var e *ServerError; return errors.As(err, &e) }},
	}
	for _, tc := range cases {
		err := apiErrorForStatus(&Error{StatusCode: tc.status, Message: "boom"})
		if !tc.check(err) {
			t.Fatalf("status %d: unexpected error type %T", tc.status, err)
		}
		// Every typed error must still be reachable as the base *Error.
		var base *Error
		if !errors.As(err, &base) {
			t.Fatalf("status %d: not reachable as *Error", tc.status)
		}
		if base.StatusCode != tc.status {
			t.Fatalf("status %d: base StatusCode mismatch: %d", tc.status, base.StatusCode)
		}
	}

	// A status with no dedicated subtype falls back to the base *Error.
	other := apiErrorForStatus(&Error{StatusCode: 418})
	if _, ok := other.(*Error); !ok {
		t.Fatalf("expected bare *Error for an unmapped status, got %T", other)
	}
}

func TestErrorIsRetryable(t *testing.T) {
	if !(&Error{StatusCode: 429}).IsRetryable() {
		t.Fatal("429 should be retryable")
	}
	if !(&Error{StatusCode: 503}).IsRetryable() {
		t.Fatal("503 should be retryable")
	}
	if (&Error{StatusCode: 404}).IsRetryable() {
		t.Fatal("404 should not be retryable")
	}
}

func TestPermissionErrorScopeMissing(t *testing.T) {
	e := &PermissionError{&Error{
		StatusCode: 403,
		Code:       "authorization.scope_missing",
		Message:    "Forbidden: missing the 'review' scope",
	}}
	if !e.IsScopeMissing() {
		t.Fatal("expected IsScopeMissing true")
	}
	if e.RequiredScope() != "review" {
		t.Fatalf("expected required scope 'review', got %q", e.RequiredScope())
	}

	other := &PermissionError{&Error{StatusCode: 403, Code: "some.other_code", Message: "nope"}}
	if other.IsScopeMissing() {
		t.Fatal("expected IsScopeMissing false for a different code")
	}
	if other.RequiredScope() != "" {
		t.Fatalf("expected empty required scope, got %q", other.RequiredScope())
	}
}

func TestInvalidRequestError(t *testing.T) {
	err := &InvalidRequestError{Msg: "`id` is required.", Path: "id"}
	if err.Error() == "" {
		t.Fatal("expected non-empty error message")
	}
	if err.Path != "id" {
		t.Fatalf("unexpected path: %q", err.Path)
	}
}
