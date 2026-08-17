# PageWeaver Go SDK

Official Go client for the [PageWeaver](https://pageweaver.io) PDF generation API. Standard library only, no dependencies.

## Install

```bash
go get github.com/pageweaverio/pageweaver-go
```

## Client

Construct a client and reach the API through resource-service fields. Every method takes a
`context.Context` first. Object responses come back as `pageweaver.Document` (a `map[string]any`)
and arrays as `[]pageweaver.Document`, so the SDK stays dependency-free and forward-compatible with
new response fields.

```go
pw := pageweaver.NewClient("pk_live_...")

// Override the base URL (e.g. in dev) or the HTTP client:
pw = pageweaver.NewClient("pk_live_...",
    pageweaver.WithBaseURL("http://localhost:4000"),
    pageweaver.WithHTTPClient(&http.Client{Timeout: 60 * time.Second}),
)
```

Resource services: `Documents`, `Templates` (with `Templates.Proposals`), `Schemas`, `Usage`,
`Comments`, `Reviews`, `ShareLinks`, `Environments`, `Deployments`, `ObjectTypes`, `Objects`,
`RelationshipTypes`, `Search`, `WorkflowDefinitions`, `FormTemplates`, `Intake` (with
`Intake.Sessions`), `ErrorCodes`, `Events`.

## Documents

```go
ctx := context.Background()

// Create and poll to completion (backing-off, 60s default timeout).
doc, err := pw.Documents.CreateAndWait(ctx, map[string]any{
    "templateId": "tmpl_invoice",
    "payload":    map[string]any{"number": "INV-001", "total": 4200},
}, pageweaver.WaitOptions{})
if err != nil {
    panic(err)
}
fmt.Println(doc["status"]) // "done"

// Or create (202), then wait separately.
created, _ := pw.Documents.Create(ctx, body, "idempotency-key-optional")
final, _ := pw.Documents.WaitFor(ctx, created["id"].(string), pageweaver.WaitOptions{})

// Download the finished PDF (resolves the signed URL automatically).
pdf, _ := pw.Documents.Download(ctx, doc["id"].(string), "")
// For a download-protected document, pass the password instead:
pdf, _ = pw.Documents.Download(ctx, doc["id"].(string), "s3cret")
```

Synchronous create (one HTTP call, no client-side polling) is content-negotiated:

```go
res, _ := pw.Documents.CreateSync(ctx, body, true /* want raw PDF bytes */)
switch res.Kind {
case "pdf":
    os.WriteFile("invoice.pdf", res.PDF, 0o644)
case "document":
    fmt.Println(res.Document["id"])
case "pending":
    final, _ := pw.Documents.WaitFor(ctx, res.ID, pageweaver.WaitOptions{})
    _ = final
}
```

Listing, integrity, and history:

```go
page, _ := pw.Documents.List(ctx, pageweaver.ListParams{Status: "done", Limit: 50})
all, _ := pw.Documents.ListAll(ctx, pageweaver.ListParams{Status: "failed"}) // follows nextCursor
proof, _ := pw.Documents.Verify(ctx, id)
again, _ := pw.Documents.Regenerate(ctx, id)
pages, _ := pw.Documents.Pages(ctx, id)
```

### Archival PDF/A

```go
doc, _ := pw.Documents.CreateAndWait(ctx, map[string]any{
    "templateId": "tmpl_invoice",
    "payload":    map[string]any{"number": "INV-001"},
    "output":     map[string]any{"pdfa": "3b"}, // "2b" | "3b" | "none"
}, pageweaver.WaitOptions{})
fmt.Println(doc["pdfa"])          // "3b"
fmt.Println(doc["outputNotices"]) // what had to change to honor the request
```

`2b` and `3b` produce a validated PDF/A. (`1b` is not offered: the conversion cannot produce one that
passes validation.) Send `"none"` to opt out of a template that defaults to archival output.

Three things change, and two are invisible in the produced document:

- **Links stop working.** Every clickable link annotation is dropped.
- **Some text stops being extractable.** Text set with OpenType feature substitution, most commonly
  `font-variant-numeric: tabular-nums`, looks identical but can no longer be selected, searched, or
  copied. A PDF/A document is therefore **not** a machine-readability guarantee.
- **`Author` is not written**, because PDF/A cannot record it conformantly.

A digital signature works alongside it: the signature is applied after the archival conversion and
the result still validates. It cannot be combined with an image format, a PDF open-password, or a `url`
render, and it adds roughly 200ms plus 25ms per page.


### Accessible PDF/UA

```go
doc, _ := pw.Documents.CreateAndWait(ctx, map[string]any{
    "templateId": "tmpl_invoice",
    "payload":    map[string]any{"number": "INV-001"},
    "output":     map[string]any{"pdfUa": "1"}, // "1" | "none"
}, pageweaver.WaitOptions{})
fmt.Println(doc["accessibility"]) // map[standard:PDF/UA-1 conformant:true ...]

report, _ := pw.Documents.Accessibility(ctx, doc["id"].(string)) // every rule, with its ISO clause
```

`"1"` is the only level (PDF/UA-2 needs PDF 2.0, which the renderer does not emit). Send `"none"` to
opt out of a template that defaults to accessible output.

**Conformance depends on your markup, not only on asking for it.** Your template must set a language
on `<html>`, have a title, give every image real alt text (an empty `alt` is not accepted, use a CSS
background for decoration), label inline SVG with `role="img"` + `aria-label`, keep headings in order
starting at `<h1>`, and use header cells in tables. The mechanical parts are handled for you: the role
map, link descriptions, the document language, marking running headers and footers as artifacts, and
the conformance declaration.

A non-conformant document is a **failed** document by default, so anything you receive with the claim
has been checked by the veraPDF reference validator. Use `"conformance": "attempt"` while adjusting a
template to get the document anyway with the violations listed. A large-print variant is the same
template and payload with `options.page.scale`, validated the same way.

Works alongside a digital signature: the conformance check runs on the signed document, so the
verdict covers the file you receive. Cannot be combined with a watermark, a PDF open-password,
PDF/A, an image format, or a `url` render.

## Other resources

```go
// Templates and proposals
tpls, _ := pw.Templates.List(ctx)
ver, _ := pw.Templates.Version(ctx, "tmpl_invoice", 3, "source")
prop, _ := pw.Templates.Proposals.Open(ctx, "tmpl_invoice", map[string]any{"fromDraft": true})
prop, _ = pw.Templates.Proposals.Promote(ctx, "tmpl_invoice", prop["id"].(string))

// Schemas
sch, _ := pw.Schemas.Get(ctx, "sch_invoice", 0 /* latest */)

// Usage
usage, _ := pw.Usage.Get(ctx)

// Review layer
thread, _ := pw.Comments.Create(ctx, map[string]any{"documentId": id, "anchor": anchor, "body": "typo"})
review, _ := pw.Reviews.Create(ctx, map[string]any{"documentId": id})
link, _ := pw.ShareLinks.Create(ctx, map[string]any{"documentId": id})

// Environments, pins, deployments (documents-as-code)
envs, _ := pw.Environments.List(ctx)
pin, _ := pw.Environments.SetPin(ctx, "production", "tmpl_invoice", 3)
plan, _ := pw.Deployments.Plan(ctx, manifestBody, "")
applied, _ := pw.Deployments.Apply(ctx, plan["id"].(string))
```

## Document lineage: trust, diff, versions, representations

```go
trust, _ := pw.Documents.Trust(ctx, "doc_1")       // one deterministic integrity + provenance manifest
diff, _ := pw.Documents.Diff(ctx, "doc_1", "doc_2") // causal diff between two documents; never renders or meters

// Reissue a template-pinned document under the same lineage (fires document.superseded):
appended, _ := pw.Documents.AppendVersion(ctx, "doc_1", map[string]any{"payload": map[string]any{"total": 51}})
versions, _ := pw.Documents.Versions(ctx, "doc_1")       // the full lineage, newest first
reps, _ := pw.Documents.Representations(ctx, "doc_1", 0) // every artifact of one version (PDF, e-invoice XML, JSON twin, ...)
```

## Typed business records (objects, object types, relationships)

Requires an API key with the matching scope — `objects:read` / `objects:write` / `object-types:manage` / `relationships:manage`; see [Scopes](#scopes).

```go
// Define a record type, then publish it (freezes an immutable version):
otype, _ := pw.ObjectTypes.Create(ctx, map[string]any{
    "key": "invoice", "nameSingular": "Invoice", "namePlural": "Invoices",
    "schema": map[string]any{"type": "object", "properties": map[string]any{"total": map[string]any{"type": "number"}}},
})
pw.ObjectTypes.Publish(ctx, otype["id"].(string), nil)

// Create + replace a record. Replace requires expectedVersion — a 409 on mismatch, never a lost update.
invoice, _ := pw.Objects.Create(ctx, map[string]any{"objectTypeKey": "invoice", "data": map[string]any{"total": 42}}, "")
pw.Objects.Replace(ctx, invoice["id"].(string), map[string]any{
    "data": map[string]any{"total": 51}, "expectedVersion": invoice["version"],
})

// Relate records, and file a rendered document against one:
pw.Objects.AddRelationship(ctx, invoice["id"].(string), map[string]any{
    "relationshipTypeKey": "billed_to", "targetObjectId": customerID,
})
pw.Objects.LinkDocument(ctx, invoice["id"].(string), map[string]any{"documentId": docID, "role": "primary"})
```

## Search, domain events, and the error registry

```go
results, _ := pw.Search.Query(ctx, pageweaver.SearchParams{Q: "acme invoice", SubjectType: "object"}) // requires search:read

// The append-only event ledger — resume from the returned cursor, not the last event you saw:
after, err := pw.Events.ListAll(ctx, pageweaver.ListEventsParams{Type: "document.completed"}, func(event pageweaver.Document) error {
    fmt.Println(event["type"], event["subjectId"])
    return nil
})

codes, _ := pw.ErrorCodes.List(ctx) // the full public error-code catalog (no API key required)
```

## Document ingestion and fillable PDFs

```go
// Bring in a PDF you already have (not a template render):
pw.Intake.Create(ctx, "", "", "internal", pageweaver.MultipartFile{Data: bytes, Filename: "scan.pdf"})

// Large files: a resumable chunked session.
session, _ := pw.Intake.Sessions.Create(ctx, map[string]any{"filename": "big.pdf", "totalBytes": totalBytes, "chunkSize": chunkSize})
pw.Intake.Sessions.UploadChunk(ctx, session["id"].(string), 0, chunk0)
pw.Intake.Sessions.Finalize(ctx, session["id"].(string))

// Fill an uploaded PDF's own AcroForm fields (not a Liquid template):
tpl, _ := pw.FormTemplates.Create(ctx, "Claim form", "", pageweaver.MultipartFile{Data: bytes, Filename: "claim.pdf"})
pw.FormTemplates.Fill(ctx, tpl["id"].(string), map[string]any{"payload": map[string]any{"claimant.fullName": "Ada Lovelace"}})
```

## Scopes

Every API key carries the baseline `read` + `render` scopes. Everything else is opt-in, set per key in the portal:

| Scope | Gates |
| --- | --- |
| `review` | Comments, reviews, share links |
| `deploy` | Environments, deployments, template proposals |
| `objects:read` / `objects:write` | Reading / writing typed business records |
| `objects:read-sensitive` | Decrypting a record's sensitive fields (stacks on `objects:read`) |
| `object-types:manage` | Defining and publishing object types |
| `relationships:manage` | Object relationships, and filing documents against objects |
| `documents:upload` | Document intake and fillable-form-template uploads |
| `search:read` | `pw.Search.Query()` |
| `workflows:read` | `pw.WorkflowDefinitions.*` |

A call missing a required scope fails with a `403` — a `*pageweaver.PermissionError` (see below).

## Retries

GET/HEAD/PUT/DELETE requests, and any POST sent with an idempotency key, are retried automatically on `429` and `5xx` with exponential backoff + jitter (honoring `Retry-After` on `429`). A plain POST with no idempotency key is never retried, since a duplicate render or record is worse than a failed request.

```go
pw := pageweaver.NewClient(apiKey,
    pageweaver.WithMaxRetries(3),
    pageweaver.WithRetryBaseDelay(500*time.Millisecond),
    pageweaver.WithRetryMaxDelay(8*time.Second),
)

// Disable retries entirely:
pw = pageweaver.NewClient(apiKey, pageweaver.WithMaxRetries(0))
```

## Errors

Non-2xx responses return a typed subtype of `*pageweaver.APIError` (aliased as `*pageweaver.Error` for
backward compatibility), selected by status, so you can catch the specific failure kind with
`errors.As` — or match `*pageweaver.APIError` to catch all of them. Every subtype promotes
`APIError`'s fields, so e.g. `ve.StatusCode` works directly on a `*pageweaver.ValidationError`.

| Type | Status | Thrown when |
| --- | --- | --- |
| `ValidationError` | 400 / 422 | The request body or query failed validation. `FieldErrors` carries field-level detail. |
| `AuthenticationError` | 401 | The API key is missing, invalid, or the account is suspended. |
| `PlanRequiredError` | 402 | A billing problem, not a credential one: the account's plan doesn't include this capability at all — no key, however scoped, can call it until the account upgrades. |
| `PermissionError` | 403 | A credential problem: the key authenticated fine but isn't allowed to do this. Check `err.IsScopeMissing()` / `err.RequiredScope()` when it's a missing scope. |
| `NotFoundError` | 404 | No such resource (or it belongs to another account). |
| `ConflictError` | 409 | An `expectedVersion`/`If-Match` mismatch, a duplicate key, or a state conflict. |
| `RateLimitError` | 429 | Rate limited or over a usage quota. `RetryAfterSeconds` when the API sent `Retry-After`. |
| `ServerError` | 5xx | The API failed unexpectedly. |
| `APIError` (`Error`) | any | The base type — every type above embeds it, and it also covers any other status. |
| `InvalidRequestError` | — | A client-side shape check failed before any request was sent (e.g. a blank id). |

```go
import (
    "errors"
    "fmt"

    "github.com/pageweaverio/pageweaver-go"
)

doc, err := pw.Documents.Create(ctx, map[string]any{"templateId": "t", "payload": payload}, "")
var ve *pageweaver.ValidationError
var rle *pageweaver.RateLimitError
var pre *pageweaver.PlanRequiredError
var pe *pageweaver.PermissionError
var apiErr *pageweaver.APIError
switch {
case errors.As(err, &ve):
    fmt.Println("validation failed:", ve.FieldErrors)
case errors.As(err, &rle):
    fmt.Println("rate limited, retry after", *rle.RetryAfterSeconds, "seconds")
case errors.As(err, &pre):
    // A billing problem: the account's plan doesn't include this feature at all.
    fmt.Println("upgrade required:", pre.Message)
case errors.As(err, &pe):
    // A credential problem: this specific API key isn't allowed to do this.
    if pe.IsScopeMissing() {
        fmt.Printf("mint a key with the '%s' scope\n", pe.RequiredScope())
    } else {
        fmt.Println("forbidden:", pe.Message)
    }
case errors.As(err, &apiErr):
    fmt.Println(apiErr.Code, apiErr.StatusCode, apiErr.RequestID)
case err != nil:
    panic(err)
}
```

Look up any `err.Code` in `pw.ErrorCodes.List(ctx)` for its cause and resolution. `PlanRequiredError` (402) and `PermissionError` (403) are easy to conflate — both read as "you can't do that" — but the fix differs: a plan error is resolved by the account upgrading, a scope error by minting a new API key with the missing scope. Branch on the type, not the status code.

## Migrating off living documents

The `/v1/living-documents/*` API surface has been retired and folded into ordinary documents (the
Go SDK never shipped a `LivingDocuments` service, so this is a note for anyone calling the old HTTP
routes directly, or migrating from another PageWeaver SDK):

- `POST /v1/living-documents` with `{templateId, payload, publicAlias}` → `pw.Documents.Create(ctx, map[string]any{"templateId": ..., "payload": ..., "publicAlias": true}, "")`. The minted link comes back as `result["alias"].(map[string]any)["token"]` instead of a separate identity.
- `POST /v1/living-documents/:id/versions` with `{payload}` (reissue) → `pw.Documents.AppendVersion(ctx, documentID, map[string]any{"payload": payload})`.
- `GET /v1/living-documents/:id` / list / `GET .../versions/:seq` → `pw.Documents.Versions(ctx, documentID)` / `pw.Documents.Version(ctx, documentID, seq)`.

## Webhooks

Verify webhook deliveries with the account signing secret. The signature header is
`sha256=<hex>` of the raw body under HMAC-SHA256.

```go
sig := r.Header.Get(pageweaver.WebhookSignatureHeader)
event, err := pageweaver.VerifyWebhook(secret, rawBody, sig)
if err != nil {
    http.Error(w, "bad signature", http.StatusBadRequest)
    return
}
fmt.Println(event["event"], event["documentId"])
```

`VerifySignature(secret, body, signature) bool` is the boolean check;
`SignWebhookBody(secret, body) string` produces the header value (mainly for tests).

## Releasing

Go has no package registry; a version is a git tag (`git tag v0.2.0 && git push origin v0.2.0`). Consumers then `go get github.com/pageweaverio/pageweaver-go@v0.2.0`. This release brings the SDK to parity with the live `/v1` API (mirroring the TS SDK's 0.2.0): new object-model, search, events, workflow-definitions, form-templates, and intake resources; document lineage (trust/diff/versions/representations); a typed error hierarchy; automatic retry with backoff; and client-side request validation.

## License

MIT
