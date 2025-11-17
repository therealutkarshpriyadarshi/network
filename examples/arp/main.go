// ARP Example
//
// This example demonstrates how to use the ARP protocol to resolve
// IP addresses to MAC addresses on a local network.
//
// Usage:
//   sudo go run examples/arp/main.go <interface> <local-ip> <target-ip>
//
// Example:
//   sudo go run examples/arp/main.go eth0 192.168.1.100 192.168.1.1
//
// This will:
// 1. Open the specified network interface
// 2. Create an ARP handler
// 3. Send an ARP request for the target IP
// 4. Wait for and display the ARP reply
// 5. Show the ARP cache contents

package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/therealutkarshpriyadarshi/network/pkg/arp"
	"github.com/therealutkarshpriyadarshi/network/pkg/common"
	"github.com/therealutkarshpriyadarshi/network/pkg/ethernet"
)

func main() {
	// Parse command-line arguments
	if len(os.Args) < 4 {
		fmt.Fprintf(os.Stderr, "Usage: %s <interface> <local-ip> <target-ip>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  sudo %s eth0 192.168.1.100 192.168.1.1\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nNote: This program requires root/sudo privileges.\n")
		os.Exit(1)
	}

	ifaceName := os.Args[1]
	localIPStr := os.Args[2]
	targetIPStr := os.Args[3]

	// Parse IP addresses
	localIP, err := common.ParseIPv4(localIPStr)
	if err != nil {
		log.Fatalf("Invalid local IP address: %v", err)
	}

	targetIP, err := common.ParseIPv4(targetIPStr)
	if err != nil {
		log.Fatalf("Invalid target IP address: %v", err)
	}

	fmt.Printf("=== ARP Resolution Example ===\n\n")
	fmt.Printf("Interface:  %s\n", ifaceName)
	fmt.Printf("Local IP:   %s\n", localIP)
	fmt.Printf("Target IP:  %s\n\n", targetIP)

	// Open network interface
	fmt.Printf("Opening interface %s...\n", ifaceName)
	iface, err := ethernet.OpenInterface(ifaceName)
	if err != nil {
		log.Fatalf("Failed to open interface: %v", err)
	}
	defer iface.Close()

	fmt.Printf("Interface opened successfully\n")
	fmt.Printf("  MAC Address: %s\n", iface.MACAddress())
	fmt.Printf("  Index:       %d\n\n", iface.Index())

	// Create ARP handler
	fmt.Printf("Creating ARP handler...\n")
	handler := arp.NewHandler(iface, localIP)
	handler.SetTimeout(3 * time.Second)
	handler.SetMaxRetries(3)

	// Start the ARP handler in background
	fmt.Printf("Starting ARP handler...\n\n")
	stop, err := handler.Start()
	if err != nil {
		log.Fatalf("Failed to start ARP handler: %v", err)
	}
	defer close(stop)

	// Give the handler a moment to start
	time.Sleep(100 * time.Millisecond)

	// Optional: Send a gratuitous ARP to announce ourselves
	fmt.Printf("Sending gratuitous ARP...\n")
	if err := handler.Announce(); err != nil {
		log.Printf("Warning: Failed to send gratuitous ARP: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Resolve the target IP
	fmt.Printf("Resolving %s...\n", targetIP)
	startTime := time.Now()

	mac, err := handler.Resolve(targetIP)
	if err != nil {
		log.Fatalf("Failed to resolve IP: %v", err)
	}

	elapsed := time.Since(startTime)

	// Display results
	fmt.Printf("\n=== Resolution Successful ===\n\n")
	fmt.Printf("IP Address:  %s\n", targetIP)
	fmt.Printf("MAC Address: %s\n", mac)
	fmt.Printf("Resolution Time: %v\n\n", elapsed)

	// Display ARP cache
	fmt.Printf("=== ARP Cache ===\n\n")
	fmt.Print(handler.Cache().String())

	// Demonstrate cache hit (should be instant)
	fmt.Printf("\n=== Testing Cache Hit ===\n\n")
	fmt.Printf("Resolving %s again (should use cache)...\n", targetIP)
	startTime = time.Now()

	mac2, err := handler.Resolve(targetIP)
	if err != nil {
		log.Fatalf("Failed to resolve IP from cache: %v", err)
	}

	elapsed = time.Since(startTime)

	if mac2 != mac {
		log.Fatalf("Cache returned different MAC: %s vs %s", mac2, mac)
	}

	fmt.Printf("MAC Address: %s (from cache)\n", mac2)
	fmt.Printf("Lookup Time: %v (should be <1ms)\n\n", elapsed)

	// List all cache entries
	entries := handler.Cache().Entries()
	fmt.Printf("Total cache entries: %d\n", len(entries))
	for ip, m := range entries {
		fmt.Printf("  %s -> %s\n", ip, m)
	}

	fmt.Printf("\n=== Example Complete ===\n")
}
