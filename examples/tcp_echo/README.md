# TCP Echo Server Example

This example demonstrates a simple TCP echo server using the custom TCP implementation.

## Features

- Accepts incoming TCP connections
- Echoes back any data received
- Handles multiple concurrent connections
- Demonstrates TCP connection lifecycle:
  - Three-way handshake
  - Data transfer
  - Connection teardown

## Usage

### Running the Server

```bash
# Run with default settings (eth0, 192.168.1.100:8080)
sudo go run main.go

# Specify interface and address
sudo go run main.go -i eth0 -addr 192.168.1.100 -port 8080
```

### Testing the Server

#### Using netcat

```bash
# Connect to the echo server
nc 192.168.1.100 8080

# Type some text and press Enter
# The server will echo it back
```

#### Using telnet

```bash
telnet 192.168.1.100 8080
```

#### Using curl

```bash
echo "Hello, TCP!" | nc 192.168.1.100 8080
```

## Implementation Notes

This example demonstrates:

1. **Socket Creation**: Creating a TCP socket with `tcp.NewSocket()`
2. **Listening**: Putting the socket in listening mode with `Listen()`
3. **Accepting Connections**: Accepting new connections with `Accept()`
4. **Sending Data**: Sending data with `Send()`
5. **Receiving Data**: Receiving data with `Recv()`
6. **Connection Management**: Handling multiple concurrent connections

## TCP Protocol Flow

### Connection Establishment (3-way handshake)

```
Client                Server
  |                     |
  |------ SYN -------->|
  |                     |
  |<--- SYN+ACK --------|
  |                     |
  |------ ACK -------->|
  |                     |
  |   (ESTABLISHED)     |
```

### Data Transfer

```
Client                Server
  |                     |
  |--- DATA (PSH) ---->|
  |                     |
  |<----- ACK ---------|
  |                     |
  |<-- DATA (PSH) -----|
  |                     |
  |------ ACK -------->|
```

### Connection Teardown (4-way handshake)

```
Client                Server
  |                     |
  |------ FIN -------->|
  |                     |
  |<----- ACK ---------|
  |                     |
  |<----- FIN ---------|
  |                     |
  |------ ACK -------->|
  |                     |
```

## Advanced Features

### Congestion Control

The TCP implementation includes:
- Slow start
- Congestion avoidance
- Fast retransmit
- Fast recovery

### Reliability

- Retransmission on timeout
- Duplicate ACK detection
- Sequence number tracking

### Flow Control

- Sliding window protocol
- Advertised window updates

## Common Issues

### Permission Denied

Raw sockets require root access:

```bash
sudo go run main.go
```

Or set capabilities:

```bash
go build -o tcp_echo main.go
sudo setcap cap_net_raw+ep ./tcp_echo
./tcp_echo
```

### Connection Timeout

Make sure:
1. Firewall allows connections on the specified port
2. IP address matches your network interface
3. Client can reach the server (use `ping` to test)

### No Response

Check:
1. Server is running and listening
2. Client is connecting to the correct IP and port
3. Network interface is up (`ip link show`)

## Learning Resources

- [RFC 793: TCP](https://tools.ietf.org/html/rfc793)
- [RFC 5681: TCP Congestion Control](https://tools.ietf.org/html/rfc5681)
- [RFC 6298: Computing TCP's Retransmission Timer](https://tools.ietf.org/html/rfc6298)
