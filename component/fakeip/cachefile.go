package fakeip

import (
	"net/netip"

	"github.com/metacubex/mihomo/component/profile/cachefile"
)

type cachefileStore struct {
	cache  *cachefile.FakeIpStore
	prefix netip.Prefix
}

func (c *cachefileStore) SnapshotEntries() ([]Entry, error) {
	items, err := c.cache.SnapshotEntries(c.prefix)
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(items))
	for _, item := range items {
		entries = append(entries, Entry{Host: item.Host, IP: item.Address.String()})
	}
	return entries, nil
}

func (c *cachefileStore) ReplaceEntries(entries []Entry, offset netip.Addr, cycle bool) error {
	items := make([]cachefile.FakeIPEntry, 0, len(entries))
	for _, entry := range entries {
		items = append(items, cachefile.FakeIPEntry{Host: entry.Host, Address: netip.MustParseAddr(entry.IP)})
	}
	return c.cache.ReplaceEntries(items, offset, cycle, offsetKey, cycleKey)
}

// GetByHost implements store.GetByHost
func (c *cachefileStore) GetByHost(host string) (netip.Addr, bool) {
	return c.cache.GetByHost(host)
}

// PutByHost implements store.PutByHost
func (c *cachefileStore) PutByHost(host string, ip netip.Addr) {
	c.cache.PutByHost(host, ip)
}

// GetByIP implements store.GetByIP
func (c *cachefileStore) GetByIP(ip netip.Addr) (string, bool) {
	return c.cache.GetByIP(ip)
}

// PutByIP implements store.PutByIP
func (c *cachefileStore) PutByIP(ip netip.Addr, host string) {
	c.cache.PutByIP(ip, host)
}

// DelByIP implements store.DelByIP
func (c *cachefileStore) DelByIP(ip netip.Addr) {
	c.cache.DelByIP(ip)
}

// Exist implements store.Exist
func (c *cachefileStore) Exist(ip netip.Addr) bool {
	_, exist := c.GetByIP(ip)
	return exist
}

// CloneTo implements store.CloneTo
// already persistence
func (c *cachefileStore) CloneTo(store store) {}

// FlushFakeIP implements store.FlushFakeIP
func (c *cachefileStore) FlushFakeIP() error {
	return c.cache.FlushFakeIP()
}

func newCachefileStore(cache *cachefile.CacheFile, prefix netip.Prefix) *cachefileStore {
	if prefix.Addr().Is6() {
		return &cachefileStore{cache: cache.FakeIpStore6(), prefix: prefix}
	} else {
		return &cachefileStore{cache: cache.FakeIpStore(), prefix: prefix}
	}
}
