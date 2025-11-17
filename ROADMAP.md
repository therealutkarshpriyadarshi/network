# TCP/IP Protocol Stack Implementation Roadmap (Go)

## Project Overview
Build a TCP/IP network protocol stack from scratch in Go, implementing the core protocols: Ethernet, ARP, IP, UDP, and TCP. This project will teach you how data travels across the internet from first principles.

## Why Go?
- **Simplicity**: Clean syntax, easy to learn
- **Concurrency**: Goroutines perfect for handling multiple network connections
- **Standard Library**: Excellent `net` package for raw sockets and interfaces
- **Performance**: Compiled language, fast enough for network operations
- **Memory Management**: Automatic GC (easier than C), no borrow checker (simpler than Rust)

---

## Implementation Strategy: Bottom-Up Approach

We'll build from Layer 2 (Data Link) up to Layer 4 (Transport), following the OSI model:

```
┌─────────────────────────────────────┐
│  Application (Your Code)            │
├─────────────────────────────────────┤
│  Layer 4: TCP / UDP                 │  ← Phase 4 & 5
├─────────────────────────────────────┤
│  Layer 3: IP (+ ICMP)               │  ← Phase 3
├─────────────────────────────────────┤
│  Layer 2: Ethernet + ARP            │  ← Phase 2
├─────────────────────────────────────┤
│  Layer 1: Raw Sockets / TUN/TAP     │  ← Phase 1
└─────────────────────────────────────┘
```

---

## Phase 1: Foundation & Setup (Week 1)
**Goal**: Set up project structure and understand packet capture

### Tasks
1. **Project Structure**
   - Initialize Go module
   - Create package structure (`pkg/ethernet`, `pkg/arp`, `pkg/ip`, `pkg/udp`, `pkg/tcp`, `pkg/common`)
   - Set up basic types and utilities

2. **Raw Socket Setup**
   - Learn to use `AF_PACKET` sockets (Linux) or `pcap` library
   - Capture packets on network interface
   - Parse Ethernet frames
   - Print packet hex dumps

3. **Helper Utilities**
   - Checksum calculation (RFC 1071)
   - Byte order conversion (network byte order = big endian)
   - Packet buffer management

### Deliverables
- [ ] Can capture and print Ethernet frames from network interface
- [ ] Checksum function tested and working
- [ ] Basic project structure in place

### Learning Resources
- Go `syscall` package documentation
- Ethernet frame format (IEEE 802.3)
- Wireshark for packet inspection

---

## Phase 2: Ethernet & ARP (Week 2)
**Goal**: Implement Layer 2 protocols

### Tasks
1. **Ethernet Frame Handling**
   - Parse Ethernet headers (Dst MAC, Src MAC, EtherType)
   - Identify packet types (IPv4=0x0800, ARP=0x0806)
   - Build Ethernet frames
   - Handle frame padding (minimum 64 bytes)

2. **ARP Protocol**
   - Parse ARP requests and replies
   - Build ARP request packets
   - Implement ARP cache (map IP → MAC address)
   - Handle ARP timeout and cache expiration

3. **Testing**
   - Send ARP request for a known IP on your network
   - Receive and parse ARP reply
   - Verify MAC address matches

### Deliverables
- [ ] Can send ARP requests
- [ ] Can receive and parse ARP replies
- [ ] ARP cache working with expiration
- [ ] Successfully resolve IP to MAC on local network

### Key Challenges
- Understanding hardware addressing vs protocol addressing
- Handling broadcast addresses (FF:FF:FF:FF:FF:FF)
- Cache invalidation strategy

---

## Phase 3: IP & ICMP (Week 3-4)
**Goal**: Implement Layer 3 routing and basic ping

### Tasks
1. **IPv4 Packet Handling**
   - Parse IP header (version, IHL, TTL, protocol, src/dst IP)
   - Verify header checksum
   - Handle IP fragmentation and reassembly
   - Implement TTL decrement
   - Build IP packets

2. **ICMP (for ping)**
   - Parse ICMP Echo Request/Reply
   - Calculate ICMP checksum
   - Respond to ping requests
   - Send ping requests

3. **Routing Table**
   - Simple routing table (destination → next hop)
   - Default gateway support
   - Route lookup for outgoing packets

4. **Testing**
   - Ping your stack from another machine
   - Your stack pings another machine
   - Test with different payload sizes
   - Verify fragmentation handling

### Deliverables
- [ ] Can parse and build IP packets
- [ ] Respond to ping (ICMP Echo)
- [ ] Can initiate ping to other hosts
- [ ] Handle fragmented packets correctly
- [ ] Basic routing table working

### Key Challenges
- IP header checksum calculation
- Fragmentation and reassembly logic
- TTL handling and preventing loops
- Dealing with IP options

---

## Phase 4: UDP (Week 5)
**Goal**: Implement unreliable datagram protocol

### Tasks
1. **UDP Packet Handling**
   - Parse UDP header (src port, dst port, length, checksum)
   - Calculate UDP checksum with pseudo-header
   - Build UDP packets
   - Port demultiplexing (route to correct application)

2. **Socket API**
   - Create UDP socket abstraction
   - `Bind()` to port
   - `SendTo()` and `RecvFrom()` operations
   - Port allocation

3. **Testing**
   - Send UDP packet to another machine
   - Receive UDP packets
   - Test with netcat or custom client
   - Verify checksums

### Deliverables
- [ ] Can send/receive UDP packets
- [ ] Port binding and demultiplexing works
- [ ] UDP checksum correct (with IP pseudo-header)
- [ ] Simple UDP echo server working

### Key Challenges
- UDP pseudo-header for checksum
- Port management and allocation
- Handling socket buffers

---

## Phase 5: TCP (Week 6-10) - THE BIG ONE
**Goal**: Implement reliable, ordered, connection-oriented protocol

This is the most complex part. Break it into sub-phases:

### 5.1: TCP Basics (Week 6-7)
1. **TCP Header Parsing**
   - Parse all TCP header fields
   - Sequence numbers, ACK numbers
   - Flags (SYN, ACK, FIN, RST, PSH, URG)
   - Window size, checksum, urgent pointer
   - Options (MSS, window scaling, timestamps)

2. **TCP State Machine**
   - Implement RFC 793 state machine
   - States: CLOSED, LISTEN, SYN_SENT, SYN_RECEIVED, ESTABLISHED, FIN_WAIT_1, FIN_WAIT_2, CLOSE_WAIT, CLOSING, LAST_ACK, TIME_WAIT
   - State transitions for events (receive SYN, send FIN, etc.)

3. **Three-Way Handshake**
   - Passive open (server): LISTEN → SYN_RECEIVED → ESTABLISHED
   - Active open (client): CLOSED → SYN_SENT → ESTABLISHED
   - Initial sequence number (ISN) generation

**Deliverables**:
- [ ] Can parse TCP packets
- [ ] State machine implemented
- [ ] Three-way handshake works (can establish connection)

### 5.2: Data Transfer (Week 8)
1. **Send Path**
   - Segment data into MSS-sized chunks
   - Assign sequence numbers
   - Add to retransmission queue
   - Send segments

2. **Receive Path**
   - Receive segments
   - Handle out-of-order packets (reassembly buffer)
   - Send ACKs
   - Deliver data to application in order

3. **ACK Processing**
   - Cumulative acknowledgments
   - Remove ACKed data from retransmission queue
   - Delayed ACKs (send ACK after 200ms or 2 segments)

**Deliverables**:
- [ ] Can send data over established connection
- [ ] Can receive data in order
- [ ] ACKs sent and processed correctly

### 5.3: Reliability (Week 9)
1. **Retransmission**
   - Timeout calculation (RTT estimation)
   - Retransmit on timeout
   - Fast retransmit (3 duplicate ACKs)

2. **Flow Control**
   - Advertise receive window
   - Respect sender's window
   - Window probe for zero window

3. **Connection Teardown**
   - Four-way handshake (FIN, ACK, FIN, ACK)
   - TIME_WAIT state (2MSL timer)
   - Handle simultaneous close

**Deliverables**:
- [ ] Retransmission working
- [ ] Flow control prevents buffer overflow
- [ ] Clean connection close

### 5.4: Congestion Control (Week 10)
1. **Slow Start**
   - Start with cwnd = 1 MSS
   - Exponential growth until ssthresh

2. **Congestion Avoidance**
   - Linear growth after ssthresh
   - Multiplicative decrease on loss

3. **Fast Recovery**
   - On fast retransmit, halve cwnd
   - Continue sending new data

**Deliverables**:
- [ ] Slow start implemented
- [ ] Congestion avoidance working
- [ ] Network performs well under load

### Key TCP Challenges
- Complex state machine (11 states!)
- Sequence number wraparound
- RTT estimation and timeout calculation
- Out-of-order packet handling
- Congestion control algorithms
- Edge cases (simultaneous close, connection reset, etc.)

---

## Phase 6: Integration & Testing (Week 11-12)
**Goal**: Polish and stress-test the stack

### Tasks
1. **End-to-End Testing**
   - HTTP server on your stack
   - Download files via your TCP implementation
   - Test with real applications

2. **Performance Testing**
   - Throughput benchmarks
   - Latency measurements
   - Packet loss scenarios

3. **Robustness Testing**
   - Handle malformed packets
   - Network failures
   - High connection count

4. **Documentation**
   - API documentation
   - Architecture diagrams
   - README with examples

### Deliverables
- [ ] Complete test suite
- [ ] Performance benchmarks
- [ ] Documentation complete
- [ ] Example applications (echo server, simple HTTP)

---

## Project Structure

```
network/
├── go.mod
├── GOAL.TXT
├── ROADMAP.md (this file)
├── README.md (usage guide)
│
├── pkg/
│   ├── common/           # Shared types and utilities
│   │   ├── types.go      # IP addresses, packet types
│   │   ├── checksum.go   # Internet checksum
│   │   └── buffer.go     # Packet buffers
│   │
│   ├── ethernet/         # Layer 2: Ethernet
│   │   ├── frame.go      # Frame parsing/building
│   │   └── interface.go  # Network interface handling
│   │
│   ├── arp/              # Layer 2: ARP
│   │   ├── arp.go        # ARP packet handling
│   │   └── cache.go      # ARP cache
│   │
│   ├── ip/               # Layer 3: IP
│   │   ├── packet.go     # IP packet parsing/building
│   │   ├── fragment.go   # Fragmentation/reassembly
│   │   └── routing.go    # Routing table
│   │
│   ├── icmp/             # Layer 3: ICMP
│   │   └── icmp.go       # ICMP messages (ping)
│   │
│   ├── udp/              # Layer 4: UDP
│   │   ├── packet.go     # UDP packet handling
│   │   └── socket.go     # UDP socket API
│   │
│   └── tcp/              # Layer 4: TCP
│       ├── packet.go     # TCP segment parsing/building
│       ├── state.go      # TCP state machine
│       ├── connection.go # TCP connection management
│       ├── send.go       # Send path
│       ├── recv.go       # Receive path
│       ├── retransmit.go # Retransmission timer
│       ├── congestion.go # Congestion control
│       └── socket.go     # TCP socket API
│
├── cmd/
│   └── netstack/
│       └── main.go       # Main entry point
│
├── examples/
│   ├── ping/             # ICMP ping example
│   ├── udp_echo/         # UDP echo server
│   ├── tcp_echo/         # TCP echo server
│   └── http_server/      # Simple HTTP server
│
└── tests/
    ├── unit/             # Unit tests for each package
    ├── integration/      # Integration tests
    └── benchmark/        # Performance benchmarks
```

---

## Testing Strategy

### Unit Tests
- Test each protocol layer independently
- Mock lower layers
- Test packet parsing/building
- Test checksum calculations

### Integration Tests
- Test cross-layer communication
- ARP → IP → UDP/TCP flow
- End-to-end packet flow

### Real-World Tests
- Ping from external host
- netcat UDP/TCP tests
- curl HTTP requests
- iperf throughput tests

### Tools
- **Wireshark**: Inspect packets
- **tcpdump**: Capture traffic
- **netcat**: TCP/UDP testing
- **curl**: HTTP testing
- **iperf**: Performance testing

---

## Development Timeline

### Weeks 1-2: Foundation
- Raw sockets, Ethernet, ARP
- Can communicate on local network

### Weeks 3-4: Internet Layer
- IP, ICMP, routing
- Can ping across networks

### Week 5: UDP
- Simple datagram service
- UDP applications work

### Weeks 6-10: TCP
- Connection establishment (Week 6-7)
- Data transfer (Week 8)
- Reliability & teardown (Week 9)
- Congestion control (Week 10)

### Weeks 11-12: Polish
- Testing, benchmarks, documentation

**Total Time**: ~3 months of focused work

---

## Learning Resources

### RFCs (The Official Specs)
- **RFC 791**: Internet Protocol (IP)
- **RFC 792**: Internet Control Message Protocol (ICMP)
- **RFC 793**: Transmission Control Protocol (TCP) - THE BIBLE
- **RFC 768**: User Datagram Protocol (UDP)
- **RFC 826**: Address Resolution Protocol (ARP)
- **RFC 1071**: Computing the Internet Checksum
- **RFC 5681**: TCP Congestion Control
- **RFC 6298**: Computing TCP's Retransmission Timer

### Books
- **TCP/IP Illustrated, Volume 1** by W. Richard Stevens (Classic!)
- **Computer Networks** by Andrew Tanenbaum
- **The TCP/IP Guide** by Charles Kozierok

### Online Resources
- **Beej's Guide to Network Programming** (C, but concepts apply)
- **Linux Kernel Networking** (see real implementation)
- **Go net package source code** (learn from the pros)

### Videos
- **Ben Eater's networking series** (YouTube)
- **Hussein Nasser's TCP explanations** (YouTube)

---

## Common Pitfalls & How to Avoid Them

### 1. Byte Order Confusion
- **Problem**: Network = Big Endian, your machine might be Little Endian
- **Solution**: Always use `binary.BigEndian` for network data

### 2. Checksum Errors
- **Problem**: Forgot pseudo-header for TCP/UDP
- **Solution**: Use Wireshark to verify your checksums

### 3. Sequence Number Wraparound
- **Problem**: Sequence numbers are uint32, they wrap around
- **Solution**: Use modular arithmetic for comparisons

### 4. State Machine Bugs
- **Problem**: TCP state machine has many edge cases
- **Solution**: Draw state diagrams, test each transition

### 5. Deadlocks
- **Problem**: Goroutines waiting on each other
- **Solution**: Careful lock ordering, use channels when possible

### 6. Buffer Management
- **Problem**: Memory leaks or buffer overruns
- **Solution**: Use ring buffers, clear references

---

## Success Metrics

### Minimum Viable Product (MVP)
- [ ] Can ping another host (ICMP)
- [ ] Can send/receive UDP packets
- [ ] Can establish TCP connection
- [ ] Can transfer data over TCP

### Stretch Goals
- [ ] HTTP server works
- [ ] Download a file via TCP
- [ ] 1 Gbps throughput
- [ ] Handle 10,000+ concurrent connections
- [ ] Implement TCP options (window scaling, timestamps)
- [ ] IPv6 support

---

## Next Steps

1. **Review this roadmap** - Understand the scope
2. **Set up development environment** - Go, Wireshark, netcat
3. **Start Phase 1** - Get raw sockets working
4. **Iterate weekly** - Follow the phases
5. **Test constantly** - Don't move forward until current phase works

---

## Getting Help

When stuck:
1. Read the RFC (dry but authoritative)
2. Check Wireshark captures (see what real packets look like)
3. Look at Linux kernel implementation (complex but correct)
4. Search for "TCP implementation tutorial" or similar
5. Ask specific questions with packet dumps

---

**Remember**: This is a marathon, not a sprint. TCP is complex because it solves hard problems. Take it one phase at a time, test thoroughly, and you'll learn more about networking than any course could teach you.

Good luck! 🚀
