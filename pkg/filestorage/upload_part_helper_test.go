package filestorage

import "testing"

func TestFilePartLength(t *testing.T) {
	cases := []struct {
		name        string
		totalSize   int64
		maxPartSize int64
		partIndex   []int64
		want        int64
	}{
		{"no_index_returns_total_size", 100, 30, nil, 100},
		{"first_full_part", 100, 30, []int64{0}, 30},
		{"middle_full_part", 100, 30, []int64{1}, 30},
		{"last_partial_part", 100, 30, []int64{3}, 10}, // 100 - 3*30 = 10
		{"exact_multiple_last_part_is_full", 90, 30, []int64{2}, 30},
		{"index_at_exact_boundary_is_empty", 90, 30, []int64{3}, 0}, // start == totalSize
		{"index_past_end_is_empty", 100, 30, []int64{10}, 0},
		{"single_part_file", 10, 30, []int64{0}, 10},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := FilePartLength(c.totalSize, c.maxPartSize, c.partIndex...)
			if got != c.want {
				t.Fatalf("FilePartLength(%d, %d, %v) = %d, want %d", c.totalSize, c.maxPartSize, c.partIndex, got, c.want)
			}
		})
	}
}

func TestUploadPartHelperUploadPartLength(t *testing.T) {
	h := &UploadPartHelperConfig{UPLOAD_PART_LENGTH: 30}

	t.Run("no_index_returns_whole_size", func(t *testing.T) {
		info := newTestInfo(100)
		if got := h.UploadPartLength(info); got != 100 {
			t.Fatalf("UploadPartLength(no index) = %d, want 100", got)
		}
	})

	t.Run("uses_helper_default_when_info_has_no_override", func(t *testing.T) {
		info := newTestInfo(100)
		if got := h.UploadPartLength(info, 0); got != 30 {
			t.Fatalf("UploadPartLength(0) = %d, want 30 (helper default)", got)
		}
	})

	t.Run("info_override_takes_precedence_over_helper_default", func(t *testing.T) {
		info := newTestInfo(100)
		info.uploadPartSize = 10
		if got := h.UploadPartLength(info, 0); got != 10 {
			t.Fatalf("UploadPartLength(0) = %d, want 10 (info override)", got)
		}
		if got := h.UploadPartLength(info, 9); got != 10 {
			t.Fatalf("UploadPartLength(9) = %d, want 10 (last full part)", got)
		}
	})
}

func TestUploadPartHelperPartCount(t *testing.T) {
	cases := []struct {
		name           string
		partLength     int64
		size           int64
		uploadPartSize int64
		want           int64
	}{
		{"exact_division", 30, 90, 0, 3},
		{"with_remainder", 30, 100, 0, 4},
		{"zero_size", 30, 0, 0, 0},
		{"negative_size", 30, -1, 0, 0},
		{"zero_helper_default_and_no_override", 0, 100, 0, 0},
		{"info_override_wins_over_helper_default", 30, 100, 25, 4}, // ceil(100/25)
		{"single_byte_file", 30, 1, 0, 1},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := &UploadPartHelperConfig{UPLOAD_PART_LENGTH: c.partLength}
			info := newTestInfo(c.size)
			info.uploadPartSize = c.uploadPartSize
			got := h.PartCount(info)
			if got != c.want {
				t.Fatalf("PartCount(size=%d, uploadPartSize=%d, helperDefault=%d) = %d, want %d",
					c.size, c.uploadPartSize, c.partLength, got, c.want)
			}
		})
	}
}
