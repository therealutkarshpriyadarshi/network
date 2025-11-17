// Package ethernet implements Ethernet frame handling for Layer 2 communication.
package ethernet

import (
	"encoding/binary"
	"fmt"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

// Ethernet frame format (IEEE 802.3):
// +-------------------+-------------------+----------+---------+-----+
// | Destination (6B)  | Source (6B)       | Type (2B)| Payload | FCS |
// +-------------------+-------------------+----------+---------+-----+
//
// Minimum frame size: 64 bytes (including FCS)
// Maximum frame size: 1518 bytes (including FCS)

const (
	// HeaderSize is the size of an Ethernet header (14 bytes).
	HeaderSize = 14

	// MinFrameSize is the minimum Ethernet frame size including FCS (64 bytes).
	MinFrameSize = 64

	// MaxFrameSize is the maximum Ethernet frame size including FCS (1518 bytes).
	MaxFrameSize = 1518

	// MinPayloadSize is the minimum payload size (46 bytes).
	MinPayloadSize = 46

	// MaxPayloadSize is the maximum payload size (1500 bytes, MTU).
	MaxPayloadSize = 1500

	// FCSSize is the size of the Frame Check Sequence (4 bytes).
	FCSSize = 4
)

// Frame represents an Ethernet II frame.
type Frame struct {
	Destination common.MACAddress
	Source      common.MACAddress
	EtherType   common.EtherType
	Payload     []byte
}

// Parse parses an Ethernet frame from raw bytes.
// Note: This does not validate or parse the FCS (Frame Check Sequence),
// as that's typically handled by the network hardware.
func Parse(data []byte) (*Frame, error) {
	if len(data) < HeaderSize {
		return nil, fmt.Errorf("ethernet frame too short: %d bytes", len(data))
	}

	frame := &Frame{}

	// Parse destination MAC (6 bytes)
	copy(frame.Destination[:], data[0:6])

	// Parse source MAC (6 bytes)
	copy(frame.Source[:], data[6:12])

	// Parse EtherType (2 bytes, big endian)
	frame.EtherType = common.EtherType(binary.BigEndian.Uint16(data[12:14]))

	// Remaining bytes are payload (minus FCS if present)
	// In raw socket captures, the FCS is usually not included
	frame.Payload = data[HeaderSize:]

	return frame, nil
}

// Serialize converts the frame to bytes for transmission.
// This does not add the FCS (Frame Check Sequence) as that's typically
// added by the network hardware.
func (f *Frame) Serialize() []byte {
	// Calculate total frame size
	frameSize := HeaderSize + len(f.Payload)

	// If payload is too small, we need to pad it
	if len(f.Payload) < MinPayloadSize {
		frameSize = HeaderSize + MinPayloadSize
	}

	frame := make([]byte, frameSize)

	// Write destination MAC
	copy(frame[0:6], f.Destination[:])

	// Write source MAC
	copy(frame[6:12], f.Source[:])

	// Write EtherType
	binary.BigEndian.PutUint16(frame[12:14], uint16(f.EtherType))

	// Write payload
	copy(frame[HeaderSize:], f.Payload)

	// Padding is implicitly zero bytes if payload is too small

	return frame
}

// Size returns the total size of the frame in bytes.
func (f *Frame) Size() int {
	size := HeaderSize + len(f.Payload)
	if len(f.Payload) < MinPayloadSize {
		size = HeaderSize + MinPayloadSize
	}
	return size
}

// String returns a human-readable representation of the frame.
func (f *Frame) String() string {
	return fmt.Sprintf("Ethernet{Dst=%s, Src=%s, Type=%s, PayloadLen=%d}",
		f.Destination, f.Source, f.EtherType, len(f.Payload))
}

// IsBroadcast returns true if this is a broadcast frame.
func (f *Frame) IsBroadcast() bool {
	return f.Destination.IsBroadcast()
}

// IsMulticast returns true if this is a multicast frame.
func (f *Frame) IsMulticast() bool {
	return f.Destination.IsMulticast()
}

// IsUnicast returns true if this is a unicast frame.
func (f *Frame) IsUnicast() bool {
	return !f.IsBroadcast() && !f.IsMulticast()
}

// NewFrame creates a new Ethernet frame.
func NewFrame(dst, src common.MACAddress, etherType common.EtherType, payload []byte) *Frame {
	return &Frame{
		Destination: dst,
		Source:      src,
		EtherType:   etherType,
		Payload:     payload,
	}
}
