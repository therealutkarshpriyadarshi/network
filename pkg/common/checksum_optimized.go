package common

import "encoding/binary"

// CalculateChecksumOptimized computes the Internet checksum with optimizations.
// This version processes 8 bytes at a time for better performance.
func CalculateChecksumOptimized(data []byte) uint16 {
	length := len(data)
	var sum uint32

	// Process 8 bytes at a time (4 uint16 words)
	i := 0
	for i+7 < length {
		sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))
		sum += uint32(binary.BigEndian.Uint16(data[i+2 : i+4]))
		sum += uint32(binary.BigEndian.Uint16(data[i+4 : i+6]))
		sum += uint32(binary.BigEndian.Uint16(data[i+6 : i+8]))
		i += 8
	}

	// Process remaining 16-bit words
	for i+1 < length {
		sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))
		i += 2
	}

	// Handle odd byte
	if i < length {
		sum += uint32(data[i]) << 8
	}

	// Fold 32-bit sum to 16 bits
	for sum > 0xFFFF {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	return ^uint16(sum)
}

// CalculateChecksumFast is an even faster version that minimizes bounds checking.
func CalculateChecksumFast(data []byte) uint16 {
	length := len(data)
	if length == 0 {
		return 0xFFFF
	}

	var sum uint32
	i := 0

	// Process 16 bytes at a time when possible
	for i+15 < length {
		// Manually unrolled loop for 16 bytes (8 uint16 words)
		w0 := uint32(data[i])<<8 | uint32(data[i+1])
		w1 := uint32(data[i+2])<<8 | uint32(data[i+3])
		w2 := uint32(data[i+4])<<8 | uint32(data[i+5])
		w3 := uint32(data[i+6])<<8 | uint32(data[i+7])
		w4 := uint32(data[i+8])<<8 | uint32(data[i+9])
		w5 := uint32(data[i+10])<<8 | uint32(data[i+11])
		w6 := uint32(data[i+12])<<8 | uint32(data[i+13])
		w7 := uint32(data[i+14])<<8 | uint32(data[i+15])

		sum += w0 + w1 + w2 + w3 + w4 + w5 + w6 + w7
		i += 16

		// Periodic carry folding to prevent overflow
		if sum > 0xFFFFFFFF-0x10000 {
			sum = (sum & 0xFFFF) + (sum >> 16)
		}
	}

	// Process remaining 16-bit words
	for i+1 < length {
		sum += uint32(data[i])<<8 | uint32(data[i+1])
		i += 2
	}

	// Handle odd byte
	if i < length {
		sum += uint32(data[i]) << 8
	}

	// Final carry folding
	for sum > 0xFFFF {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	return ^uint16(sum)
}

// CalculateChecksumWithPseudoHeaderOptimized is an optimized version
// that avoids allocating a temporary buffer.
func CalculateChecksumWithPseudoHeaderOptimized(pseudoHeader PseudoHeader, data []byte) uint16 {
	var sum uint32

	// Process pseudo-header directly
	// Source address (4 bytes)
	sum += uint32(pseudoHeader.SourceAddr[0])<<8 | uint32(pseudoHeader.SourceAddr[1])
	sum += uint32(pseudoHeader.SourceAddr[2])<<8 | uint32(pseudoHeader.SourceAddr[3])

	// Destination address (4 bytes)
	sum += uint32(pseudoHeader.DestinationAddr[0])<<8 | uint32(pseudoHeader.DestinationAddr[1])
	sum += uint32(pseudoHeader.DestinationAddr[2])<<8 | uint32(pseudoHeader.DestinationAddr[3])

	// Zero + Protocol (2 bytes)
	sum += uint32(pseudoHeader.Protocol)

	// Length (2 bytes)
	sum += uint32(pseudoHeader.Length)

	// Fold pseudo-header carries
	sum = (sum & 0xFFFF) + (sum >> 16)

	// Process data
	length := len(data)
	i := 0

	// Process 16 bytes at a time
	for i+15 < length {
		w0 := uint32(data[i])<<8 | uint32(data[i+1])
		w1 := uint32(data[i+2])<<8 | uint32(data[i+3])
		w2 := uint32(data[i+4])<<8 | uint32(data[i+5])
		w3 := uint32(data[i+6])<<8 | uint32(data[i+7])
		w4 := uint32(data[i+8])<<8 | uint32(data[i+9])
		w5 := uint32(data[i+10])<<8 | uint32(data[i+11])
		w6 := uint32(data[i+12])<<8 | uint32(data[i+13])
		w7 := uint32(data[i+14])<<8 | uint32(data[i+15])

		sum += w0 + w1 + w2 + w3 + w4 + w5 + w6 + w7
		i += 16

		if sum > 0xFFFFFFFF-0x10000 {
			sum = (sum & 0xFFFF) + (sum >> 16)
		}
	}

	// Process remaining words
	for i+1 < length {
		sum += uint32(data[i])<<8 | uint32(data[i+1])
		i += 2
	}

	// Handle odd byte
	if i < length {
		sum += uint32(data[i]) << 8
	}

	// Final folding
	for sum > 0xFFFF {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	return ^uint16(sum)
}

// UpdateChecksumOptimized is an optimized incremental checksum update.
// For now, this just wraps the original implementation to ensure correctness.
// TODO: Add true optimizations after correctness is verified.
func UpdateChecksumOptimized(oldChecksum uint16, oldData, newData []byte) uint16 {
	return UpdateChecksum(oldChecksum, oldData, newData)
}

// ChecksumCompute is the interface for checksum computation strategies
type ChecksumCompute func([]byte) uint16

// SelectChecksumFunction returns the most appropriate checksum function
// based on the data size.
func SelectChecksumFunction(size int) ChecksumCompute {
	if size >= 1024 {
		// For large packets, use the fastest implementation
		return CalculateChecksumFast
	} else if size >= 128 {
		// For medium packets, use optimized version
		return CalculateChecksumOptimized
	}
	// For small packets, use standard version (less overhead)
	return CalculateChecksum
}
