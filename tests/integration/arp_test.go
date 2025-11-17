// +build integration

// Integration tests for ARP protocol
//
// These tests require:
// - Root/sudo privileges (for raw sockets)
// - A network interface to test with
// - Network connectivity
//
// Run with: sudo go test -tags=integration ./tests/integration/...

package integration

import (
	"os"
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/network/pkg/arp"
	"github.com/therealutkarshpriyadarshi/network/pkg/common"
	"github.com/therealutkarshpriyadarshi/network/pkg/ethernet"
)

// getTestInterface returns the network interface to use for testing.
// Set the TEST_INTERFACE environment variable to override the default.
func getTestInterface() string {
	if iface := os.Getenv("TEST_INTERFACE"); iface != "" {
		return iface
	}
	return "eth0" // default
}

// getLocalIP returns the local IP to use for testing.
// Set the TEST_LOCAL_IP environment variable to override.
func getLocalIP() (common.IPv4Address, error) {
	if ip := os.Getenv("TEST_LOCAL_IP"); ip != "" {
		return common.ParseIPv4(ip)
	}
	return common.ParseIPv4("192.168.1.100") // default
}

// getTargetIP returns a known-good target IP for testing.
// Set the TEST_TARGET_IP environment variable (usually your gateway).
func getTargetIP() (common.IPv4Address, error) {
	if ip := os.Getenv("TEST_TARGET_IP"); ip != "" {
		return common.ParseIPv4(ip)
	}
	return common.ParseIPv4("192.168.1.1") // default (common gateway)
}

// skipIfNotRoot skips the test if not running with root privileges
func skipIfNotRoot(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("This test requires root privileges (try: sudo go test -tags=integration ...)")
	}
}

func TestARPResolution(t *testing.T) {
	skipIfNotRoot(t)

	ifaceName := getTestInterface()
	localIP, err := getLocalIP()
	if err != nil {
		t.Fatalf("Invalid local IP: %v", err)
	}

	targetIP, err := getTargetIP()
	if err != nil {
		t.Fatalf("Invalid target IP: %v", err)
	}

	t.Logf("Testing ARP resolution on %s", ifaceName)
	t.Logf("Local IP: %s, Target IP: %s", localIP, targetIP)

	// Open interface
	iface, err := ethernet.OpenInterface(ifaceName)
	if err != nil {
		t.Fatalf("Failed to open interface: %v (is %s available?)", err, ifaceName)
	}
	defer iface.Close()

	// Create ARP handler
	handler := arp.NewHandler(iface, localIP)
	handler.SetTimeout(5 * time.Second)
	handler.SetMaxRetries(3)

	// Start handler
	stop, err := handler.Start()
	if err != nil {
		t.Fatalf("Failed to start handler: %v", err)
	}
	defer close(stop)

	// Give handler time to start
	time.Sleep(100 * time.Millisecond)

	// Resolve target IP
	startTime := time.Now()
	mac, err := handler.Resolve(targetIP)
	elapsed := time.Since(startTime)

	if err != nil {
		t.Fatalf("Failed to resolve %s: %v (is the target reachable?)", targetIP, err)
	}

	t.Logf("Successfully resolved %s to %s in %v", targetIP, mac, elapsed)

	// Verify MAC is not zero
	zeroMAC := common.MACAddress{}
	if mac == zeroMAC {
		t.Error("Resolved MAC is zero (00:00:00:00:00:00)")
	}

	// Verify cache was updated
	cachedMAC, found := handler.Cache().Get(targetIP)
	if !found {
		t.Error("Target IP not found in cache after resolution")
	}
	if cachedMAC != mac {
		t.Errorf("Cached MAC %s doesn't match resolved MAC %s", cachedMAC, mac)
	}
}

func TestARPCacheHit(t *testing.T) {
	skipIfNotRoot(t)

	ifaceName := getTestInterface()
	localIP, err := getLocalIP()
	if err != nil {
		t.Fatalf("Invalid local IP: %v", err)
	}

	targetIP, err := getTargetIP()
	if err != nil {
		t.Fatalf("Invalid target IP: %v", err)
	}

	// Open interface
	iface, err := ethernet.OpenInterface(ifaceName)
	if err != nil {
		t.Fatalf("Failed to open interface: %v", err)
	}
	defer iface.Close()

	// Create ARP handler
	handler := arp.NewHandler(iface, localIP)
	handler.SetTimeout(5 * time.Second)

	// Start handler
	stop, err := handler.Start()
	if err != nil {
		t.Fatalf("Failed to start handler: %v", err)
	}
	defer close(stop)

	time.Sleep(100 * time.Millisecond)

	// First resolution
	mac1, err := handler.Resolve(targetIP)
	if err != nil {
		t.Fatalf("First resolution failed: %v", err)
	}

	// Second resolution (should hit cache)
	startTime := time.Now()
	mac2, err := handler.Resolve(targetIP)
	elapsed := time.Since(startTime)

	if err != nil {
		t.Fatalf("Second resolution failed: %v", err)
	}

	if mac1 != mac2 {
		t.Errorf("MACs don't match: first=%s, second=%s", mac1, mac2)
	}

	// Cache hit should be very fast (< 1ms typically)
	if elapsed > 10*time.Millisecond {
		t.Logf("Warning: Cache hit took %v (expected < 10ms)", elapsed)
	} else {
		t.Logf("Cache hit took %v", elapsed)
	}
}

func TestARPGratuitous(t *testing.T) {
	skipIfNotRoot(t)

	ifaceName := getTestInterface()
	localIP, err := getLocalIP()
	if err != nil {
		t.Fatalf("Invalid local IP: %v", err)
	}

	// Open interface
	iface, err := ethernet.OpenInterface(ifaceName)
	if err != nil {
		t.Fatalf("Failed to open interface: %v", err)
	}
	defer iface.Close()

	// Create ARP handler
	handler := arp.NewHandler(iface, localIP)

	// Start handler
	stop, err := handler.Start()
	if err != nil {
		t.Fatalf("Failed to start handler: %v", err)
	}
	defer close(stop)

	time.Sleep(100 * time.Millisecond)

	// Send gratuitous ARP
	err = handler.Announce()
	if err != nil {
		t.Fatalf("Failed to send gratuitous ARP: %v", err)
	}

	t.Log("Successfully sent gratuitous ARP")
}

func TestARPTimeout(t *testing.T) {
	skipIfNotRoot(t)

	ifaceName := getTestInterface()
	localIP, err := getLocalIP()
	if err != nil {
		t.Fatalf("Invalid local IP: %v", err)
	}

	// Use a non-existent IP on the local network
	nonExistentIP := common.IPv4Address{192, 168, 255, 254}

	// Open interface
	iface, err := ethernet.OpenInterface(ifaceName)
	if err != nil {
		t.Fatalf("Failed to open interface: %v", err)
	}
	defer iface.Close()

	// Create ARP handler with short timeout
	handler := arp.NewHandler(iface, localIP)
	handler.SetTimeout(1 * time.Second)
	handler.SetMaxRetries(1)

	// Start handler
	stop, err := handler.Start()
	if err != nil {
		t.Fatalf("Failed to start handler: %v", err)
	}
	defer close(stop)

	time.Sleep(100 * time.Millisecond)

	// Try to resolve non-existent IP (should timeout)
	startTime := time.Now()
	_, err = handler.Resolve(nonExistentIP)
	elapsed := time.Since(startTime)

	if err == nil {
		t.Error("Expected timeout error, got nil")
	}

	t.Logf("Timeout occurred after %v (error: %v)", elapsed, err)

	// Verify timeout happened in reasonable time
	if elapsed < 500*time.Millisecond {
		t.Error("Timeout happened too quickly")
	}
	if elapsed > 3*time.Second {
		t.Error("Timeout took too long")
	}
}

func TestARPMultipleTargets(t *testing.T) {
	skipIfNotRoot(t)

	ifaceName := getTestInterface()
	localIP, err := getLocalIP()
	if err != nil {
		t.Fatalf("Invalid local IP: %v", err)
	}

	// Open interface
	iface, err := ethernet.OpenInterface(ifaceName)
	if err != nil {
		t.Fatalf("Failed to open interface: %v", err)
	}
	defer iface.Close()

	// Create ARP handler
	handler := arp.NewHandler(iface, localIP)
	handler.SetTimeout(5 * time.Second)

	// Start handler
	stop, err := handler.Start()
	if err != nil {
		t.Fatalf("Failed to start handler: %v", err)
	}
	defer close(stop)

	time.Sleep(100 * time.Millisecond)

	// Try to resolve multiple common IPs on the network
	// Note: Some may not exist, which is fine
	testIPs := []string{
		"192.168.1.1", // Common gateway
		// Add more IPs if you know they exist on your network
	}

	resolved := 0
	for _, ipStr := range testIPs {
		ip, err := common.ParseIPv4(ipStr)
		if err != nil {
			continue
		}

		mac, err := handler.Resolve(ip)
		if err == nil {
			t.Logf("Resolved %s -> %s", ip, mac)
			resolved++
		} else {
			t.Logf("Could not resolve %s: %v", ip, err)
		}
	}

	if resolved == 0 {
		t.Skip("Could not resolve any test IPs (network may not be configured)")
	}

	// Verify cache has entries
	cacheSize := handler.Cache().Size()
	if cacheSize == 0 {
		t.Error("Cache is empty after resolving IPs")
	}

	t.Logf("Cache has %d entries", cacheSize)
}
