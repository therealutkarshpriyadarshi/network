package benchmarks

import (
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

// BenchmarkChecksum measures checksum calculation performance
func BenchmarkChecksum(b *testing.B) {
	dataSizes := []int{
		20,    // IP header
		40,    // TCP header
		64,    // Small packet
		512,   // Medium packet
		1024,  // 1 KB
		1500,  // MTU size
		4096,  // 4 KB
		65536, // Max IP packet
	}

	for _, size := range dataSizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			benchmarkChecksum(b, size)
		})
	}
}

func benchmarkChecksum(b *testing.B, dataSize int) {
	data := make([]byte, dataSize)
	rand.Read(data)

	b.ResetTimer()
	b.SetBytes(int64(dataSize))

	for i := 0; i < b.N; i++ {
		_ = common.CalculateChecksum(data)
	}
}

// BenchmarkVerifyChecksum measures checksum verification performance
func BenchmarkVerifyChecksum(b *testing.B) {
	data := make([]byte, 1500)
	rand.Read(data)

	// Calculate correct checksum
	checksum := common.CalculateChecksum(data[:len(data)-2])
	data[len(data)-2] = byte(checksum >> 8)
	data[len(data)-1] = byte(checksum & 0xFF)

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_ = common.VerifyChecksum(data)
	}
}

// BenchmarkUpdateChecksum measures incremental checksum update performance
func BenchmarkUpdateChecksum(b *testing.B) {
	dataSizes := []int{4, 8, 16, 32, 64, 128}

	for _, size := range dataSizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			oldData := make([]byte, size)
			newData := make([]byte, size)
			rand.Read(oldData)
			rand.Read(newData)

			// Create a packet and calculate initial checksum
			packet := make([]byte, 1500)
			copy(packet[100:100+size], oldData)
			checksum := common.CalculateChecksum(packet)

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = common.UpdateChecksum(checksum, oldData, newData)
			}
		})
	}
}

// BenchmarkChecksumWithPseudoHeader measures TCP/UDP checksum with pseudo-header
func BenchmarkChecksumWithPseudoHeader(b *testing.B) {
	pseudoHeader := common.PseudoHeader{
		SourceAddr:      common.IPv4Address{192, 168, 1, 1},
		DestinationAddr: common.IPv4Address{192, 168, 1, 2},
		Protocol:        common.ProtocolTCP,
		Length:          1460,
	}

	data := make([]byte, 1460)
	rand.Read(data)

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_ = common.CalculateChecksumWithPseudoHeader(pseudoHeader, data)
	}
}

// BenchmarkChecksumAlignment measures impact of data alignment
func BenchmarkChecksumAlignment(b *testing.B) {
	b.Run("Aligned", func(b *testing.B) {
		// 16-byte aligned data
		data := make([]byte, 1500)
		rand.Read(data)

		b.ResetTimer()
		b.SetBytes(int64(len(data)))

		for i := 0; i < b.N; i++ {
			_ = common.CalculateChecksum(data)
		}
	})

	b.Run("Unaligned", func(b *testing.B) {
		// Offset by 1 byte (unaligned)
		buffer := make([]byte, 1501)
		rand.Read(buffer)
		data := buffer[1:]

		b.ResetTimer()
		b.SetBytes(int64(len(data)))

		for i := 0; i < b.N; i++ {
			_ = common.CalculateChecksum(data)
		}
	})
}

// BenchmarkChecksumOddLength measures performance with odd-length data
func BenchmarkChecksumOddLength(b *testing.B) {
	b.Run("EvenLength", func(b *testing.B) {
		data := make([]byte, 1500)
		rand.Read(data)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = common.CalculateChecksum(data)
		}
	})

	b.Run("OddLength", func(b *testing.B) {
		data := make([]byte, 1501)
		rand.Read(data)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			_ = common.CalculateChecksum(data)
		}
	})
}

// BenchmarkChecksumParallel measures concurrent checksum calculation
func BenchmarkChecksumParallel(b *testing.B) {
	data := make([]byte, 1500)
	rand.Read(data)

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = common.CalculateChecksum(data)
		}
	})
}

// BenchmarkChecksumVsRecalculation compares update vs full recalculation
func BenchmarkChecksumVsRecalculation(b *testing.B) {
	packet := make([]byte, 1500)
	rand.Read(packet)

	oldData := make([]byte, 4)
	newData := make([]byte, 4)
	copy(oldData, packet[100:104])
	rand.Read(newData)

	b.Run("FullRecalculation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			copy(packet[100:104], newData)
			_ = common.CalculateChecksum(packet)
		}
	})

	b.Run("IncrementalUpdate", func(b *testing.B) {
		checksum := common.CalculateChecksum(packet)

		for i := 0; i < b.N; i++ {
			checksum = common.UpdateChecksum(checksum, oldData, newData)
		}
	})
}

// BenchmarkChecksumMemoryAllocation tracks memory allocations
func BenchmarkChecksumMemoryAllocation(b *testing.B) {
	b.ReportAllocs()

	data := make([]byte, 1500)
	rand.Read(data)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = common.CalculateChecksum(data)
	}
}

// BenchmarkPseudoHeaderCreation measures pseudo-header creation overhead
func BenchmarkPseudoHeaderCreation(b *testing.B) {
	b.ReportAllocs()

	pseudoHeader := common.PseudoHeader{
		SourceAddr:      common.IPv4Address{192, 168, 1, 1},
		DestinationAddr: common.IPv4Address{192, 168, 1, 2},
		Protocol:        common.ProtocolTCP,
		Length:          1460,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = pseudoHeader.Bytes()
	}
}

// BenchmarkChecksumRealWorld simulates real packet processing
func BenchmarkChecksumRealWorld(b *testing.B) {
	b.Run("TCPPacket", func(b *testing.B) {
		// Simulate TCP packet checksum calculation
		tcpHeader := make([]byte, 20)
		tcpPayload := make([]byte, 1440)
		rand.Read(tcpHeader)
		rand.Read(tcpPayload)

		tcpData := append(tcpHeader, tcpPayload...)

		pseudoHeader := common.PseudoHeader{
			SourceAddr:      common.IPv4Address{192, 168, 1, 1},
			DestinationAddr: common.IPv4Address{192, 168, 1, 2},
			Protocol:        common.ProtocolTCP,
			Length:          uint16(len(tcpData)),
		}

		b.ResetTimer()
		b.SetBytes(int64(len(tcpData)))

		for i := 0; i < b.N; i++ {
			_ = common.CalculateChecksumWithPseudoHeader(pseudoHeader, tcpData)
		}
	})

	b.Run("IPHeader", func(b *testing.B) {
		ipHeader := make([]byte, 20)
		rand.Read(ipHeader)

		b.ResetTimer()
		b.SetBytes(20)

		for i := 0; i < b.N; i++ {
			_ = common.CalculateChecksum(ipHeader)
		}
	})
}

// BenchmarkChecksumBatchProcessing measures batch checksum calculation
func BenchmarkChecksumBatchProcessing(b *testing.B) {
	numPackets := 100
	packets := make([][]byte, numPackets)

	for i := 0; i < numPackets; i++ {
		packets[i] = make([]byte, 1500)
		rand.Read(packets[i])
	}

	b.ResetTimer()
	b.SetBytes(int64(1500 * numPackets))

	for i := 0; i < b.N; i++ {
		for _, packet := range packets {
			_ = common.CalculateChecksum(packet)
		}
	}
}

// BenchmarkChecksumZeroData measures checksum of zero-filled data
func BenchmarkChecksumZeroData(b *testing.B) {
	data := make([]byte, 1500)
	// All zeros

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_ = common.CalculateChecksum(data)
	}
}

// BenchmarkChecksumMaxEntropy measures checksum of random data
func BenchmarkChecksumMaxEntropy(b *testing.B) {
	data := make([]byte, 1500)
	rand.Read(data)

	b.ResetTimer()
	b.SetBytes(int64(len(data)))

	for i := 0; i < b.N; i++ {
		_ = common.CalculateChecksum(data)
	}
}
