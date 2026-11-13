package fakeip

import (
	"net/netip"
	"path/filepath"
	"testing"

	"github.com/metacubex/bbolt"
	"github.com/metacubex/mihomo/component/profile/cachefile"
)

func testSnapshotPool(t *testing.T, prefix netip.Prefix, disk bool) *Pool {
	t.Helper()
	p, err := New(Options{IPNet: prefix, Size: 10})
	if err != nil {
		t.Fatal(err)
	}
	if disk {
		dir := t.TempDir()
		db, err := bbolt.Open(filepath.Join(dir, "cache.db"), 0600, nil)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Close() })
		p.store = newCachefileStore(&cachefile.CacheFile{DB: db}, prefix)
	}
	return p
}

func TestSnapshotTransfersMappingsAndAllocatorAcrossStores(t *testing.T) {
	for _, prefix := range []netip.Prefix{netip.MustParsePrefix("198.18.0.1/24"), netip.MustParsePrefix("fd00::1/120")} {
		for _, sourceDisk := range []bool{false, true} {
			for _, destinationDisk := range []bool{false, true} {
				source := testSnapshotPool(t, prefix, sourceDisk)
				dest := testSnapshotPool(t, prefix, destinationDisk)
				a, b := source.Lookup("a.example"), source.Lookup("b.example")
				_ = dest.Lookup("conflicting.example")
				state, err := source.ExportSnapshot()
				if err != nil {
					t.Fatal(err)
				}
				if err := dest.ImportSnapshot(state); err != nil {
					t.Fatal(err)
				}
				if got := dest.Lookup("b.example"); got != b {
					t.Fatalf("b mapping changed: %s != %s", got, b)
				}
				if got := dest.Lookup("a.example"); got != a {
					t.Fatalf("a mapping changed: %s != %s", got, a)
				}
				if host, ok := dest.LookBack(a); !ok || host != "a.example" {
					t.Fatalf("reverse mapping: %q %v", host, ok)
				}
				if next := dest.Lookup("next.example"); next == a || next == b {
					t.Fatalf("allocator reused active address %s", next)
				}
			}
		}
	}
}

func TestSnapshotRejectsInvalidMappingsWithoutChangingPool(t *testing.T) {
	p := testSnapshotPool(t, netip.MustParsePrefix("198.18.0.1/24"), false)
	original := p.Lookup("original.example")
	state, err := p.ExportSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	state.Entries = append(state.Entries, Entry{Host: "evil.example", IP: original.String()})
	if err := p.ImportSnapshot(state); err == nil {
		t.Fatal("accepted duplicate address")
	}
	if got := p.Lookup("original.example"); got != original {
		t.Fatal("invalid snapshot changed active mapping")
	}
	state.Entries = []Entry{{Host: "outside.example", IP: "203.0.113.1"}}
	if err := p.ImportSnapshot(state); err == nil {
		t.Fatal("accepted address outside fake range")
	}
}

func TestSnapshotPreservesCycledAllocator(t *testing.T) {
	prefix := netip.MustParsePrefix("198.18.0.1/29")
	source := testSnapshotPool(t, prefix, false)
	_ = source.Lookup("a.example")
	_ = source.Lookup("b.example")
	_ = source.Lookup("c.example")
	d := source.Lookup("d.example")
	state, err := source.ExportSnapshot()
	if err != nil || !state.Cycle {
		t.Fatalf("missing cycle state: %+v %v", state, err)
	}
	dest := testSnapshotPool(t, prefix, false)
	if err := dest.ImportSnapshot(state); err != nil {
		t.Fatal(err)
	}
	if host, ok := dest.LookBack(d); !ok || host != "d.example" {
		t.Fatalf("lost cycled mapping: %q %v", host, ok)
	}
	next := dest.Lookup("e.example")
	if next == d {
		t.Fatal("allocator ignored imported cycle offset")
	}
}
