package utils_test

import (
	"sync"
	"testing"

	"github.com/evgeniums/evgo/pkg/utils"
)

func TestGenerateId(t *testing.T) {

	var wg sync.WaitGroup

	gen := func(start int, wg *sync.WaitGroup) {
		defer wg.Done()
		for i := start; i < start+100; i++ {
			t.Logf("%d: %v", i, utils.GenerateID())
		}
	}

	for i := 0; i < 300; i += 100 {
		wg.Add(1)
		go gen(i, &wg)
	}

	wg.Wait()
}
