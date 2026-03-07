package utils_test

import (
	"testing"

	"github.com/evgeniums/evgo/pkg/utils"
	"github.com/stretchr/testify/assert"
)

func CheckEqual(t *testing.T, value string, sample string) {
	if value != sample {
		t.Errorf("Invalid value: expected %v, got %v", sample, value)
	}
}

func TestFloatToStr(t *testing.T) {

	f1 := float64(1000.0)
	f2 := float64(1000.10)
	f3 := float64(1000.11)
	f4 := float64(1000.111)
	f5 := float64(1000.1111)
	f6 := float64(1000.1175)
	f7 := float64(1000.1)

	// t.Logf(utils.FloatToStr2(f1))
	// t.Logf(utils.FloatToStr2(f2))
	// t.Logf(utils.FloatToStr2(f3))
	// t.Logf(utils.FloatToStr2(f4))
	// t.Logf(utils.FloatToStr2(f5))
	// t.Logf(utils.FloatToStr2(f6))

	CheckEqual(t, utils.FloatToStr(f1), "1000")
	CheckEqual(t, utils.FloatToStr2(f1), "1000.00")
	CheckEqual(t, utils.FloatToStr(f2), "1000.1")
	CheckEqual(t, utils.FloatToStr2(f2), "1000.10")
	CheckEqual(t, utils.FloatToStr2(f3), "1000.11")
	CheckEqual(t, utils.FloatToStr2(f4), "1000.11")
	CheckEqual(t, utils.FloatToStr2(f5), "1000.11")
	CheckEqual(t, utils.FloatToStr2(f6), "1000.12")
	CheckEqual(t, utils.FloatToStr(f7), "1000.1")

	ff1, e := utils.StrToFloat(utils.FloatToStr(f1))
	assert.NoError(t, e)
	assert.InDelta(t, f1, ff1, 0.0000001)

	ff2, e := utils.StrToFloat(utils.FloatToStr(f2))
	assert.NoError(t, e)
	assert.InDelta(t, f2, ff2, 0.0000001)

	ff3, e := utils.StrToFloat(utils.FloatToStr(f3))
	assert.NoError(t, e)
	assert.InDelta(t, f3, ff3, 0.0000001)

	ff4, e := utils.StrToFloat(utils.FloatToStr(f4))
	assert.NoError(t, e)
	assert.InDelta(t, f4, ff4, 0.0000001)

	ff5, e := utils.StrToFloat(utils.FloatToStr(f5))
	assert.NoError(t, e)
	assert.InDelta(t, f5, ff5, 0.0000001)

	ff6, e := utils.StrToFloat(utils.FloatToStr(f6))
	assert.NoError(t, e)
	assert.InDelta(t, f6, ff6, 0.0000001)

	ff7, e := utils.StrToFloat(utils.FloatToStr(f7))
	assert.NoError(t, e)
	assert.InDelta(t, f7, ff7, 0.0000001)
}

func TestStrToFloat(t *testing.T) {

	str := "1000.1"
	f1, e := utils.StrToFloat(str)
	assert.NoError(t, e)

	t.Logf("%s", utils.FloatToStr2(f1))
}
