package fakeip

import (
	"errors"
	"net/netip"
	"strings"
	"unicode/utf8"
)

// Entry and Snapshot are a bounded, portable representation of fake-IP state.
type Entry struct {
	Host string
	IP   string
}
type Snapshot struct {
	Prefix  string
	Offset  string
	Cycle   bool
	Entries []Entry
}

// ExportSnapshot captures allocation state while DNS allocations are locked.
func (p *Pool) ExportSnapshot() (Snapshot, error) {
	p.mux.Lock()
	defer p.mux.Unlock()
	entries, err := p.store.SnapshotEntries()
	if err != nil {
		return Snapshot{}, err
	}
	if len(entries) > 100000 {
		return Snapshot{}, errors.New("fake-ip snapshot has too many entries")
	}
	return Snapshot{Prefix: p.ipnet.String(), Offset: p.offset.String(), Cycle: p.cycle, Entries: entries}, nil
}

// ImportSnapshot validates all mappings before replacing the destination.
func (p *Pool) ImportSnapshot(state Snapshot) error {
	p.mux.Lock()
	defer p.mux.Unlock()
	if state.Prefix != p.ipnet.String() || len(state.Entries) > 100000 {
		return errors.New("incompatible fake-ip snapshot")
	}
	offset, err := netip.ParseAddr(state.Offset)
	if err != nil || offset.Less(p.first.Prev()) || !offset.Less(p.last) {
		return errors.New("invalid fake-ip allocation offset")
	}
	entries := append([]Entry(nil), state.Entries...)
	hosts := make(map[string]bool, len(state.Entries))
	addresses := make(map[netip.Addr]bool, len(state.Entries))
	for i := range entries {
		entry := &entries[i]
		entry.Host = strings.ToLower(entry.Host)
		ip, err := netip.ParseAddr(entry.IP)
		if err != nil || !utf8.ValidString(entry.Host) || len(entry.Host) == 0 || len(entry.Host) > 253 || strings.ContainsRune(entry.Host, 0) || ip.Less(p.first) || !ip.Less(p.last) || !p.ipnet.Contains(ip) || hosts[entry.Host] || addresses[ip] {
			return errors.New("invalid or duplicate fake-ip mapping")
		}
		entry.IP = ip.String()
		hosts[entry.Host], addresses[ip] = true, true
	}
	if err := p.store.ReplaceEntries(entries, offset, state.Cycle); err != nil {
		return err
	}
	p.offset, p.cycle = offset, state.Cycle
	return nil
}
