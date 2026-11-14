package resource

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/metacubex/mihomo/common/utils"
	P "github.com/metacubex/mihomo/constant/provider"
)

type pausableTestVehicle struct {
	started chan struct{}
	mu      sync.Mutex
	writes  []string
	block   bool
}

func (v *pausableTestVehicle) Read(ctx context.Context, _ utils.HashType) ([]byte, utils.HashType, error) {
	v.mu.Lock()
	block := v.block
	v.mu.Unlock()
	if block {
		v.started <- struct{}{}
		<-ctx.Done()
		return nil, utils.HashType{}, ctx.Err()
	}
	b := []byte("fresh")
	return b, utils.MakeHash(b), nil
}
func (v *pausableTestVehicle) Write(b []byte) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.writes = append(v.writes, string(b))
	return nil
}
func (*pausableTestVehicle) Path() string        { return "" }
func (*pausableTestVehicle) Url() string         { return "" }
func (*pausableTestVehicle) Proxy() string       { return "" }
func (*pausableTestVehicle) Type() P.VehicleType { return P.HTTP }

func TestPauseUpdatesCancelsConcurrentReadsAndResume(t *testing.T) {
	v := &pausableTestVehicle{started: make(chan struct{}, 2), block: true}
	f := NewFetcher("test", time.Hour, v, nil, func(b []byte) (string, error) { return string(b), nil }, nil)
	defer f.Close()
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() { _, _, err := f.Update(); results <- err }()
	}
	for i := 0; i < 2; i++ {
		<-v.started
	}
	f.PauseUpdates()
	for i := 0; i < 2; i++ {
		if err := <-results; !errors.Is(err, context.Canceled) {
			t.Fatalf("update: %v", err)
		}
	}
	if _, _, err := f.Update(); !errors.Is(err, context.Canceled) {
		t.Fatalf("paused update: %v", err)
	}
	v.mu.Lock()
	v.block = false
	v.mu.Unlock()
	f.ResumeUpdates()
	if value, _, err := f.Update(); err != nil || value != "fresh" {
		t.Fatalf("resumed: %q %v", value, err)
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if len(v.writes) != 1 || v.writes[0] != "fresh" {
		t.Fatalf("writes: %v", v.writes)
	}
}

func TestCanceledUpdateCannotCommitAfterPause(t *testing.T) {
	v := &pausableTestVehicle{started: make(chan struct{}, 1)}
	f := NewFetcher("test", time.Hour, v, nil, func(b []byte) (string, error) { return string(b), nil }, nil)
	defer f.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, _, err := f.loadBufContext(ctx, []byte("stale"), utils.MakeHash([]byte("stale")), true); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled commit: %v", err)
	}
	if len(v.writes) != 0 {
		t.Fatalf("stale write: %v", v.writes)
	}
	f.PauseUpdates()
	_ = f.Close()
	f.ResumeUpdates()
	if _, _, err := f.Update(); !errors.Is(err, context.Canceled) {
		t.Fatalf("closed update: %v", err)
	}
}

func TestPauseDoesNotWaitForBlockedCommit(t *testing.T) {
	v := &pausableTestVehicle{started: make(chan struct{}, 1)}
	entered, release := make(chan struct{}), make(chan struct{})
	f := NewFetcher("test", time.Hour, v, nil, func(b []byte) (string, error) { return string(b), nil }, func(string) { close(entered); <-release })
	defer f.Close()
	updated := make(chan error, 1)
	go func() { _, _, err := f.Update(); updated <- err }()
	<-entered
	paused := make(chan struct{})
	go func() { f.PauseUpdates(); close(paused) }()
	select {
	case <-paused:
	case <-time.After(time.Second):
		close(release)
		<-updated
		t.Fatal("pause waited for an unrelated blocked commit")
	}
	if _, _, err := f.Update(); !errors.Is(err, context.Canceled) {
		t.Fatalf("paused update: %v", err)
	}
	close(release)
	if err := <-updated; err != nil {
		t.Fatal(err)
	}
	f.ResumeUpdates()
}
