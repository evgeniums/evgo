package filestorage

import (
	"context"
	"strings"
	"testing"

	"github.com/evgeniums/evgo/pkg/common"
)

type testFileInfo struct {
	common.ObjectBase
	size           int64
	topic          string
	uploadPartSize int64
}

func (f *testFileInfo) GetContentType() string   { return "application/octet-stream" }
func (f *testFileInfo) GetSize() int64           { return f.size }
func (f *testFileInfo) GetFileName() string      { return "test.bin" }
func (f *testFileInfo) GetTopic() string         { return f.topic }
func (f *testFileInfo) GetNativeId() string      { return "" }
func (f *testFileInfo) GetUploadPartSize() int64 { return f.uploadPartSize }

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

type testUploadPartHelper struct{}

func (h *testUploadPartHelper) UploadPartLength(info FileInfo, partIndex ...int64) int64 {
	return info.GetSize()
}
func (h *testUploadPartHelper) PartCount(info FileInfo) int64 { return 1 }

func newTestUploadUrlManager(t *testing.T, h *SignedUrlHandlerBase, enableTopic bool) *UrlManagerBase {
	t.Helper()
	u := newTestUrlManager(t, h)
	u.BASE_UPLOAD_URL = "https://example.com"
	u.UPLOAD_PATH_PREFIX = "/filedata/upload"
	u.UPLOAD_METHOD = "POST"
	u.PART_PARAMETER = "part"
	u.ENABLE_TOPIC = enableTopic
	u.helper = &testUploadPartHelper{}
	return u
}

// Task debug-sending-files-to-optimized-music (server stage 2, C): regression
// test for the topic-gating asymmetry - URL generation used to add a /topic/
// segment whenever info.GetTopic()!="" regardless of ENABLE_TOPIC, while the
// route itself (filedata_service.go) only ever had that segment when
// ENABLE_TOPIC was true. Whenever a caller supplied a topic with
// ENABLE_TOPIC left at its default (false), the client was handed a URL the
// server had never registered a route for - the exact 404 this whole task
// started from.
func TestTopicSegmentGatedByEnableTopic(t *testing.T) {

	ctx := context.Background()
	info := &testFileInfo{size: 100, topic: "topic1"}
	info.InitObject()

	cases := []struct {
		name        string
		enableTopic bool
		wantTopic   bool
	}{
		{"enabled_with_topic", true, true},
		{"disabled_with_topic", false, false},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newTestHandler(t)

			uu := newTestUploadUrlManager(t, h, c.enableTopic)
			uploadResp, err := uu.GetUploadUrls(ctx, info)
			if err != nil {
				t.Fatalf("GetUploadUrls failed: %v", err)
			}
			if len(uploadResp.Urls) != 1 {
				t.Fatalf("expected exactly 1 upload url, got %d", len(uploadResp.Urls))
			}
			gotUpload := strings.Contains(uploadResp.Urls[0], "/topic/topic1/")
			if gotUpload != c.wantTopic {
				t.Fatalf("upload url topic segment presence = %v, want %v (url: %s)", gotUpload, c.wantTopic, uploadResp.Urls[0])
			}

			du := newTestUrlManager(t, h)
			du.ENABLE_TOPIC = c.enableTopic
			downloadResp, err := du.GetDownloadUrl(ctx, info)
			if err != nil {
				t.Fatalf("GetDownloadUrl failed: %v", err)
			}
			gotDownload := strings.Contains(downloadResp.Url, "/topic/topic1/")
			if gotDownload != c.wantTopic {
				t.Fatalf("download url topic segment presence = %v, want %v (url: %s)", gotDownload, c.wantTopic, downloadResp.Url)
			}
		})
	}
}

// TestGetUploadUrlsPagination locks down GetUploadUrlsOptions{FromPart,MaxCount}: the returned
// Urls slice is windowed to [FromPart, FromPart+MaxCount), but TotalUrlCount always reports the
// full part count for the file regardless of the window.
func TestGetUploadUrlsPagination(t *testing.T) {
	ctx := context.Background()
	h := newTestHandler(t)
	// testUploadPartHelper.PartCount always returns 1, so use the real helper here to get more
	// than one part to paginate over.
	u := newTestUploadUrlManager(t, h, false)
	u.helper = &UploadPartHelperConfig{UPLOAD_PART_LENGTH: 10}

	info := &testFileInfo{size: 35} // 4 parts of size 10,10,10,5
	info.InitObject()

	resp, err := u.GetUploadUrls(ctx, info, GetUploadUrlsOptions{FromPart: 1, MaxCount: 2})
	if err != nil {
		t.Fatalf("GetUploadUrls failed: %v", err)
	}
	if resp.TotalUrlCount != 4 {
		t.Fatalf("TotalUrlCount = %d, want 4 (full part count, independent of the window)", resp.TotalUrlCount)
	}
	if resp.FromPartIndex != 1 {
		t.Fatalf("FromPartIndex = %d, want 1", resp.FromPartIndex)
	}
	if len(resp.Urls) != 2 {
		t.Fatalf("len(Urls) = %d, want 2 (the requested window)", len(resp.Urls))
	}
	if !strings.Contains(resp.Urls[0], "/part/1") {
		t.Fatalf("first windowed URL should target part 1: %s", resp.Urls[0])
	}
	if !strings.Contains(resp.Urls[1], "/part/2") {
		t.Fatalf("second windowed URL should target part 2: %s", resp.Urls[1])
	}
}

// TestGetUploadUrlsWindowClampedToTotal ensures a MaxCount that overruns the file's real part
// count is clamped rather than producing out-of-range part indexes.
func TestGetUploadUrlsWindowClampedToTotal(t *testing.T) {
	ctx := context.Background()
	h := newTestHandler(t)
	u := newTestUploadUrlManager(t, h, false)
	u.helper = &UploadPartHelperConfig{UPLOAD_PART_LENGTH: 10}

	info := &testFileInfo{size: 25} // 3 parts: 10,10,5
	info.InitObject()

	resp, err := u.GetUploadUrls(ctx, info, GetUploadUrlsOptions{FromPart: 2, MaxCount: 10})
	if err != nil {
		t.Fatalf("GetUploadUrls failed: %v", err)
	}
	if len(resp.Urls) != 1 {
		t.Fatalf("len(Urls) = %d, want 1 (only part 2 remains, clamped to TotalUrlCount=3)", len(resp.Urls))
	}
}

// TestUrlManagerPublishesTlsPolicy confirms the TLS-policy config fields flow through verbatim
// into both UploadUrlInfo and DownloadUrlInfo - these are trusted client-facing signals with no
// other validation between config and wire type.
func TestUrlManagerPublishesTlsPolicy(t *testing.T) {
	ctx := context.Background()
	h := newTestHandler(t)

	uu := newTestUploadUrlManager(t, h, false)
	uu.USE_SYSTEM_CA = true
	uu.SKIP_HOST_NAME_VERIFICATION = true

	info := &testFileInfo{size: 10}
	info.InitObject()

	uploadResp, err := uu.GetUploadUrls(ctx, info)
	if err != nil {
		t.Fatalf("GetUploadUrls failed: %v", err)
	}
	if !uploadResp.UseSystemCa || !uploadResp.SkipHostNameVerification {
		t.Fatalf("expected TLS policy flags to propagate into UploadUrlInfo, got %+v", uploadResp)
	}

	du := newTestUrlManager(t, h)
	du.USE_SYSTEM_CA = true
	du.SKIP_HOST_NAME_VERIFICATION = true
	downloadResp, err := du.GetDownloadUrl(ctx, info)
	if err != nil {
		t.Fatalf("GetDownloadUrl failed: %v", err)
	}
	if !downloadResp.UseSystemCa || !downloadResp.SkipHostNameVerification {
		t.Fatalf("expected TLS policy flags to propagate into DownloadUrlInfo, got %+v", downloadResp)
	}
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
