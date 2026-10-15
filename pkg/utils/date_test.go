package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPackUnpackDatetimeRoundTrip(t *testing.T) {
	zones := []*time.Location{
		time.UTC,
		time.FixedZone("", 5*3600+30*60),
		time.FixedZone("", -8*3600),
	}
	for _, loc := range zones {
		in := time.Date(2026, 10, 15, 12, 30, 45, 123_000_000, loc)
		packed := PackDatetime(in)
		require.NotZero(t, packed)
		out := UnpackDatetime(packed)
		require.True(t, in.Equal(out), "instant must survive: %s vs %s", in, out)
		_, off := out.Zone()
		_, inOff := in.Zone()
		require.Equal(t, inOff, off)
	}
}

func TestUnpackDatetimeOfClientPackedValue(t *testing.T) {
	// hatn DateTimePacker::pack(millis, tzMinutes): (millis << 8) | (tz units & 0x7F), no epoch bit
	millis := time.Date(2026, 10, 15, 12, 0, 0, 0, time.UTC).UnixMilli()
	packed := millis << 8
	out := UnpackDatetime(packed)
	require.Equal(t, millis, out.UnixMilli())
	require.Equal(t, millis, FromHatnProtoDatetime(packed).UnixMilli())
}

func TestPackUnpackDatetimeUnsetAndEpoch(t *testing.T) {
	require.Equal(t, int64(0), PackDatetime(time.Time{}))
	require.True(t, UnpackDatetime(0).IsZero())
	require.True(t, FromHatnProtoDatetime(0).IsZero())

	epoch := time.UnixMilli(0).UTC()
	packed := PackDatetime(epoch)
	require.Equal(t, int64(epochBit), packed)
	require.False(t, UnpackDatetime(packed).IsZero())
	require.Equal(t, int64(0), UnpackDatetime(packed).UnixMilli())
}
