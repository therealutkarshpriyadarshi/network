# UDP Echo Server Example

This example demonstrates a UDP echo server implementation using the custom network stack. The server listens for UDP packets on a specified port and echoes them back to the sender.

## Features

- Raw socket packet capture and transmission
- UDP packet parsing and building
- UDP checksum calculation and verification
- Port demultiplexing
- Ethernet, IP, and UDP layer handling

## Prerequisites

- Go 1.21+
- Linux system (for raw sockets)
- Root/sudo access (for packet capture)
- Network interface with IPv4 address

## Usage

```bash
# Build
go build -o udp_echo main.go

# Run (requires root for raw sockets)
sudo ./udp_echo -i <interface> [-p <port>] [-v]

# Example
sudo ./udp_echo -i eth0 -p 8080 -v
```

### Flags

- `-i <interface>`: Network interface to use (required, e.g., eth0, wlan0)
- `-p <port>`: Port to listen on (default: 8080)
- `-v`: Verbose output (optional)

## Testing

### Using netcat (nc)

```bash
# In one terminal, start the echo server
sudo ./udp_echo -i eth0 -p 8080 -v

# In another terminal, send UDP packets
echo "Hello, UDP!" | nc -u <server_ip> 8080

# Or use interactive mode
nc -u <server_ip> 8080
Hello, UDP!
Hello, UDP!  # Server echoes back
```

### Using Python

```python
#!/usr/bin/env python3
import socket

# Create UDP socket
sock = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)

# Send message
server_address = ('192.168.1.100', 8080)
message = b'Hello from Python!'
sock.sendto(message, server_address)

# Receive echo
data, server = sock.recvfrom(4096)
print(f'Received: {data.decode()}')

sock.close()
```

### Using Go client

```go
package main

import (
    "fmt"
    "net"
)

func main() {
    conn, err := net.Dial("udp", "192.168.1.100:8080")
    if err != nil {
        panic(err)
    }
    defer conn.Close()

    // Send message
    message := []byte("Hello from Go!")
    _, err = conn.Write(message)
    if err != nil {
        panic(err)
    }

    // Receive echo
    buffer := make([]byte, 1024)
    n, err := conn.Read(buffer)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Received: %s\n", string(buffer[:n]))
}
```

## How It Works

1. **Socket Setup**: Creates a raw `AF_PACKET` socket and binds to the specified network interface
2. **Packet Reception**: Continuously receives packets from the network interface
3. **Layer Processing**:
   - Parses Ethernet frames
   - Filters for IPv4 packets
   - Parses IP packets
   - Filters for UDP packets destined for the local IP
   - Parses UDP packets and verifies checksums
4. **Echo Response**: For packets on the listening port, sends back the same data to the source
5. **Packet Transmission**: Constructs UDP → IP → Ethernet layers and sends via raw socket

## Packet Flow

```
Incoming:
┌─────────────┐
│  Network    │
└──────┬──────┘
       │
       v
┌─────────────┐
│  Ethernet   │  Parse frame
└──────┬──────┘
       │
       v
┌─────────────┐
│     IP      │  Parse packet, check destination
└──────┬──────┘
       │
       v
┌─────────────┐
│    UDP      │  Parse packet, verify checksum
└──────┬──────┘
       │
       v
┌─────────────┐
│  Echo Data  │
└─────────────┘

Outgoing:
┌─────────────┐
│  Echo Data  │
└──────┬──────┘
       │
       v
┌─────────────┐
│    UDP      │  Build packet, calculate checksum
└──────┬──────┘
       │
       v
┌─────────────┐
│     IP      │  Build packet, calculate checksum
└──────┬──────┘
       │
       v
┌─────────────┐
│  Ethernet   │  Build frame
└──────┬──────┘
       │
       v
┌─────────────┐
│  Network    │
└─────────────┘
```

## Debugging

### Using Wireshark

```bash
# Capture on the interface
sudo wireshark -i eth0

# Filter for UDP traffic on your port
udp.port == 8080
```

### Using tcpdump

```bash
# Capture UDP traffic on port 8080
sudo tcpdump -i eth0 -n udp port 8080 -v

# Save to file for analysis
sudo tcpdump -i eth0 -n udp port 8080 -w capture.pcap
```

## Common Issues

### Permission Denied
Raw sockets require root access:
```bash
sudo ./udp_echo -i eth0
```

### No Packets Received
- Check interface is up: `ip link show eth0`
- Check IP address: `ip addr show eth0`
- Check firewall rules: `sudo iptables -L`
- Verify client is sending to correct IP and port

### Checksum Errors
- Use Wireshark to verify packet format
- Check that pseudo-header is constructed correctly
- Verify byte order (network = big endian)

## Architecture

This example demonstrates the UDP implementation as part of the network stack:

```
┌─────────────────────────────────────┐
│  Application (Echo Logic)           │
├─────────────────────────────────────┤
│  UDP Socket & Demultiplexer         │  ← pkg/udp
├─────────────────────────────────────┤
│  UDP Packet Handling                │  ← pkg/udp
├─────────────────────────────────────┤
│  IP Packet Handling                 │  ← pkg/ip
├─────────────────────────────────────┤
│  Ethernet Frame Handling            │  ← pkg/ethernet
├─────────────────────────────────────┤
│  Raw Sockets                        │  ← syscall
└─────────────────────────────────────┘
```

## Learning Points

1. **UDP Protocol**: Connectionless, unreliable datagram service
2. **Checksums**: UDP pseudo-header construction for checksum calculation
3. **Port Demultiplexing**: Routing packets to correct socket based on port
4. **Raw Sockets**: Low-level network programming
5. **Byte Order**: Network byte order (big endian) handling
6. **Layer Encapsulation**: How data is wrapped in protocol headers

## References

- [RFC 768](https://tools.ietf.org/html/rfc768) - User Datagram Protocol
- [RFC 791](https://tools.ietf.org/html/rfc791) - Internet Protocol
- [RFC 1071](https://tools.ietf.org/html/rfc1071) - Computing the Internet Checksum
