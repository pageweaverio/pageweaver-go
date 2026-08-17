# Changelog

All notable changes to `github.com/pageweaverio/pageweaver-go` are documented here. Format loosely
follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). This module is versioned via git
tags (Go modules convention), not a version constant in the package.

## v0.2.1 - 2026-08-18

### Fixed

- `Documents.Get`/`Verify` were missing the same `requireID` client-side guard their new sibling
  methods got in v0.2.0, letting a blank id slip through to the network instead of failing fast with
  an `*InvalidRequestError`.

### Changed

- The base error type is renamed from `Error` to `APIError` (kept as a `type Error = APIError` alias
  for backward compatibility) — embedding a field literally named `Error` collides with the promoted
  `Error()` method and silently breaks the `error` interface. `APIError` (no `PageWeaver` prefix,
  all-caps `API`) is also the Go-idiomatic name and now matches the casing every other PageWeaver SDK
  uses for its equivalent base class.

## v0.2.0 - 2026-08-18

### Added

- Full parity with the live `/v1` API: `ObjectTypes`, `Objects`, `RelationshipTypes`, `Search`,
  `WorkflowDefinitions`, `FormTemplates`, `Intake` (+ `Sessions` with chunked/batch upload),
  `ErrorCodes`, `Events`.
- `Documents.Trust`, `Diff`, `AppendVersion`, `Versions`, `Version`, `Representations`.
- Typed error hierarchy: `ValidationError`, `AuthenticationError`, `PlanRequiredError`,
  `PermissionError` (`IsScopeMissing`/`RequiredScope`), `NotFoundError`, `ConflictError`,
  `RateLimitError`, `ServerError`, each embedding `*APIError`, plus `InvalidRequestError` for
  client-side checks.
- Automatic retry with exponential backoff + jitter on `429`/`5xx`, honoring `Retry-After`, restricted
  to safe methods (`GET`/`HEAD`/`PUT`/`DELETE` always, `POST` only with an idempotency key).
- Client-side validation (`requireID`, `requireObjectBody`, `requireOneOf`, ...) returning an
  `*InvalidRequestError` before any network call.
- Multipart upload support (`doMultipart`/`MultipartFile`).

### Notes

- `localization.direction`, `ParentMessageID`, and `GitRepo` needed no struct changes here — this
  SDK's bodies are untyped `map[string]any`. It never had a `Projects` resource or a
  `LivingDocuments` resource to remove (a README migration note was added regardless, mapping the old
  REST calls to the new document-lineage calls).

## v0.1.0 - initial release

- Initial `pageweaver-go` release: documents, templates, schemas, environments, deployments, reviews,
  proposals, comments, share links, usage, webhooks.
