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
