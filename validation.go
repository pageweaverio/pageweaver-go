package pageweaver

import "strings"

// Lightweight client-side request validation: catch shape mistakes (a blank id interpolated into a
// URL, a missing required field) before spending a network round trip. This deliberately does NOT
// re-implement the API's business rules or JSON Schema validation — the API remains the source of
// truth for those. It only guards against the class of mistake that produces a confusing generic
// 400/404 or, worse, a request sent to the wrong URL (e.g. "/v1/objects/").

// requireID checks that value is a non-empty, non-whitespace string used as a URL path segment (an
// id, a slug, an env name, ...).
func requireID(value, name string) (string, error) {
	if strings.TrimSpace(value) == "" {
		return "", &InvalidRequestError{Msg: "`" + name + "` is required and must be a non-empty string.", Path: name}
	}
	return value, nil
}

// requireString checks that value is a non-empty, non-whitespace required string field.
func requireString(value, name string) error {
	if strings.TrimSpace(value) == "" {
		return &InvalidRequestError{Msg: "`" + name + "` is required and must be a non-empty string.", Path: name}
	}
	return nil
}

// requirePositiveInt checks that value is a positive integer (a version number, a chunk index + 1,
// a page seq, ...).
func requirePositiveInt(value int, name string) error {
	if value < 1 {
		return &InvalidRequestError{Msg: "`" + name + "` must be a positive integer.", Path: name}
	}
	return nil
}

// requireNonNegativeInt checks that value is a non-negative integer (a chunk index, ...).
func requireNonNegativeInt(value int, name string) error {
	if value < 0 {
		return &InvalidRequestError{Msg: "`" + name + "` must be a non-negative integer.", Path: name}
	}
	return nil
}

// requireObjectBody checks that body is non-nil and non-empty.
func requireObjectBody(body map[string]any, name string) error {
	if body == nil {
		return &InvalidRequestError{Msg: "`" + name + "` must be an object.", Path: name}
	}
	return nil
}

// requireNonEmptySlice checks that a slice has at least one item.
func requireNonEmptySlice[T any](value []T, name string) error {
	if len(value) == 0 {
		return &InvalidRequestError{Msg: "`" + name + "` must be a non-empty array.", Path: name}
	}
	return nil
}

// requireOneOf asserts that exactly one of two mutually-exclusive optional fields is set.
func requireOneOf(a, aName, b, bName string) error {
	hasA := a != ""
	hasB := b != ""
	if hasA == hasB {
		return &InvalidRequestError{Msg: "Provide exactly one of `" + aName + "` or `" + bName + "`."}
	}
	return nil
}
