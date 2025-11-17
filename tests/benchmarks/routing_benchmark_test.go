package benchmarks

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
	"github.com/therealutkarshpriyadarshi/network/pkg/ip"
)

// BenchmarkRoutingTableLookup measures routing table lookup performance
func BenchmarkRoutingTableLookup(b *testing.B) {
	routeTableSizes := []int{10, 100, 1000, 10000}

	for _, size := range routeTableSizes {
		b.Run(fmt.Sprintf("Routes_%d", size), func(b *testing.B) {
			benchmarkRoutingLookup(b, size)
		})
	}
}

func benchmarkRoutingLookup(b *testing.B, numRoutes int) {
	rt := ip.NewRoutingTable()

	// Add routes
	for i := 0; i < numRoutes; i++ {
		route := &ip.Route{
			Destination: common.IPv4Address{
				byte((i >> 24) & 0xFF),
				byte((i >> 16) & 0xFF),
				byte((i >> 8) & 0xFF),
				0,
			},
			Netmask: common.IPv4Address{255, 255, 255, 0},
			Gateway: common.IPv4Address{192, 168, 1, 1},
			Interface: "eth0",
			Metric: i % 100,
		}
		rt.AddRoute(route)
	}

	// Test IP that should match
	testIP := common.IPv4Address{
		byte((numRoutes/2 >> 24) & 0xFF),
		byte((numRoutes/2 >> 16) & 0xFF),
		byte((numRoutes/2 >> 8) & 0xFF),
		100,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, err := rt.Lookup(testIP)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRoutingTableLookupWorstCase measures worst-case lookup (no match)
func BenchmarkRoutingTableLookupWorstCase(b *testing.B) {
	rt := ip.NewRoutingTable()

	// Add 1000 routes
	for i := 0; i < 1000; i++ {
		route := &ip.Route{
			Destination: common.IPv4Address{10, byte(i >> 8), byte(i & 0xFF), 0},
			Netmask: common.IPv4Address{255, 255, 255, 0},
			Gateway: common.IPv4Address{192, 168, 1, 1},
			Interface: "eth0",
			Metric: i,
		}
		rt.AddRoute(route)
	}

	// Default route
	defaultRoute := &ip.Route{
		Destination: common.IPv4Address{0, 0, 0, 0},
		Netmask: common.IPv4Address{0, 0, 0, 0},
		Gateway: common.IPv4Address{192, 168, 1, 1},
		Interface: "eth0",
		Metric: 1000,
	}
	rt.AddRoute(defaultRoute)

	// IP that only matches default route (worst case)
	testIP := common.IPv4Address{8, 8, 8, 8}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, err := rt.Lookup(testIP)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRoutingTableLookupBestCase measures best-case lookup (first match)
func BenchmarkRoutingTableLookupBestCase(b *testing.B) {
	rt := ip.NewRoutingTable()

	// Add specific route first
	specificRoute := &ip.Route{
		Destination: common.IPv4Address{192, 168, 1, 0},
		Netmask: common.IPv4Address{255, 255, 255, 0},
		Gateway: common.IPv4Address{192, 168, 1, 1},
		Interface: "eth0",
		Metric: 1,
	}
	rt.AddRoute(specificRoute)

	// Add 1000 other routes
	for i := 0; i < 1000; i++ {
		route := &ip.Route{
			Destination: common.IPv4Address{10, byte(i >> 8), byte(i & 0xFF), 0},
			Netmask: common.IPv4Address{255, 255, 255, 0},
			Gateway: common.IPv4Address{192, 168, 1, 1},
			Interface: "eth0",
			Metric: i + 10,
		}
		rt.AddRoute(route)
	}

	// IP that matches first route
	testIP := common.IPv4Address{192, 168, 1, 100}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, err := rt.Lookup(testIP)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRoutingTableAdd measures route addition performance
func BenchmarkRoutingTableAdd(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		rt := ip.NewRoutingTable()
		b.StartTimer()

		route := &ip.Route{
			Destination: common.IPv4Address{192, 168, byte(i >> 8), 0},
			Netmask: common.IPv4Address{255, 255, 255, 0},
			Gateway: common.IPv4Address{192, 168, 1, 1},
			Interface: "eth0",
			Metric: 1,
		}
		rt.AddRoute(route)
	}
}

// BenchmarkRoutingTableRemove measures route removal performance
func BenchmarkRoutingTableRemove(b *testing.B) {
	routes := make([]*ip.Route, b.N)
	for i := 0; i < b.N; i++ {
		routes[i] = &ip.Route{
			Destination: common.IPv4Address{192, 168, byte(i >> 8), 0},
			Netmask: common.IPv4Address{255, 255, 255, 0},
			Gateway: common.IPv4Address{192, 168, 1, 1},
			Interface: "eth0",
			Metric: 1,
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		rt := ip.NewRoutingTable()
		for _, r := range routes {
			rt.AddRoute(r)
		}
		b.StartTimer()

		rt.RemoveRoute(routes[i].Destination, routes[i].Netmask)
	}
}

// BenchmarkRoutingTableLongestPrefixMatch tests longest prefix matching
func BenchmarkRoutingTableLongestPrefixMatch(b *testing.B) {
	rt := ip.NewRoutingTable()

	// Add routes with different prefix lengths
	routes := []*ip.Route{
		{
			Destination: common.IPv4Address{192, 0, 0, 0},
			Netmask: common.IPv4Address{255, 0, 0, 0},
			Gateway: common.IPv4Address{192, 168, 1, 1},
			Interface: "eth0",
			Metric: 1,
		},
		{
			Destination: common.IPv4Address{192, 168, 0, 0},
			Netmask: common.IPv4Address{255, 255, 0, 0},
			Gateway: common.IPv4Address{192, 168, 1, 1},
			Interface: "eth0",
			Metric: 1,
		},
		{
			Destination: common.IPv4Address{192, 168, 1, 0},
			Netmask: common.IPv4Address{255, 255, 255, 0},
			Gateway: common.IPv4Address{192, 168, 1, 1},
			Interface: "eth0",
			Metric: 1,
		},
		{
			Destination: common.IPv4Address{192, 168, 1, 128},
			Netmask: common.IPv4Address{255, 255, 255, 128},
			Gateway: common.IPv4Address{192, 168, 1, 1},
			Interface: "eth0",
			Metric: 1,
		},
	}

	for _, r := range routes {
		rt.AddRoute(r)
	}

	// IP that should match most specific route
	testIP := common.IPv4Address{192, 168, 1, 200}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, err := rt.Lookup(testIP)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRoutingTableRandomLookup measures lookup with random IPs
func BenchmarkRoutingTableRandomLookup(b *testing.B) {
	rt := ip.NewRoutingTable()

	// Add 1000 routes
	for i := 0; i < 1000; i++ {
		route := &ip.Route{
			Destination: common.IPv4Address{
				byte(rand.Intn(256)),
				byte(rand.Intn(256)),
				byte(rand.Intn(256)),
				0,
			},
			Netmask: common.IPv4Address{255, 255, 255, 0},
			Gateway: common.IPv4Address{192, 168, 1, 1},
			Interface: "eth0",
			Metric: i,
		}
		rt.AddRoute(route)
	}

	// Default route
	defaultRoute := &ip.Route{
		Destination: common.IPv4Address{0, 0, 0, 0},
		Netmask: common.IPv4Address{0, 0, 0, 0},
		Gateway: common.IPv4Address{192, 168, 1, 1},
		Interface: "eth0",
		Metric: 1000,
	}
	rt.AddRoute(defaultRoute)

	// Pre-generate random IPs
	testIPs := make([]common.IPv4Address, 1000)
	for i := 0; i < 1000; i++ {
		testIPs[i] = common.IPv4Address{
			byte(rand.Intn(256)),
			byte(rand.Intn(256)),
			byte(rand.Intn(256)),
			byte(rand.Intn(256)),
		}
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		testIP := testIPs[i%len(testIPs)]
		rt.Lookup(testIP)
	}
}

// BenchmarkRoutingTableConcurrent measures concurrent lookup performance
func BenchmarkRoutingTableConcurrent(b *testing.B) {
	rt := ip.NewRoutingTable()

	// Add 1000 routes
	for i := 0; i < 1000; i++ {
		route := &ip.Route{
			Destination: common.IPv4Address{10, byte(i >> 8), byte(i & 0xFF), 0},
			Netmask: common.IPv4Address{255, 255, 255, 0},
			Gateway: common.IPv4Address{192, 168, 1, 1},
			Interface: "eth0",
			Metric: i,
		}
		rt.AddRoute(route)
	}

	testIP := common.IPv4Address{10, 1, 100, 50}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rt.Lookup(testIP)
		}
	})
}

// BenchmarkRoutingTableCIDRLookup measures CIDR-based lookups
func BenchmarkRoutingTableCIDRLookup(b *testing.B) {
	prefixLengths := []int{8, 16, 24, 32}

	for _, prefixLen := range prefixLengths {
		b.Run(fmt.Sprintf("PrefixLen_%d", prefixLen), func(b *testing.B) {
			rt := ip.NewRoutingTable()

			// Create netmask for prefix length
			var netmask common.IPv4Address
			for i := 0; i < 4; i++ {
				if prefixLen >= (i+1)*8 {
					netmask[i] = 255
				} else if prefixLen > i*8 {
					netmask[i] = byte(^uint8(0) << (8 - (prefixLen - i*8)))
				} else {
					netmask[i] = 0
				}
			}

			route := &ip.Route{
				Destination: common.IPv4Address{192, 168, 1, 0},
				Netmask: netmask,
				Gateway: common.IPv4Address{192, 168, 1, 1},
				Interface: "eth0",
				Metric: 1,
			}
			rt.AddRoute(route)

			testIP := common.IPv4Address{192, 168, 1, 100}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, _, err := rt.Lookup(testIP)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkRoutingTableMemory measures memory overhead
func BenchmarkRoutingTableMemory(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rt := ip.NewRoutingTable()

		for j := 0; j < 100; j++ {
			route := &ip.Route{
				Destination: common.IPv4Address{192, 168, byte(j), 0},
				Netmask: common.IPv4Address{255, 255, 255, 0},
				Gateway: common.IPv4Address{192, 168, 1, 1},
				Interface: "eth0",
				Metric: j,
			}
			rt.AddRoute(route)
		}
	}
}
