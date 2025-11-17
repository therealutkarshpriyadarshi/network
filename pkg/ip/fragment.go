// Package ip implements IP fragmentation and reassembly as per RFC 791.
package ip

import (
	"fmt"
	"sync"
	"time"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

const (
	// MaxFragmentSize is the maximum size of a fragment payload (must be multiple of 8).
	MaxFragmentSize = 1480 // Typical MTU (1500) - IP header (20)

	// FragmentTimeout is the maximum time to wait for all fragments.
	FragmentTimeout = 60 * time.Second
)

// FragmentKey uniquely identifies a set of fragments.
type FragmentKey struct {
	Source         common.IPv4Address
	Destination    common.IPv4Address
	Identification uint16
	Protocol       common.Protocol
}

// FragmentEntry represents a set of fragments being reassembled.
type FragmentEntry struct {
	Fragments      map[uint16][]byte // offset -> data
	TotalLength    uint16            // Total length when all fragments received
	ReceivedLength uint16            // How much data we've received so far
	LastSeen       time.Time         // Last time we received a fragment
	Complete       bool              // Whether we have all fragments
}

// Fragmenter handles IP fragmentation and reassembly.
type Fragmenter struct {
	mu         sync.RWMutex
	fragments  map[FragmentKey]*FragmentEntry
	nextID     uint16 // Next identification number for outgoing fragments
	cleanupTicker *time.Ticker
	done       chan struct{}
}

// NewFragmenter creates a new fragmenter.
func NewFragmenter() *Fragmenter {
	f := &Fragmenter{
		fragments: make(map[FragmentKey]*FragmentEntry),
		nextID:    1,
		done:      make(chan struct{}),
	}

	// Start cleanup goroutine
	f.cleanupTicker = time.NewTicker(10 * time.Second)
	go f.cleanupLoop()

	return f
}

// Close stops the fragmenter.
func (f *Fragmenter) Close() {
	close(f.done)
	if f.cleanupTicker != nil {
		f.cleanupTicker.Stop()
	}
}

// cleanupLoop periodically removes expired fragment entries.
func (f *Fragmenter) cleanupLoop() {
	for {
		select {
		case <-f.cleanupTicker.C:
			f.cleanup()
		case <-f.done:
			return
		}
	}
}

// cleanup removes expired fragment entries.
func (f *Fragmenter) cleanup() {
	f.mu.Lock()
	defer f.mu.Unlock()

	now := time.Now()
	for key, entry := range f.fragments {
		if now.Sub(entry.LastSeen) > FragmentTimeout {
			delete(f.fragments, key)
		}
	}
}

// Fragment fragments a packet into MTU-sized pieces.
func (f *Fragmenter) Fragment(pkt *Packet, mtu int) ([]*Packet, error) {
	// Calculate maximum payload size per fragment
	headerSize := int(pkt.IHL) * 4
	maxPayloadSize := mtu - headerSize

	// Must be multiple of 8 bytes (fragment offset is in 8-byte units)
	maxPayloadSize = (maxPayloadSize / 8) * 8

	if maxPayloadSize <= 0 {
		return nil, fmt.Errorf("MTU too small: %d", mtu)
	}

	payloadLen := len(pkt.Payload)
	if payloadLen <= maxPayloadSize {
		// No fragmentation needed
		return []*Packet{pkt}, nil
	}

	// Assign identification number if not set
	if pkt.Identification == 0 {
		f.mu.Lock()
		pkt.Identification = f.nextID
		f.nextID++
		f.mu.Unlock()
	}

	// Create fragments
	var fragments []*Packet
	offset := 0

	for offset < payloadLen {
		end := offset + maxPayloadSize
		lastFragment := false

		if end >= payloadLen {
			end = payloadLen
			lastFragment = true
		}

		// Create fragment
		frag := &Packet{
			Version:        pkt.Version,
			IHL:            pkt.IHL,
			DSCP:           pkt.DSCP,
			ECN:            pkt.ECN,
			Identification: pkt.Identification,
			Flags:          pkt.Flags,
			FragmentOffset: uint16(offset / 8), // Offset is in 8-byte units
			TTL:            pkt.TTL,
			Protocol:       pkt.Protocol,
			Source:         pkt.Source,
			Destination:    pkt.Destination,
			Options:        pkt.Options, // Copy options to first fragment only
			Payload:        pkt.Payload[offset:end],
		}

		// Set More Fragments flag for all but last fragment
		if !lastFragment {
			frag.Flags |= FlagMoreFragments
		}

		// Only first fragment gets options
		if offset > 0 {
			frag.Options = nil
			frag.IHL = 5
		}

		fragments = append(fragments, frag)
		offset = end
	}

	return fragments, nil
}

// Reassemble attempts to reassemble fragments into a complete packet.
// Returns the reassembled packet if complete, nil otherwise.
func (f *Fragmenter) Reassemble(pkt *Packet) (*Packet, error) {
	// If not a fragment, return as-is
	if !pkt.IsFragment() {
		return pkt, nil
	}

	key := FragmentKey{
		Source:         pkt.Source,
		Destination:    pkt.Destination,
		Identification: pkt.Identification,
		Protocol:       pkt.Protocol,
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	// Get or create fragment entry
	entry, exists := f.fragments[key]
	if !exists {
		entry = &FragmentEntry{
			Fragments: make(map[uint16][]byte),
			LastSeen:  time.Now(),
		}
		f.fragments[key] = entry
	}

	// Update last seen time
	entry.LastSeen = time.Now()

	// Calculate byte offset
	byteOffset := uint16(pkt.FragmentOffset * 8)

	// Store fragment
	entry.Fragments[byteOffset] = pkt.Payload

	// Check if this is the last fragment
	if (pkt.Flags & FlagMoreFragments) == 0 {
		// This is the last fragment, we now know the total length
		entry.TotalLength = byteOffset + uint16(len(pkt.Payload))
	}

	// Calculate how much data we have
	var receivedLength uint16
	for _, data := range entry.Fragments {
		receivedLength += uint16(len(data))
	}
	entry.ReceivedLength = receivedLength

	// Check if we have all fragments
	if entry.TotalLength > 0 && entry.ReceivedLength >= entry.TotalLength {
		// Reassemble the packet
		reassembled := &Packet{
			Version:        pkt.Version,
			IHL:            pkt.IHL,
			DSCP:           pkt.DSCP,
			ECN:            pkt.ECN,
			Identification: pkt.Identification,
			Flags:          0, // Clear fragment flags
			FragmentOffset: 0,
			TTL:            pkt.TTL,
			Protocol:       pkt.Protocol,
			Source:         pkt.Source,
			Destination:    pkt.Destination,
			Options:        pkt.Options,
			Payload:        make([]byte, entry.TotalLength),
		}

		// Copy fragments into payload
		for offset, data := range entry.Fragments {
			copy(reassembled.Payload[offset:], data)
		}

		// Verify we have all the data (no holes)
		if !f.verifyNoHoles(entry, entry.TotalLength) {
			return nil, nil // Still waiting for more fragments
		}

		// Remove from fragments map
		delete(f.fragments, key)

		return reassembled, nil
	}

	// Still waiting for more fragments
	return nil, nil
}

// verifyNoHoles checks if there are any gaps in the received fragments.
func (f *Fragmenter) verifyNoHoles(entry *FragmentEntry, totalLength uint16) bool {
	// Create a bitmap to track which bytes we have
	received := make([]bool, totalLength)

	for offset, data := range entry.Fragments {
		for i := 0; i < len(data); i++ {
			if int(offset)+i < len(received) {
				received[int(offset)+i] = true
			}
		}
	}

	// Check for holes
	for i := 0; i < len(received); i++ {
		if !received[i] {
			return false
		}
	}

	return true
}
