package hooks

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSendSemaphoreRespectsCap(t *testing.T) {
	capacity := sendSemaphoreCapacity()
	if capacity <= 0 {
		t.Fatalf("send semaphore must have positive capacity, got %d", capacity)
	}

	const total = 1000
	var inFlight atomic.Int64
	var peak atomic.Int64
	var wg sync.WaitGroup
	wg.Add(total)

	for i := 0; i < total; i++ {
		go func() {
			defer wg.Done()
			sendSemaphore <- struct{}{}
			cur := inFlight.Add(1)
			for {
				prev := peak.Load()
				if cur <= prev || peak.CompareAndSwap(prev, cur) {
					break
				}
			}
			time.Sleep(50 * time.Microsecond)
			inFlight.Add(-1)
			<-sendSemaphore
		}()
	}

	wg.Wait()

	if got := int(peak.Load()); got > capacity {
		t.Fatalf("peak concurrent sends %d exceeded semaphore cap %d", got, capacity)
	}
}
