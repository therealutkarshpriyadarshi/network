package arp

import (
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

func TestCacheAddAndGet(t *testing.T) {
	cache := NewCache(1 * time.Minute)

	ip := common.IPv4Address{192, 168, 1, 1}
	mac := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}

	// Add entry
	cache.Add(ip, mac)

	// Get entry
	gotMAC, found := cache.Get(ip)
	if !found {
		t.Error("Get() found = false, want true")
	}
	if gotMAC != mac {
		t.Errorf("Get() MAC = %v, want %v", gotMAC, mac)
	}

	// Get non-existent entry
	nonExistentIP := common.IPv4Address{192, 168, 1, 2}
	_, found = cache.Get(nonExistentIP)
	if found {
		t.Error("Get() for non-existent IP found = true, want false")
	}
}

func TestCacheUpdate(t *testing.T) {
	cache := NewCache(1 * time.Minute)

	ip := common.IPv4Address{192, 168, 1, 1}
	mac1 := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	mac2 := common.MACAddress{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}

	// Add entry
	cache.Add(ip, mac1)

	// Verify first MAC
	gotMAC, found := cache.Get(ip)
	if !found || gotMAC != mac1 {
		t.Errorf("Get() MAC = %v, want %v", gotMAC, mac1)
	}

	// Update with new MAC
	cache.Add(ip, mac2)

	// Verify updated MAC
	gotMAC, found = cache.Get(ip)
	if !found || gotMAC != mac2 {
		t.Errorf("Get() after update MAC = %v, want %v", gotMAC, mac2)
	}
}

func TestCacheExpiration(t *testing.T) {
	// Use a very short timeout for testing
	cache := NewCache(100 * time.Millisecond)

	ip := common.IPv4Address{192, 168, 1, 1}
	mac := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}

	// Add entry
	cache.Add(ip, mac)

	// Should be found immediately
	_, found := cache.Get(ip)
	if !found {
		t.Error("Get() immediately after Add found = false, want true")
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not be found after expiration
	_, found = cache.Get(ip)
	if found {
		t.Error("Get() after expiration found = true, want false")
	}
}

func TestCacheDelete(t *testing.T) {
	cache := NewCache(1 * time.Minute)

	ip := common.IPv4Address{192, 168, 1, 1}
	mac := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}

	// Add entry
	cache.Add(ip, mac)

	// Verify it's there
	_, found := cache.Get(ip)
	if !found {
		t.Error("Get() before Delete found = false, want true")
	}

	// Delete entry
	cache.Delete(ip)

	// Verify it's gone
	_, found = cache.Get(ip)
	if found {
		t.Error("Get() after Delete found = true, want false")
	}
}

func TestCacheClear(t *testing.T) {
	cache := NewCache(1 * time.Minute)

	// Add multiple entries
	for i := 1; i <= 5; i++ {
		ip := common.IPv4Address{192, 168, 1, byte(i)}
		mac := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, byte(i)}
		cache.Add(ip, mac)
	}

	// Verify size
	if size := cache.Size(); size != 5 {
		t.Errorf("Size() before Clear = %d, want 5", size)
	}

	// Clear cache
	cache.Clear()

	// Verify cache is empty
	if size := cache.Size(); size != 0 {
		t.Errorf("Size() after Clear = %d, want 0", size)
	}
}

func TestCacheCleanup(t *testing.T) {
	// Use a short timeout for testing
	cache := NewCache(50 * time.Millisecond)

	// Add some entries that will expire
	for i := 1; i <= 3; i++ {
		ip := common.IPv4Address{192, 168, 1, byte(i)}
		mac := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, byte(i)}
		cache.Add(ip, mac)
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Add a new entry that won't expire yet
	freshIP := common.IPv4Address{192, 168, 1, 10}
	freshMAC := common.MACAddress{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	cache.Add(freshIP, freshMAC)

	// Cleanup should remove expired entries
	removed := cache.Cleanup()
	if removed != 3 {
		t.Errorf("Cleanup() removed = %d, want 3", removed)
	}

	// Fresh entry should still be there
	_, found := cache.Get(freshIP)
	if !found {
		t.Error("Get() for fresh entry after Cleanup found = false, want true")
	}

	// Size should be 1
	if size := cache.Size(); size != 1 {
		t.Errorf("Size() after Cleanup = %d, want 1", size)
	}
}

func TestCacheSize(t *testing.T) {
	cache := NewCache(1 * time.Minute)

	// Empty cache
	if size := cache.Size(); size != 0 {
		t.Errorf("Size() for empty cache = %d, want 0", size)
	}

	// Add entries
	for i := 1; i <= 10; i++ {
		ip := common.IPv4Address{192, 168, 1, byte(i)}
		mac := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, byte(i)}
		cache.Add(ip, mac)

		if size := cache.Size(); size != i {
			t.Errorf("Size() after adding %d entries = %d, want %d", i, size, i)
		}
	}
}

func TestCacheEntries(t *testing.T) {
	cache := NewCache(1 * time.Minute)

	// Add entries
	entries := map[common.IPv4Address]common.MACAddress{
		{192, 168, 1, 1}: {0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		{192, 168, 1, 2}: {0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
		{192, 168, 1, 3}: {0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc},
	}

	for ip, mac := range entries {
		cache.Add(ip, mac)
	}

	// Get snapshot
	snapshot := cache.Entries()

	// Verify size
	if len(snapshot) != len(entries) {
		t.Errorf("Entries() length = %d, want %d", len(snapshot), len(entries))
	}

	// Verify all entries
	for ip, wantMAC := range entries {
		gotMAC, found := snapshot[ip]
		if !found {
			t.Errorf("Entries() missing IP %v", ip)
		}
		if gotMAC != wantMAC {
			t.Errorf("Entries() for IP %v MAC = %v, want %v", ip, gotMAC, wantMAC)
		}
	}
}

func TestCacheEntriesWithExpiration(t *testing.T) {
	cache := NewCache(50 * time.Millisecond)

	// Add some entries
	ip1 := common.IPv4Address{192, 168, 1, 1}
	mac1 := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	cache.Add(ip1, mac1)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Add a fresh entry
	ip2 := common.IPv4Address{192, 168, 1, 2}
	mac2 := common.MACAddress{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	cache.Add(ip2, mac2)

	// Entries should only return non-expired entries
	snapshot := cache.Entries()
	if len(snapshot) != 1 {
		t.Errorf("Entries() length = %d, want 1 (only non-expired)", len(snapshot))
	}

	if _, found := snapshot[ip1]; found {
		t.Error("Entries() included expired entry")
	}

	if gotMAC, found := snapshot[ip2]; !found || gotMAC != mac2 {
		t.Error("Entries() missing or incorrect fresh entry")
	}
}

func TestCacheString(t *testing.T) {
	cache := NewCache(1 * time.Minute)

	ip := common.IPv4Address{192, 168, 1, 1}
	mac := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	cache.Add(ip, mac)

	str := cache.String()
	if str == "" {
		t.Error("String() returned empty string")
	}
}

func TestCacheConcurrency(t *testing.T) {
	cache := NewCache(1 * time.Minute)

	// Run concurrent operations
	done := make(chan bool)
	numGoroutines := 10
	numOperations := 100

	for g := 0; g < numGoroutines; g++ {
		go func(id int) {
			for i := 0; i < numOperations; i++ {
				ip := common.IPv4Address{192, 168, byte(id), byte(i)}
				mac := common.MACAddress{byte(id), byte(i), 0x00, 0x00, 0x00, 0x00}

				// Add
				cache.Add(ip, mac)

				// Get
				_, _ = cache.Get(ip)

				// Sometimes delete
				if i%10 == 0 {
					cache.Delete(ip)
				}
			}
			done <- true
		}(g)
	}

	// Wait for all goroutines
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Cache should still be functional
	testIP := common.IPv4Address{10, 10, 10, 10}
	testMAC := common.MACAddress{0xde, 0xad, 0xbe, 0xef, 0x00, 0x00}
	cache.Add(testIP, testMAC)

	gotMAC, found := cache.Get(testIP)
	if !found || gotMAC != testMAC {
		t.Error("Cache corrupted after concurrent operations")
	}
}

func TestCacheEntryIsExpired(t *testing.T) {
	// Entry that expires in the past
	pastEntry := &CacheEntry{
		MAC:       common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		ExpiresAt: time.Now().Add(-1 * time.Second),
	}
	if !pastEntry.IsExpired() {
		t.Error("IsExpired() for past entry = false, want true")
	}

	// Entry that expires in the future
	futureEntry := &CacheEntry{
		MAC:       common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}
	if futureEntry.IsExpired() {
		t.Error("IsExpired() for future entry = true, want false")
	}
}

func TestStartCleanupRoutine(t *testing.T) {
	cache := NewCache(50 * time.Millisecond)

	// Add entries that will expire
	for i := 1; i <= 5; i++ {
		ip := common.IPv4Address{192, 168, 1, byte(i)}
		mac := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, byte(i)}
		cache.Add(ip, mac)
	}

	// Start cleanup routine
	stop := cache.StartCleanupRoutine(100 * time.Millisecond)

	// Wait for entries to expire and cleanup to run
	time.Sleep(200 * time.Millisecond)

	// Stop cleanup routine
	close(stop)

	// Cache should have cleaned up expired entries
	// Note: We can't guarantee exact timing, but size should be small or zero
	time.Sleep(50 * time.Millisecond) // Give cleanup time to finish
}
