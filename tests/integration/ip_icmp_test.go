package integration

import (
	"bytes"
	"testing"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
	"github.com/therealutkarshpriyadarshi/network/pkg/icmp"
	"github.com/therealutkarshpriyadarshi/network/pkg/ip"
)

// TestIPWithICMP tests IP packet containing ICMP echo request/reply.
func TestIPWithICMP(t *testing.T) {
	srcIP, _ := common.ParseIPv4("192.168.1.100")
	dstIP, _ := common.ParseIPv4("192.168.1.1")

	// Create ICMP echo request
	icmpReq := icmp.NewEchoRequest(0x1234, 1, []byte("Hello, World!"))
	icmpData, err := icmpReq.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize ICMP: %v", err)
	}

	// Create IP packet with ICMP payload
	ipPkt := ip.NewPacket(srcIP, dstIP, common.ProtocolICMP, icmpData)
	ipData, err := ipPkt.Serialize()
	if err != nil {
		t.Fatalf("Failed to serialize IP: %v", err)
	}

	// Parse IP packet
	parsedIP, err := ip.Parse(ipData)
	if err != nil {
		t.Fatalf("Failed to parse IP: %v", err)
	}

	// Verify IP fields
	if parsedIP.Source != srcIP {
		t.Errorf("Source IP = %s, want %s", parsedIP.Source, srcIP)
	}
	if parsedIP.Destination != dstIP {
		t.Errorf("Destination IP = %s, want %s", parsedIP.Destination, dstIP)
	}
	if parsedIP.Protocol != common.ProtocolICMP {
		t.Errorf("Protocol = %s, want ICMP", parsedIP.Protocol)
	}
	if !parsedIP.VerifyChecksum() {
		t.Error("IP checksum verification failed")
	}

	// Parse ICMP from IP payload
	parsedICMP, err := icmp.Parse(parsedIP.Payload)
	if err != nil {
		t.Fatalf("Failed to parse ICMP: %v", err)
	}

	// Verify ICMP fields
	if !parsedICMP.IsEchoRequest() {
		t.Error("ICMP type is not echo request")
	}
	if parsedICMP.ID != 0x1234 {
		t.Errorf("ICMP ID = 0x%04x, want 0x1234", parsedICMP.ID)
	}
	if parsedICMP.Sequence != 1 {
		t.Errorf("ICMP Sequence = %d, want 1", parsedICMP.Sequence)
	}
	if !bytes.Equal(parsedICMP.Data, []byte("Hello, World!")) {
		t.Errorf("ICMP Data = %v, want 'Hello, World!'", parsedICMP.Data)
	}
	if !parsedICMP.VerifyChecksum() {
		t.Error("ICMP checksum verification failed")
	}
}

// TestPingEchoReplyFlow tests a complete ping request/reply flow.
func TestPingEchoReplyFlow(t *testing.T) {
	clientIP, _ := common.ParseIPv4("10.0.0.1")
	serverIP, _ := common.ParseIPv4("10.0.0.2")

	// Client sends echo request
	echoReq := icmp.NewEchoRequest(0xABCD, 42, []byte("ping"))
	reqData, _ := echoReq.Serialize()

	reqPkt := ip.NewPacket(clientIP, serverIP, common.ProtocolICMP, reqData)
	reqBytes, _ := reqPkt.Serialize()

	// Server receives and parses request
	serverRcvdIP, err := ip.Parse(reqBytes)
	if err != nil {
		t.Fatalf("Server failed to parse IP: %v", err)
	}

	serverRcvdICMP, err := icmp.Parse(serverRcvdIP.Payload)
	if err != nil {
		t.Fatalf("Server failed to parse ICMP: %v", err)
	}

	if !serverRcvdICMP.IsEchoRequest() {
		t.Fatal("Server did not receive echo request")
	}

	// Server creates echo reply
	echoReply := icmp.NewEchoReply(serverRcvdICMP.ID, serverRcvdICMP.Sequence, serverRcvdICMP.Data)
	replyData, _ := echoReply.Serialize()

	// Swap source and destination for reply
	replyPkt := ip.NewPacket(serverIP, clientIP, common.ProtocolICMP, replyData)
	replyBytes, _ := replyPkt.Serialize()

	// Client receives and parses reply
	clientRcvdIP, err := ip.Parse(replyBytes)
	if err != nil {
		t.Fatalf("Client failed to parse IP: %v", err)
	}

	clientRcvdICMP, err := icmp.Parse(clientRcvdIP.Payload)
	if err != nil {
		t.Fatalf("Client failed to parse ICMP: %v", err)
	}

	// Verify reply
	if !clientRcvdICMP.IsEchoReply() {
		t.Error("Client did not receive echo reply")
	}
	if clientRcvdICMP.ID != echoReq.ID {
		t.Errorf("Reply ID = 0x%04x, want 0x%04x", clientRcvdICMP.ID, echoReq.ID)
	}
	if clientRcvdICMP.Sequence != echoReq.Sequence {
		t.Errorf("Reply Sequence = %d, want %d", clientRcvdICMP.Sequence, echoReq.Sequence)
	}
	if !bytes.Equal(clientRcvdICMP.Data, echoReq.Data) {
		t.Error("Reply data mismatch")
	}
}

// TestIPFragmentationWithICMP tests IP fragmentation with ICMP payload.
func TestIPFragmentationWithICMP(t *testing.T) {
	srcIP, _ := common.ParseIPv4("192.168.1.100")
	dstIP, _ := common.ParseIPv4("8.8.8.8")

	// Create large ICMP echo request (will require fragmentation)
	largeData := make([]byte, 3000)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	icmpReq := icmp.NewEchoRequest(0x5678, 10, largeData)
	icmpData, _ := icmpReq.Serialize()

	// Create IP packet
	ipPkt := ip.NewPacket(srcIP, dstIP, common.ProtocolICMP, icmpData)

	// Fragment the packet
	fragmenter := ip.NewFragmenter()
	defer fragmenter.Close()

	fragments, err := fragmenter.Fragment(ipPkt, 1500)
	if err != nil {
		t.Fatalf("Failed to fragment: %v", err)
	}

	if len(fragments) < 2 {
		t.Errorf("Expected multiple fragments, got %d", len(fragments))
	}

	// Serialize and parse each fragment
	var parsedFragments []*ip.Packet
	for i, frag := range fragments {
		data, err := frag.Serialize()
		if err != nil {
			t.Fatalf("Failed to serialize fragment %d: %v", i, err)
		}

		parsed, err := ip.Parse(data)
		if err != nil {
			t.Fatalf("Failed to parse fragment %d: %v", i, err)
		}

		if !parsed.VerifyChecksum() {
			t.Errorf("Fragment %d: checksum verification failed", i)
		}

		parsedFragments = append(parsedFragments, parsed)
	}

	// Reassemble fragments
	reassembler := ip.NewFragmenter()
	defer reassembler.Close()

	var reassembled *ip.Packet
	for i, frag := range parsedFragments {
		result, err := reassembler.Reassemble(frag)
		if err != nil {
			t.Fatalf("Failed to reassemble fragment %d: %v", i, err)
		}
		if result != nil {
			reassembled = result
		}
	}

	if reassembled == nil {
		t.Fatal("Failed to reassemble packet")
	}

	// Parse ICMP from reassembled payload
	reassembledICMP, err := icmp.Parse(reassembled.Payload)
	if err != nil {
		t.Fatalf("Failed to parse reassembled ICMP: %v", err)
	}

	// Verify reassembled ICMP
	if !reassembledICMP.IsEchoRequest() {
		t.Error("Reassembled ICMP is not echo request")
	}
	if reassembledICMP.ID != 0x5678 {
		t.Errorf("Reassembled ID = 0x%04x, want 0x5678", reassembledICMP.ID)
	}
	if !bytes.Equal(reassembledICMP.Data, largeData) {
		t.Error("Reassembled data mismatch")
	}
}

// TestIPRouting tests IP routing table lookups.
func TestIPRouting(t *testing.T) {
	rt := ip.NewRoutingTable()

	// Add local network route
	localNet, _ := common.ParseIPv4("192.168.1.0")
	localMask, _ := common.ParseIPv4("255.255.255.0")
	rt.AddRoute(&ip.Route{
		Destination: localNet,
		Netmask:     localMask,
		Gateway:     common.IPv4Address{0, 0, 0, 0}, // Direct
		Interface:   "eth0",
		Metric:      0,
	})

	// Add default gateway
	defaultGW, _ := common.ParseIPv4("192.168.1.1")
	rt.SetDefaultGateway(defaultGW, "eth0")

	tests := []struct {
		name        string
		dst         string
		wantGateway string
	}{
		{
			name:        "local network - direct route",
			dst:         "192.168.1.50",
			wantGateway: "192.168.1.50", // Direct, no gateway
		},
		{
			name:        "remote network - via default gateway",
			dst:         "8.8.8.8",
			wantGateway: "192.168.1.1", // Via gateway
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dstIP, _ := common.ParseIPv4(tt.dst)
			_, nextHop, err := rt.Lookup(dstIP)
			if err != nil {
				t.Fatalf("Lookup failed: %v", err)
			}

			wantNextHop, _ := common.ParseIPv4(tt.wantGateway)
			if nextHop != wantNextHop {
				t.Errorf("NextHop = %s, want %s", nextHop, wantNextHop)
			}
		})
	}
}

// TestTTLDecrement tests TTL handling in IP packets.
func TestTTLDecrement(t *testing.T) {
	srcIP, _ := common.ParseIPv4("10.0.0.1")
	dstIP, _ := common.ParseIPv4("10.0.0.2")

	icmpReq := icmp.NewEchoRequest(1, 1, []byte("test"))
	icmpData, _ := icmpReq.Serialize()

	// Create packet with TTL=2
	ipPkt := ip.NewPacket(srcIP, dstIP, common.ProtocolICMP, icmpData)
	ipPkt.TTL = 2

	// First hop
	if !ipPkt.DecrementTTL() {
		t.Error("First hop: packet should still be alive")
	}
	if ipPkt.TTL != 1 {
		t.Errorf("After first hop: TTL = %d, want 1", ipPkt.TTL)
	}

	// Second hop - packet should die
	if ipPkt.DecrementTTL() {
		t.Error("Second hop: packet should die (TTL=0)")
	}
	if ipPkt.TTL != 0 {
		t.Errorf("After second hop: TTL = %d, want 0", ipPkt.TTL)
	}
}

// TestICMPErrorMessages tests ICMP error message handling.
func TestICMPErrorMessages(t *testing.T) {
	originalData := []byte("original packet data")

	tests := []struct {
		name string
		msg  *icmp.Message
	}{
		{
			name: "destination unreachable - host unreachable",
			msg:  icmp.NewDestinationUnreachable(icmp.CodeHostUnreachable, originalData),
		},
		{
			name: "time exceeded - TTL exceeded",
			msg:  icmp.NewTimeExceeded(icmp.CodeTTLExceeded, originalData),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Serialize and parse
			data, err := tt.msg.Serialize()
			if err != nil {
				t.Fatalf("Serialize failed: %v", err)
			}

			parsed, err := icmp.Parse(data)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}

			if !parsed.IsError() {
				t.Error("Message should be marked as error")
			}

			if !bytes.Equal(parsed.Data, originalData) {
				t.Error("Error data mismatch")
			}

			if !parsed.VerifyChecksum() {
				t.Error("Checksum verification failed")
			}
		})
	}
}
