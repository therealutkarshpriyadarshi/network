// Package quic implements QUIC frames as defined in RFC 9000.
package quic

import (
	"encoding/binary"
	"fmt"
)

// FrameType represents the type of QUIC frame.
type FrameType uint8

const (
	FrameTypePadding         FrameType = 0x00
	FrameTypePing            FrameType = 0x01
	FrameTypeAck             FrameType = 0x02
	FrameTypeResetStream     FrameType = 0x04
	FrameTypeStopSending     FrameType = 0x05
	FrameTypeCrypto          FrameType = 0x06
	FrameTypeNewToken        FrameType = 0x07
	FrameTypeStream          FrameType = 0x08 // 0x08-0x0f
	FrameTypeMaxData         FrameType = 0x10
	FrameTypeMaxStreamData   FrameType = 0x11
	FrameTypeMaxStreams      FrameType = 0x12
	FrameTypeDataBlocked     FrameType = 0x14
	FrameTypeStreamDataBlocked FrameType = 0x15
	FrameTypeStreamsBlocked  FrameType = 0x16
	FrameTypeNewConnectionID FrameType = 0x18
	FrameTypeRetireConnectionID FrameType = 0x19
	FrameTypePathChallenge   FrameType = 0x1a
	FrameTypePathResponse    FrameType = 0x1b
	FrameTypeConnectionClose FrameType = 0x1c
	FrameTypeHandshakeDone   FrameType = 0x1e
)

// Frame represents a QUIC frame.
type Frame interface {
	Type() FrameType
	Serialize() ([]byte, error)
	String() string
}

// PaddingFrame represents a PADDING frame.
type PaddingFrame struct {
	Length int
}

func (f *PaddingFrame) Type() FrameType {
	return FrameTypePadding
}

func (f *PaddingFrame) Serialize() ([]byte, error) {
	return make([]byte, f.Length), nil
}

func (f *PaddingFrame) String() string {
	return fmt.Sprintf("PADDING{Len=%d}", f.Length)
}

// PingFrame represents a PING frame.
type PingFrame struct{}

func (f *PingFrame) Type() FrameType {
	return FrameTypePing
}

func (f *PingFrame) Serialize() ([]byte, error) {
	return []byte{byte(FrameTypePing)}, nil
}

func (f *PingFrame) String() string {
	return "PING"
}

// AckFrame represents an ACK frame.
type AckFrame struct {
	LargestAcknowledged uint64
	AckDelay            uint64
	AckRanges           []AckRange
}

type AckRange struct {
	Gap    uint64
	Length uint64
}

func (f *AckFrame) Type() FrameType {
	return FrameTypeAck
}

func (f *AckFrame) Serialize() ([]byte, error) {
	// Simplified serialization
	buf := make([]byte, 1+8+8+1) // Type + Largest + Delay + RangeCount
	buf[0] = byte(FrameTypeAck)
	binary.BigEndian.PutUint64(buf[1:9], f.LargestAcknowledged)
	binary.BigEndian.PutUint64(buf[9:17], f.AckDelay)
	buf[17] = byte(len(f.AckRanges))
	return buf, nil
}

func (f *AckFrame) String() string {
	return fmt.Sprintf("ACK{Largest=%d, Delay=%d, Ranges=%d}",
		f.LargestAcknowledged, f.AckDelay, len(f.AckRanges))
}

// StreamFrame represents a STREAM frame.
type StreamFrame struct {
	StreamID uint64
	Offset   uint64
	Length   uint64
	Fin      bool
	Data     []byte
}

func (f *StreamFrame) Type() FrameType {
	return FrameTypeStream
}

func (f *StreamFrame) Serialize() ([]byte, error) {
	// Calculate type byte with flags
	typeByte := byte(FrameTypeStream)
	if f.Fin {
		typeByte |= 0x01
	}
	if f.Length > 0 {
		typeByte |= 0x02
	}
	if f.Offset > 0 {
		typeByte |= 0x04
	}

	// Simplified serialization
	buf := make([]byte, 1+8+8+8+len(f.Data))
	offset := 0

	buf[offset] = typeByte
	offset++

	binary.BigEndian.PutUint64(buf[offset:], f.StreamID)
	offset += 8

	if f.Offset > 0 {
		binary.BigEndian.PutUint64(buf[offset:], f.Offset)
		offset += 8
	}

	if f.Length > 0 {
		binary.BigEndian.PutUint64(buf[offset:], f.Length)
		offset += 8
	}

	copy(buf[offset:], f.Data)

	return buf[:offset+len(f.Data)], nil
}

func (f *StreamFrame) String() string {
	return fmt.Sprintf("STREAM{ID=%d, Offset=%d, Len=%d, Fin=%v}",
		f.StreamID, f.Offset, len(f.Data), f.Fin)
}

// CryptoFrame represents a CRYPTO frame.
type CryptoFrame struct {
	Offset uint64
	Data   []byte
}

func (f *CryptoFrame) Type() FrameType {
	return FrameTypeCrypto
}

func (f *CryptoFrame) Serialize() ([]byte, error) {
	buf := make([]byte, 1+8+8+len(f.Data))
	buf[0] = byte(FrameTypeCrypto)
	binary.BigEndian.PutUint64(buf[1:9], f.Offset)
	binary.BigEndian.PutUint64(buf[9:17], uint64(len(f.Data)))
	copy(buf[17:], f.Data)
	return buf, nil
}

func (f *CryptoFrame) String() string {
	return fmt.Sprintf("CRYPTO{Offset=%d, Len=%d}", f.Offset, len(f.Data))
}

// ConnectionCloseFrame represents a CONNECTION_CLOSE frame.
type ConnectionCloseFrame struct {
	ErrorCode    uint64
	FrameType    uint64
	ReasonPhrase string
}

func (f *ConnectionCloseFrame) Type() FrameType {
	return FrameTypeConnectionClose
}

func (f *ConnectionCloseFrame) Serialize() ([]byte, error) {
	reasonBytes := []byte(f.ReasonPhrase)
	buf := make([]byte, 1+8+8+8+len(reasonBytes))
	buf[0] = byte(FrameTypeConnectionClose)
	binary.BigEndian.PutUint64(buf[1:9], f.ErrorCode)
	binary.BigEndian.PutUint64(buf[9:17], f.FrameType)
	binary.BigEndian.PutUint64(buf[17:25], uint64(len(reasonBytes)))
	copy(buf[25:], reasonBytes)
	return buf, nil
}

func (f *ConnectionCloseFrame) String() string {
	return fmt.Sprintf("CONNECTION_CLOSE{Code=%d, Reason=%s}", f.ErrorCode, f.ReasonPhrase)
}

// MaxDataFrame represents a MAX_DATA frame.
type MaxDataFrame struct {
	MaximumData uint64
}

func (f *MaxDataFrame) Type() FrameType {
	return FrameTypeMaxData
}

func (f *MaxDataFrame) Serialize() ([]byte, error) {
	buf := make([]byte, 9)
	buf[0] = byte(FrameTypeMaxData)
	binary.BigEndian.PutUint64(buf[1:9], f.MaximumData)
	return buf, nil
}

func (f *MaxDataFrame) String() string {
	return fmt.Sprintf("MAX_DATA{Max=%d}", f.MaximumData)
}

// ParseFrame parses a frame from bytes (simplified).
func ParseFrame(data []byte) (Frame, int, error) {
	if len(data) < 1 {
		return nil, 0, fmt.Errorf("frame data too short")
	}

	frameType := FrameType(data[0])

	switch frameType {
	case FrameTypePing:
		return &PingFrame{}, 1, nil

	case FrameTypePadding:
		// Count consecutive padding bytes
		count := 0
		for i := 0; i < len(data) && data[i] == 0; i++ {
			count++
		}
		return &PaddingFrame{Length: count}, count, nil

	default:
		// For other frames, return a simple representation
		return nil, 0, fmt.Errorf("unsupported frame type: 0x%02x", frameType)
	}
}
