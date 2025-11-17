package common

import "encoding/binary"

// CalculateChecksum computes the Internet checksum as defined in RFC 1071.
// The Internet checksum is a 16-bit one's complement of the one's complement sum
// of all 16-bit words in the data. If the data length is odd, the last byte is
// padded with a zero byte.
//
// This checksum is used in IP, ICMP, UDP, and TCP headers.
func CalculateChecksum(data []byte) uint16 {
	// Sum all 16-bit words
	var sum uint32
	length := len(data)

	// Process 16-bit words
	for i := 0; i < length-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))
	}

	// If length is odd, add the last byte (padded with zero)
	if length%2 == 1 {
		sum += uint32(data[length-1]) << 8
	}

	// Fold 32-bit sum to 16 bits
	// Add carry bits (high 16 bits) back to low 16 bits
	for sum > 0xFFFF {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	// Return one's complement
	return ^uint16(sum)
}

// VerifyChecksum verifies that the checksum of the data is correct.
// When calculating the checksum over data that includes the checksum field,
// the result should be 0 (or 0xFFFF, which is equivalent in one's complement).
func VerifyChecksum(data []byte) bool {
	checksum := CalculateChecksum(data)
	return checksum == 0 || checksum == 0xFFFF
}

// UpdateChecksum incrementally updates a checksum when data is modified.
// This is useful for performance when only a small portion of data changes.
// Based on RFC 1624.
//
// Parameters:
//   - oldChecksum: the existing checksum
//   - oldData: the old data being replaced
//   - newData: the new data
//
// Returns the updated checksum.
func UpdateChecksum(oldChecksum uint16, oldData, newData []byte) uint16 {
	if len(oldData) != len(newData) {
		// For different lengths, just recalculate
		// In practice, this function is used when lengths are the same
		panic("UpdateChecksum requires equal length data")
	}

	// Convert checksum to one's complement sum
	sum := ^uint32(oldChecksum)

	// Subtract old data
	for i := 0; i < len(oldData)-1; i += 2 {
		sum += 0xFFFF - uint32(binary.BigEndian.Uint16(oldData[i:i+2]))
	}
	if len(oldData)%2 == 1 {
		sum += 0xFF00 - (uint32(oldData[len(oldData)-1]) << 8)
	}

	// Add new data
	for i := 0; i < len(newData)-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(newData[i : i+2]))
	}
	if len(newData)%2 == 1 {
		sum += uint32(newData[len(newData)-1]) << 8
	}

	// Fold and return one's complement
	for sum > 0xFFFF {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	return ^uint16(sum)
}

// PseudoHeader represents the pseudo-header used for TCP and UDP checksum calculation.
// As per RFC 793 (TCP) and RFC 768 (UDP), the checksum includes a pseudo-header
// containing the source address, destination address, protocol, and length.
type PseudoHeader struct {
	SourceAddr      IPv4Address
	DestinationAddr IPv4Address
	Protocol        Protocol
	Length          uint16
}

// Bytes serializes the pseudo-header to bytes for checksum calculation.
func (ph PseudoHeader) Bytes() []byte {
	b := make([]byte, 12)
	copy(b[0:4], ph.SourceAddr[:])
	copy(b[4:8], ph.DestinationAddr[:])
	b[8] = 0 // Zero byte
	b[9] = uint8(ph.Protocol)
	binary.BigEndian.PutUint16(b[10:12], ph.Length)
	return b
}

// CalculateChecksumWithPseudoHeader calculates checksum including pseudo-header.
// This is used for TCP and UDP checksums.
func CalculateChecksumWithPseudoHeader(pseudoHeader PseudoHeader, data []byte) uint16 {
	// Combine pseudo-header and data
	phBytes := pseudoHeader.Bytes()
	combined := make([]byte, len(phBytes)+len(data))
	copy(combined, phBytes)
	copy(combined[len(phBytes):], data)

	return CalculateChecksum(combined)
}
