package cachefile

import (
	"bytes"
	"errors"
	"net/netip"

	"github.com/metacubex/mihomo/log"

	"github.com/metacubex/bbolt"
)

type FakeIPEntry struct {
	Host    string
	Address netip.Addr
}

func (c *FakeIpStore) SnapshotEntries(prefix netip.Prefix) ([]FakeIPEntry, error) {
	if c.DB == nil {
		return nil, errors.New("fake-ip cache is unavailable")
	}
	var entries []FakeIPEntry
	err := c.DB.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket(c.bucketName)
		if bucket == nil {
			return nil
		}
		return bucket.ForEach(func(key, value []byte) error {
			ip, ok := netip.AddrFromSlice(value)
			if !ok || !prefix.Contains(ip) || !bytes.Equal(bucket.Get(ip.AsSlice()), key) {
				return nil
			}
			entries = append(entries, FakeIPEntry{Host: string(key), Address: ip})
			if len(entries) > 100000 {
				return errors.New("fake-ip cache has too many entries")
			}
			return nil
		})
	})
	return entries, err
}

func (c *FakeIpStore) ReplaceEntries(entries []FakeIPEntry, offset netip.Addr, cycle bool, offsetKey, cycleKey string) error {
	if c.DB == nil {
		return errors.New("fake-ip cache is unavailable")
	}
	return c.DB.Update(func(tx *bbolt.Tx) error {
		if tx.Bucket(c.bucketName) != nil {
			if err := tx.DeleteBucket(c.bucketName); err != nil {
				return err
			}
		}
		bucket, err := tx.CreateBucketIfNotExists(c.bucketName)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if err := bucket.Put([]byte(entry.Host), entry.Address.AsSlice()); err != nil {
				return err
			}
			if err := bucket.Put(entry.Address.AsSlice(), []byte(entry.Host)); err != nil {
				return err
			}
		}
		if err := bucket.Put([]byte(offsetKey), offset.AsSlice()); err != nil {
			return err
		}
		if cycle {
			return bucket.Put([]byte(cycleKey), offset.AsSlice())
		}
		return nil
	})
}

type FakeIpStore struct {
	*CacheFile
	bucketName []byte
}

func (c *CacheFile) FakeIpStore() *FakeIpStore {
	return &FakeIpStore{c, bucketFakeip}
}

func (c *CacheFile) FakeIpStore6() *FakeIpStore {
	return &FakeIpStore{c, bucketFakeip6}
}

func (c *FakeIpStore) GetByHost(host string) (ip netip.Addr, exist bool) {
	if c.DB == nil {
		return
	}
	c.DB.View(func(t *bbolt.Tx) error {
		if bucket := t.Bucket(c.bucketName); bucket != nil {
			if v := bucket.Get([]byte(host)); v != nil {
				ip, exist = netip.AddrFromSlice(v)
			}
		}
		return nil
	})
	return
}

func (c *FakeIpStore) PutByHost(host string, ip netip.Addr) {
	if c.DB == nil {
		return
	}
	err := c.DB.Batch(func(t *bbolt.Tx) error {
		bucket, err := t.CreateBucketIfNotExists(c.bucketName)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(host), ip.AsSlice())
	})
	if err != nil {
		log.Warnln("[CacheFile] write cache to %s failed: %s", c.DB.Path(), err.Error())
	}
}

func (c *FakeIpStore) GetByIP(ip netip.Addr) (host string, exist bool) {
	if c.DB == nil {
		return
	}
	c.DB.View(func(t *bbolt.Tx) error {
		if bucket := t.Bucket(c.bucketName); bucket != nil {
			if v := bucket.Get(ip.AsSlice()); v != nil {
				host, exist = string(v), true
			}
		}
		return nil
	})
	return
}

func (c *FakeIpStore) PutByIP(ip netip.Addr, host string) {
	if c.DB == nil {
		return
	}
	err := c.DB.Batch(func(t *bbolt.Tx) error {
		bucket, err := t.CreateBucketIfNotExists(c.bucketName)
		if err != nil {
			return err
		}
		return bucket.Put(ip.AsSlice(), []byte(host))
	})
	if err != nil {
		log.Warnln("[CacheFile] write cache to %s failed: %s", c.DB.Path(), err.Error())
	}
}

func (c *FakeIpStore) DelByIP(ip netip.Addr) {
	if c.DB == nil {
		return
	}

	addr := ip.AsSlice()
	err := c.DB.Batch(func(t *bbolt.Tx) error {
		bucket, err := t.CreateBucketIfNotExists(c.bucketName)
		if err != nil {
			return err
		}
		host := bucket.Get(addr)
		err = bucket.Delete(addr)
		if len(host) > 0 {
			if err = bucket.Delete(host); err != nil {
				return err
			}
		}
		return err
	})
	if err != nil {
		log.Warnln("[CacheFile] write cache to %s failed: %s", c.DB.Path(), err.Error())
	}
}

func (c *FakeIpStore) FlushFakeIP() error {
	err := c.DB.Batch(func(t *bbolt.Tx) error {
		bucket := t.Bucket(c.bucketName)
		if bucket == nil {
			return nil
		}
		return t.DeleteBucket(c.bucketName)
	})
	return err
}
