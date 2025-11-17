// Package main provides an example of capturing and displaying Ethernet frames.
// This program demonstrates the Phase 1 capabilities: raw socket packet capture
// and Ethernet frame parsing.
//
// Usage:
//
//	sudo go run examples/capture/main.go [interface]
//
// If no interface is specified, it will list available interfaces and use the first one.
//
// Note: This program requires root/sudo privileges to access raw sockets.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
	"github.com/therealutkarshpriyadarshi/network/pkg/ethernet"
)

var (
	ifaceFlag = flag.String("i", "", "Network interface to capture on (e.g., eth0, wlan0)")
	countFlag = flag.Int("c", 0, "Number of packets to capture (0 = unlimited)")
	hexFlag   = flag.Bool("x", false, "Display hex dump of packets")
	verboseFlag = flag.Bool("v", false, "Verbose output")
)

func main() {
	flag.Parse()

	// Check if running as root
	if os.Geteuid() != 0 {
		log.Fatal("This program requires root privileges. Please run with sudo.")
	}

	// Determine which interface to use
	ifname := *ifaceFlag
	if ifname == "" {
		// List available interfaces
		interfaces, err := ethernet.ListInterfaces()
		if err != nil {
			log.Fatalf("Failed to list interfaces: %v", err)
		}

		if len(interfaces) == 0 {
			log.Fatal("No network interfaces found")
		}

		fmt.Println("Available interfaces:")
		for i, name := range interfaces {
			fmt.Printf("  %d. %s\n", i+1, name)
			if *verboseFlag {
				info, _ := ethernet.GetInterfaceInfo(name)
				fmt.Print(info)
			}
		}

		// Use first interface
		ifname = interfaces[0]
		fmt.Printf("\nUsing interface: %s\n\n", ifname)
	}

	// Display interface information
	if *verboseFlag {
		info, err := ethernet.GetInterfaceInfo(ifname)
		if err != nil {
			log.Fatalf("Failed to get interface info: %v", err)
		}
		fmt.Println(info)
		fmt.Println()
	}

	// Open interface for packet capture
	fmt.Printf("Opening interface %s for packet capture...\n", ifname)
	iface, err := ethernet.OpenInterface(ifname)
	if err != nil {
		log.Fatalf("Failed to open interface: %v", err)
	}
	defer iface.Close()

	fmt.Printf("Capturing on %s (MAC: %s)\n", iface.Name(), iface.MACAddress())
	if *countFlag > 0 {
		fmt.Printf("Will capture %d packets\n", *countFlag)
	} else {
		fmt.Println("Press Ctrl+C to stop")
	}
	fmt.Println()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Capture loop
	packetCount := 0
	done := make(chan bool)

	go func() {
		for {
			frame, err := iface.ReadFrame()
			if err != nil {
				log.Printf("Error reading frame: %v", err)
				continue
			}

			packetCount++
			displayFrame(packetCount, frame, *hexFlag)

			if *countFlag > 0 && packetCount >= *countFlag {
				done <- true
				return
			}
		}
	}()

	// Wait for completion or interrupt
	select {
	case <-sigChan:
		fmt.Println("\nInterrupted by user")
	case <-done:
		fmt.Println("\nCapture complete")
	}

	fmt.Printf("\nCaptured %d packets\n", packetCount)
}

// displayFrame prints information about a captured frame.
func displayFrame(num int, frame *ethernet.Frame, showHex bool) {
	// Basic frame info
	fmt.Printf("[%d] %s\n", num, frame)
	fmt.Printf("     %s -> %s\n", frame.Source, frame.Destination)

	// Frame type
	if frame.IsBroadcast() {
		fmt.Printf("     Type: Broadcast\n")
	} else if frame.IsMulticast() {
		fmt.Printf("     Type: Multicast\n")
	} else {
		fmt.Printf("     Type: Unicast\n")
	}

	// Payload info
	if len(frame.Payload) > 0 {
		fmt.Printf("     Payload: %d bytes\n", len(frame.Payload))

		// Try to identify protocol from payload
		if len(frame.Payload) >= 1 {
			displayProtocolInfo(frame)
		}
	}

	// Hex dump if requested
	if showHex && len(frame.Payload) > 0 {
		// Limit hex dump to first 64 bytes
		dumpSize := len(frame.Payload)
		if dumpSize > 64 {
			dumpSize = 64
		}
		fmt.Printf("     Hex dump (first %d bytes):\n", dumpSize)
		dump := common.HexDump(frame.Payload[:dumpSize])
		// Indent the hex dump
		for _, line := range []byte(dump) {
			if line == '\n' {
				fmt.Print("\n")
			} else {
				if line == dump[0] || dump[int(line)-1] == '\n' {
					fmt.Print("       ")
				}
				fmt.Printf("%c", line)
			}
		}
	}

	fmt.Println()
}

// displayProtocolInfo attempts to display protocol-specific information.
func displayProtocolInfo(frame *ethernet.Frame) {
	switch frame.EtherType {
	case common.EtherTypeIPv4:
		displayIPv4Info(frame.Payload)
	case common.EtherTypeARP:
		displayARPInfo(frame.Payload)
	case common.EtherTypeIPv6:
		fmt.Printf("     Protocol: IPv6 (not yet implemented)\n")
	default:
		fmt.Printf("     Protocol: %s\n", frame.EtherType)
	}
}

// displayIPv4Info displays basic IPv4 packet information.
func displayIPv4Info(payload []byte) {
	if len(payload) < 20 {
		fmt.Printf("     Protocol: IPv4 (truncated)\n")
		return
	}

	// Parse basic IP header fields
	version := payload[0] >> 4
	ihl := payload[0] & 0x0F
	protocol := payload[9]
	srcIP := common.IPv4Address{payload[12], payload[13], payload[14], payload[15]}
	dstIP := common.IPv4Address{payload[16], payload[17], payload[18], payload[19]}

	fmt.Printf("     Protocol: IPv4 (v%d, IHL=%d)\n", version, ihl)
	fmt.Printf("     IP: %s -> %s\n", srcIP, dstIP)

	// Display transport protocol
	switch common.Protocol(protocol) {
	case common.ProtocolTCP:
		fmt.Printf("     Transport: TCP\n")
	case common.ProtocolUDP:
		fmt.Printf("     Transport: UDP\n")
	case common.ProtocolICMP:
		fmt.Printf("     Transport: ICMP\n")
	default:
		fmt.Printf("     Transport: Protocol %d\n", protocol)
	}
}

// displayARPInfo displays basic ARP packet information.
func displayARPInfo(payload []byte) {
	if len(payload) < 28 {
		fmt.Printf("     Protocol: ARP (truncated)\n")
		return
	}

	// Parse ARP header
	opcode := uint16(payload[6])<<8 | uint16(payload[7])

	var operation string
	switch opcode {
	case 1:
		operation = "Request"
	case 2:
		operation = "Reply"
	default:
		operation = fmt.Sprintf("Unknown(%d)", opcode)
	}

	// Sender and target addresses
	senderMAC := common.MACAddress{payload[8], payload[9], payload[10], payload[11], payload[12], payload[13]}
	senderIP := common.IPv4Address{payload[14], payload[15], payload[16], payload[17]}
	targetMAC := common.MACAddress{payload[18], payload[19], payload[20], payload[21], payload[22], payload[23]}
	targetIP := common.IPv4Address{payload[24], payload[25], payload[26], payload[27]}

	fmt.Printf("     Protocol: ARP %s\n", operation)
	fmt.Printf("     Sender: %s (%s)\n", senderIP, senderMAC)
	fmt.Printf("     Target: %s (%s)\n", targetIP, targetMAC)
}
