// Package tcp implements TCP socket API.
package tcp

import (
	"fmt"
	"sync"
	"time"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

// Socket represents a TCP socket.
type Socket struct {
	localAddr  common.IPv4Address
	localPort  uint16
	remoteAddr common.IPv4Address
	remotePort uint16

	conn *Connection

	// For listening sockets
	isListening    bool
	backlog        int
	acceptQueue    chan *Connection
	pendingConns   map[string]*Connection // Key: "remoteIP:remotePort"
	pendingConnsMu sync.Mutex

	// For sending packets
	sendFunc func(*Segment, common.IPv4Address, common.IPv4Address) error

	// Data channel
	dataReady chan []byte

	mu sync.RWMutex
}

// NewSocket creates a new TCP socket.
func NewSocket(localAddr common.IPv4Address, localPort uint16) *Socket {
	return &Socket{
		localAddr:    localAddr,
		localPort:    localPort,
		isListening:  false,
		backlog:      128,
		acceptQueue:  make(chan *Connection, 128),
		pendingConns: make(map[string]*Connection),
		dataReady:    make(chan []byte, 100),
	}
}

// SetSendFunc sets the function to call when sending segments.
func (s *Socket) SetSendFunc(f func(*Segment, common.IPv4Address, common.IPv4Address) error) {
	s.sendFunc = f
}

// Bind binds the socket to a local address and port.
func (s *Socket) Bind(addr common.IPv4Address, port uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.localAddr = addr
	s.localPort = port
	return nil
}

// Listen puts the socket in listening mode.
func (s *Socket) Listen(backlog int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isListening {
		return fmt.Errorf("socket already listening")
	}

	s.isListening = true
	s.backlog = backlog
	s.acceptQueue = make(chan *Connection, backlog)

	return nil
}

// Accept accepts a new connection.
// Blocks until a connection is available.
func (s *Socket) Accept() (*Socket, error) {
	if !s.isListening {
		return nil, fmt.Errorf("socket not listening")
	}

	// Wait for a connection
	conn, ok := <-s.acceptQueue
	if !ok {
		return nil, fmt.Errorf("accept queue closed")
	}

	// Create a new socket for this connection
	newSocket := &Socket{
		localAddr:  conn.LocalAddr,
		localPort:  conn.LocalPort,
		remoteAddr: conn.RemoteAddr,
		remotePort: conn.RemotePort,
		conn:       conn,
		sendFunc:   s.sendFunc,
		dataReady:  make(chan []byte, 100),
	}

	// Set up connection callbacks
	conn.onSegmentReady = func(seg *Segment) error {
		if newSocket.sendFunc != nil {
			return newSocket.sendFunc(seg, conn.LocalAddr, conn.RemoteAddr)
		}
		return nil
	}

	conn.onDataReady = func(data []byte) {
		newSocket.dataReady <- data
	}

	conn.onClose = func() {
		close(newSocket.dataReady)
	}

	return newSocket, nil
}

// Connect connects to a remote address and port.
func (s *Socket) Connect(remoteAddr common.IPv4Address, remotePort uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn != nil {
		return fmt.Errorf("socket already connected")
	}

	s.remoteAddr = remoteAddr
	s.remotePort = remotePort

	// Create connection
	s.conn = NewConnection(s.localAddr, s.localPort, remoteAddr, remotePort)

	// Set up callbacks
	s.conn.onSegmentReady = func(seg *Segment) error {
		if s.sendFunc != nil {
			return s.sendFunc(seg, s.localAddr, remoteAddr)
		}
		return nil
	}

	s.conn.onDataReady = func(data []byte) {
		s.dataReady <- data
	}

	s.conn.onClose = func() {
		close(s.dataReady)
	}

	// Initiate connection
	if err := s.conn.ActiveOpen(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	// Wait for connection to be established (with timeout)
	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("connection timeout")
		case <-ticker.C:
			if s.conn.GetState() == StateEstablished {
				return nil
			}
			if s.conn.GetState() == StateClosed {
				return fmt.Errorf("connection failed")
			}
		}
	}
}

// Send sends data over the connection.
func (s *Socket) Send(data []byte) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.conn == nil {
		return 0, fmt.Errorf("not connected")
	}

	if err := s.conn.Send(data); err != nil {
		return 0, err
	}

	return len(data), nil
}

// Recv receives data from the connection.
// Blocks until data is available.
func (s *Socket) Recv(buf []byte) (int, error) {
	data, ok := <-s.dataReady
	if !ok {
		return 0, fmt.Errorf("connection closed")
	}

	n := copy(buf, data)
	return n, nil
}

// RecvTimeout receives data with a timeout.
func (s *Socket) RecvTimeout(buf []byte, timeout time.Duration) (int, error) {
	select {
	case data, ok := <-s.dataReady:
		if !ok {
			return 0, fmt.Errorf("connection closed")
		}
		n := copy(buf, data)
		return n, nil
	case <-time.After(timeout):
		return 0, fmt.Errorf("receive timeout")
	}
}

// Close closes the socket.
func (s *Socket) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isListening {
		close(s.acceptQueue)
		s.isListening = false
		return nil
	}

	if s.conn != nil {
		return s.conn.Close()
	}

	return nil
}

// HandleIncomingSegment handles an incoming TCP segment.
// This should be called by the network stack when a TCP segment is received.
func (s *Socket) HandleIncomingSegment(seg *Segment, srcIP common.IPv4Address, dstIP common.IPv4Address) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isListening {
		return s.handleListeningSegment(seg, srcIP, dstIP)
	}

	if s.conn != nil {
		return s.conn.HandleSegment(seg)
	}

	// No connection and not listening - send RST
	return s.sendRST(seg, dstIP, srcIP)
}

// handleListeningSegment handles segments for a listening socket.
func (s *Socket) handleListeningSegment(seg *Segment, srcIP common.IPv4Address, dstIP common.IPv4Address) error {
	connKey := fmt.Sprintf("%s:%d", srcIP, seg.SourcePort)

	// Check if we have a pending connection
	s.pendingConnsMu.Lock()
	conn, exists := s.pendingConns[connKey]
	s.pendingConnsMu.Unlock()

	if exists {
		// Handle segment for existing pending connection
		if err := conn.HandleSegment(seg); err != nil {
			return err
		}

		// Check if connection is established
		if conn.GetState() == StateEstablished {
			// Move to accept queue
			s.pendingConnsMu.Lock()
			delete(s.pendingConns, connKey)
			s.pendingConnsMu.Unlock()

			select {
			case s.acceptQueue <- conn:
				// Connection added to accept queue
			default:
				// Accept queue full - drop connection
				conn.Close()
			}
		}

		return nil
	}

	// New connection attempt
	if seg.HasFlag(FlagSYN) && !seg.HasFlag(FlagACK) {
		// Create new connection
		newConn := NewConnection(dstIP, s.localPort, srcIP, seg.SourcePort)

		// Set up callbacks
		newConn.onSegmentReady = func(outSeg *Segment) error {
			if s.sendFunc != nil {
				return s.sendFunc(outSeg, dstIP, srcIP)
			}
			return nil
		}

		// Transition to LISTEN state
		newConn.state.SetState(StateListen)

		// Handle the SYN segment
		if err := newConn.HandleSegment(seg); err != nil {
			return err
		}

		// Add to pending connections
		s.pendingConnsMu.Lock()
		s.pendingConns[connKey] = newConn
		s.pendingConnsMu.Unlock()

		return nil
	}

	// Unexpected segment - send RST
	return s.sendRST(seg, dstIP, srcIP)
}

// sendRST sends a RST segment.
func (s *Socket) sendRST(seg *Segment, srcIP common.IPv4Address, dstIP common.IPv4Address) error {
	var rst *Segment

	if seg.HasFlag(FlagACK) {
		// If incoming has ACK, RST seq = incoming ACK
		rst = NewSegment(seg.DestinationPort, seg.SourcePort, seg.AckNumber, 0, FlagRST, 0, nil)
	} else {
		// Otherwise, RST ACK = incoming seq + len
		seqLen := uint32(len(seg.Data))
		if seg.HasFlag(FlagSYN) {
			seqLen++
		}
		if seg.HasFlag(FlagFIN) {
			seqLen++
		}
		rst = NewSegment(seg.DestinationPort, seg.SourcePort, 0, seg.SequenceNumber+seqLen, FlagRST|FlagACK, 0, nil)
	}

	checksum, err := rst.CalculateChecksum(srcIP, dstIP)
	if err != nil {
		return err
	}
	rst.Checksum = checksum

	if s.sendFunc != nil {
		return s.sendFunc(rst, srcIP, dstIP)
	}

	return nil
}

// GetLocalAddr returns the local address.
func (s *Socket) GetLocalAddr() common.IPv4Address {
	return s.localAddr
}

// GetLocalPort returns the local port.
func (s *Socket) GetLocalPort() uint16 {
	return s.localPort
}

// GetRemoteAddr returns the remote address.
func (s *Socket) GetRemoteAddr() common.IPv4Address {
	return s.remoteAddr
}

// GetRemotePort returns the remote port.
func (s *Socket) GetRemotePort() uint16 {
	return s.remotePort
}

// GetState returns the connection state.
func (s *Socket) GetState() State {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.conn != nil {
		return s.conn.GetState()
	}

	if s.isListening {
		return StateListen
	}

	return StateClosed
}
