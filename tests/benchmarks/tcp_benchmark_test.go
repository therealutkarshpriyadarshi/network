package benchmarks

import (
	"fmt"
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
	"github.com/therealutkarshpriyadarshi/network/pkg/tcp"
)

// BenchmarkTCPSegmentCreation measures TCP segment creation performance
func BenchmarkTCPSegmentCreation(b *testing.B) {
	payloadSizes := []int{
		64,      // Tiny packets
		512,     // Small packets
		1024,    // 1 KB
		4096,    // 4 KB
		8192,    // 8 KB
		16384,   // 16 KB
	}

	for _, size := range payloadSizes {
		b.Run(fmt.Sprintf("PayloadSize_%dB", size), func(b *testing.B) {
			benchmarkSegmentCreation(b, size)
		})
	}
}

func benchmarkSegmentCreation(b *testing.B, payloadSize int) {
	payload := make([]byte, payloadSize)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	b.ResetTimer()
	b.SetBytes(int64(payloadSize))

	for i := 0; i < b.N; i++ {
		seg := tcp.NewSegment(
			8080,
			9090,
			uint32(i)*1000,
			2000,
			tcp.FlagACK,
			65535,
			payload,
		)
		_ = seg
	}
}

// BenchmarkTCPChecksumCalculation measures TCP checksum performance
func BenchmarkTCPChecksumCalculation(b *testing.B) {
	localAddr := common.IPv4Address{127, 0, 0, 1}
	remoteAddr := common.IPv4Address{127, 0, 0, 2}

	payloadSizes := []int{64, 512, 1024, 4096, 16384}

	for _, size := range payloadSizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			payload := make([]byte, size)
			seg := tcp.NewSegment(8080, 9090, 1000, 2000, tcp.FlagACK, 65535, payload)

			b.ResetTimer()
			b.SetBytes(int64(size))

			for i := 0; i < b.N; i++ {
				seg.CalculateChecksum(localAddr, remoteAddr)
			}
		})
	}
}

// BenchmarkTCPThroughput measures theoretical TCP throughput
func BenchmarkTCPThroughput(b *testing.B) {
	localAddr := common.IPv4Address{127, 0, 0, 1}
	remoteAddr := common.IPv4Address{127, 0, 0, 2}

	payloadSizes := []int{512, 1024, 4096, 8192, 16384}

	for _, size := range payloadSizes {
		b.Run(fmt.Sprintf("PayloadSize_%dB", size), func(b *testing.B) {
			payload := make([]byte, size)
			for i := range payload {
				payload[i] = byte(i % 256)
			}

			b.ResetTimer()
			b.SetBytes(int64(size))

			var seqNum uint32 = 1000
			for i := 0; i < b.N; i++ {
				seg := tcp.NewSegment(8080, 9090, seqNum, 2000, tcp.FlagACK, 65535, payload)
				seg.CalculateChecksum(localAddr, remoteAddr)
				seqNum += uint32(len(payload))
			}

			totalBytes := int64(b.N) * int64(size)
			throughputGbps := float64(totalBytes*8) / b.Elapsed().Seconds() / 1e9
			b.ReportMetric(throughputGbps, "Gbps")
		})
	}
}

// BenchmarkTCPConnectionCreation measures connection object creation
func BenchmarkTCPConnectionCreation(b *testing.B) {
	localAddr := common.IPv4Address{127, 0, 0, 1}
	remoteAddr := common.IPv4Address{127, 0, 0, 2}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		conn := tcp.NewConnection(localAddr, 8080, remoteAddr, 9090)
		_ = conn
	}
}

// BenchmarkTCPStateMachine measures state machine overhead
func BenchmarkTCPStateMachine(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sm := tcp.NewStateMachine()
		sm.SetState(tcp.StateClosed)
		sm.Transition(tcp.EventActiveOpen)
		sm.Transition(tcp.EventReceiveSynAck)
		sm.GetState()
	}
}

// BenchmarkTCPBufferOperations measures buffer read/write performance
func BenchmarkTCPBufferOperations(b *testing.B) {
	bufferSizes := []int{4096, 16384, 65536}

	for _, size := range bufferSizes {
		b.Run(fmt.Sprintf("BufferSize_%d", size), func(b *testing.B) {
			buffer := tcp.NewReceiveBuffer(size)
			data := make([]byte, 1024)

			b.ResetTimer()
			b.SetBytes(1024)

			for i := 0; i < b.N; i++ {
				buffer.Write(data)
				buffer.Read(512)
			}
		})
	}
}

// BenchmarkTCPSegmentSerialization measures segment serialization
func BenchmarkTCPSegmentSerialization(b *testing.B) {
	payload := make([]byte, 1024)
	seg := tcp.NewSegment(8080, 9090, 1000, 2000, tcp.FlagACK, 65535, payload)

	b.ResetTimer()
	b.SetBytes(int64(1024 + 20)) // payload + header

	for i := 0; i < b.N; i++ {
		bytes, _ := seg.Serialize()
		_ = bytes
	}
}

// BenchmarkTCPSegmentParsing measures segment parsing
func BenchmarkTCPSegmentParsing(b *testing.B) {
	payload := make([]byte, 1024)
	seg := tcp.NewSegment(8080, 9090, 1000, 2000, tcp.FlagACK, 65535, payload)
	bytes, _ := seg.Serialize()

	b.ResetTimer()
	b.SetBytes(int64(len(bytes)))

	for i := 0; i < b.N; i++ {
		_, err := tcp.Parse(bytes)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkTCPMemoryAllocation tracks memory allocations
func BenchmarkTCPMemoryAllocation(b *testing.B) {
	b.ReportAllocs()

	localAddr := common.IPv4Address{127, 0, 0, 1}
	remoteAddr := common.IPv4Address{127, 0, 0, 2}
	payload := make([]byte, 1024)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		seg := tcp.NewSegment(8080, 9090, uint32(i), 2000, tcp.FlagACK, 65535, payload)
		seg.CalculateChecksum(localAddr, remoteAddr)
	}
}

// BenchmarkTCPLatency measures segment processing latency
func BenchmarkTCPLatency(b *testing.B) {
	localAddr := common.IPv4Address{127, 0, 0, 1}
	remoteAddr := common.IPv4Address{127, 0, 0, 2}

	payload := []byte("PING")

	b.ResetTimer()

	var totalLatency time.Duration
	for i := 0; i < b.N; i++ {
		start := time.Now()

		seg := tcp.NewSegment(8080, 9090, uint32(i), 2000, tcp.FlagACK, 65535, payload)
		seg.CalculateChecksum(localAddr, remoteAddr)

		latency := time.Since(start)
		totalLatency += latency
	}

	if b.N > 0 {
		avgLatencyMicros := float64(totalLatency.Microseconds()) / float64(b.N)
		b.ReportMetric(avgLatencyMicros, "µs/op")
	}
}

// BenchmarkTCPParallel measures parallel segment processing
func BenchmarkTCPParallel(b *testing.B) {
	localAddr := common.IPv4Address{127, 0, 0, 1}
	remoteAddr := common.IPv4Address{127, 0, 0, 2}
	payload := make([]byte, 1024)

	b.ResetTimer()
	b.SetBytes(1024)

	b.RunParallel(func(pb *testing.PB) {
		var seqNum uint32
		for pb.Next() {
			seg := tcp.NewSegment(8080, 9090, seqNum, 2000, tcp.FlagACK, 65535, payload)
			seg.CalculateChecksum(localAddr, remoteAddr)
			seqNum += 1024
		}
	})
}

// BenchmarkTCPRealWorldPattern simulates HTTP-like traffic
func BenchmarkTCPRealWorldPattern(b *testing.B) {
	localAddr := common.IPv4Address{127, 0, 0, 1}
	remoteAddr := common.IPv4Address{127, 0, 0, 2}

	request := make([]byte, 512)   // Small request
	response := make([]byte, 16384) // Larger response

	b.ResetTimer()
	b.SetBytes(int64(len(request) + len(response)))

	for i := 0; i < b.N; i++ {
		// Request
		reqSeg := tcp.NewSegment(8080, 9090, uint32(i)*20000, 2000, tcp.FlagACK, 65535, request)
		reqSeg.CalculateChecksum(localAddr, remoteAddr)

		// Response
		respSeg := tcp.NewSegment(9090, 8080, 2000, uint32(i)*20000+512, tcp.FlagACK, 65535, response)
		respSeg.CalculateChecksum(remoteAddr, localAddr)
	}
}

// BenchmarkTCPRetransmitQueue measures retransmit queue operations
func BenchmarkTCPRetransmitQueue(b *testing.B) {
	queue := tcp.NewRetransmitQueue()
	payload := make([]byte, 1024)
	seg := tcp.NewSegment(8080, 9090, 1000, 2000, tcp.FlagACK, 65535, payload)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		queue.Add(uint32(i), seg, time.Now())
		queue.Remove(uint32(i))
	}
}
