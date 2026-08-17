package pageweaver

import "regexp"

// scopeMissingCode is the API's stable code for "this API key authenticated fine but lacks a
// required scope".
const scopeMissingCode = "authorization.scope_missing"

// requiredScopePattern best-effort parses "missing the 'X' scope" out of a permission error's
// message. Falls back to no match rather than guessing.
var requiredScopePattern = regexp.MustCompile(`missing the '([^']+)' scope`)

// IsRetryable reports whether retrying this exact request (with the same idempotency key, if any)
// may succeed: a 429 or a 5xx.
func (e *APIError) IsRetryable() bool {
	return e.StatusCode == 429 || e.StatusCode >= 500
}

// Every typed error below embeds *APIError (not *Error — the same type under its alias, but
// spelled APIError here deliberately: an anonymous field takes the embedded type's own name, and
// naming it "Error" would collide with the promoted Error() method and silently break the `error`
// interface). The embedding still promotes every APIError field (StatusCode, Message, Code, ...)
// and its Error()/IsRetryable() methods, so e.g. `ve.StatusCode` and `ve.Error()` both work
// directly on a *ValidationError.

// ValidationError is a 400/422: the request body or query failed validation. FieldErrors carries
// the raw parsed body's "errors" field-level detail when present.
type ValidationError struct{ *APIError }

// Unwrap exposes the embedded *APIError (also *Error) to errors.As/errors.Is.
func (e *ValidationError) Unwrap() error { return e.APIError }

// AuthenticationError is a 401: the API key is missing, malformed, revoked, or the account is
// suspended/scheduled for deletion.
type AuthenticationError struct{ *APIError }

func (e *AuthenticationError) Unwrap() error { return e.APIError }

// PlanRequiredError is a 402 — a billing problem, not a credential one: the account's plan doesn't
// include this capability at all (e.g. provenance receipts, proof packs, document versioning,
// deployments, digital signing, structured e-invoice output, public alias links). No API key,
// however scoped, can call this successfully until the account upgrades. Contrast with
// PermissionError, where the feature is available but this specific key isn't allowed to use it.
type PlanRequiredError struct{ *APIError }

func (e *PlanRequiredError) Unwrap() error { return e.APIError }

// PermissionError is a 403: the API key authenticated fine but is not allowed to do this — either
// it lacks a required scope (IsScopeMissing true, RequiredScope names it) or the account can't see
// this resource for another reason (e.g. an object-type access policy).
type PermissionError struct{ *APIError }

func (e *PermissionError) Unwrap() error { return e.APIError }

// IsScopeMissing reports whether this 403 is specifically a missing-scope refusal
// (Code == "authorization.scope_missing").
func (e *PermissionError) IsScopeMissing() bool {
	return e.Code == scopeMissingCode
}

// RequiredScope returns the scope name the key is missing (e.g. "review"), when IsScopeMissing is
// true and the API's message named it. Mint a new API key with that scope in the portal to resolve
// it. Returns "" when unknown.
func (e *PermissionError) RequiredScope() string {
	if !e.IsScopeMissing() {
		return ""
	}
	if m := requiredScopePattern.FindStringSubmatch(e.Message); m != nil {
		return m[1]
	}
	return ""
}

// NotFoundError is a 404: no such resource, or it belongs to another tenant (the API never
// distinguishes the two).
type NotFoundError struct{ *APIError }

func (e *NotFoundError) Unwrap() error { return e.APIError }

// ConflictError is a 409: an optimistic-concurrency mismatch (expectedVersion/If-Match), a
// duplicate key, or a state conflict.
type ConflictError struct{ *APIError }

func (e *ConflictError) Unwrap() error { return e.APIError }

// RateLimitError is a 429: rate limited or over a usage quota. RetryAfterSeconds is set when the
// API sent Retry-After.
type RateLimitError struct{ *APIError }

func (e *RateLimitError) Unwrap() error { return e.APIError }

// ServerError is a 5xx: the API failed unexpectedly. Safe to retry (the client already retries
// these automatically).
type ServerError struct{ *APIError }

func (e *ServerError) Unwrap() error { return e.APIError }

// InvalidRequestError is an SDK-side error: a request body failed a client-side shape check before
// it was sent — no network call was made. Path names the offending field when known.
type InvalidRequestError struct {
	Msg  string
	Path string
}

func (e *InvalidRequestError) Error() string {
	return "pageweaver: " + e.Msg
}

// apiErrorForStatus builds the right typed error for a status code, wrapping a populated
// *APIError.
func apiErrorForStatus(base *APIError) error {
	switch base.StatusCode {
	case 400, 422:
		return &ValidationError{base}
	case 401:
		return &AuthenticationError{base}
	case 402:
		return &PlanRequiredError{base}
	case 403:
		return &PermissionError{base}
	case 404:
		return &NotFoundError{base}
	case 409:
		return &ConflictError{base}
	case 429:
		return &RateLimitError{base}
	default:
		if base.StatusCode >= 500 {
			return &ServerError{base}
		}
		return base
	}
}
