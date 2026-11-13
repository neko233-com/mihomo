package log

import (
	"sync"
	"testing"
)

func TestLevelCanChangeWhileLogging(t *testing.T) {
	previous := Level()
	defer SetLevel(previous)
	var workers sync.WaitGroup
	workers.Add(2)
	go func() {
		defer workers.Done()
		for i := 0; i < 10000; i++ {
			SetLevel(INFO)
			SetLevel(WARNING)
		}
	}()
	go func() {
		defer workers.Done()
		for i := 0; i < 10000; i++ {
			_ = Level()
			print(Event{LogLevel: DEBUG})
		}
	}()
	workers.Wait()
}
