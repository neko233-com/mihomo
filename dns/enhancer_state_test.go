package dns

import (
	"net/netip"
	"testing"

	"github.com/metacubex/mihomo/component/fakeip"
	C "github.com/metacubex/mihomo/constant"
)

func TestExportAndPatchPreserveRestoredFakeIP(t *testing.T) {
	prefix := netip.MustParsePrefix("198.18.0.1/24")
	old, err := fakeip.New(fakeip.Options{IPNet: prefix, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	restored, err := fakeip.New(fakeip.Options{IPNet: prefix, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	_ = old.Lookup("stale.example")
	want := restored.Lookup("current.example")
	previous := NewEnhancer(EnhancerConfig{EnhancedMode: C.DNSFakeIP, FakeIPPool: old})
	current := NewEnhancer(EnhancerConfig{EnhancedMode: C.DNSFakeIP, FakeIPPool: restored})
	current.PatchMappingFrom(previous)
	if got := restored.Lookup("current.example"); got != want {
		t.Fatal("restored fake-IP was overwritten")
	}
	state, err := current.ExportFakeIPState()
	if err != nil || state.IPv4 == nil || len(state.IPv4.Entries) != 1 || state.IPv4.Entries[0].Host != "current.example" {
		t.Fatalf("state: %+v %v", state, err)
	}
}
