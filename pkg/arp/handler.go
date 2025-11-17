package arp

import (
	"fmt"
	"sync"
	"time"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
	"github.com/therealutkarshpriyadarshi/network/pkg/ethernet"
)

// DefaultRequestTimeout is the default timeout for ARP requests.
const DefaultRequestTimeout = 3 * time.Second

// DefaultMaxRetries is the default number of retries for ARP requests.
const DefaultMaxRetries = 3

// Handler handles ARP protocol operations including resolving IP addresses,
// responding to requests, and maintaining the ARP cache.
type Handler struct {
	iface        *ethernet.Interface
	cache        *Cache
	localIP      common.IPv4Address
	requestQueue map[common.IPv4Address]chan common.MACAddress
	mu           sync.RWMutex
	timeout      time.Duration
	maxRetries   int
}

// NewHandler creates a new ARP handler for the given interface.
func NewHandler(iface *ethernet.Interface, localIP common.IPv4Address) *Handler {
	return &Handler{
		iface:        iface,
		cache:        NewDefaultCache(),
		localIP:      localIP,
		requestQueue: make(map[common.IPv4Address]chan common.MACAddress),
		timeout:      DefaultRequestTimeout,
		maxRetries:   DefaultMaxRetries,
	}
}

// SetTimeout sets the timeout for ARP requests.
func (h *Handler) SetTimeout(timeout time.Duration) {
	h.timeout = timeout
}

// SetMaxRetries sets the maximum number of retries for ARP requests.
func (h *Handler) SetMaxRetries(retries int) {
	h.maxRetries = retries
}

// Cache returns the ARP cache.
func (h *Handler) Cache() *Cache {
	return h.cache
}

// Resolve resolves an IP address to a MAC address using ARP.
// It first checks the cache, and if not found, sends an ARP request.
// This function blocks until a response is received or timeout occurs.
func (h *Handler) Resolve(targetIP common.IPv4Address) (common.MACAddress, error) {
	// Check cache first
	if mac, found := h.cache.Get(targetIP); found {
		return mac, nil
	}

	// Send ARP request and wait for reply
	return h.sendRequestAndWait(targetIP)
}

// sendRequestAndWait sends an ARP request and waits for a reply.
func (h *Handler) sendRequestAndWait(targetIP common.IPv4Address) (common.MACAddress, error) {
	// Create a response channel for this IP
	h.mu.Lock()
	responseChan, exists := h.requestQueue[targetIP]
	if !exists {
		responseChan = make(chan common.MACAddress, 1)
		h.requestQueue[targetIP] = responseChan
	}
	h.mu.Unlock()

	// If another goroutine is already waiting for this IP, just wait with it
	if exists {
		select {
		case mac := <-responseChan:
			return mac, nil
		case <-time.After(h.timeout):
			return common.MACAddress{}, fmt.Errorf("ARP request timeout for %s", targetIP)
		}
	}

	// Clean up when done
	defer func() {
		h.mu.Lock()
		delete(h.requestQueue, targetIP)
		close(responseChan)
		h.mu.Unlock()
	}()

	// Send ARP request with retries
	var lastErr error
	for attempt := 0; attempt < h.maxRetries; attempt++ {
		if err := h.SendRequest(targetIP); err != nil {
			lastErr = err
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Wait for response
		select {
		case mac := <-responseChan:
			return mac, nil
		case <-time.After(h.timeout / time.Duration(h.maxRetries)):
			lastErr = fmt.Errorf("ARP request timeout for %s (attempt %d/%d)", targetIP, attempt+1, h.maxRetries)
		}
	}

	return common.MACAddress{}, lastErr
}

// SendRequest sends an ARP request for the given IP address.
func (h *Handler) SendRequest(targetIP common.IPv4Address) error {
	// Create ARP request packet
	arpPacket := NewRequest(h.iface.MACAddress(), h.localIP, targetIP)

	// Create Ethernet frame with broadcast destination
	frame := ethernet.NewFrame(
		common.BroadcastMAC,
		h.iface.MACAddress(),
		common.EtherTypeARP,
		arpPacket.Serialize(),
	)

	// Send the frame
	return h.iface.WriteFrame(frame)
}

// HandlePacket processes an incoming ARP packet.
// This should be called when an ARP packet is received from the network.
func (h *Handler) HandlePacket(packet *Packet) error {
	if packet.IsRequest() {
		return h.handleRequest(packet)
	} else if packet.IsReply() {
		return h.handleReply(packet)
	}
	return fmt.Errorf("unknown ARP operation: %d", packet.Operation)
}

// handleRequest processes an ARP request.
// If the request is for our IP, send a reply.
func (h *Handler) handleRequest(packet *Packet) error {
	// Update cache with sender's information
	h.cache.Add(packet.SenderIP, packet.SenderMAC)

	// Check if the request is for our IP
	if packet.TargetIP != h.localIP {
		// Not for us, ignore
		return nil
	}

	// Send ARP reply
	return h.SendReply(packet.SenderMAC, packet.SenderIP)
}

// handleReply processes an ARP reply.
// Update the cache and notify any waiting goroutines.
func (h *Handler) handleReply(packet *Packet) error {
	// Update cache
	h.cache.Add(packet.SenderIP, packet.SenderMAC)

	// Notify any waiting goroutines
	h.mu.RLock()
	responseChan, exists := h.requestQueue[packet.SenderIP]
	h.mu.RUnlock()

	if exists {
		select {
		case responseChan <- packet.SenderMAC:
		default:
			// Channel already has a response or is closed
		}
	}

	return nil
}

// SendReply sends an ARP reply to the given MAC/IP address.
func (h *Handler) SendReply(targetMAC common.MACAddress, targetIP common.IPv4Address) error {
	// Create ARP reply packet
	arpPacket := NewReply(h.iface.MACAddress(), h.localIP, targetMAC, targetIP)

	// Create Ethernet frame
	frame := ethernet.NewFrame(
		targetMAC,
		h.iface.MACAddress(),
		common.EtherTypeARP,
		arpPacket.Serialize(),
	)

	// Send the frame
	return h.iface.WriteFrame(frame)
}

// Announce sends a gratuitous ARP to announce our IP/MAC mapping.
// This is useful when an interface comes up or changes IP address.
func (h *Handler) Announce() error {
	// Gratuitous ARP: sender IP == target IP, broadcast
	arpPacket := NewRequest(h.iface.MACAddress(), h.localIP, h.localIP)

	frame := ethernet.NewFrame(
		common.BroadcastMAC,
		h.iface.MACAddress(),
		common.EtherTypeARP,
		arpPacket.Serialize(),
	)

	return h.iface.WriteFrame(frame)
}

// Start starts the ARP handler, processing incoming ARP packets.
// This should be run in a separate goroutine.
// It returns a channel that can be closed to stop the handler.
func (h *Handler) Start() (chan<- struct{}, error) {
	stop := make(chan struct{})

	go func() {
		for {
			select {
			case <-stop:
				return
			default:
				// Read frame from interface (with timeout to check stop channel)
				frame, err := h.iface.ReadFrame()
				if err != nil {
					// Log error or handle appropriately
					time.Sleep(10 * time.Millisecond)
					continue
				}

				// Only process ARP frames
				if frame.EtherType != common.EtherTypeARP {
					continue
				}

				// Parse ARP packet
				packet, err := Parse(frame.Payload)
				if err != nil {
					// Log error or handle appropriately
					continue
				}

				// Handle the packet
				if err := h.HandlePacket(packet); err != nil {
					// Log error or handle appropriately
					continue
				}
			}
		}
	}()

	return stop, nil
}

// String returns a human-readable representation of the handler state.
func (h *Handler) String() string {
	return fmt.Sprintf("ARP Handler{Interface=%s, LocalIP=%s, MAC=%s}\n%s",
		h.iface.Name(),
		h.localIP,
		h.iface.MACAddress(),
		h.cache.String(),
	)
}
