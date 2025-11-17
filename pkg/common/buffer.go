package common

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

// PacketBuffer provides a buffer for reading and writing network packets.
// It handles byte order conversion and provides utilities for packet manipulation.
type PacketBuffer struct {
	data []byte
	pos  int // Current read position
}

// NewPacketBuffer creates a new packet buffer with the given size.
func NewPacketBuffer(size int) *PacketBuffer {
	return &PacketBuffer{
		data: make([]byte, size),
		pos:  0,
	}
}

// NewPacketBufferFromBytes creates a packet buffer from existing data.
func NewPacketBufferFromBytes(data []byte) *PacketBuffer {
	return &PacketBuffer{
		data: data,
		pos:  0,
	}
}

// Bytes returns the underlying byte slice.
func (pb *PacketBuffer) Bytes() []byte {
	return pb.data
}

// Data returns the data from current position to end.
func (pb *PacketBuffer) Data() []byte {
	if pb.pos >= len(pb.data) {
		return nil
	}
	return pb.data[pb.pos:]
}

// Len returns the total length of the buffer.
func (pb *PacketBuffer) Len() int {
	return len(pb.data)
}

// Remaining returns the number of bytes remaining from current position.
func (pb *PacketBuffer) Remaining() int {
	return len(pb.data) - pb.pos
}

// Position returns the current read position.
func (pb *PacketBuffer) Position() int {
	return pb.pos
}

// SetPosition sets the read position.
func (pb *PacketBuffer) SetPosition(pos int) error {
	if pos < 0 || pos > len(pb.data) {
		return fmt.Errorf("position %d out of range [0, %d]", pos, len(pb.data))
	}
	pb.pos = pos
	return nil
}

// Reset resets the read position to the beginning.
func (pb *PacketBuffer) Reset() {
	pb.pos = 0
}

// Skip advances the position by n bytes.
func (pb *PacketBuffer) Skip(n int) error {
	if pb.pos+n > len(pb.data) {
		return io.EOF
	}
	pb.pos += n
	return nil
}

// ReadByte reads a single byte.
func (pb *PacketBuffer) ReadByte() (byte, error) {
	if pb.pos >= len(pb.data) {
		return 0, io.EOF
	}
	b := pb.data[pb.pos]
	pb.pos++
	return b, nil
}

// ReadBytes reads n bytes and advances the position.
func (pb *PacketBuffer) ReadBytes(n int) ([]byte, error) {
	if pb.pos+n > len(pb.data) {
		return nil, io.EOF
	}
	data := pb.data[pb.pos : pb.pos+n]
	pb.pos += n
	return data, nil
}

// ReadUint16 reads a 16-bit unsigned integer in network byte order (big endian).
func (pb *PacketBuffer) ReadUint16() (uint16, error) {
	if pb.pos+2 > len(pb.data) {
		return 0, io.EOF
	}
	val := binary.BigEndian.Uint16(pb.data[pb.pos : pb.pos+2])
	pb.pos += 2
	return val, nil
}

// ReadUint32 reads a 32-bit unsigned integer in network byte order (big endian).
func (pb *PacketBuffer) ReadUint32() (uint32, error) {
	if pb.pos+4 > len(pb.data) {
		return 0, io.EOF
	}
	val := binary.BigEndian.Uint32(pb.data[pb.pos : pb.pos+4])
	pb.pos += 4
	return val, nil
}

// ReadMAC reads a 6-byte MAC address.
func (pb *PacketBuffer) ReadMAC() (MACAddress, error) {
	if pb.pos+6 > len(pb.data) {
		return MACAddress{}, io.EOF
	}
	var mac MACAddress
	copy(mac[:], pb.data[pb.pos:pb.pos+6])
	pb.pos += 6
	return mac, nil
}

// ReadIPv4 reads a 4-byte IPv4 address.
func (pb *PacketBuffer) ReadIPv4() (IPv4Address, error) {
	if pb.pos+4 > len(pb.data) {
		return IPv4Address{}, io.EOF
	}
	var ip IPv4Address
	copy(ip[:], pb.data[pb.pos:pb.pos+4])
	pb.pos += 4
	return ip, nil
}

// WriteByte writes a single byte.
func (pb *PacketBuffer) WriteByte(b byte) error {
	if pb.pos >= len(pb.data) {
		return io.EOF
	}
	pb.data[pb.pos] = b
	pb.pos++
	return nil
}

// WriteBytes writes a byte slice.
func (pb *PacketBuffer) WriteBytes(data []byte) error {
	if pb.pos+len(data) > len(pb.data) {
		return io.EOF
	}
	copy(pb.data[pb.pos:], data)
	pb.pos += len(data)
	return nil
}

// WriteUint16 writes a 16-bit unsigned integer in network byte order (big endian).
func (pb *PacketBuffer) WriteUint16(val uint16) error {
	if pb.pos+2 > len(pb.data) {
		return io.EOF
	}
	binary.BigEndian.PutUint16(pb.data[pb.pos:pb.pos+2], val)
	pb.pos += 2
	return nil
}

// WriteUint32 writes a 32-bit unsigned integer in network byte order (big endian).
func (pb *PacketBuffer) WriteUint32(val uint32) error {
	if pb.pos+4 > len(pb.data) {
		return io.EOF
	}
	binary.BigEndian.PutUint32(pb.data[pb.pos:pb.pos+4], val)
	pb.pos += 4
	return nil
}

// WriteMAC writes a 6-byte MAC address.
func (pb *PacketBuffer) WriteMAC(mac MACAddress) error {
	if pb.pos+6 > len(pb.data) {
		return io.EOF
	}
	copy(pb.data[pb.pos:pb.pos+6], mac[:])
	pb.pos += 6
	return nil
}

// WriteIPv4 writes a 4-byte IPv4 address.
func (pb *PacketBuffer) WriteIPv4(ip IPv4Address) error {
	if pb.pos+4 > len(pb.data) {
		return io.EOF
	}
	copy(pb.data[pb.pos:pb.pos+4], ip[:])
	pb.pos += 4
	return nil
}

// HexDump returns a hex dump of the buffer for debugging.
func (pb *PacketBuffer) HexDump() string {
	return HexDump(pb.data)
}

// HexDump formats a byte slice as a hex dump with offsets and ASCII representation.
// This is useful for debugging network packets.
func HexDump(data []byte) string {
	var sb strings.Builder
	const bytesPerLine = 16

	for i := 0; i < len(data); i += bytesPerLine {
		// Write offset
		sb.WriteString(fmt.Sprintf("%04x  ", i))

		// Write hex bytes
		lineEnd := i + bytesPerLine
		if lineEnd > len(data) {
			lineEnd = len(data)
		}

		line := data[i:lineEnd]
		hexStr := hex.EncodeToString(line)

		// Format hex with spaces every 2 characters
		for j := 0; j < len(hexStr); j += 2 {
			sb.WriteString(hexStr[j : j+2])
			sb.WriteString(" ")
			if j == 14 {
				sb.WriteString(" ") // Extra space in the middle
			}
		}

		// Pad if line is short
		for j := len(line); j < bytesPerLine; j++ {
			sb.WriteString("   ")
			if j == 7 {
				sb.WriteString(" ")
			}
		}

		// Write ASCII representation
		sb.WriteString(" |")
		for _, b := range line {
			if b >= 32 && b <= 126 {
				sb.WriteByte(b)
			} else {
				sb.WriteByte('.')
			}
		}
		sb.WriteString("|\n")
	}

	return sb.String()
}
