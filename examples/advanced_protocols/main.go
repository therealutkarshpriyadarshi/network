// Package main demonstrates advanced networking protocols.
package main

import (
	"fmt"
	"log"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
	"github.com/therealutkarshpriyadarshi/network/pkg/ipv6"
	"github.com/therealutkarshpriyadarshi/network/pkg/multicast"
	"github.com/therealutkarshpriyadarshi/network/pkg/quic"
	"github.com/therealutkarshpriyadarshi/network/pkg/tcp"
	"github.com/therealutkarshpriyadarshi/network/pkg/tls"
)

func main() {
	fmt.Println("=== Advanced Networking Protocols Demo ===\n")

	demonstrateIPv6()
	demonstrateTCPOptions()
	demonstrateTLS()
	demonstrateTCPFastOpen()
	demonstrateQUIC()
	demonstrateMulticast()
}

func demonstrateIPv6() {
	fmt.Println("1. IPv6 Dual-Stack Support")
	fmt.Println("---------------------------")

	// Create IPv6 addresses
	src, _ := common.ParseIPv6("2001:db8::1")
	dst, _ := common.ParseIPv6("2001:db8::2")

	// Create an IPv6 packet
	payload := []byte("Hello IPv6!")
	pkt := ipv6.NewPacket(src, dst, common.ProtocolTCP, payload)

	// Serialize and display
	data, _ := pkt.Serialize()
	fmt.Printf("Created IPv6 packet: %s\n", pkt)
	fmt.Printf("Packet size: %d bytes\n", len(data))

	// Dual-stack address
	dualIPv4 := ipv6.NewIPv4Address(common.IPv4Address{192, 168, 1, 1})
	dualIPv6 := ipv6.NewIPv6Address(dst)
	fmt.Printf("Dual-stack IPv4: %s\n", dualIPv4)
	fmt.Printf("Dual-stack IPv6: %s\n\n", dualIPv6)
}

func demonstrateTCPOptions() {
	fmt.Println("2. TCP Options (Window Scaling, Timestamps, SACK)")
	fmt.Println("--------------------------------------------------")

	// Create a TCP segment with various options
	seg := tcp.NewSegment(8080, 80, 1000, 0, tcp.FlagSYN, 65535, nil)

	// Add MSS option
	mssOpt := tcp.BuildMSSOption(1460)
	fmt.Printf("MSS Option: %v\n", mssOpt)

	// Add Window Scale option
	wsOpt := tcp.BuildWindowScaleOption(7)
	fmt.Printf("Window Scale Option: %v\n", wsOpt)

	// Add Timestamp option
	tsOpt := tcp.BuildTimestampOption(12345, 0)
	fmt.Printf("Timestamp Option: %v\n", tsOpt)

	// Add SACK Permitted option
	sackPermOpt := tcp.BuildSACKPermittedOption()
	fmt.Printf("SACK Permitted Option: %v\n", sackPermOpt)

	// Add SACK option with blocks
	sackBlocks := []tcp.SACKBlock{
		{LeftEdge: 1000, RightEdge: 2000},
		{LeftEdge: 3000, RightEdge: 4000},
	}
	sackOpt := tcp.BuildSACKOption(sackBlocks)
	fmt.Printf("SACK Option: %v\n", sackOpt)

	// Combine options
	var options []byte
	options = append(options, mssOpt...)
	options = append(options, wsOpt...)
	options = append(options, tsOpt...)
	options = append(options, sackPermOpt...)
	seg.Options = options

	fmt.Printf("TCP Segment with options: %s\n\n", seg)
}

func demonstrateTLS() {
	fmt.Println("3. TLS/SSL Secure Communication")
	fmt.Println("--------------------------------")

	// Create TLS configuration
	config := tls.DefaultConfig()
	fmt.Printf("TLS Config: MinVersion=%s, MaxVersion=%s\n",
		config.MinVersion, config.MaxVersion)

	// Generate self-signed certificate
	certPEM, keyPEM, err := tls.GenerateSelfSignedCert("localhost")
	if err != nil {
		log.Printf("Failed to generate certificate: %v\n", err)
		return
	}

	fmt.Printf("Generated self-signed certificate (%d bytes)\n", len(certPEM))
	fmt.Printf("Generated private key (%d bytes)\n", len(keyPEM))

	// Load certificate
	cert, err := tls.LoadCertificate(certPEM, keyPEM)
	if err != nil {
		log.Printf("Failed to load certificate: %v\n", err)
		return
	}

	fmt.Printf("Loaded certificate with %d certs\n", len(cert.Certificate))

	// Display cipher suites
	fmt.Println("Supported cipher suites:")
	for _, cs := range config.CipherSuites {
		fmt.Printf("  - %s\n", cs)
	}
	fmt.Println()
}

func demonstrateTCPFastOpen() {
	fmt.Println("4. TCP Fast Open")
	fmt.Println("----------------")

	// Create TFO state
	tfoState, err := tcp.NewTFOState()
	if err != nil {
		log.Printf("Failed to create TFO state: %v\n", err)
		return
	}

	// Create TFO connection
	tfoConn := tcp.NewTFOConnection(tfoState)

	// Queue data to send with SYN
	tfoConn.QueueData([]byte("Fast Open Data"))
	fmt.Printf("Queued data for TFO: %d bytes\n", len(tfoConn.GetQueuedData()))

	// Build TFO option (cookie request)
	tfoOpt := tcp.BuildTFOOption(nil)
	fmt.Printf("TFO Cookie Request Option: %v\n", tfoOpt)

	// Build TFO option with cookie
	cookie := make([]byte, tcp.TFOCookieLen)
	copy(cookie, "example-cookie!!")
	tfoOptWithCookie := tcp.BuildTFOOption(cookie)
	fmt.Printf("TFO Option with Cookie: %v\n\n", tfoOptWithCookie)
}

func demonstrateQUIC() {
	fmt.Println("5. QUIC Protocol")
	fmt.Println("----------------")

	// Create QUIC packets
	destConnID := []byte{0x01, 0x02, 0x03, 0x04}
	srcConnID := []byte{0x05, 0x06, 0x07, 0x08}

	// Initial packet
	initialPkt := quic.NewInitialPacket(destConnID, srcConnID, nil, []byte("Initial data"))
	fmt.Printf("Initial packet: %s\n", initialPkt)

	// Handshake packet
	handshakePkt := quic.NewHandshakePacket(destConnID, srcConnID, []byte("Handshake data"))
	fmt.Printf("Handshake packet: %s\n", handshakePkt)

	// 1-RTT packet
	rttPkt := quic.New1RTTPacket(destConnID, []byte("Application data"))
	fmt.Printf("1-RTT packet: %s\n", rttPkt)

	// QUIC frames
	pingFrame := &quic.PingFrame{}
	fmt.Printf("PING frame: %s\n", pingFrame)

	streamFrame := &quic.StreamFrame{
		StreamID: 4,
		Offset:   0,
		Data:     []byte("Stream data"),
		Fin:      false,
	}
	fmt.Printf("STREAM frame: %s\n", streamFrame)

	ackFrame := &quic.AckFrame{
		LargestAcknowledged: 100,
		AckDelay:            25,
		AckRanges:           []quic.AckRange{{Gap: 0, Length: 10}},
	}
	fmt.Printf("ACK frame: %s\n\n", ackFrame)
}

func demonstrateMulticast() {
	fmt.Println("6. Multicast Support")
	fmt.Println("--------------------")

	// IPv4 multicast
	mcastAddr4 := multicast.AllHostsMulticast
	fmt.Printf("IPv4 All-Hosts Multicast: %s\n", mcastAddr4)
	fmt.Printf("Is multicast: %v\n", multicast.IsMulticastIPv4(mcastAddr4))

	// IPv6 multicast
	mcastAddr6 := multicast.AllNodesMulticast
	fmt.Printf("IPv6 All-Nodes Multicast: %s\n", mcastAddr6)
	fmt.Printf("Is multicast: %v\n", multicast.IsMulticastIPv6(mcastAddr6))
	fmt.Printf("Multicast scope: %d\n", multicast.GetIPv6MulticastScope(mcastAddr6))

	// Create multicast group
	group := multicast.NewIPv4Group(mcastAddr4, 0)
	group.AddMember("client-1")
	group.AddMember("client-2")
	fmt.Printf("Multicast group: %s\n", group)
	fmt.Printf("Member count: %d\n", group.MemberCount())

	// IGMP messages
	igmpQuery := multicast.NewMembershipQuery(mcastAddr4, 100)
	fmt.Printf("IGMP Query: %s\n", igmpQuery)

	igmpReport := multicast.NewMembershipReport(mcastAddr4)
	fmt.Printf("IGMP Report: %s\n", igmpReport)

	// MLD messages
	mldQuery := multicast.NewMLDQuery(mcastAddr6, 1000)
	fmt.Printf("MLD Query: %s\n", mldQuery)

	mldReport := multicast.NewMLDReport(mcastAddr6)
	fmt.Printf("MLD Report: %s\n", mldReport)

	fmt.Println("\n=== Demo Complete ===")
}
