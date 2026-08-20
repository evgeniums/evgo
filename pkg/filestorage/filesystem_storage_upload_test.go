package filestorage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"

	"github.com/evgeniums/evgo/pkg/utils"
)

// newTestStorage builds a FilesystemStorage wired with a real UploadPartHelperConfig (not the
// always-whole-file testUploadPartHelper stub from url_manager_test.go), so part boundaries are
// computed the same way production code computes them. Init() is deliberately not exercised here
// (that would require a full app_context + config asset) - fields are set directly, matching the
// convention already used by TestTempPathAndPathNoTenancyDoesNotPanic.
func newTestStorage(t *testing.T, partLength int64) *FilesystemStorage {
	t.Helper()
	f := &FilesystemStorage{}
	f.BASE_DIR = t.TempDir()
	f.TEMP_DIR = t.TempDir()
	f.TENANCY_SUBFOLDER = "tenancy"
	f.TOPIC_SUBFOLDER = "topic"
	f.helper = &UploadPartHelperConfig{UPLOAD_PART_LENGTH: partLength}
	return f
}

func newTestInfo(size int64) *testFileInfo {
	info := &testFileInfo{size: size}
	info.InitObject()
	return info
}

func uploadAllParts(t *testing.T, f *FilesystemStorage, ctx context.Context, info FileInfo, data []byte, partLength int64) int64 {
	t.Helper()
	count := f.helper.PartCount(info)
	for i := int64(0); i < count; i++ {
		start := i * partLength
		end := start + f.helper.UploadPartLength(info, i)
		if err := f.UploadPart(ctx, info, bytes.NewReader(data[start:end]), i); err != nil {
			t.Fatalf("UploadPart(%d) failed: %v", i, err)
		}
	}
	return count
}

// TestUploadPartFinalizeFetchRoundTrip exercises the whole chunked-upload lifecycle end to end:
// StartUpload, three UploadPart calls spanning a partial last part, FinalizeUpload concatenating
// them in order, then Fetch reading the result back - the exact sequence FileDataControllerBase
// drives in production but that no existing test in this package covers.
func TestUploadPartFinalizeFetchRoundTrip(t *testing.T) {
	ctx := context.Background()
	const partLength = 4
	data := []byte("0123456789") // 10 bytes -> parts "0123", "4567", "89"

	f := newTestStorage(t, partLength)
	info := newTestInfo(int64(len(data)))

	if err := f.StartUpload(ctx, info); err != nil {
		t.Fatalf("StartUpload failed: %v", err)
	}
	if _, err := os.Stat(f.TempPath(ctx, info)); err != nil {
		t.Fatalf("StartUpload did not create the temp directory: %v", err)
	}

	count := uploadAllParts(t, f, ctx, info, data, partLength)
	if count != 3 {
		t.Fatalf("expected 3 parts for a 10-byte file with part length 4, got %d", count)
	}

	if err := f.FinalizeUpload(ctx, info, count); err != nil {
		t.Fatalf("FinalizeUpload failed: %v", err)
	}

	// the temp directory for this file must be gone after finalization
	if _, err := os.Stat(f.TempPath(ctx, info)); !os.IsNotExist(err) {
		t.Fatalf("expected temp dir to be removed after FinalizeUpload, stat err = %v", err)
	}

	rc, err := f.Fetch(ctx, info)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("reading fetched data failed: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("fetched data = %q, want %q", got, data)
	}
}

func TestFetchWithOffset(t *testing.T) {
	ctx := context.Background()
	const partLength = 100 // single part, whole file in one UploadPart call
	data := []byte("hello world")

	f := newTestStorage(t, partLength)
	info := newTestInfo(int64(len(data)))

	uploadAllParts(t, f, ctx, info, data, partLength)
	if err := f.FinalizeUpload(ctx, info, 1); err != nil {
		t.Fatalf("FinalizeUpload failed: %v", err)
	}

	rc, err := f.Fetch(ctx, info, 6)
	if err != nil {
		t.Fatalf("Fetch with offset failed: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("reading fetched data failed: %v", err)
	}
	if string(got) != "world" {
		t.Fatalf("fetched data with offset 6 = %q, want %q", got, "world")
	}
}

func TestFetchRangeLimitsReadLength(t *testing.T) {
	ctx := context.Background()
	const partLength = 100
	data := []byte("0123456789ABCDEF")

	f := newTestStorage(t, partLength)
	info := newTestInfo(int64(len(data)))

	uploadAllParts(t, f, ctx, info, data, partLength)
	if err := f.FinalizeUpload(ctx, info, 1); err != nil {
		t.Fatalf("FinalizeUpload failed: %v", err)
	}

	rc, err := f.FetchRange(ctx, info, 2, 5)
	if err != nil {
		t.Fatalf("FetchRange failed: %v", err)
	}
	defer rc.Close()

	// read well past the requested length: the limit reader must stop at 5 bytes
	buf := make([]byte, 100)
	n, err := io.ReadFull(rc, buf)
	if err != nil && err != io.ErrUnexpectedEOF {
		t.Fatalf("unexpected error reading range: %v", err)
	}
	if n != 5 {
		t.Fatalf("FetchRange(2,5) yielded %d bytes, want 5", n)
	}
	if string(buf[:5]) != "23456" {
		t.Fatalf("FetchRange(2,5) = %q, want %q", buf[:5], "23456")
	}
}

func TestFetchNonexistentFile(t *testing.T) {
	ctx := context.Background()
	f := newTestStorage(t, 4)
	info := newTestInfo(10)

	if _, err := f.Fetch(ctx, info); err == nil {
		t.Fatal("expected Fetch to fail for a file that was never finalized")
	}
}

// TestUploadPartRejectsShortSource locks down the under-length guard: io.CopyN surfaces io.EOF
// when the source has fewer bytes than the part length the helper computed, and UploadPart must
// propagate that as a failure rather than silently writing a short part.
func TestUploadPartRejectsShortSource(t *testing.T) {
	ctx := context.Background()
	f := newTestStorage(t, 10)
	info := newTestInfo(10) // helper will demand a 10-byte part

	err := f.UploadPart(ctx, info, bytes.NewReader([]byte("short")), 0)
	if err == nil {
		t.Fatal("expected UploadPart to fail when the source is shorter than the part length")
	}
	if !errors.Is(err, io.EOF) {
		t.Fatalf("expected the error to be io.EOF (from io.CopyN), got %v", err)
	}

	// the scratch file must not have been left behind, and the real part path must not exist
	if _, statErr := os.Stat(f.TempPath(ctx, info, 0)); !os.IsNotExist(statErr) {
		t.Fatalf("expected part path to not exist after a failed upload, stat err = %v", statErr)
	}
}

// TestUploadPartRejectsOversizedSource locks down the "(n+1)th byte" guard in UploadPart: a
// source with more bytes than the part length must be rejected even though the first
// partLength bytes alone would have copied successfully.
func TestUploadPartRejectsOversizedSource(t *testing.T) {
	ctx := context.Background()
	f := newTestStorage(t, 4)
	info := newTestInfo(4)

	err := f.UploadPart(ctx, info, bytes.NewReader([]byte("12345")), 0)
	if err == nil {
		t.Fatal("expected UploadPart to fail when the source exceeds the part length")
	}
	if err.Error() != "source exceeds part size" {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Stat(f.TempPath(ctx, info, 0)); !os.IsNotExist(statErr) {
		t.Fatalf("expected part path to not exist after a failed upload, stat err = %v", statErr)
	}
}

func TestUploadPartExactLengthSucceeds(t *testing.T) {
	ctx := context.Background()
	f := newTestStorage(t, 4)
	info := newTestInfo(4)

	if err := f.UploadPart(ctx, info, bytes.NewReader([]byte("1234")), 0); err != nil {
		t.Fatalf("UploadPart with an exact-length source failed: %v", err)
	}
	if _, err := os.Stat(f.TempPath(ctx, info, 0)); err != nil {
		t.Fatalf("expected part file to exist after a successful upload: %v", err)
	}
}

// TestFinalizeUploadMissingPartFails asserts that finalizing with a partsCount larger than the
// number of parts actually uploaded fails cleanly (os.Open ENOENT) and leaves no target file
// behind, rather than silently producing a truncated file.
func TestFinalizeUploadMissingPartFails(t *testing.T) {
	ctx := context.Background()
	f := newTestStorage(t, 4)
	info := newTestInfo(8)

	if err := f.UploadPart(ctx, info, bytes.NewReader([]byte("1234")), 0); err != nil {
		t.Fatalf("UploadPart(0) failed: %v", err)
	}
	// part 1 is deliberately never uploaded

	err := f.FinalizeUpload(ctx, info, 2)
	if err == nil {
		t.Fatal("expected FinalizeUpload to fail when a required part is missing")
	}
	if _, statErr := os.Stat(f.Path(ctx, info)); !os.IsNotExist(statErr) {
		t.Fatalf("expected target file to not exist after a failed finalize, stat err = %v", statErr)
	}
}

// TestFinalizeUploadIgnoresExtraParts documents the documented-by-code default: FinalizeUpload's
// partsCount defaults to 1 when omitted, and parts at/after that count are simply never read (and
// then discarded with the rest of the temp dir), not validated against info.GetSize().
func TestFinalizeUploadIgnoresExtraParts(t *testing.T) {
	ctx := context.Background()
	f := newTestStorage(t, 4)
	info := newTestInfo(8)

	if err := f.UploadPart(ctx, info, bytes.NewReader([]byte("1234")), 0); err != nil {
		t.Fatalf("UploadPart(0) failed: %v", err)
	}
	if err := f.UploadPart(ctx, info, bytes.NewReader([]byte("5678")), 1); err != nil {
		t.Fatalf("UploadPart(1) failed: %v", err)
	}

	// omit partsCount entirely -> defaults to 1, part 1 is never touched
	if err := f.FinalizeUpload(ctx, info); err != nil {
		t.Fatalf("FinalizeUpload failed: %v", err)
	}

	rc, err := f.Fetch(ctx, info)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	defer rc.Close()
	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("reading fetched data failed: %v", err)
	}
	if string(got) != "1234" {
		t.Fatalf("finalized content = %q, want %q (only part 0)", got, "1234")
	}
}

func TestDeleteFileIsIdempotent(t *testing.T) {
	ctx := context.Background()
	f := newTestStorage(t, 100)
	info := newTestInfo(5)

	if err := f.UploadPart(ctx, info, bytes.NewReader([]byte("hello")), 0); err != nil {
		t.Fatalf("UploadPart failed: %v", err)
	}
	if err := f.FinalizeUpload(ctx, info, 1); err != nil {
		t.Fatalf("FinalizeUpload failed: %v", err)
	}

	if err := f.DeleteFile(ctx, info); err != nil {
		t.Fatalf("first DeleteFile failed: %v", err)
	}
	if _, err := os.Stat(f.Path(ctx, info)); !os.IsNotExist(err) {
		t.Fatalf("expected file to be gone after DeleteFile, stat err = %v", err)
	}
	// calling it again on an already-deleted (or never-finalized) file must not error -
	// this is what lets the TTL expiry runner delete abandoned uploads safely
	if err := f.DeleteFile(ctx, info); err != nil {
		t.Fatalf("second DeleteFile on an already-deleted file returned an error: %v", err)
	}
}

func TestDeletePathPrefixRemovesSubtree(t *testing.T) {
	ctx := context.Background()
	f := newTestStorage(t, 100)
	info := newTestInfo(5)

	if err := f.UploadPart(ctx, info, bytes.NewReader([]byte("hello")), 0); err != nil {
		t.Fatalf("UploadPart failed: %v", err)
	}
	if err := f.FinalizeUpload(ctx, info, 1); err != nil {
		t.Fatalf("FinalizeUpload failed: %v", err)
	}

	if err := f.Delete(ctx, "."); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if _, err := os.Stat(f.Path(ctx, info)); !os.IsNotExist(err) {
		t.Fatalf("expected file to be gone after Delete(BASE_DIR), stat err = %v", err)
	}
}

// TestDeleteTempRemovesOnlyOlderDates locks down DeleteTemp's string-comparison-based date cutoff
// (entry.Name() < toDate.AsNumber()): it relies on the zero-padded 8-digit AsNumber() encoding to
// make lexicographic and numeric ordering agree.
func TestDeleteTempRemovesOnlyOlderDates(t *testing.T) {
	ctx := context.Background()
	f := newTestStorage(t, 100)

	older := []string{"20250101", "20250601"}
	kept := []string{"20260101", "20260615"}
	for _, name := range append(append([]string{}, older...), kept...) {
		if err := os.MkdirAll(f.TEMP_DIR+"/"+name, 0755); err != nil {
			t.Fatalf("failed to seed temp dir %s: %v", name, err)
		}
	}
	_ = ctx

	var cutoff utils.Date
	cutoff.Set(2026, 1, 1)
	if err := f.DeleteTemp(context.Background(), cutoff); err != nil {
		t.Fatalf("DeleteTemp failed: %v", err)
	}

	for _, name := range older {
		if _, err := os.Stat(f.TEMP_DIR + "/" + name); !os.IsNotExist(err) {
			t.Fatalf("expected %s to be removed by DeleteTemp, stat err = %v", name, err)
		}
	}
	for _, name := range kept {
		if _, err := os.Stat(f.TEMP_DIR + "/" + name); err != nil {
			t.Fatalf("expected %s to survive DeleteTemp, stat err = %v", name, err)
		}
	}
}

func TestPathAndTempPathIncludeTopicSubfolder(t *testing.T) {
	ctx := context.Background()
	f := newTestStorage(t, 100)
	info := newTestInfo(5)
	info.topic = "topic1"

	path := f.Path(ctx, info)
	if !bytes.Contains([]byte(path), []byte("/topic/topic1/")) {
		t.Fatalf("Path() = %q, want it to contain a /topic/topic1/ segment", path)
	}

	tempPath := f.TempPath(ctx, info, 0)
	if !bytes.Contains([]byte(tempPath), []byte("/topic/topic1/")) {
		t.Fatalf("TempPath() = %q, want it to contain a /topic/topic1/ segment", tempPath)
	}
}
