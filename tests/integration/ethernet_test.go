// +build integration

// Integration tests for Ethernet protocol
//
// These tests verify Ethernet frame handling, MAC addressing, and EtherType processing.
//
// Run with: go test -tags=integration ./tests/integration/...

package integration

import (
	"bytes"
	"testing"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
	"github.com/therealutkarshpriyadarshi/network/pkg/ethernet"
)

// TestEthernetFrameTypes tests different EtherType values.
func TestEthernetFrameTypes(t *testing.T) {
	src := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dst := common.MACAddress{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}
	payload := []byte("test payload")

	tests := []struct {
		name      string
		etherType common.EtherType
	}{
		{"IPv4", common.EtherTypeIPv4},
		{"IPv6", common.EtherTypeIPv6},
		{"ARP", common.EtherTypeARP},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := ethernet.NewFrame(dst, src, tt.etherType, payload)
			data := frame.Serialize()

			parsed, err := ethernet.Parse(data)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if parsed.EtherType != tt.etherType {
				t.Errorf("EtherType = %v, want %v", parsed.EtherType, tt.etherType)
			}
			if parsed.Source != src {
				t.Errorf("Source = %v, want %v", parsed.Source, src)
			}
			if parsed.Destination != dst {
				t.Errorf("Destination = %v, want %v", parsed.Destination, dst)
			}
		})
	}
}

// TestEthernetBroadcast tests broadcast MAC address handling.
func TestEthernetBroadcast(t *testing.T) {
	src := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	payload := []byte("broadcast message")

	frame := ethernet.NewFrame(common.BroadcastMAC, src, common.EtherTypeIPv4, payload)
	data := frame.Serialize()

	parsed, err := ethernet.Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.Destination != common.BroadcastMAC {
		t.Errorf("Destination = %v, want broadcast %v", parsed.Destination, common.BroadcastMAC)
	}

	// Check if IsBroadcast method works (if it exists)
	expectedBroadcast := common.MACAddress{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}
	if parsed.Destination != expectedBroadcast {
		t.Error("Broadcast MAC not correctly set")
	}
}

// TestEthernetMulticast tests multicast MAC address handling.
func TestEthernetMulticast(t *testing.T) {
	src := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	// Multicast MAC: LSB of first octet is 1
	multicastDst := common.MACAddress{0x01, 0x00, 0x5E, 0x00, 0x00, 0x01}
	payload := []byte("multicast")

	frame := ethernet.NewFrame(multicastDst, src, common.EtherTypeIPv4, payload)
	data := frame.Serialize()

	parsed, err := ethernet.Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.Destination != multicastDst {
		t.Errorf("Destination = %v, want %v", parsed.Destination, multicastDst)
	}

	// Verify it's a multicast address (LSB of first byte is 1)
	if (parsed.Destination[0] & 0x01) == 0 {
		t.Error("Destination should be multicast (LSB of first byte should be 1)")
	}
}

// TestEthernetMinimumPayload tests minimum payload padding.
func TestEthernetMinimumPayload(t *testing.T) {
	src := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dst := common.MACAddress{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}

	// Small payload (less than minimum)
	smallPayload := []byte{0x01, 0x02}

	frame := ethernet.NewFrame(dst, src, common.EtherTypeIPv4, smallPayload)
	data := frame.Serialize()

	// Ethernet minimum frame size is 64 bytes (header + 46 byte min payload)
	// Header is 14 bytes, so minimum payload is 46 bytes
	const minFrameSize = 60 // Without FCS
	if len(data) < minFrameSize {
		t.Errorf("Frame size = %d, want at least %d (should be padded)", len(data), minFrameSize)
	}

	// Parse and verify original payload is preserved
	parsed, err := ethernet.Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// The payload should contain at least our original data
	if len(parsed.Payload) < len(smallPayload) {
		t.Error("Payload was truncated")
	}
	if !bytes.Equal(parsed.Payload[:len(smallPayload)], smallPayload) {
		t.Error("Original payload data not preserved")
	}
}

// TestEthernetMaximumPayload tests maximum payload size.
func TestEthernetMaximumPayload(t *testing.T) {
	src := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dst := common.MACAddress{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}

	// Maximum Ethernet payload is 1500 bytes (MTU)
	maxPayload := make([]byte, 1500)
	for i := range maxPayload {
		maxPayload[i] = byte(i % 256)
	}

	frame := ethernet.NewFrame(dst, src, common.EtherTypeIPv4, maxPayload)
	data := frame.Serialize()

	parsed, err := ethernet.Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !bytes.Equal(parsed.Payload, maxPayload) {
		t.Error("Maximum payload data mismatch")
	}

	t.Logf("Successfully handled maximum payload (%d bytes)", len(maxPayload))
}

// TestEthernetRoundTrip tests serialization and parsing round-trip.
func TestEthernetRoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		src       common.MACAddress
		dst       common.MACAddress
		etherType common.EtherType
		payload   []byte
	}{
		{
			name:      "Small frame",
			src:       common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
			dst:       common.MACAddress{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
			etherType: common.EtherTypeIPv4,
			payload:   []byte("test"),
		},
		{
			name:      "Large frame",
			src:       common.MACAddress{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC},
			dst:       common.MACAddress{0xFE, 0xDC, 0xBA, 0x98, 0x76, 0x54},
			etherType: common.EtherTypeIPv6,
			payload:   bytes.Repeat([]byte("X"), 1000),
		},
		{
			name:      "Empty payload",
			src:       common.MACAddress{0x00, 0x00, 0x00, 0x00, 0x00, 0x01},
			dst:       common.MACAddress{0x00, 0x00, 0x00, 0x00, 0x00, 0x02},
			etherType: common.EtherTypeARP,
			payload:   []byte{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := ethernet.NewFrame(tt.dst, tt.src, tt.etherType, tt.payload)
			data := frame.Serialize()

			parsed, err := ethernet.Parse(data)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if parsed.Source != tt.src {
				t.Errorf("Source = %v, want %v", parsed.Source, tt.src)
			}
			if parsed.Destination != tt.dst {
				t.Errorf("Destination = %v, want %v", parsed.Destination, tt.dst)
			}
			if parsed.EtherType != tt.etherType {
				t.Errorf("EtherType = %v, want %v", parsed.EtherType, tt.etherType)
			}

			// For empty payload, account for padding
			if len(tt.payload) > 0 {
				if !bytes.Equal(parsed.Payload[:len(tt.payload)], tt.payload) {
					t.Error("Payload mismatch")
				}
			}
		})
	}
}

// TestEthernetVLANTagging tests VLAN-tagged frames (802.1Q).
func TestEthernetVLANTagging(t *testing.T) {
	src := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dst := common.MACAddress{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}

	// 802.1Q VLAN tag: 0x8100
	vlanEtherType := common.EtherType(0x8100)

	// VLAN payload: TCI (2 bytes) + actual EtherType (2 bytes) + data
	vlanPayload := []byte{
		0x00, 0x64, // TCI: VLAN ID 100
		0x08, 0x00, // EtherType: IPv4
		0x45, 0x00, // Start of IP packet
	}

	frame := ethernet.NewFrame(dst, src, vlanEtherType, vlanPayload)
	data := frame.Serialize()

	parsed, err := ethernet.Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if parsed.EtherType != vlanEtherType {
		t.Errorf("EtherType = 0x%04X, want 0x8100", parsed.EtherType)
	}

	t.Log("VLAN-tagged frame handled correctly")
}

// TestEthernetMACAddressFormats tests various MAC address formats.
func TestEthernetMACAddressFormats(t *testing.T) {
	tests := []struct {
		name string
		mac  common.MACAddress
		desc string
	}{
		{
			name: "Unicast",
			mac:  common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
			desc: "Regular unicast address",
		},
		{
			name: "Broadcast",
			mac:  common.MACAddress{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF},
			desc: "Broadcast address",
		},
		{
			name: "Multicast",
			mac:  common.MACAddress{0x01, 0x00, 0x5E, 0x00, 0x00, 0x01},
			desc: "IPv4 multicast address",
		},
		{
			name: "Zero",
			mac:  common.MACAddress{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			desc: "All zeros",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
			payload := []byte("test")

			frame := ethernet.NewFrame(tt.mac, src, common.EtherTypeIPv4, payload)
			data := frame.Serialize()

			parsed, err := ethernet.Parse(data)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if parsed.Destination != tt.mac {
				t.Errorf("Destination = %v, want %v", parsed.Destination, tt.mac)
			}

			t.Logf("%s: %v", tt.desc, tt.mac)
		})
	}
}

// TestEthernetJumboFrames tests jumbo frames (> 1500 byte MTU).
func TestEthernetJumboFrames(t *testing.T) {
	src := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dst := common.MACAddress{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}

	// Jumbo frame: 9000 bytes
	jumboPayload := make([]byte, 9000)
	for i := range jumboPayload {
		jumboPayload[i] = byte(i % 256)
	}

	frame := ethernet.NewFrame(dst, src, common.EtherTypeIPv4, jumboPayload)
	data := frame.Serialize()

	parsed, err := ethernet.Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if len(parsed.Payload) != len(jumboPayload) {
		t.Errorf("Payload length = %d, want %d", len(parsed.Payload), len(jumboPayload))
	}

	if !bytes.Equal(parsed.Payload, jumboPayload) {
		t.Error("Jumbo frame payload mismatch")
	}

	t.Logf("Successfully handled jumbo frame (%d bytes)", len(jumboPayload))
}

// TestEthernetPayloadExtraction tests extracting specific protocol payloads.
func TestEthernetPayloadExtraction(t *testing.T) {
	src := common.MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	dst := common.MACAddress{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF}

	// Create an IPv4 packet header as payload
	ipv4Header := []byte{
		0x45, 0x00, // Version, IHL, ToS
		0x00, 0x54, // Total length
		0x12, 0x34, // Identification
		0x40, 0x00, // Flags, Fragment offset
		0x40, 0x11, // TTL, Protocol (UDP)
		0x00, 0x00, // Checksum
		0xC0, 0xA8, 0x01, 0x64, // Source IP
		0xC0, 0xA8, 0x01, 0x01, // Dest IP
	}

	frame := ethernet.NewFrame(dst, src, common.EtherTypeIPv4, ipv4Header)
	data := frame.Serialize()

	parsed, err := ethernet.Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify we can extract the IPv4 header
	if parsed.EtherType == common.EtherTypeIPv4 {
		if len(parsed.Payload) < 20 {
			t.Error("IPv4 payload too short")
		}

		// Check IPv4 version
		version := parsed.Payload[0] >> 4
		if version != 4 {
			t.Errorf("IPv4 version = %d, want 4", version)
		}

		t.Log("Successfully extracted IPv4 payload from Ethernet frame")
	}
}
