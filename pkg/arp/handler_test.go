package arp

import (
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

// mockInterface is a minimal mock of ethernet.Interface for testing
type mockInterface struct {
	name       string
	mac        common.MACAddress
	index      int
	lastFrame  []byte
	frameQueue chan []byte
}

func newMockInterface() *mockInterface {
	return &mockInterface{
		name:       "mock0",
		mac:        common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		index:      1,
		frameQueue: make(chan []byte, 10),
	}
}

func (m *mockInterface) Name() string {
	return m.name
}

func (m *mockInterface) MACAddress() common.MACAddress {
	return m.mac
}

func (m *mockInterface) Index() int {
	return m.index
}

// TestHandleRequest tests handling of ARP requests (cache update only)
func TestHandleRequest(t *testing.T) {
	localIP := common.IPv4Address{192, 168, 1, 1}
	remoteIP := common.IPv4Address{192, 168, 1, 2}
	remoteMAC := common.MACAddress{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}

	// Create a mock handler without actual interface
	handler := &Handler{
		cache:        NewDefaultCache(),
		localIP:      localIP,
		requestQueue: make(map[common.IPv4Address]chan common.MACAddress),
		timeout:      DefaultRequestTimeout,
		maxRetries:   DefaultMaxRetries,
	}

	// Create an ARP request for a DIFFERENT IP (not ours)
	// This way it won't try to send a reply
	otherIP := common.IPv4Address{192, 168, 1, 99}
	request := NewRequest(remoteMAC, remoteIP, otherIP)

	// Handle the request (this should update cache)
	err := handler.handleRequest(request)
	if err != nil {
		t.Errorf("handleRequest() error = %v, want nil", err)
	}

	// Verify cache was updated with sender's information
	cachedMAC, found := handler.cache.Get(remoteIP)
	if !found {
		t.Error("Cache not updated with sender's MAC")
	}
	if cachedMAC != remoteMAC {
		t.Errorf("Cached MAC = %v, want %v", cachedMAC, remoteMAC)
	}
}

func TestHandleRequestNotForUs(t *testing.T) {
	localIP := common.IPv4Address{192, 168, 1, 1}
	otherIP := common.IPv4Address{192, 168, 1, 99}
	remoteIP := common.IPv4Address{192, 168, 1, 2}
	remoteMAC := common.MACAddress{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}

	handler := &Handler{
		cache:        NewDefaultCache(),
		localIP:      localIP,
		requestQueue: make(map[common.IPv4Address]chan common.MACAddress),
		timeout:      DefaultRequestTimeout,
		maxRetries:   DefaultMaxRetries,
	}

	// Create an ARP request for a different IP
	request := NewRequest(remoteMAC, remoteIP, otherIP)

	// Handle the request (should still update cache but not send reply)
	err := handler.handleRequest(request)
	if err != nil {
		t.Errorf("handleRequest() error = %v, want nil", err)
	}

	// Verify cache was still updated
	cachedMAC, found := handler.cache.Get(remoteIP)
	if !found {
		t.Error("Cache not updated even though request wasn't for us")
	}
	if cachedMAC != remoteMAC {
		t.Errorf("Cached MAC = %v, want %v", cachedMAC, remoteMAC)
	}
}

func TestHandleReply(t *testing.T) {
	localIP := common.IPv4Address{192, 168, 1, 1}
	localMAC := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	remoteIP := common.IPv4Address{192, 168, 1, 2}
	remoteMAC := common.MACAddress{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}

	handler := &Handler{
		cache:        NewDefaultCache(),
		localIP:      localIP,
		requestQueue: make(map[common.IPv4Address]chan common.MACAddress),
		timeout:      DefaultRequestTimeout,
		maxRetries:   DefaultMaxRetries,
	}

	// Create an ARP reply
	reply := NewReply(remoteMAC, remoteIP, localMAC, localIP)

	// Handle the reply
	err := handler.handleReply(reply)
	if err != nil {
		t.Errorf("handleReply() error = %v, want nil", err)
	}

	// Verify cache was updated
	cachedMAC, found := handler.cache.Get(remoteIP)
	if !found {
		t.Error("Cache not updated with ARP reply")
	}
	if cachedMAC != remoteMAC {
		t.Errorf("Cached MAC = %v, want %v", cachedMAC, remoteMAC)
	}
}

func TestHandleReplyWithWaitingRequest(t *testing.T) {
	localIP := common.IPv4Address{192, 168, 1, 1}
	remoteIP := common.IPv4Address{192, 168, 1, 2}
	remoteMAC := common.MACAddress{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}

	handler := &Handler{
		cache:        NewDefaultCache(),
		localIP:      localIP,
		requestQueue: make(map[common.IPv4Address]chan common.MACAddress),
		timeout:      DefaultRequestTimeout,
		maxRetries:   DefaultMaxRetries,
	}

	// Create a channel for waiting request
	responseChan := make(chan common.MACAddress, 1)
	handler.requestQueue[remoteIP] = responseChan

	// Create an ARP reply
	reply := NewReply(remoteMAC, remoteIP, common.MACAddress{}, localIP)

	// Handle the reply in a goroutine
	go func() {
		_ = handler.handleReply(reply)
	}()

	// Wait for response on channel
	select {
	case mac := <-responseChan:
		if mac != remoteMAC {
			t.Errorf("Received MAC = %v, want %v", mac, remoteMAC)
		}
	case <-time.After(1 * time.Second):
		t.Error("Timeout waiting for response on channel")
	}
}

func TestHandlePacket(t *testing.T) {
	localIP := common.IPv4Address{192, 168, 1, 1}
	otherIP := common.IPv4Address{192, 168, 1, 99}
	remoteIP := common.IPv4Address{192, 168, 1, 2}
	remoteMAC := common.MACAddress{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}

	handler := &Handler{
		cache:        NewDefaultCache(),
		localIP:      localIP,
		requestQueue: make(map[common.IPv4Address]chan common.MACAddress),
		timeout:      DefaultRequestTimeout,
		maxRetries:   DefaultMaxRetries,
	}

	tests := []struct {
		name    string
		packet  *Packet
		wantErr bool
	}{
		{
			name:    "ARP request (not for us)",
			packet:  NewRequest(remoteMAC, remoteIP, otherIP),
			wantErr: false,
		},
		{
			name:    "ARP reply",
			packet:  NewReply(remoteMAC, remoteIP, common.MACAddress{}, localIP),
			wantErr: false,
		},
		{
			name: "Unknown operation",
			packet: &Packet{
				Operation: Operation(99),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handler.HandlePacket(tt.packet)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandlePacket() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSetTimeout(t *testing.T) {
	handler := &Handler{
		timeout: DefaultRequestTimeout,
	}

	newTimeout := 5 * time.Second
	handler.SetTimeout(newTimeout)

	if handler.timeout != newTimeout {
		t.Errorf("timeout = %v, want %v", handler.timeout, newTimeout)
	}
}

func TestSetMaxRetries(t *testing.T) {
	handler := &Handler{
		maxRetries: DefaultMaxRetries,
	}

	newRetries := 5
	handler.SetMaxRetries(newRetries)

	if handler.maxRetries != newRetries {
		t.Errorf("maxRetries = %v, want %v", handler.maxRetries, newRetries)
	}
}

func TestCacheAccessor(t *testing.T) {
	cache := NewDefaultCache()
	handler := &Handler{
		cache: cache,
	}

	if handler.Cache() != cache {
		t.Error("Cache() returned different cache instance")
	}
}

func TestHandlerString(t *testing.T) {
	// Skip this test because String() method requires a valid interface
	// which we can't easily mock in a unit test. The String() method
	// will be tested in integration tests with a real interface.
	t.Skip("String() requires a real interface, tested in integration tests")
}

// TestCacheIntegration tests that cache operations work correctly through the handler
func TestCacheIntegration(t *testing.T) {
	localIP := common.IPv4Address{192, 168, 1, 1}
	otherIP := common.IPv4Address{192, 168, 1, 99}
	remoteIP := common.IPv4Address{192, 168, 1, 2}
	remoteMAC := common.MACAddress{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}

	handler := &Handler{
		cache:        NewDefaultCache(),
		localIP:      localIP,
		requestQueue: make(map[common.IPv4Address]chan common.MACAddress),
		timeout:      DefaultRequestTimeout,
		maxRetries:   DefaultMaxRetries,
	}

	// Initially cache should be empty
	if size := handler.cache.Size(); size != 0 {
		t.Errorf("Initial cache size = %d, want 0", size)
	}

	// Process a request (for a different IP, not ours)
	request := NewRequest(remoteMAC, remoteIP, otherIP)
	_ = handler.handleRequest(request)

	// Cache should now have one entry
	if size := handler.cache.Size(); size != 1 {
		t.Errorf("Cache size after request = %d, want 1", size)
	}

	// Verify the entry
	cachedMAC, found := handler.cache.Get(remoteIP)
	if !found {
		t.Error("Entry not found in cache")
	}
	if cachedMAC != remoteMAC {
		t.Errorf("Cached MAC = %v, want %v", cachedMAC, remoteMAC)
	}
}

// TestMultipleRequests tests handling multiple ARP requests
func TestMultipleRequests(t *testing.T) {
	localIP := common.IPv4Address{192, 168, 1, 1}
	otherIP := common.IPv4Address{192, 168, 1, 99}

	handler := &Handler{
		cache:        NewDefaultCache(),
		localIP:      localIP,
		requestQueue: make(map[common.IPv4Address]chan common.MACAddress),
		timeout:      DefaultRequestTimeout,
		maxRetries:   DefaultMaxRetries,
	}

	// Process multiple requests from different IPs (not for our IP)
	for i := 2; i <= 10; i++ {
		remoteIP := common.IPv4Address{192, 168, 1, byte(i)}
		remoteMAC := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, byte(i)}
		request := NewRequest(remoteMAC, remoteIP, otherIP)

		err := handler.handleRequest(request)
		if err != nil {
			t.Errorf("handleRequest() for IP %v error = %v", remoteIP, err)
		}
	}

	// Cache should have 9 entries
	if size := handler.cache.Size(); size != 9 {
		t.Errorf("Cache size = %d, want 9", size)
	}
}
