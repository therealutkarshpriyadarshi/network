// +build integration

// Extended integration tests for IP protocol
//
// These tests extend the existing IP/ICMP tests to improve coverage
// of routing, fragmentation, TTL, and other IP features.
//
// Run with: go test -tags=integration ./tests/integration/...

package integration

import (
	"bytes"
	"testing"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
	"github.com/therealutkarshpriyadarshi/network/pkg/ip"
)

// TestIPPacketSerialization tests various IP packet configurations.
func TestIPPacketSerialization(t *testing.T) {
	srcIP, _ := common.ParseIPv4("192.168.1.100")
	dstIP, _ := common.ParseIPv4("192.168.1.1")

	tests := []struct {
		name     string
		protocol common.Protocol
		data     []byte
	}{
		{
			name:     "TCP packet",
			protocol: common.ProtocolTCP,
			data:     []byte("TCP data"),
		},
		{
			name:     "UDP packet",
			protocol: common.ProtocolUDP,
			data:     []byte("UDP data"),
		},
		{
			name:     "ICMP packet",
			protocol: common.ProtocolICMP,
			data:     []byte{0x08, 0x00}, // ICMP echo request
		},
		{
			name:     "Empty payload",
			protocol: common.ProtocolTCP,
			data:     []byte{},
		},
		{
			name:     "Large payload",
			protocol: common.ProtocolTCP,
			data:     make([]byte, 1400),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkt := ip.NewPacket(srcIP, dstIP, tt.protocol, tt.data)

			// Serialize
			data, err := pkt.Serialize()
			if err != nil {
				t.Fatalf("Serialize failed: %v", err)
			}

			// Parse
			parsed, err := ip.Parse(data)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			// Verify
			if parsed.Source != srcIP {
				t.Errorf("Source = %v, want %v", parsed.Source, srcIP)
			}
			if parsed.Destination != dstIP {
				t.Errorf("Destination = %v, want %v", parsed.Destination, dstIP)
			}
			if parsed.Protocol != tt.protocol {
				t.Errorf("Protocol = %v, want %v", parsed.Protocol, tt.protocol)
			}
			if !bytes.Equal(parsed.Payload, tt.data) {
				t.Error("Payload mismatch")
			}
			if !parsed.VerifyChecksum() {
				t.Error("Checksum verification failed")
			}
		})
	}
}

// TestIPRoutingTableOperations tests routing table operations.
func TestIPRoutingTableOperations(t *testing.T) {
	rt := ip.NewRoutingTable()

	// Add multiple routes
	routes := []struct {
		dest    string
		mask    string
		gateway string
		iface   string
		metric  int
	}{
		{"192.168.1.0", "255.255.255.0", "0.0.0.0", "eth0", 0},         // Direct route
		{"10.0.0.0", "255.0.0.0", "192.168.1.1", "eth0", 10},           // Via gateway
		{"172.16.0.0", "255.240.0.0", "192.168.1.1", "eth0", 20},       // Different metric
		{"192.168.2.0", "255.255.255.0", "192.168.1.254", "eth1", 5},   // Different interface
	}

	for _, r := range routes {
		dest, _ := common.ParseIPv4(r.dest)
		mask, _ := common.ParseIPv4(r.mask)
		gw, _ := common.ParseIPv4(r.gateway)

		rt.AddRoute(&ip.Route{
			Destination: dest,
			Netmask:     mask,
			Gateway:     gw,
			Interface:   r.iface,
			Metric:      r.metric,
		})
	}

	// Test route lookups
	tests := []struct {
		name        string
		dst         string
		wantIface   string
		wantGateway string
	}{
		{
			name:        "Local network - direct",
			dst:         "192.168.1.50",
			wantIface:   "eth0",
			wantGateway: "192.168.1.50", // Direct delivery
		},
		{
			name:        "Class A network",
			dst:         "10.5.6.7",
			wantIface:   "eth0",
			wantGateway: "192.168.1.1", // Via gateway
		},
		{
			name:        "Class B network",
			dst:         "172.20.1.1",
			wantIface:   "eth0",
			wantGateway: "192.168.1.1", // Via gateway
		},
		{
			name:        "Different subnet",
			dst:         "192.168.2.100",
			wantIface:   "eth1",
			wantGateway: "192.168.1.254", // Via different gateway
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dstIP, _ := common.ParseIPv4(tt.dst)
			route, nextHop, err := rt.Lookup(dstIP)

			if err != nil {
				t.Fatalf("Lookup failed: %v", err)
			}

			if route.Interface != tt.wantIface {
				t.Errorf("Interface = %s, want %s", route.Interface, tt.wantIface)
			}

			wantNextHop, _ := common.ParseIPv4(tt.wantGateway)
			if nextHop != wantNextHop {
				t.Errorf("NextHop = %v, want %v", nextHop, wantNextHop)
			}
		})
	}
}

// TestIPRoutingDefaultGateway tests default gateway handling.
func TestIPRoutingDefaultGateway(t *testing.T) {
	rt := ip.NewRoutingTable()

	// Add specific route
	localNet, _ := common.ParseIPv4("192.168.1.0")
	localMask, _ := common.ParseIPv4("255.255.255.0")
	rt.AddRoute(&ip.Route{
		Destination: localNet,
		Netmask:     localMask,
		Gateway:     common.IPv4Address{0, 0, 0, 0},
		Interface:   "eth0",
		Metric:      0,
	})

	// Set default gateway
	defaultGW, _ := common.ParseIPv4("192.168.1.1")
	rt.SetDefaultGateway(defaultGW, "eth0")

	// Test lookup for remote address (should use default gateway)
	remoteIP, _ := common.ParseIPv4("8.8.8.8")
	route, nextHop, err := rt.Lookup(remoteIP)

	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}

	if route.Interface != "eth0" {
		t.Errorf("Interface = %s, want eth0", route.Interface)
	}

	if nextHop != defaultGW {
		t.Errorf("NextHop = %v, want %v (default gateway)", nextHop, defaultGW)
	}

	t.Log("Default gateway routing works correctly")
}

// TestIPFragmentationSizes tests fragmentation with various sizes.
func TestIPFragmentationSizes(t *testing.T) {
	srcIP, _ := common.ParseIPv4("10.0.0.1")
	dstIP, _ := common.ParseIPv4("10.0.0.2")

	tests := []struct {
		name        string
		payloadSize int
		mtu         int
		wantFrags   int
	}{
		{
			name:        "No fragmentation needed",
			payloadSize: 100,
			mtu:         1500,
			wantFrags:   1,
		},
		{
			name:        "Two fragments",
			payloadSize: 2000,
			mtu:         1500,
			wantFrags:   2,
		},
		{
			name:        "Three fragments",
			payloadSize: 4000,
			mtu:         1500,
			wantFrags:   3,
		},
		{
			name:        "Small MTU",
			payloadSize: 1000,
			mtu:         500,
			wantFrags:   3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := make([]byte, tt.payloadSize)
			for i := range payload {
				payload[i] = byte(i % 256)
			}

			pkt := ip.NewPacket(srcIP, dstIP, common.ProtocolTCP, payload)
			fragmenter := ip.NewFragmenter()
			defer fragmenter.Close()

			fragments, err := fragmenter.Fragment(pkt, tt.mtu)
			if err != nil {
				t.Fatalf("Fragmentation failed: %v", err)
			}

			if len(fragments) != tt.wantFrags {
				t.Errorf("Fragment count = %d, want %d", len(fragments), tt.wantFrags)
			}

			// Verify fragment flags and offsets
			for i, frag := range fragments {
				isLast := i == len(fragments)-1

				if !isLast {
					// All but last should have MF (More Fragments) flag
					if (frag.FragmentOffset & 0x2000) == 0 {
						t.Errorf("Fragment %d should have MF flag set", i)
					}
				} else {
					// Last fragment should NOT have MF flag
					if (frag.FragmentOffset & 0x2000) != 0 {
						t.Errorf("Last fragment should not have MF flag")
					}
				}
			}

			// Reassemble
			reassembler := ip.NewFragmenter()
			defer reassembler.Close()

			var reassembled *ip.Packet
			for _, frag := range fragments {
				result, err := reassembler.Reassemble(frag)
				if err != nil {
					t.Fatalf("Reassembly failed: %v", err)
				}
				if result != nil {
					reassembled = result
				}
			}

			if reassembled == nil {
				t.Fatal("Reassembly did not complete")
			}

			if !bytes.Equal(reassembled.Payload, payload) {
				t.Error("Reassembled payload mismatch")
			}
		})
	}
}

// TestIPFragmentationReordering tests reassembly with out-of-order fragments.
func TestIPFragmentationReordering(t *testing.T) {
	srcIP, _ := common.ParseIPv4("10.0.0.1")
	dstIP, _ := common.ParseIPv4("10.0.0.2")

	// Create packet that will be fragmented
	payload := make([]byte, 3000)
	for i := range payload {
		payload[i] = byte(i % 256)
	}

	pkt := ip.NewPacket(srcIP, dstIP, common.ProtocolTCP, payload)

	fragmenter := ip.NewFragmenter()
	defer fragmenter.Close()

	fragments, err := fragmenter.Fragment(pkt, 1500)
	if err != nil {
		t.Fatalf("Fragmentation failed: %v", err)
	}

	if len(fragments) < 2 {
		t.Skip("Need at least 2 fragments for reordering test")
	}

	// Reassemble in different order: last, first, middle
	reassembler := ip.NewFragmenter()
	defer reassembler.Close()

	order := []int{len(fragments) - 1, 0}
	for i := 1; i < len(fragments)-1; i++ {
		order = append(order, i)
	}

	var reassembled *ip.Packet
	for _, idx := range order {
		result, err := reassembler.Reassemble(fragments[idx])
		if err != nil {
			t.Fatalf("Reassembly failed: %v", err)
		}
		if result != nil {
			reassembled = result
		}
	}

	if reassembled == nil {
		t.Fatal("Reassembly did not complete")
	}

	if !bytes.Equal(reassembled.Payload, payload) {
		t.Error("Reassembled payload mismatch with out-of-order fragments")
	}

	t.Log("Out-of-order reassembly successful")
}

// TestIPTTLBehavior tests various TTL scenarios.
func TestIPTTLBehavior(t *testing.T) {
	srcIP, _ := common.ParseIPv4("10.0.0.1")
	dstIP, _ := common.ParseIPv4("10.0.0.2")

	tests := []struct {
		name       string
		initialTTL uint8
		hops       int
		shouldDie  bool
	}{
		{"High TTL", 64, 10, false},
		{"TTL expires exactly", 5, 5, true},
		{"TTL expires before", 3, 5, true},
		{"Single hop", 1, 1, true},
		{"Zero TTL", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkt := ip.NewPacket(srcIP, dstIP, common.ProtocolTCP, []byte("test"))
			pkt.TTL = tt.initialTTL

			alive := true
			for i := 0; i < tt.hops && alive; i++ {
				alive = pkt.DecrementTTL()
			}

			if tt.shouldDie && alive {
				t.Error("Packet should have died but is still alive")
			}
			if !tt.shouldDie && !alive {
				t.Error("Packet died prematurely")
			}
		})
	}
}

// TestIPTypesOfService tests ToS/DSCP field handling.
func TestIPTypesOfService(t *testing.T) {
	srcIP, _ := common.ParseIPv4("192.168.1.1")
	dstIP, _ := common.ParseIPv4("192.168.1.2")

	tests := []struct {
		name string
		tos  uint8
		desc string
	}{
		{"Routine", 0x00, "Normal"},
		{"Priority", 0x20, "Priority"},
		{"Immediate", 0x40, "Immediate"},
		{"Flash", 0x60, "Flash"},
		{"Flash Override", 0x80, "Flash Override"},
		{"CRITIC/ECP", 0xA0, "Critical"},
		{"Internetwork Control", 0xC0, "Network Control"},
		{"Network Control", 0xE0, "Network Control"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkt := ip.NewPacket(srcIP, dstIP, common.ProtocolTCP, []byte("test"))
			pkt.DSCP = tt.tos >> 2 // DSCP is 6 bits, ToS is 8 bits

			data, err := pkt.Serialize()
			if err != nil {
				t.Fatalf("Serialize failed: %v", err)
			}

			parsed, err := ip.Parse(data)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if parsed.DSCP != tt.tos>>2 {
				t.Errorf("DSCP = 0x%02X, want 0x%02X", parsed.DSCP, tt.tos>>2)
			}

			t.Logf("%s: DSCP = 0x%02X", tt.desc, tt.tos>>2)
		})
	}
}

// TestIPChecksumWithCorruption tests checksum detection of corrupted packets.
func TestIPChecksumWithCorruption(t *testing.T) {
	srcIP, _ := common.ParseIPv4("192.168.1.1")
	dstIP, _ := common.ParseIPv4("192.168.1.2")

	pkt := ip.NewPacket(srcIP, dstIP, common.ProtocolTCP, []byte("test data"))

	data, err := pkt.Serialize()
	if err != nil {
		t.Fatalf("Serialize failed: %v", err)
	}

	// Parse original packet
	original, err := ip.Parse(data)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !original.VerifyChecksum() {
		t.Error("Original packet checksum should be valid")
	}

	// Corrupt various fields and verify checksum fails
	corruptionTests := []struct {
		name   string
		offset int
	}{
		{"TTL field", 8},
		{"Protocol field", 9},
		{"Source IP", 12},
		{"Destination IP", 16},
	}

	for _, ct := range corruptionTests {
		t.Run(ct.name, func(t *testing.T) {
			corruptedData := make([]byte, len(data))
			copy(corruptedData, data)

			// Flip a bit
			corruptedData[ct.offset] ^= 0x01

			corrupted, err := ip.Parse(corruptedData)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if corrupted.VerifyChecksum() {
				t.Errorf("Corrupted %s should fail checksum", ct.name)
			}
		})
	}
}

// TestIPPacketIdentification tests identification field for fragmentation.
func TestIPPacketIdentification(t *testing.T) {
	srcIP, _ := common.ParseIPv4("10.0.0.1")
	dstIP, _ := common.ParseIPv4("10.0.0.2")

	// Create multiple packets
	packets := make([]*ip.Packet, 5)
	for i := range packets {
		packets[i] = ip.NewPacket(srcIP, dstIP, common.ProtocolTCP, []byte("test"))
	}

	// Check that identification numbers are set
	for i, pkt := range packets {
		if pkt.Identification == 0 {
			t.Errorf("Packet %d: Identification should be non-zero", i)
		}
	}

	t.Log("Packet identification fields are set correctly")
}

// TestIPRouteMetrics tests routing table metric-based selection.
func TestIPRouteMetrics(t *testing.T) {
	rt := ip.NewRoutingTable()

	// Add two routes to same destination with different metrics
	dest, _ := common.ParseIPv4("10.0.0.0")
	mask, _ := common.ParseIPv4("255.0.0.0")
	gw1, _ := common.ParseIPv4("192.168.1.1")
	gw2, _ := common.ParseIPv4("192.168.1.2")

	// Lower metric (preferred)
	rt.AddRoute(&ip.Route{
		Destination: dest,
		Netmask:     mask,
		Gateway:     gw1,
		Interface:   "eth0",
		Metric:      10,
	})

	// Higher metric
	rt.AddRoute(&ip.Route{
		Destination: dest,
		Netmask:     mask,
		Gateway:     gw2,
		Interface:   "eth1",
		Metric:      20,
	})

	// Lookup should return route with lower metric
	testIP, _ := common.ParseIPv4("10.5.6.7")
	route, nextHop, err := rt.Lookup(testIP)

	if err != nil {
		t.Fatalf("Lookup failed: %v", err)
	}

	// Should use eth0 (lower metric)
	if route.Interface != "eth0" {
		t.Errorf("Interface = %s, want eth0 (lower metric)", route.Interface)
	}

	if nextHop != gw1 {
		t.Errorf("NextHop = %v, want %v (lower metric gateway)", nextHop, gw1)
	}

	t.Log("Metric-based route selection works correctly")
}
