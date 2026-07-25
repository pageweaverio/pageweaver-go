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
`Comments`, `Reviews`, `ShareLinks`, `Environments`, `Deployments`.

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

It cannot be combined with an image format, a PDF open-password, a digital signature, or a `url`
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

## Errors

Non-2xx responses return a `*pageweaver.Error` carrying `StatusCode`, `Body`, and (when the JSON
body has them) `Message` and `Code`.

```go
doc, err := pw.Documents.Get(ctx, "doc_missing")
var apiErr *pageweaver.Error
if errors.As(err, &apiErr) {
    fmt.Println(apiErr.StatusCode, apiErr.Message)
}
```

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

Go has no package registry; a version is a git tag (`git tag v0.1.0 && git push origin v0.1.0`). Consumers then `go get github.com/pageweaverio/pageweaver-go@v0.1.0`.

## License

MIT
