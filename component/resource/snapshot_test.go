package resource

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/metacubex/mihomo/common/utils"
	P "github.com/metacubex/mihomo/constant/provider"
)

type blockedVehicle struct {
	started chan struct{}
	allow   chan struct{}
	data    []byte
}

func (*blockedVehicle) Type() P.VehicleType { return P.HTTP }
func (*blockedVehicle) Path() string        { return "cache" }
func (*blockedVehicle) Url() string         { return "https://example.invalid" }
func (*blockedVehicle) Proxy() string       { return "" }
func (*blockedVehicle) Read(context.Context, utils.HashType) ([]byte, utils.HashType, error) {
	return nil, utils.HashType{}, errors.New("unused")
}
func (v *blockedVehicle) Write(b []byte) error {
	close(v.started)
	<-v.allow
	v.data = append([]byte(nil), b...)
	return nil
}

func TestSnapshotAndCloseWaitForCommittedWrite(t *testing.T) {
	v := &blockedVehicle{started: make(chan struct{}), allow: make(chan struct{})}
	f := NewFetcher("test", 0, v, nil, func(b []byte) (string, error) { return string(b), nil }, nil)
	updated := make(chan error, 1)
	go func() { _, _, err := f.SideUpdate([]byte("new")); updated <- err }()
	<-v.started
	snapshot := make(chan string, 1)
	go func() {
		_ = f.WithSnapshot(func(_ P.Vehicle, _ utils.HashType) error { snapshot <- string(v.data); return nil })
	}()
	closed := make(chan error, 1)
	go func() { closed <- f.Close() }()
	select {
	case <-snapshot:
		t.Fatal("snapshot observed an incomplete write")
	case <-closed:
		t.Fatal("close returned during write")
	case <-time.After(30 * time.Millisecond):
	}
	close(v.allow)
	if err := <-updated; err != nil {
		t.Fatal(err)
	}
	if err := <-closed; err != nil {
		t.Fatal(err)
	}
	if got := <-snapshot; got != "new" {
		t.Fatalf("snapshot = %q", got)
	}
	if _, _, err := f.SideUpdate([]byte("late")); !errors.Is(err, context.Canceled) {
		t.Fatalf("write after close: %v", err)
	}
}
