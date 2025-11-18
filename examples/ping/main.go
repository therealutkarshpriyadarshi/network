// Package main implements a simple ping utility using ICMP echo requests.
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
	"time"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
	"github.com/therealutkarshpriyadarshi/network/pkg/ethernet"
	"github.com/therealutkarshpriyadarshi/network/pkg/icmp"
	"github.com/therealutkarshpriyadarshi/network/pkg/ip"
)

const (
	defaultCount    = 4
	defaultInterval = 1 * time.Second
	defaultTimeout  = 5 * time.Second
	defaultDataSize = 56 // Standard ping data size
)

var (
	count    = flag.Int("c", defaultCount, "Number of pings to send")
	interval = flag.Duration("i", defaultInterval, "Interval between pings")
	timeout  = flag.Duration("W", defaultTimeout, "Timeout for each ping")
	dataSize = flag.Int("s", defaultDataSize, "Size of ping data")
)

type pingStats struct {
	transmitted int
	received    int
	minRTT      time.Duration
	maxRTT      time.Duration
	totalRTT    time.Duration
}

func main() {
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <destination>\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	destination := flag.Arg(0)

	// Parse destination IP
	dstIP, err := common.ParseIPv4(destination)
	if err != nil {
		log.Fatalf("Invalid destination IP: %v", err)
	}

	// Get local network interface
	iface, srcIP, err := getNetworkInterface()
	if err != nil {
		log.Fatalf("Failed to get network interface: %v", err)
	}

	fmt.Printf("PING %s (%s) %d bytes of data.\n", destination, dstIP, *dataSize)

	// Run ping
	stats, err := runPing(iface, srcIP, dstIP, *count, *interval, *timeout, *dataSize)
	if err != nil {
		log.Fatalf("Ping failed: %v", err)
	}

	// Print statistics
	printStats(destination, stats)
}

func runPing(iface *net.Interface, srcIP, dstIP common.IPv4Address, count int, interval, timeout time.Duration, dataSize int) (*pingStats, error) {
	// Create raw socket
	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(htons(syscall.ETH_P_ALL)))
	if err != nil {
		return nil, fmt.Errorf("failed to create socket: %w", err)
	}
	defer syscall.Close(fd)

	// Bind to interface
	addr := syscall.SockaddrLinklayer{
		Protocol: htons(syscall.ETH_P_ALL),
		Ifindex:  iface.Index,
	}
	if err := syscall.Bind(fd, &addr); err != nil {
		return nil, fmt.Errorf("failed to bind socket: %w", err)
	}

	stats := &pingStats{
		minRTT: time.Duration(1<<63 - 1), // Max duration
	}

	// Set up signal handler for graceful exit
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Send pings
	seq := uint16(0)
	pid := uint16(os.Getpid() & 0xFFFF)

	for i := 0; i < count; i++ {
		select {
		case <-sigChan:
			fmt.Println("\nInterrupted.")
			return stats, nil
		default:
		}

		seq++
		start := time.Now()

		// Send ICMP echo request
		err := sendPing(fd, iface, srcIP, dstIP, pid, seq, dataSize)
		if err != nil {
			log.Printf("Failed to send ping: %v", err)
			continue
		}
		stats.transmitted++

		// Wait for reply
		replied, rtt := waitForReply(fd, dstIP, pid, seq, timeout)
		if replied {
			stats.received++
			stats.totalRTT += rtt
			if rtt < stats.minRTT {
				stats.minRTT = rtt
			}
			if rtt > stats.maxRTT {
				stats.maxRTT = rtt
			}

			fmt.Printf("%d bytes from %s: icmp_seq=%d ttl=64 time=%.3f ms\n",
				dataSize+8, dstIP, seq, float64(rtt.Microseconds())/1000.0)
		} else {
			fmt.Printf("From %s: icmp_seq=%d Destination Host Unreachable\n", dstIP, seq)
		}

		// Wait for interval (unless last ping)
		if i < count-1 {
			elapsed := time.Since(start)
			if elapsed < interval {
				time.Sleep(interval - elapsed)
			}
		}
	}

	return stats, nil
}

func sendPing(fd int, iface *net.Interface, srcIP, dstIP common.IPv4Address, id, seq uint16, dataSize int) error {
	// Create ICMP echo request
	data := make([]byte, dataSize)
	// Fill with pattern
	for i := range data {
		data[i] = byte(i % 256)
	}

	icmpMsg := icmp.NewEchoRequest(id, seq, data)
	icmpData, err := icmpMsg.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize ICMP: %w", err)
	}

	// Create IP packet
	ipPkt := ip.NewPacket(srcIP, dstIP, common.ProtocolICMP, icmpData)
	ipData, err := ipPkt.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize IP: %w", err)
	}

	// Note: In a real implementation, you would need to resolve the MAC address
	// using ARP first. For simplicity, we're using broadcast MAC here.
	dstMAC := common.BroadcastMAC

	// Create Ethernet frame
	ethFrame := &ethernet.Frame{
		Destination: dstMAC,
		Source:      bytesToMAC(iface.HardwareAddr),
		EtherType:   common.EtherTypeIPv4,
		Payload:     ipData,
	}

	frameData := ethFrame.Serialize()

	// Send packet
	err = syscall.Sendto(fd, frameData, 0, &syscall.SockaddrLinklayer{
		Protocol: htons(syscall.ETH_P_ALL),
		Ifindex:  iface.Index,
	})
	if err != nil {
		return fmt.Errorf("failed to send packet: %w", err)
	}

	return nil
}

func waitForReply(fd int, expectedSrc common.IPv4Address, expectedID, expectedSeq uint16, timeout time.Duration) (bool, time.Duration) {
	deadline := time.Now().Add(timeout)
	buf := make([]byte, 65535)

	// Set socket timeout
	tv := syscall.NsecToTimeval(timeout.Nanoseconds())
	syscall.SetsockoptTimeval(fd, syscall.SOL_SOCKET, syscall.SO_RCVTIMEO, &tv)

	for time.Now().Before(deadline) {
		start := time.Now()

		n, _, err := syscall.Recvfrom(fd, buf, 0)
		if err != nil {
			return false, 0
		}

		// Parse Ethernet frame
		frame, err := ethernet.Parse(buf[:n])
		if err != nil {
			continue
		}

		// Check if IPv4
		if frame.EtherType != common.EtherTypeIPv4 {
			continue
		}

		// Parse IP packet
		ipPkt, err := ip.Parse(frame.Payload)
		if err != nil {
			continue
		}

		// Check if from expected source and ICMP
		if ipPkt.Source != expectedSrc || ipPkt.Protocol != common.ProtocolICMP {
			continue
		}

		// Parse ICMP message
		icmpMsg, err := icmp.Parse(ipPkt.Payload)
		if err != nil {
			continue
		}

		// Check if echo reply with matching ID and sequence
		if icmpMsg.IsEchoReply() && icmpMsg.ID == expectedID && icmpMsg.Sequence == expectedSeq {
			rtt := time.Since(start)
			return true, rtt
		}
	}

	return false, 0
}

func printStats(destination string, stats *pingStats) {
	fmt.Printf("\n--- %s ping statistics ---\n", destination)
	fmt.Printf("%d packets transmitted, %d received, %.1f%% packet loss\n",
		stats.transmitted, stats.received,
		float64(stats.transmitted-stats.received)/float64(stats.transmitted)*100.0)

	if stats.received > 0 {
		avgRTT := stats.totalRTT / time.Duration(stats.received)
		fmt.Printf("rtt min/avg/max = %.3f/%.3f/%.3f ms\n",
			float64(stats.minRTT.Microseconds())/1000.0,
			float64(avgRTT.Microseconds())/1000.0,
			float64(stats.maxRTT.Microseconds())/1000.0)
	}
}

func getNetworkInterface() (*net.Interface, common.IPv4Address, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, common.IPv4Address{}, err
	}

	// Find first non-loopback interface with IPv4 address
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
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

			var srcIP common.IPv4Address
			copy(srcIP[:], ipv4)
			return &iface, srcIP, nil
		}
	}

	return nil, common.IPv4Address{}, fmt.Errorf("no suitable network interface found")
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
