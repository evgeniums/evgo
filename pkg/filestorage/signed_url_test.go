package filestorage

import (
	"context"
	"testing"
)

type testSignParams struct {
	values []string
}

func (p *testSignParams) Values() []string {
	return p.values
}

func newTestHandler(t *testing.T) *SignedUrlHandlerBase {
	t.Helper()
	h := NewSignedUrl()
	h.SECRET = "test-secret"
	h.EXPIRATION = 3600
	h.EXPIRY_PARAM = "e"
	h.SIGNATURE_PARAM = "s"
	return h
}

// Regression test for the bug where SignUrl computed the HMAC before the expiry was
// written into the URL's RawQuery, and calcHmac read the literal query key "expiry"
// instead of EXPIRY_PARAM ("e"). Both together meant SignUrl always signed an empty
// expiry while CheckUrl verified against the real one, so every signed URL with a
// non-zero EXPIRATION failed CheckUrl.
func TestSignedUrlRoundTrip(t *testing.T) {

	ctx := context.Background()
	h := newTestHandler(t)
	params := &testSignParams{values: []string{"PUT", "12345"}}

	signed, err := h.SignUrl(ctx, "https://example.com/upload/part1", params)
	if err != nil {
		t.Fatalf("SignUrl failed: %v", err)
	}

	if err := h.CheckUrlString(ctx, signed, params); err != nil {
		t.Fatalf("CheckUrl failed for a freshly signed URL: %v", err)
	}
}

func TestSignedUrlRejectsTamperedParameters(t *testing.T) {

	ctx := context.Background()
	h := newTestHandler(t)
	signed, err := h.SignUrl(ctx, "https://example.com/upload/part1", &testSignParams{values: []string{"PUT", "12345"}})
	if err != nil {
		t.Fatalf("SignUrl failed: %v", err)
	}

	// same URL, but checked against a different content-length: must fail
	if err := h.CheckUrlString(ctx, signed, &testSignParams{values: []string{"PUT", "99999"}}); err == nil {
		t.Fatalf("expected CheckUrl to reject a signature computed over different parameters")
	}
}

func TestSignedUrlRejectsExpired(t *testing.T) {

	ctx := context.Background()
	h := newTestHandler(t)
	h.EXPIRATION = 0 // build the URL manually with an already-expired timestamp

	params := &testSignParams{values: []string{"PUT", "12345"}}
	signed, err := h.SignUrl(ctx, "https://example.com/upload/part1?e=1", params)
	if err != nil {
		t.Fatalf("SignUrl failed: %v", err)
	}

	if err := h.CheckUrlString(ctx, signed, params); err == nil {
		t.Fatalf("expected CheckUrl to reject an expired URL")
	}
}
