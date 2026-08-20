package filestorage

import (
	"context"
	"net/url"
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

// TestSignedUrlRejectsTamperedPath locks down that the URL path itself participates in the HMAC
// (calcHmac/checkHmac both fold in u.Path) - swapping the path after signing must invalidate the
// signature, even though the query string (where the signature lives) is untouched.
func TestSignedUrlRejectsTamperedPath(t *testing.T) {
	ctx := context.Background()
	h := newTestHandler(t)
	params := &testSignParams{values: []string{"PUT", "12345"}}

	signed, err := h.SignUrl(ctx, "https://example.com/upload/part1", params)
	if err != nil {
		t.Fatalf("SignUrl failed: %v", err)
	}

	u, err := url.Parse(signed)
	if err != nil {
		t.Fatalf("failed to parse signed URL: %v", err)
	}
	u.Path = "/upload/part2"

	if err := h.CheckUrl(ctx, u, params); err == nil {
		t.Fatalf("expected CheckUrl to reject a signed URL whose path was changed after signing")
	}
}

// TestSignedUrlNoExpirationParamNeverExpires documents that EXPIRATION=0 means SignUrl never
// writes an expiry parameter at all (see the `if s.EXPIRATION != 0` guard), so CheckUrl's
// expiry check - gated on the parameter being present - never rejects such a URL for being old.
func TestSignedUrlNoExpirationParamNeverExpires(t *testing.T) {
	ctx := context.Background()
	h := newTestHandler(t)
	h.EXPIRATION = 0

	params := &testSignParams{values: []string{"GET"}}
	signed, err := h.SignUrl(ctx, "https://example.com/fetch/id1", params)
	if err != nil {
		t.Fatalf("SignUrl failed: %v", err)
	}
	parsed, err := url.Parse(signed)
	if err != nil {
		t.Fatalf("failed to parse signed URL: %v", err)
	}
	if parsed.Query().Get(h.EXPIRY_PARAM) != "" {
		t.Fatalf("expected no expiry parameter in a URL signed with EXPIRATION=0, got %s", signed)
	}
	if err := h.CheckUrlString(ctx, signed, params); err != nil {
		t.Fatalf("expected a non-expiring signed URL to always pass CheckUrl, got %v", err)
	}
}
