// Package tcp implements the Transmission Control Protocol (TCP) as defined in RFC 793.
package tcp

import (
	"encoding/binary"
	"fmt"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

const (
	// MinHeaderLength is the minimum TCP header length (20 bytes).
	MinHeaderLength = 20

	// MaxHeaderLength is the maximum TCP header length (60 bytes).
	MaxHeaderLength = 60

	// MaxSegmentSize is the default maximum segment size.
	DefaultMSS = 1460 // 1500 (MTU) - 20 (IP header) - 20 (TCP header)
)

// TCP Flags
const (
	FlagFIN uint8 = 1 << 0 // Finish - no more data from sender
	FlagSYN uint8 = 1 << 1 // Synchronize - establish connection
	FlagRST uint8 = 1 << 2 // Reset - abort connection
	FlagPSH uint8 = 1 << 3 // Push - deliver data immediately
	FlagACK uint8 = 1 << 4 // Acknowledgment - ACK field is valid
	FlagURG uint8 = 1 << 5 // Urgent - urgent pointer is valid
	FlagECE uint8 = 1 << 6 // ECN Echo
	FlagCWR uint8 = 1 << 7 // Congestion Window Reduced
)

// TCP Option kinds
const (
	OptionKindEOL            = 0  // End of Option List
	OptionKindNOP            = 1  // No Operation
	OptionKindMSS            = 2  // Maximum Segment Size
	OptionKindWindowScale    = 3  // Window Scale
	OptionKindSACKPermitted  = 4  // SACK Permitted
	OptionKindSACK           = 5  // SACK
	OptionKindTimestamp      = 8  // Timestamp
	OptionKindTFO            = 34 // TCP Fast Open
)

// Segment represents a TCP segment.
type Segment struct {
	// Header fields
	SourcePort      uint16 // Source port number
	DestinationPort uint16 // Destination port number
	SequenceNumber  uint32 // Sequence number
	AckNumber       uint32 // Acknowledgment number (if ACK flag is set)
	DataOffset      uint8  // Data offset (header length in 32-bit words)
	Flags           uint8  // Control flags (FIN, SYN, RST, PSH, ACK, URG, ECE, CWR)
	WindowSize      uint16 // Window size (for flow control)
	Checksum        uint16 // Checksum
	UrgentPointer   uint16 // Urgent pointer (if URG flag is set)
	Options         []byte // TCP options (if DataOffset > 5)

	// Payload
	Data []byte // Segment data
}

// Parse parses a TCP segment from raw bytes.
func Parse(data []byte) (*Segment, error) {
	if len(data) < MinHeaderLength {
		return nil, fmt.Errorf("TCP segment too short: %d bytes (minimum %d)", len(data), MinHeaderLength)
	}

	seg := &Segment{
		SourcePort:      binary.BigEndian.Uint16(data[0:2]),
		DestinationPort: binary.BigEndian.Uint16(data[2:4]),
		SequenceNumber:  binary.BigEndian.Uint32(data[4:8]),
		AckNumber:       binary.BigEndian.Uint32(data[8:12]),
	}

	// Parse data offset and flags
	dataOffsetReserved := data[12]
	seg.DataOffset = dataOffsetReserved >> 4
	seg.Flags = data[13]

	// Validate data offset
	if seg.DataOffset < 5 {
		return nil, fmt.Errorf("invalid data offset: %d (minimum 5)", seg.DataOffset)
	}

	headerLength := int(seg.DataOffset) * 4
	if headerLength > MaxHeaderLength {
		return nil, fmt.Errorf("invalid header length: %d (maximum %d)", headerLength, MaxHeaderLength)
	}

	if len(data) < headerLength {
		return nil, fmt.Errorf("segment too short for header: %d bytes (expected %d)", len(data), headerLength)
	}

	// Parse remaining fields
	seg.WindowSize = binary.BigEndian.Uint16(data[14:16])
	seg.Checksum = binary.BigEndian.Uint16(data[16:18])
	seg.UrgentPointer = binary.BigEndian.Uint16(data[18:20])

	// Parse options (if any)
	if headerLength > MinHeaderLength {
		seg.Options = make([]byte, headerLength-MinHeaderLength)
		copy(seg.Options, data[MinHeaderLength:headerLength])
	}

	// Extract data
	if len(data) > headerLength {
		seg.Data = make([]byte, len(data)-headerLength)
		copy(seg.Data, data[headerLength:])
	}

	return seg, nil
}

// Serialize converts the TCP segment to bytes.
// Note: This does NOT calculate the checksum. Use CalculateChecksum separately.
func (s *Segment) Serialize() ([]byte, error) {
	// Calculate header length
	headerLength := MinHeaderLength + len(s.Options)

	// Pad options to 4-byte boundary
	if len(s.Options) > 0 {
		padding := (4 - (len(s.Options) % 4)) % 4
		if padding > 0 {
			s.Options = append(s.Options, make([]byte, padding)...)
			headerLength += padding
		}
	}

	if headerLength > MaxHeaderLength {
		return nil, fmt.Errorf("header too large: %d bytes (maximum %d)", headerLength, MaxHeaderLength)
	}

	s.DataOffset = uint8(headerLength / 4)

	// Allocate buffer
	buf := make([]byte, headerLength+len(s.Data))

	// Set source and destination ports
	binary.BigEndian.PutUint16(buf[0:2], s.SourcePort)
	binary.BigEndian.PutUint16(buf[2:4], s.DestinationPort)

	// Set sequence and acknowledgment numbers
	binary.BigEndian.PutUint32(buf[4:8], s.SequenceNumber)
	binary.BigEndian.PutUint32(buf[8:12], s.AckNumber)

	// Set data offset and flags
	buf[12] = s.DataOffset << 4 // Upper 4 bits: data offset, lower 4 bits: reserved (0)
	buf[13] = s.Flags

	// Set window size, checksum, and urgent pointer
	binary.BigEndian.PutUint16(buf[14:16], s.WindowSize)
	binary.BigEndian.PutUint16(buf[16:18], s.Checksum)
	binary.BigEndian.PutUint16(buf[18:20], s.UrgentPointer)

	// Copy options
	if len(s.Options) > 0 {
		copy(buf[MinHeaderLength:headerLength], s.Options)
	}

	// Copy data
	if len(s.Data) > 0 {
		copy(buf[headerLength:], s.Data)
	}

	return buf, nil
}

// CalculateChecksum calculates the TCP checksum with the given pseudo-header.
// The pseudo-header is constructed from the IP header fields:
// - Source IP (4 bytes)
// - Destination IP (4 bytes)
// - Zero byte (1 byte)
// - Protocol (1 byte) = 6 for TCP
// - TCP Length (2 bytes)
func (s *Segment) CalculateChecksum(srcIP, dstIP common.IPv4Address) (uint16, error) {
	// Serialize the TCP segment first
	tcpData, err := s.Serialize()
	if err != nil {
		return 0, err
	}

	// Construct pseudo-header
	pseudoHeader := make([]byte, 12)
	copy(pseudoHeader[0:4], srcIP[:])
	copy(pseudoHeader[4:8], dstIP[:])
	pseudoHeader[8] = 0 // Zero
	pseudoHeader[9] = uint8(common.ProtocolTCP)
	binary.BigEndian.PutUint16(pseudoHeader[10:12], uint16(len(tcpData)))

	// Combine pseudo-header and TCP segment
	combined := append(pseudoHeader, tcpData...)

	// Calculate checksum
	checksum := common.CalculateChecksum(combined)

	return checksum, nil
}

// VerifyChecksum verifies the TCP checksum with the given pseudo-header.
func (s *Segment) VerifyChecksum(srcIP, dstIP common.IPv4Address) bool {
	// For verification, we check by calculating checksum of the whole thing
	// (including the checksum field) - it should equal 0 or 0xFFFF
	tcpData, err := s.Serialize()
	if err != nil {
		return false
	}

	// Construct pseudo-header
	pseudoHeader := make([]byte, 12)
	copy(pseudoHeader[0:4], srcIP[:])
	copy(pseudoHeader[4:8], dstIP[:])
	pseudoHeader[8] = 0
	pseudoHeader[9] = uint8(common.ProtocolTCP)
	binary.BigEndian.PutUint16(pseudoHeader[10:12], uint16(len(tcpData)))

	// Combine pseudo-header and TCP segment
	combined := append(pseudoHeader, tcpData...)

	// Calculate checksum - should be 0 or 0xFFFF if valid
	checksum := common.CalculateChecksum(combined)

	return checksum == 0 || checksum == 0xFFFF
}

// HasFlag checks if the segment has the specified flag set.
func (s *Segment) HasFlag(flag uint8) bool {
	return s.Flags&flag != 0
}

// SetFlag sets the specified flag.
func (s *Segment) SetFlag(flag uint8) {
	s.Flags |= flag
}

// ClearFlag clears the specified flag.
func (s *Segment) ClearFlag(flag uint8) {
	s.Flags &^= flag
}

// String returns a human-readable representation of the TCP segment.
func (s *Segment) String() string {
	flags := ""
	if s.HasFlag(FlagFIN) {
		flags += "F"
	}
	if s.HasFlag(FlagSYN) {
		flags += "S"
	}
	if s.HasFlag(FlagRST) {
		flags += "R"
	}
	if s.HasFlag(FlagPSH) {
		flags += "P"
	}
	if s.HasFlag(FlagACK) {
		flags += "A"
	}
	if s.HasFlag(FlagURG) {
		flags += "U"
	}
	if flags == "" {
		flags = "."
	}

	return fmt.Sprintf("TCP{SrcPort=%d, DstPort=%d, Seq=%d, Ack=%d, Flags=%s, Win=%d, DataLen=%d}",
		s.SourcePort, s.DestinationPort, s.SequenceNumber, s.AckNumber, flags, s.WindowSize, len(s.Data))
}

// NewSegment creates a new TCP segment with the given parameters.
func NewSegment(srcPort, dstPort uint16, seqNum, ackNum uint32, flags uint8, window uint16, data []byte) *Segment {
	return &Segment{
		SourcePort:      srcPort,
		DestinationPort: dstPort,
		SequenceNumber:  seqNum,
		AckNumber:       ackNum,
		DataOffset:      5, // Minimum size (20 bytes)
		Flags:           flags,
		WindowSize:      window,
		Checksum:        0, // Will be calculated later
		UrgentPointer:   0,
		Options:         nil,
		Data:            data,
	}
}

// ParseOptions parses TCP options from the options bytes.
func (s *Segment) ParseOptions() (map[uint8][]byte, error) {
	options := make(map[uint8][]byte)
	data := s.Options
	i := 0

	for i < len(data) {
		kind := data[i]

		// End of Option List
		if kind == OptionKindEOL {
			break
		}

		// No Operation
		if kind == OptionKindNOP {
			i++
			continue
		}

		// Options with length
		if i+1 >= len(data) {
			return nil, fmt.Errorf("incomplete option at offset %d", i)
		}

		length := int(data[i+1])
		if length < 2 || i+length > len(data) {
			return nil, fmt.Errorf("invalid option length %d at offset %d", length, i)
		}

		// Extract option data (excluding kind and length bytes)
		optData := make([]byte, length-2)
		copy(optData, data[i+2:i+length])
		options[kind] = optData

		i += length
	}

	return options, nil
}

// BuildMSSOption builds a Maximum Segment Size option.
func BuildMSSOption(mss uint16) []byte {
	opt := make([]byte, 4)
	opt[0] = OptionKindMSS
	opt[1] = 4 // Length
	binary.BigEndian.PutUint16(opt[2:4], mss)
	return opt
}

// BuildWindowScaleOption builds a Window Scale option.
func BuildWindowScaleOption(shift uint8) []byte {
	return []byte{OptionKindWindowScale, 3, shift}
}

// BuildTimestampOption builds a Timestamp option.
func BuildTimestampOption(tsVal, tsEcr uint32) []byte {
	opt := make([]byte, 10)
	opt[0] = OptionKindTimestamp
	opt[1] = 10 // Length
	binary.BigEndian.PutUint32(opt[2:6], tsVal)
	binary.BigEndian.PutUint32(opt[6:10], tsEcr)
	return opt
}

// BuildSACKPermittedOption builds a SACK Permitted option.
func BuildSACKPermittedOption() []byte {
	return []byte{OptionKindSACKPermitted, 2}
}

// SACKBlock represents a single SACK block.
type SACKBlock struct {
	LeftEdge  uint32
	RightEdge uint32
}

// BuildSACKOption builds a SACK option with the given blocks.
func BuildSACKOption(blocks []SACKBlock) []byte {
	if len(blocks) == 0 || len(blocks) > 4 {
		return nil // SACK can have at most 4 blocks
	}

	length := 2 + len(blocks)*8 // Kind + Length + blocks
	opt := make([]byte, length)
	opt[0] = OptionKindSACK
	opt[1] = uint8(length)

	offset := 2
	for _, block := range blocks {
		binary.BigEndian.PutUint32(opt[offset:offset+4], block.LeftEdge)
		binary.BigEndian.PutUint32(opt[offset+4:offset+8], block.RightEdge)
		offset += 8
	}

	return opt
}

// GetMSS extracts the MSS value from options, returns DefaultMSS if not found.
func (s *Segment) GetMSS() (uint16, error) {
	opts, err := s.ParseOptions()
	if err != nil {
		return DefaultMSS, err
	}

	if mssData, ok := opts[OptionKindMSS]; ok {
		if len(mssData) != 2 {
			return DefaultMSS, fmt.Errorf("invalid MSS option length: %d", len(mssData))
		}
		return binary.BigEndian.Uint16(mssData), nil
	}

	return DefaultMSS, nil
}

// GetWindowScale extracts the window scale value from options.
func (s *Segment) GetWindowScale() (uint8, error) {
	opts, err := s.ParseOptions()
	if err != nil {
		return 0, err
	}

	if wsData, ok := opts[OptionKindWindowScale]; ok {
		if len(wsData) != 1 {
			return 0, fmt.Errorf("invalid window scale option length: %d", len(wsData))
		}
		return wsData[0], nil
	}

	return 0, fmt.Errorf("window scale option not found")
}

// GetTimestamp extracts timestamp values from options.
func (s *Segment) GetTimestamp() (tsVal, tsEcr uint32, err error) {
	opts, err := s.ParseOptions()
	if err != nil {
		return 0, 0, err
	}

	if tsData, ok := opts[OptionKindTimestamp]; ok {
		if len(tsData) != 8 {
			return 0, 0, fmt.Errorf("invalid timestamp option length: %d", len(tsData))
		}
		tsVal = binary.BigEndian.Uint32(tsData[0:4])
		tsEcr = binary.BigEndian.Uint32(tsData[4:8])
		return tsVal, tsEcr, nil
	}

	return 0, 0, fmt.Errorf("timestamp option not found")
}

// GetSACKBlocks extracts SACK blocks from options.
func (s *Segment) GetSACKBlocks() ([]SACKBlock, error) {
	opts, err := s.ParseOptions()
	if err != nil {
		return nil, err
	}

	if sackData, ok := opts[OptionKindSACK]; ok {
		if len(sackData)%8 != 0 {
			return nil, fmt.Errorf("invalid SACK option length: %d", len(sackData))
		}

		numBlocks := len(sackData) / 8
		blocks := make([]SACKBlock, numBlocks)

		for i := 0; i < numBlocks; i++ {
			offset := i * 8
			blocks[i].LeftEdge = binary.BigEndian.Uint32(sackData[offset : offset+4])
			blocks[i].RightEdge = binary.BigEndian.Uint32(sackData[offset+4 : offset+8])
		}

		return blocks, nil
	}

	return nil, fmt.Errorf("SACK option not found")
}

// HasSACKPermitted checks if the SACK Permitted option is present.
func (s *Segment) HasSACKPermitted() bool {
	opts, err := s.ParseOptions()
	if err != nil {
		return false
	}
	_, ok := opts[OptionKindSACKPermitted]
	return ok
}
