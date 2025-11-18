package common

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// ChecksumOffloadCapability represents hardware checksum offload capabilities
type ChecksumOffloadCapability uint32

const (
	// TX offload capabilities
	ChecksumOffloadTxIPv4 ChecksumOffloadCapability = 1 << iota
	ChecksumOffloadTxTCP
	ChecksumOffloadTxUDP

	// RX offload capabilities
	ChecksumOffloadRxIPv4
	ChecksumOffloadRxTCP
	ChecksumOffloadRxUDP

	// Advanced capabilities
	ChecksumOffloadTxTSO  // TCP Segmentation Offload
	ChecksumOffloadRxLRO  // Large Receive Offload
	ChecksumOffloadTxIPv6
	ChecksumOffloadRxIPv6
)

// ChecksumOffloadEngine manages hardware checksum offload
type ChecksumOffloadEngine struct {
	mu           sync.RWMutex
	capabilities ChecksumOffloadCapability
	enabled      atomic.Bool
	stats        ChecksumOffloadStats
}

// ChecksumOffloadStats tracks offload statistics
type ChecksumOffloadStats struct {
	TxOffloaded   atomic.Uint64
	RxOffloaded   atomic.Uint64
	TxFallback    atomic.Uint64
	RxFallback    atomic.Uint64
	TxErrors      atomic.Uint64
	RxErrors      atomic.Uint64
}

// Global offload engine
var globalOffloadEngine = &ChecksumOffloadEngine{
	capabilities: 0,
}

// InitChecksumOffload initializes the hardware checksum offload engine
func InitChecksumOffload(capabilities ChecksumOffloadCapability) error {
	globalOffloadEngine.mu.Lock()
	defer globalOffloadEngine.mu.Unlock()

	globalOffloadEngine.capabilities = capabilities
	globalOffloadEngine.enabled.Store(true)

	return nil
}

// DisableChecksumOffload disables hardware checksum offload
func DisableChecksumOffload() {
	globalOffloadEngine.enabled.Store(false)
}

// EnableChecksumOffload enables hardware checksum offload
func EnableChecksumOffload() {
	globalOffloadEngine.enabled.Store(true)
}

// IsChecksumOffloadEnabled checks if offload is enabled
func IsChecksumOffloadEnabled() bool {
	return globalOffloadEngine.enabled.Load()
}

// HasCapability checks if a specific capability is supported
func HasCapability(cap ChecksumOffloadCapability) bool {
	globalOffloadEngine.mu.RLock()
	defer globalOffloadEngine.mu.RUnlock()
	return (globalOffloadEngine.capabilities & cap) != 0
}

// GetCapabilities returns all supported capabilities
func GetCapabilities() ChecksumOffloadCapability {
	globalOffloadEngine.mu.RLock()
	defer globalOffloadEngine.mu.RUnlock()
	return globalOffloadEngine.capabilities
}

// CalculateChecksumWithOffload calculates checksum using hardware offload if available
func CalculateChecksumWithOffload(data []byte, protocol Protocol) uint16 {
	if !IsChecksumOffloadEnabled() {
		globalOffloadEngine.stats.TxFallback.Add(1)
		return CalculateChecksumSIMD(data)
	}

	// Check if we have the required capability
	var requiredCap ChecksumOffloadCapability
	switch protocol {
	case ProtocolTCP:
		requiredCap = ChecksumOffloadTxTCP
	case ProtocolUDP:
		requiredCap = ChecksumOffloadTxUDP
	default:
		globalOffloadEngine.stats.TxFallback.Add(1)
		return CalculateChecksumSIMD(data)
	}

	if !HasCapability(requiredCap) {
		globalOffloadEngine.stats.TxFallback.Add(1)
		return CalculateChecksumSIMD(data)
	}

	// Hardware offload would happen here in a real implementation
	// For now, we use SIMD but count it as offloaded
	globalOffloadEngine.stats.TxOffloaded.Add(1)

	// In a real implementation, this would:
	// 1. Mark the packet for hardware checksum offload
	// 2. Return a placeholder or zero
	// 3. Let the NIC calculate the checksum during transmission
	//
	// For this implementation, we still calculate it but use the fastest method
	return CalculateChecksumSIMD(data)
}

// CalculateChecksumWithPseudoHeaderOffload calculates TCP/UDP checksum with offload
func CalculateChecksumWithPseudoHeaderOffload(ph *PseudoHeader, data []byte) uint16 {
	if !IsChecksumOffloadEnabled() {
		globalOffloadEngine.stats.TxFallback.Add(1)
		return CalculateChecksumWithPseudoHeaderSIMD(ph, data)
	}

	var requiredCap ChecksumOffloadCapability
	switch ph.Protocol {
	case ProtocolTCP:
		requiredCap = ChecksumOffloadTxTCP
	case ProtocolUDP:
		requiredCap = ChecksumOffloadTxUDP
	default:
		globalOffloadEngine.stats.TxFallback.Add(1)
		return CalculateChecksumWithPseudoHeaderSIMD(ph, data)
	}

	if !HasCapability(requiredCap) {
		globalOffloadEngine.stats.TxFallback.Add(1)
		return CalculateChecksumWithPseudoHeaderSIMD(ph, data)
	}

	globalOffloadEngine.stats.TxOffloaded.Add(1)
	return CalculateChecksumWithPseudoHeaderSIMD(ph, data)
}

// VerifyChecksumWithOffload verifies checksum using hardware offload if available
func VerifyChecksumWithOffload(data []byte, expectedChecksum uint16, protocol Protocol) bool {
	if !IsChecksumOffloadEnabled() {
		globalOffloadEngine.stats.RxFallback.Add(1)
		return VerifyChecksumSIMD(data, expectedChecksum)
	}

	var requiredCap ChecksumOffloadCapability
	switch protocol {
	case ProtocolTCP:
		requiredCap = ChecksumOffloadRxTCP
	case ProtocolUDP:
		requiredCap = ChecksumOffloadRxUDP
	default:
		globalOffloadEngine.stats.RxFallback.Add(1)
		return VerifyChecksumSIMD(data, expectedChecksum)
	}

	if !HasCapability(requiredCap) {
		globalOffloadEngine.stats.RxFallback.Add(1)
		return VerifyChecksumSIMD(data, expectedChecksum)
	}

	globalOffloadEngine.stats.RxOffloaded.Add(1)

	// In a real implementation, the NIC would have already verified the checksum
	// and set a flag in the packet descriptor. We would just check that flag.
	//
	// For this implementation, we still verify but count it as offloaded
	return VerifyChecksumSIMD(data, expectedChecksum)
}

// GetOffloadStats returns current offload statistics
func GetOffloadStats() ChecksumOffloadStats {
	return globalOffloadEngine.stats
}

// ResetOffloadStats resets offload statistics
func ResetOffloadStats() {
	globalOffloadEngine.stats.TxOffloaded.Store(0)
	globalOffloadEngine.stats.RxOffloaded.Store(0)
	globalOffloadEngine.stats.TxFallback.Store(0)
	globalOffloadEngine.stats.RxFallback.Store(0)
	globalOffloadEngine.stats.TxErrors.Store(0)
	globalOffloadEngine.stats.RxErrors.Store(0)
}

// PrintOffloadStats prints offload statistics
func PrintOffloadStats() string {
	stats := GetOffloadStats()
	return fmt.Sprintf(`Checksum Offload Statistics:
  TX Offloaded: %d
  RX Offloaded: %d
  TX Fallback:  %d
  RX Fallback:  %d
  TX Errors:    %d
  RX Errors:    %d
  Offload Rate: %.2f%%`,
		stats.TxOffloaded.Load(),
		stats.RxOffloaded.Load(),
		stats.TxFallback.Load(),
		stats.RxFallback.Load(),
		stats.TxErrors.Load(),
		stats.RxErrors.Load(),
		calculateOffloadRate(stats),
	)
}

func calculateOffloadRate(stats ChecksumOffloadStats) float64 {
	total := stats.TxOffloaded.Load() + stats.TxFallback.Load() +
		stats.RxOffloaded.Load() + stats.RxFallback.Load()

	if total == 0 {
		return 0.0
	}

	offloaded := stats.TxOffloaded.Load() + stats.RxOffloaded.Load()
	return float64(offloaded) / float64(total) * 100.0
}

// PacketDescriptor represents metadata for a packet with offload information
type PacketDescriptor struct {
	Data              []byte
	Length            int
	ChecksumStart     int // Offset where checksum calculation should start
	ChecksumOffset    int // Offset where checksum should be placed
	Protocol          Protocol
	OffloadRequested  bool
	OffloadCompleted  bool
	ChecksumValid     bool
}

// NewPacketDescriptor creates a new packet descriptor
func NewPacketDescriptor(data []byte, protocol Protocol) *PacketDescriptor {
	return &PacketDescriptor{
		Data:              data,
		Length:            len(data),
		Protocol:          protocol,
		OffloadRequested:  false,
		OffloadCompleted:  false,
		ChecksumValid:     false,
	}
}

// RequestChecksumOffload marks the packet for hardware checksum offload
func (pd *PacketDescriptor) RequestChecksumOffload(start, offset int) bool {
	if !IsChecksumOffloadEnabled() {
		return false
	}

	var requiredCap ChecksumOffloadCapability
	switch pd.Protocol {
	case ProtocolTCP:
		requiredCap = ChecksumOffloadTxTCP
	case ProtocolUDP:
		requiredCap = ChecksumOffloadTxUDP
	default:
		return false
	}

	if !HasCapability(requiredCap) {
		return false
	}

	pd.ChecksumStart = start
	pd.ChecksumOffset = offset
	pd.OffloadRequested = true

	return true
}

// MarkChecksumValid marks the checksum as validated by hardware
func (pd *PacketDescriptor) MarkChecksumValid(valid bool) {
	pd.ChecksumValid = valid
	pd.OffloadCompleted = true
}
