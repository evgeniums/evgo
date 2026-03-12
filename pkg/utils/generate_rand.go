package utils

import (
	"fmt"
	"math"
	"math/rand/v2"
	"sync/atomic"
	"time"
)

var idCount atomic.Uint32

func GenerateID() string {
	t := time.Now().UnixMilli()
	r1 := rand.Uint32()

	count := idCount.Add(1)

	id := fmt.Sprintf("%11x%06x%08x", t, count&0xFFFFFF, r1)
	return id
}

func GenerateRand64() string {
	r1 := rand.Uint64()
	id := fmt.Sprintf("%016x", r1)
	return id
}

func GenerateRandInt(length ...int) string {
	r1 := rand.Uint64()
	id := fmt.Sprintf("%d", r1)
	if len(length) != 0 {
		return id[:length[0]]
	}
	return id
}

func GenerateRandIntPadded(length ...int) string {
	r1 := rand.Uint64()
	id := fmt.Sprintf("%020d", r1)
	if len(length) != 0 {
		return id[:length[0]]
	}
	return id
}

func GenerateRandIntFrom(from int, length ...int) string {
	var min uint64 = uint64(10000)
	var max uint64 = math.MaxUint64

	var n uint64 = rand.Uint64N(max-min) + min

	id := fmt.Sprintf("%d", n)
	if len(length) != 0 {
		return id[:length[0]]
	}
	return id
}
