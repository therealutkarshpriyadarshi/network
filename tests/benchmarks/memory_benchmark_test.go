package benchmarks

import (
	"crypto/rand"
	"fmt"
	"runtime"
	"testing"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
	"github.com/therealutkarshpriyadarshi/network/pkg/tcp"
)

// BenchmarkMemoryAllocation measures overall memory allocation patterns
func BenchmarkMemoryAllocation(b *testing.B) {
	b.ReportAllocs()

	localAddr := common.IPv4Address{127, 0, 0, 1}
	remoteAddr := common.IPv4Address{127, 0, 0, 2}
	payload := make([]byte, 1024)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		conn := tcp.NewConnection(localAddr, 8080, remoteAddr, 9090)
		seg := tcp.NewSegment(8080, 9090, uint32(i), 2000, tcp.FlagACK, 65535, payload)
		seg.CalculateChecksum(localAddr, remoteAddr)
		_ = conn
	}
}

// BenchmarkBufferAllocation measures buffer allocation overhead
func BenchmarkBufferAllocation(b *testing.B) {
	sizes := []int{1024, 4096, 16384, 65536}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size_%d", size), func(b *testing.B) {
			b.ReportAllocs()

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				buf := make([]byte, size)
				_ = buf
			}
		})
	}
}

// BenchmarkBufferReuse measures buffer reuse patterns
func BenchmarkBufferReuse(b *testing.B) {
	b.Run("NoReuse", func(b *testing.B) {
		b.ReportAllocs()

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := make([]byte, 1500)
			rand.Read(buf)
		}
	})

	b.Run("WithReuse", func(b *testing.B) {
		b.ReportAllocs()

		buf := make([]byte, 1500)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			rand.Read(buf)
		}
	})
}

// BenchmarkGCPressure measures GC impact under load
func BenchmarkGCPressure(b *testing.B) {
	b.Run("HighAllocation", func(b *testing.B) {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		allocsBefore := ms.Mallocs

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Allocate many small objects (high GC pressure)
			for j := 0; j < 100; j++ {
				buf := make([]byte, 128)
				_ = buf
			}
		}

		b.StopTimer()
		runtime.ReadMemStats(&ms)
		allocsAfter := ms.Mallocs

		b.ReportMetric(float64(allocsAfter-allocsBefore)/float64(b.N), "allocs/op")
	})

	b.Run("LowAllocation", func(b *testing.B) {
		var ms runtime.MemStats
		runtime.ReadMemStats(&ms)
		allocsBefore := ms.Mallocs

		// Reuse buffer
		buf := make([]byte, 12800)

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Reuse existing buffer (low GC pressure)
			for j := 0; j < 100; j++ {
				_ = buf[j*128 : (j+1)*128]
			}
		}

		b.StopTimer()
		runtime.ReadMemStats(&ms)
		allocsAfter := ms.Mallocs

		b.ReportMetric(float64(allocsAfter-allocsBefore)/float64(b.N), "allocs/op")
	})
}

// BenchmarkConnectionMemory measures per-connection memory overhead
func BenchmarkConnectionMemory(b *testing.B) {
	b.ReportAllocs()

	localAddr := common.IPv4Address{127, 0, 0, 1}

	var ms runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&ms)
	memBefore := ms.Alloc

	b.ResetTimer()

	connections := make([]*tcp.Connection, b.N)
	for i := 0; i < b.N; i++ {
		remoteAddr := common.IPv4Address{
			127, 0,
			byte((i >> 8) & 0xFF),
			byte(i & 0xFF),
		}
		connections[i] = tcp.NewConnection(localAddr, 8080, remoteAddr, 9090)
	}

	b.StopTimer()
	runtime.ReadMemStats(&ms)
	memAfter := ms.Alloc

	if b.N > 0 {
		b.ReportMetric(float64(memAfter-memBefore)/float64(b.N), "bytes/conn")
	}
}

// BenchmarkSegmentMemory measures per-segment memory overhead
func BenchmarkSegmentMemory(b *testing.B) {
	b.ReportAllocs()

	payload := make([]byte, 1024)

	var ms runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&ms)
	memBefore := ms.Alloc

	b.ResetTimer()

	segments := make([]*tcp.Segment, b.N)
	for i := 0; i < b.N; i++ {
		segments[i] = tcp.NewSegment(
			8080, 9090,
			uint32(i*1024), uint32(i*1024),
			tcp.FlagACK, 65535,
			payload,
		)
	}

	b.StopTimer()
	runtime.ReadMemStats(&ms)
	memAfter := ms.Alloc

	if b.N > 0 {
		b.ReportMetric(float64(memAfter-memBefore)/float64(b.N), "bytes/segment")
	}
}

// BenchmarkReceiveBufferMemory measures receive buffer memory usage
func BenchmarkReceiveBufferMemory(b *testing.B) {
	sizes := []int{4096, 16384, 65536}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("BufferSize_%d", size), func(b *testing.B) {
			b.ReportAllocs()

			var ms runtime.MemStats
			runtime.GC()
			runtime.ReadMemStats(&ms)
			memBefore := ms.Alloc

			b.ResetTimer()

			buffers := make([]*tcp.ReceiveBuffer, b.N)
			for i := 0; i < b.N; i++ {
				buffers[i] = tcp.NewReceiveBuffer(size)
			}

			b.StopTimer()
			runtime.ReadMemStats(&ms)
			memAfter := ms.Alloc

			if b.N > 0 {
				b.ReportMetric(float64(memAfter-memBefore)/float64(b.N), "bytes/buffer")
			}
		})
	}
}

// BenchmarkMemoryFragmentation simulates fragmentation patterns
func BenchmarkMemoryFragmentation(b *testing.B) {
	b.ReportAllocs()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Allocate varying sizes (simulates fragmentation)
		small := make([]byte, 64)
		medium := make([]byte, 512)
		large := make([]byte, 4096)

		_ = small
		_ = medium
		_ = large
	}
}

// BenchmarkHeapVsStack measures heap vs stack allocation
func BenchmarkHeapVsStack(b *testing.B) {
	b.Run("StackAllocation", func(b *testing.B) {
		b.ReportAllocs()

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Small array on stack
			var buf [128]byte
			_ = buf
		}
	})

	b.Run("HeapAllocation", func(b *testing.B) {
		b.ReportAllocs()

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			// Slice allocated on heap
			buf := make([]byte, 128)
			_ = buf
		}
	})
}

// BenchmarkPooledBuffers measures buffer pool performance
func BenchmarkPooledBuffers(b *testing.B) {
	b.ReportAllocs()

	// Simple buffer pool simulation
	pool := make(chan []byte, 100)
	for i := 0; i < 100; i++ {
		pool <- make([]byte, 1500)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		select {
		case buf := <-pool:
			// Use buffer
			_ = buf
			// Return to pool
			pool <- buf
		default:
			// Pool empty, allocate new
			buf := make([]byte, 1500)
			_ = buf
		}
	}
}

// BenchmarkSliceCapacity measures impact of slice capacity
func BenchmarkSliceCapacity(b *testing.B) {
	b.Run("ExactCapacity", func(b *testing.B) {
		b.ReportAllocs()

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := make([]byte, 1500, 1500)
			_ = buf
		}
	})

	b.Run("ExcessCapacity", func(b *testing.B) {
		b.ReportAllocs()

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := make([]byte, 1500, 4096)
			_ = buf
		}
	})
}

// BenchmarkAppendPattern measures different append patterns
func BenchmarkAppendPattern(b *testing.B) {
	b.Run("PreAllocated", func(b *testing.B) {
		b.ReportAllocs()

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := make([]byte, 0, 1500)
			for j := 0; j < 150; j++ {
				buf = append(buf, byte(j))
			}
		}
	})

	b.Run("GrowAsNeeded", func(b *testing.B) {
		b.ReportAllocs()

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			var buf []byte
			for j := 0; j < 150; j++ {
				buf = append(buf, byte(j))
			}
		}
	})
}

// BenchmarkMemoryLeakDetection checks for potential memory leaks
func BenchmarkMemoryLeakDetection(b *testing.B) {
	localAddr := common.IPv4Address{127, 0, 0, 1}
	remoteAddr := common.IPv4Address{127, 0, 0, 2}

	var ms runtime.MemStats

	// Warmup
	for i := 0; i < 100; i++ {
		conn := tcp.NewConnection(localAddr, 8080, remoteAddr, 9090)
		_ = conn
	}

	runtime.GC()
	runtime.ReadMemStats(&ms)
	startAlloc := ms.Alloc

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		conn := tcp.NewConnection(localAddr, 8080, remoteAddr, 9090)
		_ = conn

		if i%1000 == 0 {
			runtime.GC()
		}
	}

	b.StopTimer()
	runtime.GC()
	runtime.ReadMemStats(&ms)
	endAlloc := ms.Alloc

	b.ReportMetric(float64(endAlloc-startAlloc)/(1024*1024), "MB_growth")
}

// BenchmarkCopyVsAssignment measures copy vs assignment overhead
func BenchmarkCopyVsAssignment(b *testing.B) {
	src := make([]byte, 1500)
	rand.Read(src)

	b.Run("Copy", func(b *testing.B) {
		b.ReportAllocs()

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			dst := make([]byte, len(src))
			copy(dst, src)
		}
	})

	b.Run("Append", func(b *testing.B) {
		b.ReportAllocs()

		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			dst := append([]byte(nil), src...)
			_ = dst
		}
	})
}
