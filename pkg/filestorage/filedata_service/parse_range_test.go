package filedata_service

import "testing"

func TestParseRange(t *testing.T) {
	const totalSize = 1000

	cases := []struct {
		name       string
		header     string
		wantOffset int64
		wantLength int64
		wantErr    bool
	}{
		{"specific_range", "bytes=0-499", 0, 500, false},
		{"from_offset_to_end", "bytes=500-", 500, 500, false},
		{"suffix_range", "bytes=-200", 800, 200, false},
		{"suffix_larger_than_total_is_clamped_to_whole_file", "bytes=-2000", 0, 1000, false},
		{"end_beyond_total_is_clamped", "bytes=900-1500", 900, 100, false},
		{"single_byte_range", "bytes=0-0", 0, 1, false},
		{"last_byte_range", "bytes=999-999", 999, 1, false},

		{"missing_bytes_prefix", "0-499", 0, 0, true},
		{"empty_spec", "bytes=", 0, 0, true},
		{"empty_spec_dash_only", "bytes=-", 0, 0, true},
		{"multi_range_unsupported", "bytes=0-1,5-6", 0, 0, true},
		{"offset_at_total_size_yields_empty_range", "bytes=1000-1999", 0, 0, true},
		{"offset_past_total_size", "bytes=1500-1999", 0, 0, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			offset, length, err := ParseRange(c.header, totalSize)
			if c.wantErr {
				if err == nil {
					t.Fatalf("ParseRange(%q) succeeded (offset=%d, length=%d), want an error", c.header, offset, length)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseRange(%q) failed: %v", c.header, err)
			}
			if offset != c.wantOffset || length != c.wantLength {
				t.Fatalf("ParseRange(%q) = (offset=%d, length=%d), want (offset=%d, length=%d)",
					c.header, offset, length, c.wantOffset, c.wantLength)
			}
		})
	}
}

// TestParseRangeMalformedNumbersAreSilentlyZero documents a real gotcha: strconv.ParseInt errors
// inside ParseRange are discarded (`offset, _ = strconv.ParseInt(...)`), so a non-numeric bound
// silently becomes 0 instead of producing an error. This is existing, intentional-looking
// behavior being pinned down, not a defect this test is asserting should be fixed.
func TestParseRangeMalformedNumbersAreSilentlyZero(t *testing.T) {
	offset, length, err := ParseRange("bytes=abc-def", 1000)
	if err != nil {
		t.Fatalf("expected malformed numbers to parse as 0 rather than error, got err=%v", err)
	}
	if offset != 0 || length != 1 {
		t.Fatalf("ParseRange(\"bytes=abc-def\", 1000) = (offset=%d, length=%d), want (offset=0, length=1)", offset, length)
	}
}
