package ip

import (
	"testing"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

func TestRoutingTable_AddRoute(t *testing.T) {
	rt := NewRoutingTable()

	destIP, _ := common.ParseIPv4("192.168.1.0")
	netmask, _ := common.ParseIPv4("255.255.255.0")
	gateway, _ := common.ParseIPv4("192.168.1.1")

	route := &Route{
		Destination: destIP,
		Netmask:     netmask,
		Gateway:     gateway,
		Interface:   "eth0",
		Metric:      0,
	}

	err := rt.AddRoute(route)
	if err != nil {
		t.Fatalf("AddRoute() error = %v", err)
	}

	routes := rt.GetRoutes()
	if len(routes) != 1 {
		t.Errorf("Expected 1 route, got %d", len(routes))
	}
}

func TestRoutingTable_RemoveRoute(t *testing.T) {
	rt := NewRoutingTable()

	destIP, _ := common.ParseIPv4("192.168.1.0")
	netmask, _ := common.ParseIPv4("255.255.255.0")
	gateway, _ := common.ParseIPv4("192.168.1.1")

	route := &Route{
		Destination: destIP,
		Netmask:     netmask,
		Gateway:     gateway,
		Interface:   "eth0",
		Metric:      0,
	}

	rt.AddRoute(route)

	// Remove the route
	removed := rt.RemoveRoute(destIP, netmask)
	if !removed {
		t.Error("RemoveRoute() = false, want true")
	}

	routes := rt.GetRoutes()
	if len(routes) != 0 {
		t.Errorf("Expected 0 routes, got %d", len(routes))
	}

	// Try to remove again
	removed = rt.RemoveRoute(destIP, netmask)
	if removed {
		t.Error("RemoveRoute() = true for non-existent route, want false")
	}
}

func TestRoutingTable_Lookup(t *testing.T) {
	rt := NewRoutingTable()

	// Add a local network route
	localNet, _ := common.ParseIPv4("192.168.1.0")
	localMask, _ := common.ParseIPv4("255.255.255.0")
	rt.AddRoute(&Route{
		Destination: localNet,
		Netmask:     localMask,
		Gateway:     common.IPv4Address{0, 0, 0, 0}, // Direct route
		Interface:   "eth0",
		Metric:      0,
	})

	// Add a default route
	defaultGW, _ := common.ParseIPv4("192.168.1.1")
	rt.SetDefaultGateway(defaultGW, "eth0")

	tests := []struct {
		name           string
		dst            string
		wantInterface  string
		wantNextHop    string
		wantErr        bool
	}{
		{
			name:          "local network",
			dst:           "192.168.1.100",
			wantInterface: "eth0",
			wantNextHop:   "192.168.1.100", // Direct route
			wantErr:       false,
		},
		{
			name:          "via default gateway",
			dst:           "8.8.8.8",
			wantInterface: "eth0",
			wantNextHop:   "192.168.1.1", // Via gateway
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dstIP, _ := common.ParseIPv4(tt.dst)
			route, nextHop, err := rt.Lookup(dstIP)

			if (err != nil) != tt.wantErr {
				t.Errorf("Lookup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if route.Interface != tt.wantInterface {
					t.Errorf("Interface = %s, want %s", route.Interface, tt.wantInterface)
				}

				wantNextHopIP, _ := common.ParseIPv4(tt.wantNextHop)
				if nextHop != wantNextHopIP {
					t.Errorf("NextHop = %s, want %s", nextHop, wantNextHopIP)
				}
			}
		})
	}
}

func TestRoutingTable_LongestPrefixMatch(t *testing.T) {
	rt := NewRoutingTable()

	// Add multiple routes with different prefix lengths
	// 192.168.1.0/24
	net1, _ := common.ParseIPv4("192.168.1.0")
	mask1, _ := common.ParseIPv4("255.255.255.0")
	gw1, _ := common.ParseIPv4("192.168.1.1")
	rt.AddRoute(&Route{
		Destination: net1,
		Netmask:     mask1,
		Gateway:     gw1,
		Interface:   "eth0",
		Metric:      0,
	})

	// 192.168.1.128/25 (more specific)
	net2, _ := common.ParseIPv4("192.168.1.128")
	mask2, _ := common.ParseIPv4("255.255.255.128")
	gw2, _ := common.ParseIPv4("192.168.1.129")
	rt.AddRoute(&Route{
		Destination: net2,
		Netmask:     mask2,
		Gateway:     gw2,
		Interface:   "eth1",
		Metric:      0,
	})

	// Lookup should return the more specific route
	dstIP, _ := common.ParseIPv4("192.168.1.200")
	route, nextHop, err := rt.Lookup(dstIP)
	if err != nil {
		t.Fatalf("Lookup() error = %v", err)
	}

	// Should match the /25 route (more specific)
	if route.Interface != "eth1" {
		t.Errorf("Interface = %s, want eth1 (more specific route)", route.Interface)
	}

	if nextHop != gw2 {
		t.Errorf("NextHop = %s, want %s", nextHop, gw2)
	}
}

func TestRoutingTable_SetDefaultGateway(t *testing.T) {
	rt := NewRoutingTable()

	gateway, _ := common.ParseIPv4("192.168.1.1")
	err := rt.SetDefaultGateway(gateway, "eth0")
	if err != nil {
		t.Fatalf("SetDefaultGateway() error = %v", err)
	}

	defaultGW := rt.GetDefaultGateway()
	if defaultGW == nil {
		t.Fatal("GetDefaultGateway() returned nil")
	}

	if defaultGW.Gateway != gateway {
		t.Errorf("Gateway = %s, want %s", defaultGW.Gateway, gateway)
	}

	if defaultGW.Interface != "eth0" {
		t.Errorf("Interface = %s, want eth0", defaultGW.Interface)
	}
}

func TestRoutingTable_LocalInterface(t *testing.T) {
	rt := NewRoutingTable()

	ip, _ := common.ParseIPv4("192.168.1.100")
	rt.AddLocalInterface("eth0", ip)

	// Test GetLocalInterface
	retrievedIP, exists := rt.GetLocalInterface("eth0")
	if !exists {
		t.Error("GetLocalInterface() returned false for existing interface")
	}
	if retrievedIP != ip {
		t.Errorf("IP = %s, want %s", retrievedIP, ip)
	}

	// Test non-existent interface
	_, exists = rt.GetLocalInterface("eth1")
	if exists {
		t.Error("GetLocalInterface() returned true for non-existent interface")
	}
}

func TestRoutingTable_IsLocalAddress(t *testing.T) {
	rt := NewRoutingTable()

	localIP, _ := common.ParseIPv4("192.168.1.100")
	remoteIP, _ := common.ParseIPv4("8.8.8.8")

	rt.AddLocalInterface("eth0", localIP)

	if !rt.IsLocalAddress(localIP) {
		t.Error("IsLocalAddress() = false for local IP, want true")
	}

	if rt.IsLocalAddress(remoteIP) {
		t.Error("IsLocalAddress() = true for remote IP, want false")
	}
}

func TestRoutingTable_matches(t *testing.T) {
	rt := NewRoutingTable()

	tests := []struct {
		name    string
		ip      string
		network string
		netmask string
		want    bool
	}{
		{
			name:    "exact match /32",
			ip:      "192.168.1.1",
			network: "192.168.1.1",
			netmask: "255.255.255.255",
			want:    true,
		},
		{
			name:    "match in /24",
			ip:      "192.168.1.100",
			network: "192.168.1.0",
			netmask: "255.255.255.0",
			want:    true,
		},
		{
			name:    "no match different network",
			ip:      "192.168.2.100",
			network: "192.168.1.0",
			netmask: "255.255.255.0",
			want:    false,
		},
		{
			name:    "default route 0.0.0.0/0",
			ip:      "8.8.8.8",
			network: "0.0.0.0",
			netmask: "0.0.0.0",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip, _ := common.ParseIPv4(tt.ip)
			network, _ := common.ParseIPv4(tt.network)
			netmask, _ := common.ParseIPv4(tt.netmask)

			got := rt.matches(ip, network, netmask)
			if got != tt.want {
				t.Errorf("matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRoutingTable_countOnes(t *testing.T) {
	rt := NewRoutingTable()

	tests := []struct {
		netmask string
		want    int
	}{
		{"255.255.255.255", 32},
		{"255.255.255.0", 24},
		{"255.255.0.0", 16},
		{"255.0.0.0", 8},
		{"0.0.0.0", 0},
		{"255.255.255.128", 25},
	}

	for _, tt := range tests {
		t.Run(tt.netmask, func(t *testing.T) {
			netmask, _ := common.ParseIPv4(tt.netmask)
			got := rt.countOnes(netmask)
			if got != tt.want {
				t.Errorf("countOnes() = %d, want %d", got, tt.want)
			}
		})
	}
}

func BenchmarkLookup(b *testing.B) {
	rt := NewRoutingTable()

	// Add some routes
	for i := 0; i < 100; i++ {
		destIP := common.IPv4Address{192, 168, byte(i), 0}
		netmask := common.IPv4Address{255, 255, 255, 0}
		gateway := common.IPv4Address{192, 168, byte(i), 1}
		rt.AddRoute(&Route{
			Destination: destIP,
			Netmask:     netmask,
			Gateway:     gateway,
			Interface:   "eth0",
			Metric:      0,
		})
	}

	dstIP, _ := common.ParseIPv4("192.168.50.100")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = rt.Lookup(dstIP)
	}
}
