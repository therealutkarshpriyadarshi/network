// TCP Echo Server Example
//
// This example demonstrates a simple TCP echo server using the custom TCP implementation.
// The server listens on a specified port and echoes back any data it receives.
//
// Usage:
//   sudo go run main.go -i eth0 -addr 192.168.1.100 -port 8080
//
// Test with netcat:
//   nc 192.168.1.100 8080
//
package main

import (
	"flag"
	"log"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
	"github.com/therealutkarshpriyadarshi/network/pkg/tcp"
)

var (
	interfaceName = flag.String("i", "eth0", "Network interface name")
	listenAddr    = flag.String("addr", "192.168.1.100", "IP address to listen on")
	listenPort    = flag.Int("port", 8080, "Port to listen on")
)

func main() {
	flag.Parse()

	log.SetFlags(log.Ltime | log.Lshortfile)
	log.Printf("Starting TCP echo server on %s:%d", *listenAddr, *listenPort)

	// Parse listen address
	addr, err := common.ParseIPv4(*listenAddr)
	if err != nil {
		log.Fatalf("Invalid IP address: %v", err)
	}

	// Create TCP socket
	socket := tcp.NewSocket(addr, uint16(*listenPort))

	// Set up send function (in a real implementation, this would send via the network stack)
	socket.SetSendFunc(func(seg *tcp.Segment, srcIP, dstIP common.IPv4Address) error {
		log.Printf("Sending segment: %s", seg)
		// In a real implementation, this would:
		// 1. Wrap segment in IP packet
		// 2. Wrap IP packet in Ethernet frame
		// 3. Send via raw socket
		return nil
	})

	// Listen for connections
	if err := socket.Listen(10); err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("Listening on %s:%d", *listenAddr, *listenPort)

	// Accept connections in a loop
	for {
		conn, err := socket.Accept()
		if err != nil {
			log.Printf("Accept error: %v", err)
			continue
		}

		log.Printf("Accepted connection from %s:%d",
			conn.GetRemoteAddr(), conn.GetRemotePort())

		// Handle connection in a goroutine
		go handleConnection(conn)
	}
}

func handleConnection(conn *tcp.Socket) {
	defer conn.Close()

	log.Printf("Handling connection from %s:%d",
		conn.GetRemoteAddr(), conn.GetRemotePort())

	buf := make([]byte, 4096)

	for {
		// Receive data
		n, err := conn.Recv(buf)
		if err != nil {
			log.Printf("Receive error: %v", err)
			return
		}

		data := buf[:n]
		log.Printf("Received %d bytes: %s", n, string(data))

		// Echo back
		sent, err := conn.Send(data)
		if err != nil {
			log.Printf("Send error: %v", err)
			return
		}

		log.Printf("Sent %d bytes", sent)
	}
}
