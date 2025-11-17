// Package quic implements QUIC connection management.
package quic

import (
	"crypto/rand"
	"fmt"
	"net"
	"sync"
	"time"
)

// ConnectionState represents the state of a QUIC connection.
type ConnectionState uint8

const (
	StateIdle        ConnectionState = 0
	StateHandshaking ConnectionState = 1
	StateEstablished ConnectionState = 2
	StateClosing     ConnectionState = 3
	StateClosed      ConnectionState = 4
)

// Connection represents a QUIC connection.
type Connection struct {
	mu sync.RWMutex

	// Connection identifiers
	LocalConnID  []byte
	RemoteConnID []byte

	// Network
	conn       net.PacketConn
	remoteAddr net.Addr

	// State
	state          ConnectionState
	version        uint32
	maxStreamData  uint64
	maxData        uint64

	// Streams
	streams map[uint64]*Stream

	// Packet handling
	packetNumber uint64
	largestAcked uint64

	// Timing
	created time.Time
	lastSeen time.Time
}

// Stream represents a QUIC stream.
type Stream struct {
	ID       uint64
	sendBuf  []byte
	recvBuf  []byte
	offset   uint64
	finished bool
}

// NewConnection creates a new QUIC connection.
func NewConnection(conn net.PacketConn, remoteAddr net.Addr) (*Connection, error) {
	// Generate random connection ID
	localConnID := make([]byte, 8)
	if _, err := rand.Read(localConnID); err != nil {
		return nil, fmt.Errorf("failed to generate connection ID: %w", err)
	}

	return &Connection{
		LocalConnID:  localConnID,
		conn:         conn,
		remoteAddr:   remoteAddr,
		state:        StateIdle,
		version:      Version1,
		maxStreamData: 1024 * 1024,  // 1MB
		maxData:      10 * 1024 * 1024, // 10MB
		streams:      make(map[uint64]*Stream),
		packetNumber: 0,
		created:      time.Now(),
		lastSeen:     time.Now(),
	}, nil
}

// SendPacket sends a QUIC packet.
func (c *Connection) SendPacket(pkt *Packet) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := pkt.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize packet: %w", err)
	}

	_, err = c.conn.WriteTo(data, c.remoteAddr)
	if err != nil {
		return fmt.Errorf("failed to send packet: %w", err)
	}

	c.packetNumber++
	return nil
}

// ReceivePacket receives and processes a QUIC packet.
func (c *Connection) ReceivePacket() (*Packet, error) {
	buf := make([]byte, 65535)
	n, addr, err := c.conn.ReadFrom(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to receive packet: %w", err)
	}

	c.mu.Lock()
	c.remoteAddr = addr
	c.lastSeen = time.Now()
	c.mu.Unlock()

	pkt, err := Parse(buf[:n])
	if err != nil {
		return nil, fmt.Errorf("failed to parse packet: %w", err)
	}

	return pkt, nil
}

// OpenStream opens a new stream.
func (c *Connection) OpenStream() (*Stream, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state != StateEstablished {
		return nil, fmt.Errorf("connection not established")
	}

	streamID := uint64(len(c.streams) * 4) // Client-initiated bidirectional stream
	stream := &Stream{
		ID:      streamID,
		sendBuf: make([]byte, 0),
		recvBuf: make([]byte, 0),
		offset:  0,
	}

	c.streams[streamID] = stream
	return stream, nil
}

// GetStream gets a stream by ID.
func (c *Connection) GetStream(streamID uint64) (*Stream, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stream, ok := c.streams[streamID]
	if !ok {
		return nil, fmt.Errorf("stream not found: %d", streamID)
	}

	return stream, nil
}

// SendStreamData sends data on a stream.
func (c *Connection) SendStreamData(streamID uint64, data []byte, fin bool) error {
	stream, err := c.GetStream(streamID)
	if err != nil {
		return err
	}

	// Create STREAM frame
	frame := &StreamFrame{
		StreamID: streamID,
		Offset:   stream.offset,
		Data:     data,
		Fin:      fin,
	}

	frameData, err := frame.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize frame: %w", err)
	}

	// Create packet with frame
	pkt := New1RTTPacket(c.RemoteConnID, frameData)

	err = c.SendPacket(pkt)
	if err != nil {
		return err
	}

	stream.offset += uint64(len(data))
	if fin {
		stream.finished = true
	}

	return nil
}

// Close closes the connection.
func (c *Connection) Close(errorCode uint64, reason string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state == StateClosed {
		return nil
	}

	// Send CONNECTION_CLOSE frame
	frame := &ConnectionCloseFrame{
		ErrorCode:    errorCode,
		ReasonPhrase: reason,
	}

	frameData, err := frame.Serialize()
	if err != nil {
		return fmt.Errorf("failed to serialize close frame: %w", err)
	}

	pkt := New1RTTPacket(c.RemoteConnID, frameData)
	_ = c.SendPacket(pkt) // Best effort

	c.state = StateClosed
	return nil
}

// GetState returns the current connection state.
func (c *Connection) GetState() ConnectionState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// SetState sets the connection state.
func (c *Connection) SetState(state ConnectionState) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state = state
}

// String returns a string representation of the connection.
func (c *Connection) String() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stateStr := "Unknown"
	switch c.state {
	case StateIdle:
		stateStr = "Idle"
	case StateHandshaking:
		stateStr = "Handshaking"
	case StateEstablished:
		stateStr = "Established"
	case StateClosing:
		stateStr = "Closing"
	case StateClosed:
		stateStr = "Closed"
	}

	return fmt.Sprintf("QUIC{State=%s, LocalConnID=%x, RemoteConnID=%x, Streams=%d}",
		stateStr, c.LocalConnID, c.RemoteConnID, len(c.streams))
}

// String returns a string representation of the state.
func (s ConnectionState) String() string {
	switch s {
	case StateIdle:
		return "Idle"
	case StateHandshaking:
		return "Handshaking"
	case StateEstablished:
		return "Established"
	case StateClosing:
		return "Closing"
	case StateClosed:
		return "Closed"
	default:
		return "Unknown"
	}
}
