# Phase 5: TCP Implementation

This document describes the implementation of the Transmission Control Protocol (TCP) for the network stack project.

## Overview

TCP is a connection-oriented, reliable, byte-stream transport protocol. It provides:

- **Reliable delivery**: Lost packets are retransmitted
- **Ordered delivery**: Data arrives in the order it was sent
- **Flow control**: Prevents overwhelming the receiver
- **Congestion control**: Adapts to network conditions
- **Full-duplex communication**: Data flows in both directions

## Implementation Structure

The TCP implementation is divided into several modules:

```
pkg/tcp/
├── packet.go        # TCP segment parsing and building
├── state.go         # TCP state machine (11 states)
├── connection.go    # Connection management
├── send.go          # Send buffer management
├── recv.go          # Receive buffer management
├── retransmit.go    # Retransmission queue
├── congestion.go    # Congestion control algorithms
└── socket.go        # High-level socket API
```

## TCP State Machine

TCP uses a complex state machine with 11 states:

### States

1. **CLOSED**: No connection exists
2. **LISTEN**: Waiting for incoming connections (server)
3. **SYN_SENT**: Waiting for SYN+ACK after sending SYN (client)
4. **SYN_RECEIVED**: Waiting for ACK after receiving SYN and sending SYN+ACK
5. **ESTABLISHED**: Connection established, data transfer phase
6. **FIN_WAIT_1**: Waiting for FIN ACK after sending FIN
7. **FIN_WAIT_2**: Waiting for FIN from peer
8. **CLOSE_WAIT**: Waiting for application to close after receiving FIN
9. **CLOSING**: Waiting for FIN ACK (simultaneous close)
10. **LAST_ACK**: Waiting for final ACK
11. **TIME_WAIT**: Waiting 2*MSL before closing

### Connection Establishment (3-way handshake)

```
Client                    Server
CLOSED                    LISTEN
  |                          |
  |-------- SYN ------------>|
  |     (seq=x)              |
SYN_SENT                 SYN_RECEIVED
  |                          |
  |<----- SYN+ACK -----------|
  |   (seq=y, ack=x+1)       |
  |                          |
  |-------- ACK ------------>|
  |     (ack=y+1)            |
ESTABLISHED              ESTABLISHED
```

### Connection Teardown (4-way handshake)

```
Client                    Server
ESTABLISHED              ESTABLISHED
  |                          |
  |-------- FIN ------------>|
  |                          |
FIN_WAIT_1               CLOSE_WAIT
  |                          |
  |<------- ACK -------------|
  |                          |
FIN_WAIT_2                   |
  |                          |
  |<------- FIN -------------|
  |                          |
  |-------- ACK ------------>|
  |                          |
TIME_WAIT                CLOSED
  |
  | (2*MSL timeout)
  |
CLOSED
```

## Key Features

### 1. Segment Structure

TCP segments contain:

```
 0                   1                   2                   3
 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1 2 3 4 5 6 7 8 9 0 1
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|          Source Port          |       Destination Port        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                        Sequence Number                        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Acknowledgment Number                      |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|  Data |       |C|E|U|A|P|R|S|F|                               |
| Offset| Rsrvd |W|C|R|C|S|S|Y|I|            Window             |
|       |       |R|E|G|K|H|T|N|N|                               |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|           Checksum            |         Urgent Pointer        |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                    Options                    |    Padding    |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
|                             data                              |
+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
```

### 2. Reliability

**Retransmission**: Segments that are not acknowledged within RTO (Retransmission Timeout) are retransmitted.

**RTT Estimation**: Round-trip time is estimated using:
- SRTT (Smoothed RTT) = (1 - α) * SRTT + α * RTT_measured
- RTTVAR = (1 - β) * RTTVAR + β * |SRTT - RTT_measured|
- RTO = SRTT + 4 * RTTVAR

**Fast Retransmit**: When 3 duplicate ACKs are received, retransmit immediately without waiting for timeout.

### 3. Flow Control

TCP uses a **sliding window protocol**:

- Receiver advertises available buffer space (receive window)
- Sender limits outstanding data to min(cwnd, rwnd)
- Prevents receiver buffer overflow

### 4. Congestion Control

Implementation includes standard TCP congestion control algorithms:

#### Slow Start
- Start with cwnd = 2 * MSS
- Exponential growth: cwnd += bytes_acked
- Continue until cwnd >= ssthresh

#### Congestion Avoidance
- Linear growth: cwnd += MSS² / cwnd (approximately MSS per RTT)
- Entered when cwnd >= ssthresh

#### Fast Recovery
- Triggered on 3 duplicate ACKs
- Set ssthresh = cwnd / 2
- Set cwnd = ssthresh + 3 * MSS
- Continue transmitting new data

#### Timeout Recovery
- Set ssthresh = cwnd / 2
- Set cwnd = MSS
- Enter slow start

### 5. Sequence Numbers

TCP uses 32-bit sequence numbers that wrap around:

```go
// Comparison functions handle wraparound
func seqBefore(seq1, seq2 uint32) bool {
    return int32(seq1 - seq2) < 0
}

func seqAfter(seq1, seq2 uint32) bool {
    return int32(seq1 - seq2) > 0
}
```

### 6. Options

Supported TCP options:

- **MSS (Maximum Segment Size)**: Negotiated during connection setup
- **Window Scale**: Allows windows larger than 64KB
- **Timestamps**: For RTT measurement and PAWS (Protection Against Wrapped Sequences)

## Socket API

The TCP implementation provides a BSD-socket-like API:

```go
// Server side
socket := tcp.NewSocket(localAddr, localPort)
socket.Listen(backlog)
conn, _ := socket.Accept()
n, _ := conn.Recv(buf)
conn.Send(data)
conn.Close()

// Client side
socket := tcp.NewSocket(localAddr, localPort)
socket.Connect(remoteAddr, remotePort)
socket.Send(data)
n, _ := socket.Recv(buf)
socket.Close()
```

## Testing

### Unit Tests

Comprehensive unit tests cover:

- Segment parsing and serialization
- Checksum calculation and verification
- State machine transitions
- Send/receive buffer operations
- Retransmission queue management
- Sequence number wraparound handling

Run tests:

```bash
go test ./pkg/tcp -v
```

### Integration Tests

Test the full TCP stack:

```bash
# Run TCP echo server
sudo go run ./examples/tcp_echo/main.go -i eth0 -addr 192.168.1.100 -port 8080

# In another terminal, connect with netcat
nc 192.168.1.100 8080
```

## Performance Considerations

### Buffer Management

- Send and receive buffers use efficient slice operations
- Avoid unnecessary allocations by reusing buffers
- Buffer sizes are configurable

### Concurrency

- State machine operations are protected by mutexes
- Each connection can run in its own goroutine
- Lock-free operations where possible

### Memory Usage

- Retransmission queue stores only unacknowledged segments
- Buffers are sized appropriately to avoid waste
- Old connections are cleaned up promptly

## Known Limitations

1. **Options**: Limited support for TCP options (only MSS implemented)
2. **Selective ACK**: SACK is not implemented
3. **Window Scaling**: Not fully implemented
4. **Path MTU Discovery**: Not implemented
5. **Timestamps**: Option parsing exists but not used for RTT
6. **Out-of-order handling**: Basic implementation, could be improved

## Future Enhancements

1. **SACK (Selective Acknowledgment)**: Better handling of packet loss
2. **Window Scaling**: Support for high-bandwidth networks
3. **TCP Fast Open**: Reduce connection establishment latency
4. **Better congestion control**: Implement modern algorithms (BBR, CUBIC)
5. **Zero-copy optimizations**: Reduce memory copies
6. **Connection pooling**: Reuse connections efficiently

## References

- [RFC 793](https://tools.ietf.org/html/rfc793) - Transmission Control Protocol
- [RFC 5681](https://tools.ietf.org/html/rfc5681) - TCP Congestion Control
- [RFC 6298](https://tools.ietf.org/html/rfc6298) - Computing TCP's Retransmission Timer
- [RFC 2018](https://tools.ietf.org/html/rfc2018) - TCP Selective Acknowledgment
- [RFC 7323](https://tools.ietf.org/html/rfc7323) - TCP Extensions for High Performance

## Debugging Tips

### Enable Verbose Logging

Add logging to see segment flow:

```go
log.Printf("Sending: %s", segment)
log.Printf("Received: %s", segment)
log.Printf("State: %s -> %s", oldState, newState)
```

### Use Wireshark

Capture and analyze TCP traffic:

```bash
sudo wireshark -i eth0 -f "tcp port 8080"
```

### Monitor Connection State

Check connection state during debugging:

```go
fmt.Printf("Connection state: %s\n", conn.GetState())
fmt.Printf("Send window: %d, Congestion window: %d\n", conn.sndWnd, conn.cwnd)
```

### Common Issues

**Connection hangs**: Check for deadlocks in state transitions
**Data loss**: Verify retransmission logic and sequence numbers
**Performance**: Monitor congestion window and RTT estimates
**Checksum errors**: Ensure pseudo-header is calculated correctly

## Conclusion

Phase 5 completes the transport layer implementation with a fully functional TCP stack. The implementation handles connection management, reliable data transfer, flow control, and congestion control, providing a solid foundation for building network applications.

The TCP implementation demonstrates:
- Complex state machine management
- Reliability through retransmission
- Flow control with sliding windows
- Congestion control algorithms
- Practical socket API design

This phase represents the culmination of the network stack project, bringing together all previous layers (Ethernet, ARP, IP, ICMP, UDP) to create a complete, working TCP/IP implementation.
