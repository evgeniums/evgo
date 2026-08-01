package filestorage

import (
	"context"
	"testing"

	"github.com/evgeniums/evgo/pkg/common"
)

type testFileInfo struct {
	common.ObjectBase
	size  int64
	topic string
}

func (f *testFileInfo) GetContentType() string   { return "application/octet-stream" }
func (f *testFileInfo) GetSize() int64           { return f.size }
func (f *testFileInfo) GetFileName() string      { return "test.bin" }
func (f *testFileInfo) GetTopic() string         { return f.topic }
func (f *testFileInfo) GetNativeId() string      { return "" }
func (f *testFileInfo) GetUploadPartSize() int64 { return 0 }

func newTestUrlManager(t *testing.T, h *SignedUrlHandlerBase) *UrlManagerBase {
	t.Helper()
	u := NewUrlManager()
	u.BASE_DOWNLOAD_URL = "https://example.com"
	u.DOWNLOAD_PATH_PREFIX = "/filedata/fetch"
	u.DOWNLOAD_METHOD = "GET"
	u.ID_PARAMETER = "id"
	u.TOPIC_PARAMETER = "topic"
	u.TENANCY_PARAMETER = "tenancy"
	u.SHADOW_TENANCY_PATH = true
	u.signedUrlHandler = h
	return u
}

// Regression test for the second signed-download-URL bug (url_manager.go's
// GetDownloadUrl): it used to sign with SignUrlValues{Method,ContentLength:size},
// but FileDataControllerBase.checkUrl (filedata_service/filedata_controller.go)
// only ever sets ContentLength for uploads (`if !download {...}`) - for downloads it
// checks against SignUrlValues{Method} alone. The two value sets never matched
// whenever EXPIRATION!=0 (the shipped default), so every signed download URL failed
// verification unconditionally. Found while implementing the files2 client download
// queue (task 5), independent of and in addition to the HMAC/RawQuery-ordering bug
// already covered by signed_url_test.go.
func TestGetDownloadUrlMatchesCheckUrl(t *testing.T) {

	ctx := context.Background()
	h := newTestHandler(t)
	u := newTestUrlManager(t, h)

	info := &testFileInfo{size: 12345, topic: "topic1"}
	info.InitObject()

	resp, err := u.GetDownloadUrl(ctx, info)
	if err != nil {
		t.Fatalf("GetDownloadUrl failed: %v", err)
	}

	// This is exactly the SignUrlValues checkUrl builds for a download request
	// (filedata_controller.go's checkUrl: `v := filestorage.SignUrlValues{Method:
	// r.GetRequestMethod()}`, ContentLength left unset since `download` is true).
	downloadCheckValues := &SignUrlValues{Method: u.DOWNLOAD_METHOD}
	if err := h.CheckUrlString(ctx, resp.Url, downloadCheckValues); err != nil {
		t.Fatalf("a GetDownloadUrl-signed URL failed CheckUrl with checkUrl's own download value set: %v", err)
	}

	// Guard against a vacuous test: confirm ContentLength genuinely participates in
	// the signature, i.e. checking with it included (the pre-fix value set) is
	// rejected - otherwise this test would pass even if SignUrl still signed over
	// ContentLength internally.
	uploadShapedValues := &SignUrlValues{Method: u.DOWNLOAD_METHOD, ContentLength: "12345"}
	if err := h.CheckUrlString(ctx, resp.Url, uploadShapedValues); err == nil {
		t.Fatalf("expected CheckUrl to reject a download URL checked with an upload-shaped (ContentLength-included) value set")
	}
}
