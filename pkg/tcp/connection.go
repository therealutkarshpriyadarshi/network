// Package tcp implements TCP connection management.
package tcp

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

// Connection represents a TCP connection.
type Connection struct {
	// Connection identification
	LocalAddr  common.IPv4Address
	LocalPort  uint16
	RemoteAddr common.IPv4Address
	RemotePort uint16

	// State machine
	state *StateMachine
	mu    sync.RWMutex

	// Sequence numbers
	sndUna uint32 // Send unacknowledged
	sndNxt uint32 // Send next
	sndWnd uint16 // Send window
	iss    uint32 // Initial send sequence number

	rcvNxt uint32 // Receive next
	rcvWnd uint16 // Receive window
	irs    uint32 // Initial receive sequence number

	// Buffers
	sendBuffer    *SendBuffer
	receiveBuffer *ReceiveBuffer

	// Retransmission
	retransmitQueue *RetransmitQueue
	rto             time.Duration // Retransmission timeout
	srtt            time.Duration // Smoothed round-trip time
	rttvar          time.Duration // Round-trip time variation

	// Congestion control
	cwnd      uint32 // Congestion window (in bytes)
	ssthresh  uint32 // Slow start threshold
	dupAckCnt int    // Duplicate ACK count

	// Options
	mss         uint16 // Maximum segment size
	windowScale uint8  // Window scale factor

	// Timers
	timeWaitTimer *time.Timer
	retransmitTimer *time.Timer

	// Callbacks
	onSegmentReady func(*Segment) error // Called when a segment is ready to send
	onDataReady    func([]byte)         // Called when data is ready to deliver to app
	onClose        func()               // Called when connection is closed
}

// NewConnection creates a new TCP connection.
func NewConnection(localAddr common.IPv4Address, localPort uint16, remoteAddr common.IPv4Address, remotePort uint16) *Connection {
	conn := &Connection{
		LocalAddr:       localAddr,
		LocalPort:       localPort,
		RemoteAddr:      remoteAddr,
		RemotePort:      remotePort,
		state:           NewStateMachine(),
		rcvWnd:          65535, // Default receive window
		sndWnd:          65535, // Default send window (will be updated)
		sendBuffer:      NewSendBuffer(),
		receiveBuffer:   NewReceiveBuffer(65535),
		retransmitQueue: NewRetransmitQueue(),
		rto:             time.Second,     // Initial RTO = 1 second
		srtt:            0,
		rttvar:          0,
		cwnd:            DefaultMSS * 2,  // Initial cwnd = 2 * MSS
		ssthresh:         65535,          // Initial ssthresh = max window
		mss:             DefaultMSS,
		windowScale:     0,
	}

	return conn
}

// GetState returns the current connection state.
func (c *Connection) GetState() State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state.GetState()
}

// SetState sets the connection state.
func (c *Connection) SetState(state State) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.state.SetState(state)
}

// ActiveOpen initiates an active open (client-side connection).
func (c *Connection) ActiveOpen() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state.GetState() != StateClosed {
		return fmt.Errorf("connection not in CLOSED state")
	}

	// Generate initial sequence number
	c.iss = c.generateISN()
	c.sndUna = c.iss
	c.sndNxt = c.iss

	// Create SYN segment
	seg := NewSegment(c.LocalPort, c.RemotePort, c.iss, 0, FlagSYN, c.rcvWnd, nil)
	seg.Options = BuildMSSOption(c.mss)

	// Calculate checksum
	checksum, err := seg.CalculateChecksum(c.LocalAddr, c.RemoteAddr)
	if err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}
	seg.Checksum = checksum

	// Transition state
	if err := c.state.Transition(EventActiveOpen); err != nil {
		return err
	}

	// Send SYN segment
	if c.onSegmentReady != nil {
		if err := c.onSegmentReady(seg); err != nil {
			return fmt.Errorf("failed to send SYN: %w", err)
		}
	}

	// Add to retransmit queue
	c.retransmitQueue.Add(c.iss, seg, time.Now())
	c.sndNxt = c.iss + 1

	return nil
}

// PassiveOpen initiates a passive open (server-side listen).
func (c *Connection) PassiveOpen() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.state.GetState() != StateClosed {
		return fmt.Errorf("connection not in CLOSED state")
	}

	// Transition to LISTEN state
	return c.state.Transition(EventPassiveOpen)
}

// HandleSegment processes an incoming TCP segment.
func (c *Connection) HandleSegment(seg *Segment) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	state := c.state.GetState()

	// Verify checksum
	if !seg.VerifyChecksum(c.RemoteAddr, c.LocalAddr) {
		return fmt.Errorf("checksum verification failed")
	}

	// State-specific processing
	switch state {
	case StateListen:
		return c.handleSegmentListen(seg)
	case StateSynSent:
		return c.handleSegmentSynSent(seg)
	case StateSynReceived:
		return c.handleSegmentSynReceived(seg)
	case StateEstablished:
		return c.handleSegmentEstablished(seg)
	case StateFinWait1:
		return c.handleSegmentFinWait1(seg)
	case StateFinWait2:
		return c.handleSegmentFinWait2(seg)
	case StateCloseWait:
		return c.handleSegmentCloseWait(seg)
	case StateClosing:
		return c.handleSegmentClosing(seg)
	case StateLastAck:
		return c.handleSegmentLastAck(seg)
	case StateTimeWait:
		return c.handleSegmentTimeWait(seg)
	default:
		return fmt.Errorf("invalid state: %s", state)
	}
}

// handleSegmentListen handles segments in LISTEN state.
func (c *Connection) handleSegmentListen(seg *Segment) error {
	if seg.HasFlag(FlagSYN) && !seg.HasFlag(FlagACK) {
		// Received SYN, transition to SYN_RECEIVED
		c.irs = seg.SequenceNumber
		c.rcvNxt = seg.SequenceNumber + 1

		// Generate ISN for this connection
		c.iss = c.generateISN()
		c.sndUna = c.iss
		c.sndNxt = c.iss

		// Extract MSS from options
		if mss, err := seg.GetMSS(); err == nil {
			c.mss = mss
		}

		// Send SYN+ACK
		reply := NewSegment(c.LocalPort, c.RemotePort, c.iss, c.rcvNxt, FlagSYN|FlagACK, c.rcvWnd, nil)
		reply.Options = BuildMSSOption(c.mss)

		checksum, err := reply.CalculateChecksum(c.LocalAddr, c.RemoteAddr)
		if err != nil {
			return err
		}
		reply.Checksum = checksum

		if err := c.state.Transition(EventReceiveSyn); err != nil {
			return err
		}

		if c.onSegmentReady != nil {
			if err := c.onSegmentReady(reply); err != nil {
				return err
			}
		}

		c.retransmitQueue.Add(c.iss, reply, time.Now())
		c.sndNxt = c.iss + 1

		return nil
	}

	return fmt.Errorf("expected SYN in LISTEN state")
}

// handleSegmentSynSent handles segments in SYN_SENT state.
func (c *Connection) handleSegmentSynSent(seg *Segment) error {
	if seg.HasFlag(FlagSYN) && seg.HasFlag(FlagACK) {
		// Received SYN+ACK
		if seg.AckNumber != c.sndNxt {
			return fmt.Errorf("invalid ACK number: got %d, expected %d", seg.AckNumber, c.sndNxt)
		}

		c.irs = seg.SequenceNumber
		c.rcvNxt = seg.SequenceNumber + 1
		c.sndUna = seg.AckNumber

		// Extract MSS from options
		if mss, err := seg.GetMSS(); err == nil {
			if mss < c.mss {
				c.mss = mss
			}
		}

		// Update send window
		c.sndWnd = seg.WindowSize

		// Remove SYN from retransmit queue
		c.retransmitQueue.Remove(c.iss)

		// Send ACK
		ack := NewSegment(c.LocalPort, c.RemotePort, c.sndNxt, c.rcvNxt, FlagACK, c.rcvWnd, nil)
		checksum, err := ack.CalculateChecksum(c.LocalAddr, c.RemoteAddr)
		if err != nil {
			return err
		}
		ack.Checksum = checksum

		if err := c.state.Transition(EventReceiveSynAck); err != nil {
			return err
		}

		if c.onSegmentReady != nil {
			if err := c.onSegmentReady(ack); err != nil {
				return err
			}
		}

		return nil
	}

	return fmt.Errorf("expected SYN+ACK in SYN_SENT state")
}

// handleSegmentSynReceived handles segments in SYN_RECEIVED state.
func (c *Connection) handleSegmentSynReceived(seg *Segment) error {
	if seg.HasFlag(FlagACK) {
		if seg.AckNumber != c.sndNxt {
			return fmt.Errorf("invalid ACK number: got %d, expected %d", seg.AckNumber, c.sndNxt)
		}

		c.sndUna = seg.AckNumber
		c.sndWnd = seg.WindowSize

		// Remove SYN from retransmit queue
		c.retransmitQueue.Remove(c.iss)

		// Transition to ESTABLISHED
		return c.state.Transition(EventReceiveAck)
	}

	return fmt.Errorf("expected ACK in SYN_RECEIVED state")
}

// handleSegmentEstablished handles segments in ESTABLISHED state.
func (c *Connection) handleSegmentEstablished(seg *Segment) error {
	// Process ACK
	if seg.HasFlag(FlagACK) {
		c.processAck(seg)
	}

	// Process data
	if len(seg.Data) > 0 {
		c.processData(seg)
	}

	// Process FIN
	if seg.HasFlag(FlagFIN) {
		c.rcvNxt = seg.SequenceNumber + uint32(len(seg.Data)) + 1

		// Send ACK for FIN
		ack := NewSegment(c.LocalPort, c.RemotePort, c.sndNxt, c.rcvNxt, FlagACK, c.rcvWnd, nil)
		checksum, _ := ack.CalculateChecksum(c.LocalAddr, c.RemoteAddr)
		ack.Checksum = checksum

		if c.onSegmentReady != nil {
			c.onSegmentReady(ack)
		}

		return c.state.Transition(EventReceiveFin)
	}

	return nil
}

// handleSegmentFinWait1 handles segments in FIN_WAIT_1 state.
func (c *Connection) handleSegmentFinWait1(seg *Segment) error {
	// Process ACK
	if seg.HasFlag(FlagACK) {
		c.processAck(seg)

		// Check if FIN was ACKed
		if seg.AckNumber > c.sndUna {
			c.retransmitQueue.Remove(c.sndNxt - 1)
			c.sndUna = seg.AckNumber
		}
	}

	// Process FIN
	if seg.HasFlag(FlagFIN) {
		c.rcvNxt = seg.SequenceNumber + uint32(len(seg.Data)) + 1

		// Send ACK for FIN
		ack := NewSegment(c.LocalPort, c.RemotePort, c.sndNxt, c.rcvNxt, FlagACK, c.rcvWnd, nil)
		checksum, _ := ack.CalculateChecksum(c.LocalAddr, c.RemoteAddr)
		ack.Checksum = checksum

		if c.onSegmentReady != nil {
			c.onSegmentReady(ack)
		}

		if seg.HasFlag(FlagACK) {
			return c.state.Transition(EventReceiveFinAck)
		}
		return c.state.Transition(EventReceiveFin)
	}

	if seg.HasFlag(FlagACK) && !seg.HasFlag(FlagFIN) {
		return c.state.Transition(EventReceiveAck)
	}

	return nil
}

// handleSegmentFinWait2 handles segments in FIN_WAIT_2 state.
func (c *Connection) handleSegmentFinWait2(seg *Segment) error {
	if seg.HasFlag(FlagFIN) {
		c.rcvNxt = seg.SequenceNumber + uint32(len(seg.Data)) + 1

		// Send ACK for FIN
		ack := NewSegment(c.LocalPort, c.RemotePort, c.sndNxt, c.rcvNxt, FlagACK, c.rcvWnd, nil)
		checksum, _ := ack.CalculateChecksum(c.LocalAddr, c.RemoteAddr)
		ack.Checksum = checksum

		if c.onSegmentReady != nil {
			c.onSegmentReady(ack)
		}

		// Start TIME_WAIT timer (2 * MSL)
		c.startTimeWaitTimer()

		return c.state.Transition(EventReceiveFin)
	}

	return nil
}

// handleSegmentCloseWait handles segments in CLOSE_WAIT state.
func (c *Connection) handleSegmentCloseWait(seg *Segment) error {
	// Can still send data, just process ACKs
	if seg.HasFlag(FlagACK) {
		c.processAck(seg)
	}
	return nil
}

// handleSegmentClosing handles segments in CLOSING state.
func (c *Connection) handleSegmentClosing(seg *Segment) error {
	if seg.HasFlag(FlagACK) {
		c.processAck(seg)

		// Start TIME_WAIT timer
		c.startTimeWaitTimer()

		return c.state.Transition(EventReceiveAck)
	}
	return nil
}

// handleSegmentLastAck handles segments in LAST_ACK state.
func (c *Connection) handleSegmentLastAck(seg *Segment) error {
	if seg.HasFlag(FlagACK) {
		c.processAck(seg)

		if err := c.state.Transition(EventReceiveAck); err != nil {
			return err
		}

		// Connection is closed
		if c.onClose != nil {
			c.onClose()
		}

		return nil
	}
	return nil
}

// handleSegmentTimeWait handles segments in TIME_WAIT state.
func (c *Connection) handleSegmentTimeWait(seg *Segment) error {
	// If we receive a FIN, re-ACK it and restart timer
	if seg.HasFlag(FlagFIN) {
		ack := NewSegment(c.LocalPort, c.RemotePort, c.sndNxt, c.rcvNxt, FlagACK, c.rcvWnd, nil)
		checksum, _ := ack.CalculateChecksum(c.LocalAddr, c.RemoteAddr)
		ack.Checksum = checksum

		if c.onSegmentReady != nil {
			c.onSegmentReady(ack)
		}

		c.startTimeWaitTimer()
	}

	return nil
}

// processAck processes an ACK segment.
func (c *Connection) processAck(seg *Segment) {
	// Update send window
	c.sndWnd = seg.WindowSize

	// Check if this ACKs new data
	if seg.AckNumber > c.sndUna {
		// New ACK received
		bytesAcked := seg.AckNumber - c.sndUna
		c.sndUna = seg.AckNumber

		// Remove ACKed segments from retransmit queue
		c.retransmitQueue.RemoveBefore(seg.AckNumber)

		// Update congestion window
		c.updateCongestionWindow(bytesAcked)

		// Reset duplicate ACK counter
		c.dupAckCnt = 0
	} else if seg.AckNumber == c.sndUna && len(seg.Data) == 0 {
		// Duplicate ACK
		c.dupAckCnt++

		// Fast retransmit on 3 duplicate ACKs
		if c.dupAckCnt == 3 {
			c.fastRetransmit()
		}
	}
}

// processData processes data in a segment.
func (c *Connection) processData(seg *Segment) {
	if seg.SequenceNumber == c.rcvNxt {
		// In-order data
		c.receiveBuffer.Write(seg.Data)
		c.rcvNxt += uint32(len(seg.Data))

		// Deliver data to application
		if c.onDataReady != nil {
			c.onDataReady(seg.Data)
		}

		// Send ACK
		ack := NewSegment(c.LocalPort, c.RemotePort, c.sndNxt, c.rcvNxt, FlagACK, c.rcvWnd, nil)
		checksum, _ := ack.CalculateChecksum(c.LocalAddr, c.RemoteAddr)
		ack.Checksum = checksum

		if c.onSegmentReady != nil {
			c.onSegmentReady(ack)
		}
	} else {
		// Out-of-order data - store in receive buffer
		// TODO: Implement out-of-order handling
	}
}

// Send sends data over the connection.
func (c *Connection) Send(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.state.GetState().CanSendData() {
		return fmt.Errorf("cannot send data in state %s", c.state.GetState())
	}

	// Add data to send buffer
	c.sendBuffer.Write(data)

	// Send as much as we can
	return c.sendData()
}

// sendData sends data from the send buffer.
func (c *Connection) sendData() error {
	for {
		// Check if we can send more data (window check)
		availableWindow := int(c.sndWnd) - int(c.sndNxt-c.sndUna)
		if availableWindow <= 0 {
			break
		}

		// Check congestion window
		if int(c.sndNxt-c.sndUna) >= int(c.cwnd) {
			break
		}

		// Read from send buffer
		data := c.sendBuffer.Read(int(c.mss))
		if len(data) == 0 {
			break
		}

		// Create segment
		seg := NewSegment(c.LocalPort, c.RemotePort, c.sndNxt, c.rcvNxt, FlagACK|FlagPSH, c.rcvWnd, data)
		checksum, err := seg.CalculateChecksum(c.LocalAddr, c.RemoteAddr)
		if err != nil {
			return err
		}
		seg.Checksum = checksum

		// Send segment
		if c.onSegmentReady != nil {
			if err := c.onSegmentReady(seg); err != nil {
				return err
			}
		}

		// Add to retransmit queue
		c.retransmitQueue.Add(c.sndNxt, seg, time.Now())

		// Update sequence number
		c.sndNxt += uint32(len(data))
	}

	return nil
}

// Close closes the connection.
func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	state := c.state.GetState()
	if state == StateClosed {
		return fmt.Errorf("connection already closed")
	}

	// Send FIN
	fin := NewSegment(c.LocalPort, c.RemotePort, c.sndNxt, c.rcvNxt, FlagFIN|FlagACK, c.rcvWnd, nil)
	checksum, err := fin.CalculateChecksum(c.LocalAddr, c.RemoteAddr)
	if err != nil {
		return err
	}
	fin.Checksum = checksum

	if c.onSegmentReady != nil {
		if err := c.onSegmentReady(fin); err != nil {
			return err
		}
	}

	// Add FIN to retransmit queue
	c.retransmitQueue.Add(c.sndNxt, fin, time.Now())
	c.sndNxt++

	// Transition state
	return c.state.Transition(EventClose)
}

// generateISN generates a random initial sequence number.
func (c *Connection) generateISN() uint32 {
	var isn [4]byte
	rand.Read(isn[:])
	return binary.BigEndian.Uint32(isn[:])
}

// startTimeWaitTimer starts the TIME_WAIT timer (2 * MSL).
func (c *Connection) startTimeWaitTimer() {
	if c.timeWaitTimer != nil {
		c.timeWaitTimer.Stop()
	}

	c.timeWaitTimer = time.AfterFunc(2*time.Minute, func() {
		c.mu.Lock()
		defer c.mu.Unlock()

		c.state.Transition(EventTimeout)

		if c.onClose != nil {
			c.onClose()
		}
	})
}

// updateCongestionWindow updates the congestion window.
func (c *Connection) updateCongestionWindow(bytesAcked uint32) {
	if c.cwnd < c.ssthresh {
		// Slow start: exponential growth
		c.cwnd += bytesAcked
	} else {
		// Congestion avoidance: linear growth
		c.cwnd += uint32(c.mss) * bytesAcked / c.cwnd
	}
}

// fastRetransmit performs fast retransmit.
func (c *Connection) fastRetransmit() {
	// Retransmit the first unacknowledged segment
	if seg := c.retransmitQueue.GetFirst(); seg != nil {
		if c.onSegmentReady != nil {
			c.onSegmentReady(seg)
		}
	}

	// Fast recovery: set ssthresh and cwnd
	c.ssthresh = c.cwnd / 2
	if c.ssthresh < uint32(c.mss)*2 {
		c.ssthresh = uint32(c.mss) * 2
	}
	c.cwnd = c.ssthresh
}
