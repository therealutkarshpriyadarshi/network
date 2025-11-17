package ip

import (
	"bytes"
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

func TestFragmenter_Fragment(t *testing.T) {
	f := NewFragmenter()
	defer f.Close()

	srcIP, _ := common.ParseIPv4("192.168.1.100")
	dstIP, _ := common.ParseIPv4("192.168.1.1")

	// Create a large payload that will require fragmentation
	payload := make([]byte, 3000)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	pkt := NewPacket(srcIP, dstIP, common.ProtocolICMP, payload)
	pkt.Identification = 0x1234

	// Fragment with MTU of 1500
	fragments, err := f.Fragment(pkt, 1500)
	if err != nil {
		t.Fatalf("Fragment() error = %v", err)
	}

	if len(fragments) < 2 {
		t.Errorf("Expected multiple fragments, got %d", len(fragments))
	}

	// Check that all fragments have the same ID
	for i, frag := range fragments {
		if frag.Identification != 0x1234 {
			t.Errorf("Fragment %d: ID = 0x%04x, want 0x1234", i, frag.Identification)
		}
	}

	// Check that all but last fragment have MoreFragments flag
	for i := 0; i < len(fragments)-1; i++ {
		if (fragments[i].Flags & FlagMoreFragments) == 0 {
			t.Errorf("Fragment %d: missing MoreFragments flag", i)
		}
	}

	// Check that last fragment doesn't have MoreFragments flag
	lastFrag := fragments[len(fragments)-1]
	if (lastFrag.Flags & FlagMoreFragments) != 0 {
		t.Error("Last fragment: has MoreFragments flag")
	}

	// Verify total payload length
	totalPayload := 0
	for _, frag := range fragments {
		totalPayload += len(frag.Payload)
	}
	if totalPayload != len(payload) {
		t.Errorf("Total payload = %d, want %d", totalPayload, len(payload))
	}
}

func TestFragmenter_Fragment_NoFragmentation(t *testing.T) {
	f := NewFragmenter()
	defer f.Close()

	srcIP, _ := common.ParseIPv4("192.168.1.100")
	dstIP, _ := common.ParseIPv4("192.168.1.1")

	// Small payload that doesn't need fragmentation
	payload := []byte("Hello, World!")
	pkt := NewPacket(srcIP, dstIP, common.ProtocolICMP, payload)

	fragments, err := f.Fragment(pkt, 1500)
	if err != nil {
		t.Fatalf("Fragment() error = %v", err)
	}

	if len(fragments) != 1 {
		t.Errorf("Expected 1 fragment, got %d", len(fragments))
	}

	if !bytes.Equal(fragments[0].Payload, payload) {
		t.Error("Payload mismatch")
	}
}

func TestFragmenter_Reassemble(t *testing.T) {
	f := NewFragmenter()
	defer f.Close()

	srcIP, _ := common.ParseIPv4("192.168.1.100")
	dstIP, _ := common.ParseIPv4("192.168.1.1")

	// Create a large payload
	originalPayload := make([]byte, 3000)
	for i := range originalPayload {
		originalPayload[i] = byte(i % 256)
	}

	pkt := NewPacket(srcIP, dstIP, common.ProtocolICMP, originalPayload)
	pkt.Identification = 0x5678

	// Fragment the packet
	fragments, err := f.Fragment(pkt, 1500)
	if err != nil {
		t.Fatalf("Fragment() error = %v", err)
	}

	// Reassemble the fragments
	var reassembled *Packet
	for i, frag := range fragments {
		result, err := f.Reassemble(frag)
		if err != nil {
			t.Fatalf("Reassemble() error = %v at fragment %d", err, i)
		}

		// Only the last fragment should return the reassembled packet
		if i < len(fragments)-1 {
			if result != nil {
				t.Errorf("Reassemble() returned packet at fragment %d, want nil", i)
			}
		} else {
			if result == nil {
				t.Error("Reassemble() returned nil for last fragment")
			} else {
				reassembled = result
			}
		}
	}

	// Verify reassembled packet
	if reassembled == nil {
		t.Fatal("Failed to reassemble packet")
	}

	if !bytes.Equal(reassembled.Payload, originalPayload) {
		t.Errorf("Reassembled payload length = %d, want %d", len(reassembled.Payload), len(originalPayload))
		// Check first few bytes
		for i := 0; i < 10 && i < len(reassembled.Payload); i++ {
			if reassembled.Payload[i] != originalPayload[i] {
				t.Errorf("Payload mismatch at byte %d: got 0x%02x, want 0x%02x",
					i, reassembled.Payload[i], originalPayload[i])
			}
		}
	}

	if reassembled.IsFragment() {
		t.Error("Reassembled packet is still marked as fragment")
	}
}

func TestFragmenter_Reassemble_OutOfOrder(t *testing.T) {
	f := NewFragmenter()
	defer f.Close()

	srcIP, _ := common.ParseIPv4("192.168.1.100")
	dstIP, _ := common.ParseIPv4("192.168.1.1")

	// Create a large payload
	originalPayload := make([]byte, 3000)
	for i := range originalPayload {
		originalPayload[i] = byte(i % 256)
	}

	pkt := NewPacket(srcIP, dstIP, common.ProtocolICMP, originalPayload)
	pkt.Identification = 0xABCD

	// Fragment the packet
	fragments, err := f.Fragment(pkt, 1500)
	if err != nil {
		t.Fatalf("Fragment() error = %v", err)
	}

	if len(fragments) < 3 {
		t.Skip("Need at least 3 fragments for out-of-order test")
	}

	// Reassemble in reverse order
	var reassembled *Packet
	for i := len(fragments) - 1; i >= 0; i-- {
		result, err := f.Reassemble(fragments[i])
		if err != nil {
			t.Fatalf("Reassemble() error = %v at fragment %d", err, i)
		}
		if result != nil {
			reassembled = result
		}
	}

	// Verify reassembled packet
	if reassembled == nil {
		t.Fatal("Failed to reassemble packet")
	}

	if !bytes.Equal(reassembled.Payload, originalPayload) {
		t.Error("Reassembled payload mismatch")
	}
}

func TestFragmenter_Reassemble_NonFragment(t *testing.T) {
	f := NewFragmenter()
	defer f.Close()

	srcIP, _ := common.ParseIPv4("192.168.1.100")
	dstIP, _ := common.ParseIPv4("192.168.1.1")

	// Non-fragmented packet
	payload := []byte("Hello, World!")
	pkt := NewPacket(srcIP, dstIP, common.ProtocolICMP, payload)

	result, err := f.Reassemble(pkt)
	if err != nil {
		t.Fatalf("Reassemble() error = %v", err)
	}

	if result != pkt {
		t.Error("Reassemble() should return the same packet for non-fragments")
	}
}

func TestFragmenter_Cleanup(t *testing.T) {
	f := NewFragmenter()
	defer f.Close()

	srcIP, _ := common.ParseIPv4("192.168.1.100")
	dstIP, _ := common.ParseIPv4("192.168.1.1")

	// Create a fragment
	pkt := NewPacket(srcIP, dstIP, common.ProtocolICMP, []byte("test"))
	pkt.Identification = 0x9999
	pkt.Flags = FlagMoreFragments
	pkt.FragmentOffset = 0

	// Reassemble (will be incomplete)
	_, err := f.Reassemble(pkt)
	if err != nil {
		t.Fatalf("Reassemble() error = %v", err)
	}

	// Check that entry was created
	f.mu.RLock()
	initialCount := len(f.fragments)
	f.mu.RUnlock()

	if initialCount == 0 {
		t.Error("Fragment entry was not created")
	}

	// Manually set last seen to past
	f.mu.Lock()
	for _, entry := range f.fragments {
		entry.LastSeen = time.Now().Add(-2 * FragmentTimeout)
	}
	f.mu.Unlock()

	// Run cleanup
	f.cleanup()

	// Check that entry was removed
	f.mu.RLock()
	finalCount := len(f.fragments)
	f.mu.RUnlock()

	if finalCount != 0 {
		t.Errorf("Cleanup did not remove expired entries: %d remaining", finalCount)
	}
}

func BenchmarkFragment(b *testing.B) {
	f := NewFragmenter()
	defer f.Close()

	srcIP, _ := common.ParseIPv4("192.168.1.100")
	dstIP, _ := common.ParseIPv4("192.168.1.1")
	payload := make([]byte, 3000)
	pkt := NewPacket(srcIP, dstIP, common.ProtocolICMP, payload)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = f.Fragment(pkt, 1500)
	}
}

func BenchmarkReassemble(b *testing.B) {
	f := NewFragmenter()
	defer f.Close()

	srcIP, _ := common.ParseIPv4("192.168.1.100")
	dstIP, _ := common.ParseIPv4("192.168.1.1")
	payload := make([]byte, 3000)
	pkt := NewPacket(srcIP, dstIP, common.ProtocolICMP, payload)
	fragments, _ := f.Fragment(pkt, 1500)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Create a new fragmenter for each iteration
		f := NewFragmenter()
		for _, frag := range fragments {
			_, _ = f.Reassemble(frag)
		}
		f.Close()
	}
}
