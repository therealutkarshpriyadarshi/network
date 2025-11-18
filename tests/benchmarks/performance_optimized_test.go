package benchmarks

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
	"github.com/therealutkarshpriyadarshi/network/pkg/ip"
)

// Benchmark trie-based routing lookup performance
func BenchmarkTrieRoutingLookup(b *testing.B) {
	sizes := []int{10, 100, 1000, 10000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Routes_%d", size), func(b *testing.B) {
			table := ip.NewTrieRoutingTable()

			// Add routes
			for i := 0; i < size; i++ {
				route := &ip.Route{
					Destination: common.IPv4Address{byte(i >> 24), byte(i >> 16), byte(i >> 8), byte(i)},
					Netmask:     common.IPv4Address{255, 255, 255, 0},
					Gateway:     common.IPv4Address{192, 168, 1, 1},
					Interface:   "eth0",
					Metric:      10,
				}
				table.AddRoute(route)
			}

			// Test lookup
			dst := common.IPv4Address{10, 0, 0, 1}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				table.Lookup(dst)
			}
		})
	}
}

// Benchmark trie vs optimized routing comparison
func BenchmarkRoutingComparison(b *testing.B) {
	const numRoutes = 1000

	// Create routes
	routes := make([]*ip.Route, numRoutes)
	for i := 0; i < numRoutes; i++ {
		routes[i] = &ip.Route{
			Destination: common.IPv4Address{byte(i >> 24), byte(i >> 16), byte(i >> 8), 0},
			Netmask:     common.IPv4Address{255, 255, 255, 0},
			Gateway:     common.IPv4Address{192, 168, 1, 1},
			Interface:   "eth0",
			Metric:      10,
		}
	}

	dst := common.IPv4Address{10, 0, 0, 1}

	b.Run("OptimizedRouting", func(b *testing.B) {
		table := ip.NewOptimizedRoutingTable()
		for _, route := range routes {
			table.AddRoute(route)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			table.Lookup(dst)
		}
	})

	b.Run("TrieRouting", func(b *testing.B) {
		table := ip.NewTrieRoutingTable()
		for _, route := range routes {
			table.AddRoute(route)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			table.Lookup(dst)
		}
	})

	b.Run("TrieRoutingWithCache", func(b *testing.B) {
		table := ip.NewTrieRoutingTableWithCache()
		for _, route := range routes {
			table.AddRoute(route)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			table.Lookup(dst)
		}
	})
}

// Benchmark random lookups (realistic scenario)
func BenchmarkTrieRoutingRandomLookup(b *testing.B) {
	table := ip.NewTrieRoutingTable()

	// Add 1000 routes
	for i := 0; i < 1000; i++ {
		route := &ip.Route{
			Destination: common.IPv4Address{byte(rand.Intn(256)), byte(rand.Intn(256)), byte(rand.Intn(256)), 0},
			Netmask:     common.IPv4Address{255, 255, 255, 0},
			Gateway:     common.IPv4Address{192, 168, 1, 1},
			Interface:   "eth0",
			Metric:      10,
		}
		table.AddRoute(route)
	}

	// Generate random IPs
	ips := make([]common.IPv4Address, 1000)
	for i := 0; i < 1000; i++ {
		ips[i] = common.IPv4Address{
			byte(rand.Intn(256)),
			byte(rand.Intn(256)),
			byte(rand.Intn(256)),
			byte(rand.Intn(256)),
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		table.Lookup(ips[i%1000])
	}
}

// Benchmark concurrent trie lookups
func BenchmarkTrieRoutingConcurrent(b *testing.B) {
	table := ip.NewTrieRoutingTable()

	// Add routes
	for i := 0; i < 1000; i++ {
		route := &ip.Route{
			Destination: common.IPv4Address{byte(i >> 24), byte(i >> 16), byte(i >> 8), 0},
			Netmask:     common.IPv4Address{255, 255, 255, 0},
			Gateway:     common.IPv4Address{192, 168, 1, 1},
			Interface:   "eth0",
			Metric:      10,
		}
		table.AddRoute(route)
	}

	dst := common.IPv4Address{10, 0, 0, 1}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			table.Lookup(dst)
		}
	})
}

// Benchmark SIMD checksum vs standard
func BenchmarkChecksumSIMDComparison(b *testing.B) {
	sizes := []int{64, 512, 1024, 1500, 4096}

	for _, size := range sizes {
		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i)
		}

		b.Run(fmt.Sprintf("Standard_%d", size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				common.CalculateChecksum(data)
			}
		})

		b.Run(fmt.Sprintf("Optimized_%d", size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				common.CalculateChecksumFast(data)
			}
		})

		b.Run(fmt.Sprintf("SIMD_%d", size), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				common.CalculateChecksumSIMD(data)
			}
		})
	}
}

// Benchmark checksum with hardware offload
func BenchmarkChecksumWithOffload(b *testing.B) {
	data := make([]byte, 1500)
	for i := range data {
		data[i] = byte(i)
	}

	// Initialize offload with all capabilities
	common.InitChecksumOffload(
		common.ChecksumOffloadTxIPv4 |
			common.ChecksumOffloadTxTCP |
			common.ChecksumOffloadTxUDP |
			common.ChecksumOffloadRxIPv4 |
			common.ChecksumOffloadRxTCP |
			common.ChecksumOffloadRxUDP,
	)

	b.Run("WithoutOffload", func(b *testing.B) {
		common.DisableChecksumOffload()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			common.CalculateChecksumWithOffload(data, common.ProtocolTCP)
		}
	})

	b.Run("WithOffload", func(b *testing.B) {
		common.EnableChecksumOffload()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			common.CalculateChecksumWithOffload(data, common.ProtocolTCP)
		}
	})
}

// Benchmark end-to-end routing + checksum performance
func BenchmarkEndToEndPerformance(b *testing.B) {
	// Setup routing table
	table := ip.NewTrieRoutingTableWithCache()
	for i := 0; i < 1000; i++ {
		route := &ip.Route{
			Destination: common.IPv4Address{byte(i >> 24), byte(i >> 16), byte(i >> 8), 0},
			Netmask:     common.IPv4Address{255, 255, 255, 0},
			Gateway:     common.IPv4Address{192, 168, 1, 1},
			Interface:   "eth0",
			Metric:      10,
		}
		table.AddRoute(route)
	}

	// Setup packet
	dst := common.IPv4Address{10, 0, 0, 1}
	data := make([]byte, 1500)
	for i := range data {
		data[i] = byte(i)
	}

	// Enable checksum offload
	common.InitChecksumOffload(common.ChecksumOffloadTxTCP | common.ChecksumOffloadTxUDP)
	common.EnableChecksumOffload()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Lookup route
		_, _, _ = table.Lookup(dst)

		// Calculate checksum
		_ = common.CalculateChecksumWithOffload(data, common.ProtocolTCP)
	}

	// Report latency in microseconds
	nsPerOp := b.Elapsed().Nanoseconds() / int64(b.N)
	usPerOp := float64(nsPerOp) / 1000.0
	b.ReportMetric(usPerOp, "µs/op")
}

// Benchmark worst-case scenario
func BenchmarkWorstCase(b *testing.B) {
	table := ip.NewTrieRoutingTable()

	// Add 10000 routes (stress test)
	for i := 0; i < 10000; i++ {
		route := &ip.Route{
			Destination: common.IPv4Address{byte(i >> 24), byte(i >> 16), byte(i >> 8), 0},
			Netmask:     common.IPv4Address{255, 255, 255, 0},
			Gateway:     common.IPv4Address{192, 168, 1, 1},
			Interface:   "eth0",
			Metric:      10,
		}
		table.AddRoute(route)
	}

	// Lookup non-existent route (worst case)
	dst := common.IPv4Address{255, 255, 255, 255}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		table.Lookup(dst)
	}

	// Report latency
	nsPerOp := b.Elapsed().Nanoseconds() / int64(b.N)
	usPerOp := float64(nsPerOp) / 1000.0
	b.ReportMetric(usPerOp, "µs/op")
}

// Benchmark best-case scenario (cached lookup)
func BenchmarkBestCase(b *testing.B) {
	table := ip.NewTrieRoutingTableWithCache()

	// Add 1000 routes
	for i := 0; i < 1000; i++ {
		route := &ip.Route{
			Destination: common.IPv4Address{byte(i >> 24), byte(i >> 16), byte(i >> 8), 0},
			Netmask:     common.IPv4Address{255, 255, 255, 0},
			Gateway:     common.IPv4Address{192, 168, 1, 1},
			Interface:   "eth0",
			Metric:      10,
		}
		table.AddRoute(route)
	}

	dst := common.IPv4Address{0, 0, 0, 1}

	// Warm up cache
	table.Lookup(dst)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		table.Lookup(dst)
	}

	// Report latency
	nsPerOp := b.Elapsed().Nanoseconds() / int64(b.N)
	usPerOp := float64(nsPerOp) / 1000.0
	b.ReportMetric(usPerOp, "µs/op")
}

// Benchmark realistic internet routing table
func BenchmarkRealisticRoutingTable(b *testing.B) {
	table := ip.NewTrieRoutingTableWithCache()

	// Simulate a realistic routing table with common prefixes
	commonPrefixes := []struct {
		prefix  common.IPv4Address
		mask    common.IPv4Address
		gateway common.IPv4Address
	}{
		// Private networks
		{common.IPv4Address{10, 0, 0, 0}, common.IPv4Address{255, 0, 0, 0}, common.IPv4Address{10, 0, 0, 1}},
		{common.IPv4Address{172, 16, 0, 0}, common.IPv4Address{255, 240, 0, 0}, common.IPv4Address{172, 16, 0, 1}},
		{common.IPv4Address{192, 168, 0, 0}, common.IPv4Address{255, 255, 0, 0}, common.IPv4Address{192, 168, 0, 1}},

		// Public internet routes (simulated)
		{common.IPv4Address{8, 8, 8, 0}, common.IPv4Address{255, 255, 255, 0}, common.IPv4Address{192, 168, 1, 1}},
		{common.IPv4Address{1, 1, 1, 0}, common.IPv4Address{255, 255, 255, 0}, common.IPv4Address{192, 168, 1, 1}},
	}

	// Add common routes
	for _, p := range commonPrefixes {
		route := &ip.Route{
			Destination: p.prefix,
			Netmask:     p.mask,
			Gateway:     p.gateway,
			Interface:   "eth0",
			Metric:      10,
		}
		table.AddRoute(route)
	}

	// Add more specific routes
	for i := 0; i < 100; i++ {
		route := &ip.Route{
			Destination: common.IPv4Address{byte(rand.Intn(256)), byte(rand.Intn(256)), byte(rand.Intn(256)), 0},
			Netmask:     common.IPv4Address{255, 255, 255, 0},
			Gateway:     common.IPv4Address{192, 168, 1, 1},
			Interface:   "eth0",
			Metric:      10,
		}
		table.AddRoute(route)
	}

	// Test with common destinations
	testAddrs := []common.IPv4Address{
		{10, 0, 0, 1},       // Private
		{192, 168, 1, 100},  // Private
		{8, 8, 8, 8},        // Google DNS
		{1, 1, 1, 1},        // Cloudflare DNS
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		table.Lookup(testAddrs[i%len(testAddrs)])
	}

	// Report latency
	nsPerOp := b.Elapsed().Nanoseconds() / int64(b.N)
	usPerOp := float64(nsPerOp) / 1000.0
	b.ReportMetric(usPerOp, "µs/op")

	if usPerOp < 1.0 {
		b.Logf("✓ Target achieved: %.3f µs/op (target: <1 µs)", usPerOp)
	} else {
		b.Logf("✗ Target not met: %.3f µs/op (target: <1 µs)", usPerOp)
	}
}
