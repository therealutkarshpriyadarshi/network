// Package main implements a UDP echo server using the custom network stack.
package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
	"github.com/therealutkarshpriyadarshi/network/pkg/ethernet"
	"github.com/therealutkarshpriyadarshi/network/pkg/ip"
	"github.com/therealutkarshpriyadarshi/network/pkg/udp"
)

var (
	port      = flag.Int("p", 8080, "Port to listen on")
	iface     = flag.String("i", "", "Network interface to use (e.g., eth0)")
	verbose   = flag.Bool("v", false, "Verbose output")
	maxPacket = 1500 // Maximum packet size to receive
)

func main() {
	flag.Parse()

	if *iface == "" {
		fmt.Fprintf(os.Stderr, "Usage: %s -i <interface> [-p <port>] [-v]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Example: sudo %s -i eth0 -p 8080 -v\n", os.Args[0])
		os.Exit(1)
	}

	// Get network interface
	netIface, localIP, err := getNetworkInterface(*iface)
	if err != nil {
		log.Fatalf("Failed to get network interface: %v", err)
	}

	fmt.Printf("UDP Echo Server starting on %s:%d\n", localIP, *port)
	fmt.Printf("Interface: %s (%s)\n", netIface.Name, bytesToMAC(netIface.HardwareAddr))
	fmt.Printf("Press Ctrl+C to stop\n\n")

	// Create raw socket
	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(htons(syscall.ETH_P_ALL)))
	if err != nil {
		log.Fatalf("Failed to create socket (need root): %v", err)
	}
	defer syscall.Close(fd)

	// Bind to interface
	addr := syscall.SockaddrLinklayer{
		Protocol: htons(syscall.ETH_P_ALL),
		Ifindex:  netIface.Index,
	}
	if err := syscall.Bind(fd, &addr); err != nil {
		log.Fatalf("Failed to bind socket: %v", err)
	}

	// Set up signal handler for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Create UDP demultiplexer and socket
	demux := udp.NewDemultiplexer()
	socket := udp.NewSocket()

	// Bind socket to port
	localAddr := udp.Address{
		IP:   localIP,
		Port: uint16(*port),
	}
	if err := socket.Bind(localAddr); err != nil {
		log.Fatalf("Failed to bind socket: %v", err)
	}

	assignedPort, err := demux.Bind(socket, uint16(*port))
	if err != nil {
		log.Fatalf("Failed to bind to demultiplexer: %v", err)
	}

	fmt.Printf("Listening on UDP port %d...\n\n", assignedPort)

	// Start packet receiving loop in goroutine
	go func() {
		buf := make([]byte, maxPacket)
		for {
			n, _, err := syscall.Recvfrom(fd, buf, 0)
			if err != nil {
				continue
			}

			// Process packet
			if err := processPacket(fd, netIface, localIP, demux, socket, buf[:n]); err != nil {
				if *verbose {
					log.Printf("Error processing packet: %v", err)
				}
			}
		}
	}()

	// Wait for interrupt signal
	<-sigChan
	fmt.Println("\nShutting down...")
	socket.Close()
}

func processPacket(fd int, netIface *net.Interface, localIP common.IPv4Address, demux *udp.Demultiplexer, socket *udp.Socket, data []byte) error {
	// Parse Ethernet frame
	frame, err := ethernet.Parse(data)
	if err != nil {
		return fmt.Errorf("failed to parse ethernet frame: %w", err)
	}

	// Only process IPv4 packets
	if frame.EtherType != common.EtherTypeIPv4 {
		return nil
	}

	// Parse IP packet
	ipPkt, err := ip.Parse(frame.Payload)
	if err != nil {
		return fmt.Errorf("failed to parse IP packet: %w", err)
	}

	// Only process UDP packets destined for us
	if ipPkt.Protocol != common.ProtocolUDP || ipPkt.Destination != localIP {
		return nil
	}

	// Parse UDP packet
	udpPkt, err := udp.Parse(ipPkt.Payload)
	if err != nil {
		return fmt.Errorf("failed to parse UDP packet: %w", err)
	}

	// Verify checksum if present
	if udpPkt.Checksum != 0 && !udpPkt.VerifyChecksum(ipPkt.Source, ipPkt.Destination) {
		if *verbose {
			log.Printf("UDP checksum verification failed")
		}
		return fmt.Errorf("UDP checksum verification failed")
	}

	// Only process packets for our port
	if udpPkt.DestinationPort != uint16(*port) {
		return nil
	}

	srcAddr := udp.Address{
		IP:   ipPkt.Source,
		Port: udpPkt.SourcePort,
	}

	fmt.Printf("Received %d bytes from %s: %s\n", len(udpPkt.Data), srcAddr, string(udpPkt.Data))

	// Echo back the data
	return sendUDPPacket(fd, netIface, localIP, srcAddr.IP, uint16(*port), srcAddr.Port, udpPkt.Data, frame.Source)
}

func sendUDPPacket(fd int, netIface *net.Interface, srcIP, dstIP common.IPv4Address, srcPort, dstPort uint16, data []byte, dstMAC common.MACAddress) error {
	// Create UDP packet
	udpPkt := udp.NewPacket(srcPort, dstPort, data)

	// Calculate UDP checksum
	checksum, err := udpPkt.CalculateChecksum(srcIP, dstIP)
	if err != nil {
		return fmt.Errorf("failed to calculate UDP checksum: %w", err)
	}
	udpPkt.Checksum = checksum

	// Serialize UDP packet
	udpData, err := udpPkt.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize UDP packet: %w", err)
	}

	// Create IP packet
	ipPkt := ip.NewPacket(srcIP, dstIP, common.ProtocolUDP, udpData)
	ipData, err := ipPkt.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize IP packet: %w", err)
	}

	// Create Ethernet frame
	ethFrame := &ethernet.Frame{
		Destination: dstMAC,
		Source:      bytesToMAC(netIface.HardwareAddr),
		EtherType:   common.EtherTypeIPv4,
		Payload:     ipData,
	}

	frameData := ethFrame.Serialize()

	// Send packet
	err = syscall.Sendto(fd, frameData, 0, &syscall.SockaddrLinklayer{
		Protocol: htons(syscall.ETH_P_ALL),
		Ifindex:  netIface.Index,
	})
	if err != nil {
		return fmt.Errorf("failed to send packet: %w", err)
	}

	if *verbose {
		fmt.Printf("Echoed %d bytes to %s:%d\n", len(data), dstIP, dstPort)
	}

	return nil
}

func getNetworkInterface(name string) (*net.Interface, common.IPv4Address, error) {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return nil, common.IPv4Address{}, fmt.Errorf("interface not found: %w", err)
	}

	if iface.Flags&net.FlagUp == 0 {
		return nil, common.IPv4Address{}, fmt.Errorf("interface %s is down", name)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return nil, common.IPv4Address{}, fmt.Errorf("failed to get addresses: %w", err)
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}

		ipv4 := ipNet.IP.To4()
		if ipv4 == nil {
			continue
		}

		var localIP common.IPv4Address
		copy(localIP[:], ipv4)
		return iface, localIP, nil
	}

	return nil, common.IPv4Address{}, fmt.Errorf("no IPv4 address found on interface %s", name)
}

func htons(v uint16) uint16 {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, v)
	return binary.LittleEndian.Uint16(buf)
}

func bytesToMAC(b []byte) common.MACAddress {
	var mac common.MACAddress
	copy(mac[:], b)
	return mac
}
