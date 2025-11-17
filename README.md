# Network Protocol Stack (Go Implementation)

A TCP/IP network protocol stack implementation from scratch in Go. This project implements Ethernet, ARP, IP, UDP, and TCP protocols to understand how data travels across the internet.

## Quick Start

### Prerequisites
- Go 1.21+ installed
- Linux system (for raw sockets)
- Root/sudo access (for packet capture)
- Wireshark (recommended for debugging)

### Installation

```bash
# Clone the repository
git clone https://github.com/therealutkarshpriyadarshi/network.git
cd network

# Initialize Go module (already done)
go mod tidy

# Build the project
go build ./cmd/netstack
```

### Running Examples

```bash
# Example 1: Packet capture
sudo go run ./examples/capture/main.go

# Example 2: ARP resolution
sudo go run ./examples/arp/main.go eth0 192.168.1.100 192.168.1.1

# Example 3: Ping (ICMP)
sudo go run ./examples/ping/main.go 192.168.1.1

# Example 4: UDP echo server
sudo go run ./examples/udp_echo/main.go -i eth0 -p 8080

# Example 5: TCP echo server
sudo go run ./examples/tcp_echo/main.go
```

## Project Status

This is an educational project. Current implementation status:

- [x] Project structure
- [x] Phase 1: Raw sockets & packet capture
- [x] Phase 2: Ethernet & ARP
  - [x] Ethernet frame parsing and building
  - [x] ARP request/reply handling
  - [x] ARP cache with expiration
  - [x] Comprehensive tests and examples
- [x] Phase 3: IP & ICMP (ping)
  - [x] IPv4 packet parsing and building
  - [x] IP header checksum verification
  - [x] IP fragmentation and reassembly
  - [x] Routing table with longest prefix match
  - [x] ICMP echo request/reply (ping)
  - [x] ICMP error messages
  - [x] TTL handling
  - [x] Comprehensive tests and ping example
- [x] Phase 4: UDP
  - [x] UDP packet parsing and building
  - [x] UDP checksum with pseudo-header
  - [x] Port demultiplexing
  - [x] Socket API (Bind, SendTo, RecvFrom)
  - [x] Port allocation (ephemeral ports)
  - [x] Comprehensive tests and UDP echo server example
- [x] Phase 5: TCP
  - [x] Connection establishment (3-way handshake)
  - [x] Data transfer (send/receive buffers)
  - [x] Reliability (retransmission with RTO)
  - [x] Flow control (sliding window)
  - [x] Congestion control (slow start, congestion avoidance, fast retransmit/recovery)
  - [x] TCP state machine (11 states)
  - [x] Socket API (Listen, Accept, Connect, Send, Recv, Close)
  - [x] Comprehensive tests and TCP echo server example
- [ ] Phase 6: Testing & optimization

See [ROADMAP.md](ROADMAP.md) for detailed implementation plan.

## Architecture

```
Application Layer
      ↓
┌─────────────────┐
│   TCP / UDP     │  Layer 4: Transport
├─────────────────┤
│   IP / ICMP     │  Layer 3: Network
├─────────────────┤
│ Ethernet / ARP  │  Layer 2: Data Link
├─────────────────┤
│  Raw Sockets    │  Layer 1: Physical
└─────────────────┘
```

## Project Structure

```
network/
├── pkg/              # Core protocol implementations
│   ├── common/       # Shared utilities (checksum, types)
│   ├── ethernet/     # Ethernet frame handling
│   ├── arp/          # ARP protocol
│   ├── ip/           # IPv4 protocol
│   ├── icmp/         # ICMP (ping)
│   ├── udp/          # UDP protocol
│   └── tcp/          # TCP protocol (state machine, congestion control)
│
├── cmd/              # Main applications
│   └── netstack/     # Network stack daemon
│
├── examples/         # Example programs
│   ├── capture/      # Packet capture example
│   ├── ping/         # Ping implementation
│   ├── udp_echo/     # UDP echo server
│   └── tcp_echo/     # TCP echo server
│
└── tests/            # Test suites
    ├── unit/         # Unit tests
    └── integration/  # Integration tests
```

## Learning Goals

This project teaches:

1. **Network Protocols**: How TCP/IP actually works
2. **State Machines**: TCP's complex state management
3. **Concurrency**: Goroutines for handling multiple connections
4. **Systems Programming**: Raw sockets, byte manipulation
5. **Algorithms**: Congestion control, retransmission timers
6. **Testing**: Network testing strategies

## Key Concepts Implemented

### Ethernet (Layer 2)
- Frame parsing and building
- MAC addressing
- EtherType identification

### ARP (Address Resolution Protocol)
- IP to MAC address resolution
- ARP cache with expiration
- Request/Reply handling

### IP (Internet Protocol)
- Packet routing
- Fragmentation and reassembly
- TTL handling
- Header checksum verification

### ICMP (Internet Control Message Protocol)
- Echo request/reply (ping)
- Error messaging

### UDP (User Datagram Protocol)
- Connectionless communication
- Port multiplexing
- Checksum with pseudo-header

### TCP (Transmission Control Protocol)
- Connection establishment (3-way handshake)
- Reliable, ordered delivery
- Flow control (sliding window)
- Congestion control (slow start, congestion avoidance)
- Retransmission on timeout
- State machine (11 states)

## Testing

```bash
# Run unit tests
go test ./pkg/...

# Run with coverage
go test -cover ./pkg/...

# Run specific package tests
go test ./pkg/tcp

# Run integration tests
sudo go test ./tests/integration/...

# Benchmarks
go test -bench=. ./pkg/...
```

## Debugging Tips

### 1. Use Wireshark
```bash
# Capture on interface
sudo wireshark -i eth0

# Filter for your traffic
tcp.port == 8080
```

### 2. Enable Debug Logging
```go
// In your code
import "log"
log.SetFlags(log.Ltime | log.Lshortfile)
log.Printf("Received packet: %+v", packet)
```

### 3. Packet Dumps
```bash
# Capture packets to file
sudo tcpdump -i eth0 -w capture.pcap

# Read packet dump
tcpdump -r capture.pcap -v
```

## Common Issues

### Permission Denied
Raw sockets require root access:
```bash
sudo go run main.go
# OR
sudo setcap cap_net_raw+ep ./netstack
./netstack
```

### No Packets Received
Check interface is up:
```bash
ip link show
sudo ip link set eth0 up
```

### Checksum Errors
Verify checksum calculation matches RFC 1071:
```go
// Internet checksum includes pseudo-header for TCP/UDP
checksum := common.CalculateChecksum(data)
```

## Resources

### RFCs (Specifications)
- [RFC 791](https://tools.ietf.org/html/rfc791) - Internet Protocol
- [RFC 793](https://tools.ietf.org/html/rfc793) - TCP
- [RFC 768](https://tools.ietf.org/html/rfc768) - UDP
- [RFC 826](https://tools.ietf.org/html/rfc826) - ARP

### Books
- TCP/IP Illustrated, Volume 1 by W. Richard Stevens
- Computer Networks by Andrew Tanenbaum

### Online
- [Beej's Guide to Network Programming](https://beej.us/guide/bgnet/)
- [Go net package](https://pkg.go.dev/net)

## Contributing

This is an educational project. Feel free to:
- Report issues
- Suggest improvements
- Submit pull requests
- Use as learning material

## License

MIT License - See LICENSE file

## Acknowledgments

Inspired by the goal of understanding networking from first principles. Based on the challenge: "When you send data over the internet, it gets broken into packets, routed through multiple computers, possibly arriving out of order or not at all. TCP makes this look like a simple stream of bytes - building this teaches you how reliability emerges from unreliability."

---

**Status**: 🚧 Under Development

For detailed implementation plan, see [ROADMAP.md](ROADMAP.md)
