# PageWeaver Go SDK

Official Go client for the [PageWeaver](https://pageweaver.io) PDF generation API. Standard library only, no dependencies.

## Install

```bash
go get github.com/pageweaverio/pageweaver-go
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"time"

	pageweaver "github.com/pageweaverio/pageweaver-go"
)

func main() {
	pw := pageweaver.NewClient("pk_live_...")

	doc, err := pw.CreateAndWait(context.Background(), map[string]any{
		"templateId": "tmpl_invoice",
		"payload":    map[string]any{"number": "INV-001", "total": 4200},
	}, time.Second, 60*time.Second)
	if err != nil {
		panic(err)
	}
	fmt.Println(doc["status"]) // "done"
}
```

Non-2xx responses return a `*pageweaver.Error` carrying `StatusCode` and `Body`.

## Releasing

Go has no package registry; a version is a git tag (`git tag v0.1.0 && git push origin v0.1.0`). Consumers then `go get github.com/pageweaverio/pageweaver-go@v0.1.0`.

## License

MIT
