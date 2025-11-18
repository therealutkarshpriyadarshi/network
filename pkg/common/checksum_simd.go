// +build amd64

package common

import (
	"sync"
	"unsafe"
)

// CPU capabilities cache
var (
	cpuOnce      sync.Once
	cpuHasAVX2   bool
	cpuHasSSE2   bool
)

// initCPUCaps initializes CPU capabilities cache
func initCPUCaps() {
	cpuOnce.Do(func() {
		cpuHasAVX2 = hasAVX2()
		cpuHasSSE2 = hasSSE2()
	})
}

// CalculateChecksumSIMD uses SIMD instructions for ultra-fast checksum calculation
// This function is optimized for amd64 with AVX2 support
func CalculateChecksumSIMD(data []byte) uint16 {
	if len(data) == 0 {
		return 0
	}

	// For very small packets, use the optimized scalar implementation
	// SIMD overhead is not worth it for small data
	if len(data) < 64 {
		return CalculateChecksumFast(data)
	}

	// Initialize CPU capabilities once
	initCPUCaps()

	var sum uint64

	// Process 32-byte chunks with AVX2 (if available)
	if cpuHasAVX2 {
		sum = checksumAVX2(data)
	} else if cpuHasSSE2 {
		// Fallback to SSE2
		sum = checksumSSE2(data)
	} else {
		// No SIMD support, use optimized scalar
		return CalculateChecksumFast(data)
	}

	// Fold 64-bit sum to 16-bit
	for sum > 0xFFFF {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	return ^uint16(sum)
}

// checksumAVX2 processes data in 32-byte chunks using AVX2
//go:noescape
func checksumAVX2(data []byte) uint64

// checksumSSE2 processes data in 16-byte chunks using SSE2
//go:noescape
func checksumSSE2(data []byte) uint64

// hasAVX2 checks if CPU supports AVX2
//go:noescape
func hasAVX2() bool

// hasSSE2 checks if CPU supports SSE2
//go:noescape
func hasSSE2() bool

// CalculateChecksumWithPseudoHeaderSIMD combines pseudo-header and data checksums using SIMD
func CalculateChecksumWithPseudoHeaderSIMD(ph *PseudoHeader, data []byte) uint16 {
	// Use stack-allocated buffer for pseudo-header to avoid heap allocation
	var phBytes [12]byte

	// Manually serialize pseudo-header (inline to avoid function call overhead)
	phBytes[0] = ph.SourceAddr[0]
	phBytes[1] = ph.SourceAddr[1]
	phBytes[2] = ph.SourceAddr[2]
	phBytes[3] = ph.SourceAddr[3]
	phBytes[4] = ph.DestinationAddr[0]
	phBytes[5] = ph.DestinationAddr[1]
	phBytes[6] = ph.DestinationAddr[2]
	phBytes[7] = ph.DestinationAddr[3]
	phBytes[8] = 0 // Reserved
	phBytes[9] = byte(ph.Protocol)
	phBytes[10] = byte(ph.Length >> 8)
	phBytes[11] = byte(ph.Length)

	// Calculate checksum for pseudo-header (always 12 bytes)
	var sum uint64

	// Process 4-byte words
	sum += uint64(uint32(phBytes[0])<<24 | uint32(phBytes[1])<<16 | uint32(phBytes[2])<<8 | uint32(phBytes[3]))
	sum += uint64(uint32(phBytes[4])<<24 | uint32(phBytes[5])<<16 | uint32(phBytes[6])<<8 | uint32(phBytes[7]))
	sum += uint64(uint32(phBytes[8])<<24 | uint32(phBytes[9])<<16 | uint32(phBytes[10])<<8 | uint32(phBytes[11]))

	// Initialize CPU capabilities once
	initCPUCaps()

	// Add data checksum using SIMD
	if len(data) >= 64 && cpuHasAVX2 {
		sum += checksumAVX2(data)
	} else if len(data) >= 32 && cpuHasSSE2 {
		sum += checksumSSE2(data)
	} else {
		// Small data, process with scalar code
		sum += uint64(calculateChecksumPartial(data))
	}

	// Fold to 16 bits
	for sum > 0xFFFF {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	return ^uint16(sum)
}

// calculateChecksumPartial returns partial sum (not inverted)
func calculateChecksumPartial(data []byte) uint32 {
	var sum uint32
	length := len(data)

	// Process 4-byte chunks
	i := 0
	for i+3 < length {
		sum += uint32(data[i])<<24 | uint32(data[i+1])<<16 | uint32(data[i+2])<<8 | uint32(data[i+3])
		i += 4
	}

	// Process remaining bytes
	for i < length {
		if i+1 < length {
			sum += uint32(data[i])<<8 | uint32(data[i+1])
			i += 2
		} else {
			sum += uint32(data[i]) << 8
			i++
		}
	}

	// Fold carries
	for sum > 0xFFFF {
		sum = (sum & 0xFFFF) + (sum >> 16)
	}

	return sum
}

// UpdateChecksumSIMD performs incremental checksum update (RFC 1624)
// Optimized for small data modifications
func UpdateChecksumSIMD(oldChecksum uint16, oldData, newData []byte) uint16 {
	if len(oldData) != len(newData) {
		// Lengths must match for incremental update
		return oldChecksum
	}

	if len(oldData) == 0 {
		return oldChecksum
	}

	// For very small updates, use direct calculation
	if len(oldData) <= 8 {
		return UpdateChecksum(oldChecksum, oldData, newData)
	}

	// Calculate difference and update checksum
	var diff int64

	// Process in 8-byte chunks
	i := 0
	for i+7 < len(oldData) {
		oldVal := uint64(oldData[i])<<56 | uint64(oldData[i+1])<<48 |
			uint64(oldData[i+2])<<40 | uint64(oldData[i+3])<<32 |
			uint64(oldData[i+4])<<24 | uint64(oldData[i+5])<<16 |
			uint64(oldData[i+6])<<8 | uint64(oldData[i+7])

		newVal := uint64(newData[i])<<56 | uint64(newData[i+1])<<48 |
			uint64(newData[i+2])<<40 | uint64(newData[i+3])<<32 |
			uint64(newData[i+4])<<24 | uint64(newData[i+5])<<16 |
			uint64(newData[i+6])<<8 | uint64(newData[i+7])

		diff += int64(newVal) - int64(oldVal)
		i += 8
	}

	// Process remaining bytes
	for i < len(oldData) {
		if i+1 < len(oldData) {
			oldWord := uint16(oldData[i])<<8 | uint16(oldData[i+1])
			newWord := uint16(newData[i])<<8 | uint16(newData[i+1])
			diff += int64(newWord) - int64(oldWord)
			i += 2
		} else {
			oldByte := uint16(oldData[i]) << 8
			newByte := uint16(newData[i]) << 8
			diff += int64(newByte) - int64(oldByte)
			i++
		}
	}

	// Apply difference to checksum
	sum := int64(^oldChecksum) + diff

	// Handle carries and borrows
	for sum > 0xFFFF || sum < 0 {
		if sum > 0xFFFF {
			sum = (sum & 0xFFFF) + (sum >> 16)
		}
		if sum < 0 {
			sum += 0x10000
		}
	}

	return ^uint16(sum)
}

// VerifyChecksumSIMD verifies a checksum using SIMD
func VerifyChecksumSIMD(data []byte, expectedChecksum uint16) bool {
	calculated := CalculateChecksumSIMD(data)
	return calculated == expectedChecksum
}

// Batch checksum calculation for multiple packets
type ChecksumBatch struct {
	packets [][]byte
	results []uint16
}

// NewChecksumBatch creates a new batch processor
func NewChecksumBatch(capacity int) *ChecksumBatch {
	return &ChecksumBatch{
		packets: make([][]byte, 0, capacity),
		results: make([]uint16, 0, capacity),
	}
}

// Add adds a packet to the batch
func (cb *ChecksumBatch) Add(data []byte) {
	cb.packets = append(cb.packets, data)
}

// ProcessSIMD processes all packets in the batch using SIMD
func (cb *ChecksumBatch) ProcessSIMD() []uint16 {
	cb.results = cb.results[:0]

	for _, packet := range cb.packets {
		checksum := CalculateChecksumSIMD(packet)
		cb.results = append(cb.results, checksum)
	}

	return cb.results
}

// Reset clears the batch
func (cb *ChecksumBatch) Reset() {
	cb.packets = cb.packets[:0]
	cb.results = cb.results[:0]
}

// Inline assembly helpers for direct memory access
func ptr(b []byte) unsafe.Pointer {
	if len(b) == 0 {
		return unsafe.Pointer(nil)
	}
	return unsafe.Pointer(&b[0])
}
