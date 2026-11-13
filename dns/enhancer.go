package dns

import (
	"errors"
	"net/netip"

	"github.com/metacubex/mihomo/common/lru"
	"github.com/metacubex/mihomo/component/fakeip"
	C "github.com/metacubex/mihomo/constant"
)

type ResolverEnhancer struct {
	ipv6          bool
	mode          C.DNSMode
	fakeIPPool    *fakeip.Pool
	fakeIPPool6   *fakeip.Pool
	fakeIPSkipper *fakeip.Skipper
	fakeIPTTL     int
	mapping       *lru.LruCache[netip.Addr, string]
	useHosts      bool
}

// FakeIPState carries the current v4 and v6 fake-IP allocations between
// kernel processes without changing the configured address ranges.
type FakeIPState struct {
	IPv4 *fakeip.Snapshot
	IPv6 *fakeip.Snapshot
}

func (h *ResolverEnhancer) ExportFakeIPState() (FakeIPState, error) {
	var state FakeIPState
	if h == nil || !h.FakeIPEnabled() {
		return state, nil
	}
	if h.fakeIPPool != nil {
		snapshot, err := h.fakeIPPool.ExportSnapshot()
		if err != nil {
			return state, err
		}
		state.IPv4 = &snapshot
	}
	if h.fakeIPPool6 != nil {
		snapshot, err := h.fakeIPPool6.ExportSnapshot()
		if err != nil {
			return state, err
		}
		state.IPv6 = &snapshot
	}
	return state, nil
}

func (h *ResolverEnhancer) FakeIPEnabled() bool {
	return h.mode == C.DNSFakeIP
}

func (h *ResolverEnhancer) MappingEnabled() bool {
	return h.mode == C.DNSFakeIP || h.mode == C.DNSMapping
}

func (h *ResolverEnhancer) IsExistFakeIP(ip netip.Addr) bool {
	if !h.FakeIPEnabled() {
		return false
	}

	if pool := h.fakeIPPool; pool != nil {
		if pool.Exist(ip) {
			return true
		}
	}

	if pool6 := h.fakeIPPool6; pool6 != nil {
		if pool6.Exist(ip) {
			return true
		}
	}

	return false
}

func (h *ResolverEnhancer) IsFakeIP(ip netip.Addr) bool {
	if !h.FakeIPEnabled() {
		return false
	}

	if pool := h.fakeIPPool; pool != nil {
		if pool.IPNet().Contains(ip) && ip != pool.Gateway() && ip != pool.Broadcast() {
			return true
		}
	}

	if pool6 := h.fakeIPPool6; pool6 != nil {
		if pool6.IPNet().Contains(ip) && ip != pool6.Gateway() && ip != pool6.Broadcast() {
			return true
		}
	}

	return false
}

func (h *ResolverEnhancer) IsFakeBroadcastIP(ip netip.Addr) bool {
	if !h.FakeIPEnabled() {
		return false
	}

	if pool := h.fakeIPPool; pool != nil {
		if pool.Broadcast() == ip {
			return true
		}
	}

	if pool6 := h.fakeIPPool6; pool6 != nil {
		if pool6.Broadcast() == ip {
			return true
		}
	}

	return false
}

func (h *ResolverEnhancer) FindHostByIP(ip netip.Addr) (string, bool) {
	if pool := h.fakeIPPool; pool != nil {
		if host, existed := pool.LookBack(ip); existed {
			return host, true
		}
	}

	if pool6 := h.fakeIPPool6; pool6 != nil {
		if host, existed := pool6.LookBack(ip); existed {
			return host, true
		}
	}

	if mapping := h.mapping; mapping != nil {
		if host, existed := h.mapping.Get(ip); existed {
			return host, true
		}
	}

	return "", false
}

func (h *ResolverEnhancer) InsertHostByIP(ip netip.Addr, host string) {
	if mapping := h.mapping; mapping != nil {
		h.mapping.Set(ip, host)
	}
}

func (h *ResolverEnhancer) FlushFakeIP() error {
	var errs []error
	if pool := h.fakeIPPool; pool != nil {
		if err := pool.FlushFakeIP(); err != nil {
			errs = append(errs, err)
		}
	}
	if pool6 := h.fakeIPPool6; pool6 != nil {
		if err := pool6.FlushFakeIP(); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (h *ResolverEnhancer) PatchFrom(o *ResolverEnhancer) {
	h.PatchMappingFrom(o)

	if h.fakeIPPool != nil && o.fakeIPPool != nil {
		h.fakeIPPool.CloneFrom(o.fakeIPPool)
	}

	if h.fakeIPPool6 != nil && o.fakeIPPool6 != nil {
		h.fakeIPPool6.CloneFrom(o.fakeIPPool6)
	}
}

// PatchMappingFrom preserves non-fake DNS mappings without overwriting an
// explicitly restored fake-IP pool.
func (h *ResolverEnhancer) PatchMappingFrom(o *ResolverEnhancer) {
	if h.mapping != nil && o.mapping != nil {
		o.mapping.CloneTo(h.mapping)
	}
}

func (h *ResolverEnhancer) StoreFakePoolState() {
	if h.fakeIPPool != nil {
		h.fakeIPPool.StoreState()
	}

	if h.fakeIPPool6 != nil {
		h.fakeIPPool6.StoreState()
	}
}

type EnhancerConfig struct {
	IPv6          bool
	EnhancedMode  C.DNSMode
	FakeIPPool    *fakeip.Pool
	FakeIPPool6   *fakeip.Pool
	FakeIPSkipper *fakeip.Skipper
	FakeIPTTL     int
	UseHosts      bool
}

func NewEnhancer(cfg EnhancerConfig) *ResolverEnhancer {
	e := &ResolverEnhancer{
		ipv6:     cfg.IPv6,
		mode:     cfg.EnhancedMode,
		useHosts: cfg.UseHosts,
	}

	if cfg.EnhancedMode != C.DNSNormal {
		e.fakeIPPool = cfg.FakeIPPool
		if cfg.IPv6 {
			e.fakeIPPool6 = cfg.FakeIPPool6
		}
		e.fakeIPSkipper = cfg.FakeIPSkipper
		e.fakeIPTTL = cfg.FakeIPTTL
		if e.fakeIPTTL < 1 {
			e.fakeIPTTL = 1
		}
		e.mapping = lru.New(lru.WithSize[netip.Addr, string](4096))
	}

	return e
}
