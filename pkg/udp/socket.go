package udp

import (
	"fmt"
	"sync"
	"time"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

const (
	// DefaultReceiveBufferSize is the default size of the receive buffer.
	DefaultReceiveBufferSize = 100

	// DefaultReceiveTimeout is the default timeout for receiving packets.
	DefaultReceiveTimeout = 5 * time.Second

	// EphemeralPortStart is the start of the ephemeral port range.
	EphemeralPortStart = 49152

	// EphemeralPortEnd is the end of the ephemeral port range.
	EphemeralPortEnd = 65535
)

// Address represents a UDP endpoint (IP address and port).
type Address struct {
	IP   common.IPv4Address
	Port uint16
}

// String returns a human-readable representation of the address.
func (a Address) String() string {
	return fmt.Sprintf("%s:%d", a.IP, a.Port)
}

// Message represents a received UDP message with its source address.
type Message struct {
	Data []byte
	From Address
}

// Socket represents a UDP socket.
type Socket struct {
	// Local address (IP and port this socket is bound to)
	localAddr Address

	// Receive buffer
	receiveBuf chan Message

	// Socket state
	bound  bool
	closed bool

	// Mutex for thread-safety
	mu sync.RWMutex

	// Handler function for receiving packets (set by the network stack)
	receiveHandler func(*Packet, Address)
}

// NewSocket creates a new UDP socket.
func NewSocket() *Socket {
	return &Socket{
		receiveBuf: make(chan Message, DefaultReceiveBufferSize),
		bound:      false,
		closed:     false,
	}
}

// Bind binds the socket to a local address and port.
// If port is 0, an ephemeral port will be assigned.
func (s *Socket) Bind(addr Address) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.bound {
		return fmt.Errorf("socket already bound to %s", s.localAddr)
	}

	if s.closed {
		return fmt.Errorf("socket is closed")
	}

	s.localAddr = addr
	s.bound = true

	return nil
}

// LocalAddr returns the local address this socket is bound to.
func (s *Socket) LocalAddr() (Address, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.bound {
		return Address{}, fmt.Errorf("socket not bound")
	}

	return s.localAddr, nil
}

// SendTo sends data to the specified destination address.
// This returns the UDP packet that should be sent (the caller is responsible
// for wrapping it in an IP packet and sending it on the network).
func (s *Socket) SendTo(data []byte, to Address) (*Packet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return nil, fmt.Errorf("socket is closed")
	}

	if !s.bound {
		return nil, fmt.Errorf("socket not bound")
	}

	// Create UDP packet
	pkt := NewPacket(s.localAddr.Port, to.Port, data)

	return pkt, nil
}

// RecvFrom receives data from the socket with a timeout.
// It returns the data and the source address.
func (s *Socket) RecvFrom(timeout time.Duration) ([]byte, Address, error) {
	s.mu.RLock()
	if s.closed {
		s.mu.RUnlock()
		return nil, Address{}, fmt.Errorf("socket is closed")
	}
	if !s.bound {
		s.mu.RUnlock()
		return nil, Address{}, fmt.Errorf("socket not bound")
	}
	s.mu.RUnlock()

	// Wait for message or timeout
	select {
	case msg := <-s.receiveBuf:
		return msg.Data, msg.From, nil
	case <-time.After(timeout):
		return nil, Address{}, fmt.Errorf("receive timeout")
	}
}

// Receive is called by the network stack when a UDP packet is received.
// This is an internal method used by the UDP demultiplexer.
func (s *Socket) Receive(data []byte, from Address) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		return fmt.Errorf("socket is closed")
	}

	if !s.bound {
		return fmt.Errorf("socket not bound")
	}

	// Create message
	msg := Message{
		Data: make([]byte, len(data)),
		From: from,
	}
	copy(msg.Data, data)

	// Try to send to receive buffer
	select {
	case s.receiveBuf <- msg:
		return nil
	default:
		// Buffer full, drop packet
		return fmt.Errorf("receive buffer full, packet dropped")
	}
}

// Close closes the socket.
func (s *Socket) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("socket already closed")
	}

	s.closed = true
	close(s.receiveBuf)

	return nil
}

// IsClosed returns true if the socket is closed.
func (s *Socket) IsClosed() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.closed
}

// IsBound returns true if the socket is bound to a local address.
func (s *Socket) IsBound() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.bound
}

// Demultiplexer manages UDP sockets and routes incoming packets to the correct socket.
type Demultiplexer struct {
	// Map of port -> socket
	sockets map[uint16]*Socket

	// Next ephemeral port to assign
	nextEphemeralPort uint16

	// Mutex for thread-safety
	mu sync.RWMutex
}

// NewDemultiplexer creates a new UDP demultiplexer.
func NewDemultiplexer() *Demultiplexer {
	return &Demultiplexer{
		sockets:           make(map[uint16]*Socket),
		nextEphemeralPort: EphemeralPortStart,
	}
}

// Bind binds a socket to a port.
// If the requested port is 0, an ephemeral port is assigned.
func (d *Demultiplexer) Bind(socket *Socket, port uint16) (uint16, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Assign ephemeral port if requested
	if port == 0 {
		assignedPort, err := d.allocateEphemeralPort()
		if err != nil {
			return 0, err
		}
		port = assignedPort
	}

	// Check if port is already in use
	if _, exists := d.sockets[port]; exists {
		return 0, fmt.Errorf("port %d already in use", port)
	}

	// Register socket
	d.sockets[port] = socket

	return port, nil
}

// Unbind removes a socket from the demultiplexer.
func (d *Demultiplexer) Unbind(port uint16) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if _, exists := d.sockets[port]; !exists {
		return fmt.Errorf("port %d not bound", port)
	}

	delete(d.sockets, port)
	return nil
}

// Deliver delivers an incoming UDP packet to the appropriate socket.
func (d *Demultiplexer) Deliver(pkt *Packet, srcAddr Address) error {
	d.mu.RLock()
	socket, exists := d.sockets[pkt.DestinationPort]
	d.mu.RUnlock()

	if !exists {
		// No socket bound to this port - packet is dropped
		// In a real implementation, we might send an ICMP Port Unreachable
		return fmt.Errorf("no socket bound to port %d", pkt.DestinationPort)
	}

	// Deliver to socket
	return socket.Receive(pkt.Data, srcAddr)
}

// allocateEphemeralPort allocates an ephemeral port.
// Must be called with d.mu held.
func (d *Demultiplexer) allocateEphemeralPort() (uint16, error) {
	// Try to find a free port
	startPort := d.nextEphemeralPort
	for {
		if _, exists := d.sockets[d.nextEphemeralPort]; !exists {
			port := d.nextEphemeralPort
			d.nextEphemeralPort++
			if d.nextEphemeralPort > EphemeralPortEnd {
				d.nextEphemeralPort = EphemeralPortStart
			}
			return port, nil
		}

		d.nextEphemeralPort++
		if d.nextEphemeralPort > EphemeralPortEnd {
			d.nextEphemeralPort = EphemeralPortStart
		}

		// Check if we've wrapped around
		if d.nextEphemeralPort == startPort {
			return 0, fmt.Errorf("no ephemeral ports available")
		}
	}
}
