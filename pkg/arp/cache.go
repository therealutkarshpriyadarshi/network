package arp

import (
	"fmt"
	"sync"
	"time"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

// DefaultCacheTimeout is the default time after which an ARP cache entry expires.
// RFC 826 doesn't specify a timeout, but typical implementations use 60-300 seconds.
const DefaultCacheTimeout = 5 * time.Minute

// CacheEntry represents a single entry in the ARP cache.
type CacheEntry struct {
	MAC       common.MACAddress
	ExpiresAt time.Time
}

// IsExpired returns true if this cache entry has expired.
func (e *CacheEntry) IsExpired() bool {
	return time.Now().After(e.ExpiresAt)
}

// Cache implements a thread-safe ARP cache that maps IP addresses to MAC addresses.
// Entries automatically expire after a configured timeout.
type Cache struct {
	mu      sync.RWMutex
	entries map[common.IPv4Address]*CacheEntry
	timeout time.Duration
}

// NewCache creates a new ARP cache with the specified timeout.
func NewCache(timeout time.Duration) *Cache {
	return &Cache{
		entries: make(map[common.IPv4Address]*CacheEntry),
		timeout: timeout,
	}
}

// NewDefaultCache creates a new ARP cache with the default timeout.
func NewDefaultCache() *Cache {
	return NewCache(DefaultCacheTimeout)
}

// Add adds or updates an entry in the ARP cache.
func (c *Cache) Add(ip common.IPv4Address, mac common.MACAddress) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[ip] = &CacheEntry{
		MAC:       mac,
		ExpiresAt: time.Now().Add(c.timeout),
	}
}

// Get retrieves a MAC address for the given IP address.
// Returns the MAC address and true if found and not expired, or zero MAC and false otherwise.
func (c *Cache) Get(ip common.IPv4Address) (common.MACAddress, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[ip]
	if !exists {
		return common.MACAddress{}, false
	}

	// Check if entry has expired
	if entry.IsExpired() {
		return common.MACAddress{}, false
	}

	return entry.MAC, true
}

// Delete removes an entry from the ARP cache.
func (c *Cache) Delete(ip common.IPv4Address) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.entries, ip)
}

// Clear removes all entries from the ARP cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[common.IPv4Address]*CacheEntry)
}

// Cleanup removes all expired entries from the cache.
// This should be called periodically to prevent the cache from growing indefinitely.
func (c *Cache) Cleanup() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	removed := 0
	for ip, entry := range c.entries {
		if entry.IsExpired() {
			delete(c.entries, ip)
			removed++
		}
	}

	return removed
}

// Size returns the number of entries currently in the cache (including expired ones).
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return len(c.entries)
}

// Entries returns a snapshot of all non-expired entries in the cache.
// The returned map is a copy and can be safely modified by the caller.
func (c *Cache) Entries() map[common.IPv4Address]common.MACAddress {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snapshot := make(map[common.IPv4Address]common.MACAddress)
	for ip, entry := range c.entries {
		if !entry.IsExpired() {
			snapshot[ip] = entry.MAC
		}
	}

	return snapshot
}

// String returns a human-readable representation of the cache.
func (c *Cache) String() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := fmt.Sprintf("ARP Cache (%d entries):\n", len(c.entries))
	for ip, entry := range c.entries {
		status := "valid"
		if entry.IsExpired() {
			status = "expired"
		}
		ttl := time.Until(entry.ExpiresAt)
		result += fmt.Sprintf("  %s -> %s (%s, TTL: %v)\n", ip, entry.MAC, status, ttl)
	}

	return result
}

// StartCleanupRoutine starts a goroutine that periodically cleans up expired entries.
// The cleanup runs at the specified interval.
// Returns a channel that can be closed to stop the cleanup routine.
func (c *Cache) StartCleanupRoutine(interval time.Duration) chan<- struct{} {
	stop := make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				removed := c.Cleanup()
				if removed > 0 {
					// Optional: log cleanup activity
					_ = removed
				}
			case <-stop:
				return
			}
		}
	}()

	return stop
}
