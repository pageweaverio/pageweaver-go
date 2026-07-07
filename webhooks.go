package pageweaver

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
)

// Webhook header names. PageWeaver signs each delivery with HMAC-SHA256 over the exact request
// body, keyed by your account webhook secret, and sends it in the signature header formatted
// sha256=<hex>. Verify it on your endpoint before trusting the payload.
const (
	// WebhookSignatureHeader carries the sha256=<hex> signature.
	WebhookSignatureHeader = "x-pageweaver-signature"
	// WebhookEventHeader carries the event name.
	WebhookEventHeader = "x-pageweaver-event"
	// WebhookTimestampHeader carries the unix-seconds send time.
	WebhookTimestampHeader = "x-pageweaver-timestamp"
)

// ErrInvalidWebhookSignature is returned by VerifyWebhook when the signature does not match.
var ErrInvalidWebhookSignature = errors.New("pageweaver: invalid webhook signature")

// SignWebhookBody computes the sha256=<hex> signature for a raw body. Exposed mainly for tests.
func SignWebhookBody(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// VerifySignature reports whether signature is a valid sha256=<hex> HMAC of body under secret,
// using a constant-time comparison. It never panics.
func VerifySignature(secret string, body []byte, signature string) bool {
	if signature == "" {
		return false
	}
	expected := SignWebhookBody(secret, body)
	return hmac.Equal([]byte(expected), []byte(signature))
}

// VerifyWebhook verifies a webhook signature and returns the parsed event body. It returns
// ErrInvalidWebhookSignature if the signature is missing or wrong.
func VerifyWebhook(secret string, body []byte, signature string) (map[string]any, error) {
	if !VerifySignature(secret, body, signature) {
		return nil, ErrInvalidWebhookSignature
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload, nil
}
