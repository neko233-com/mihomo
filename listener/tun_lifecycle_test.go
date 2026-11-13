package listener

import (
	"sync"
	"testing"
)

func TestTUNCleanupResetsStateAndSynchronizesReaders(t *testing.T) {
	tunMux.Lock()
	LastTunConf.Enable = true
	tunMux.Unlock()
	Cleanup()
	if GetTunConf().Enable {
		t.Fatal("cleanup retained enabled TUN state")
	}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				_ = GetTunConf()
			}
		}()
	}
	for i := 0; i < 50; i++ {
		Cleanup()
	}
	wg.Wait()
}
