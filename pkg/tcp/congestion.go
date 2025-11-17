// Package tcp implements TCP congestion control algorithms.
package tcp

import (
	"time"
)

// CongestionState represents the congestion control state.
type CongestionState int

const (
	// SlowStart is the slow start phase.
	SlowStart CongestionState = iota

	// CongestionAvoidance is the congestion avoidance phase.
	CongestionAvoidance

	// FastRecovery is the fast recovery phase.
	FastRecovery
)

// String returns the string representation of the congestion state.
func (cs CongestionState) String() string {
	switch cs {
	case SlowStart:
		return "SLOW_START"
	case CongestionAvoidance:
		return "CONGESTION_AVOIDANCE"
	case FastRecovery:
		return "FAST_RECOVERY"
	default:
		return "UNKNOWN"
	}
}

// CongestionControl manages TCP congestion control.
type CongestionControl struct {
	// Congestion window (in bytes)
	cwnd uint32

	// Slow start threshold (in bytes)
	ssthresh uint32

	// Current state
	state CongestionState

	// MSS (Maximum Segment Size)
	mss uint16

	// Duplicate ACK count
	dupAckCount int

	// Recovery sequence number (for fast recovery)
	recoverySeq uint32
}

// NewCongestionControl creates a new congestion control manager.
func NewCongestionControl(mss uint16) *CongestionControl {
	return &CongestionControl{
		cwnd:        uint32(mss) * 2, // Initial cwnd = 2 * MSS (RFC 5681)
		ssthresh:    65535,            // Initial ssthresh = max window
		state:       SlowStart,
		mss:         mss,
		dupAckCount: 0,
	}
}

// GetCwnd returns the current congestion window size.
func (cc *CongestionControl) GetCwnd() uint32 {
	return cc.cwnd
}

// GetSsthresh returns the current slow start threshold.
func (cc *CongestionControl) GetSsthresh() uint32 {
	return cc.ssthresh
}

// GetState returns the current congestion control state.
func (cc *CongestionControl) GetState() CongestionState {
	return cc.state
}

// OnAck is called when a new ACK is received.
func (cc *CongestionControl) OnAck(bytesAcked uint32, seqNum uint32) {
	switch cc.state {
	case SlowStart:
		cc.onAckSlowStart(bytesAcked)
	case CongestionAvoidance:
		cc.onAckCongestionAvoidance(bytesAcked)
	case FastRecovery:
		cc.onAckFastRecovery(bytesAcked, seqNum)
	}

	// Reset duplicate ACK count on new ACK
	cc.dupAckCount = 0
}

// OnDuplicateAck is called when a duplicate ACK is received.
func (cc *CongestionControl) OnDuplicateAck(seqNum uint32) bool {
	cc.dupAckCount++

	// Fast retransmit on 3 duplicate ACKs
	if cc.dupAckCount == 3 {
		cc.onFastRetransmit(seqNum)
		return true // Indicate that fast retransmit should occur
	}

	// Increment cwnd during fast recovery
	if cc.state == FastRecovery {
		cc.cwnd += uint32(cc.mss)
	}

	return false
}

// OnTimeout is called when a retransmission timeout occurs.
func (cc *CongestionControl) OnTimeout() {
	// Set ssthresh to half of current cwnd
	cc.ssthresh = cc.cwnd / 2
	if cc.ssthresh < uint32(cc.mss)*2 {
		cc.ssthresh = uint32(cc.mss) * 2
	}

	// Reset cwnd to 1 MSS (conservative restart)
	cc.cwnd = uint32(cc.mss)

	// Enter slow start
	cc.state = SlowStart
	cc.dupAckCount = 0
}

// onAckSlowStart handles ACKs during slow start.
func (cc *CongestionControl) onAckSlowStart(bytesAcked uint32) {
	// Exponential growth: increase cwnd by bytes ACKed
	cc.cwnd += bytesAcked

	// Transition to congestion avoidance when cwnd >= ssthresh
	if cc.cwnd >= cc.ssthresh {
		cc.state = CongestionAvoidance
	}
}

// onAckCongestionAvoidance handles ACKs during congestion avoidance.
func (cc *CongestionControl) onAckCongestionAvoidance(bytesAcked uint32) {
	// Linear growth: increase cwnd by (MSS * MSS) / cwnd per ACK
	// This results in approximately 1 MSS increase per RTT
	increment := (uint32(cc.mss) * uint32(cc.mss)) / cc.cwnd
	if increment == 0 {
		increment = 1
	}
	cc.cwnd += increment
}

// onAckFastRecovery handles ACKs during fast recovery.
func (cc *CongestionControl) onAckFastRecovery(bytesAcked uint32, seqNum uint32) {
	// Check if this ACK acknowledges data beyond the recovery point
	if seqAfter(seqNum, cc.recoverySeq) {
		// Exit fast recovery
		cc.cwnd = cc.ssthresh
		cc.state = CongestionAvoidance
		cc.dupAckCount = 0
	}
}

// onFastRetransmit handles fast retransmit.
func (cc *CongestionControl) onFastRetransmit(seqNum uint32) {
	// Set ssthresh to half of current cwnd
	cc.ssthresh = cc.cwnd / 2
	if cc.ssthresh < uint32(cc.mss)*2 {
		cc.ssthresh = uint32(cc.mss) * 2
	}

	// Set cwnd to ssthresh + 3 * MSS (for the 3 duplicate ACKs)
	cc.cwnd = cc.ssthresh + uint32(cc.mss)*3

	// Enter fast recovery
	cc.state = FastRecovery
	cc.recoverySeq = seqNum
}

// CanSend returns true if we can send data based on the congestion window.
func (cc *CongestionControl) CanSend(bytesInFlight uint32) bool {
	return bytesInFlight < cc.cwnd
}

// RTTEstimator manages RTT estimation and RTO calculation.
type RTTEstimator struct {
	srtt   time.Duration // Smoothed RTT
	rttvar time.Duration // RTT variance
	rto    time.Duration // Retransmission timeout

	alpha float64 // SRTT smoothing factor (1/8)
	beta  float64 // RTTVAR smoothing factor (1/4)

	minRTO time.Duration // Minimum RTO (1 second per RFC 6298)
	maxRTO time.Duration // Maximum RTO (60 seconds)
}

// NewRTTEstimator creates a new RTT estimator.
func NewRTTEstimator() *RTTEstimator {
	return &RTTEstimator{
		srtt:   0,
		rttvar: 0,
		rto:    time.Second, // Initial RTO = 1 second
		alpha:  1.0 / 8.0,
		beta:   1.0 / 4.0,
		minRTO: time.Second,
		maxRTO: 60 * time.Second,
	}
}

// UpdateRTT updates the RTT estimate with a new measurement.
func (re *RTTEstimator) UpdateRTT(measuredRTT time.Duration) {
	if re.srtt == 0 {
		// First RTT measurement
		re.srtt = measuredRTT
		re.rttvar = measuredRTT / 2
	} else {
		// Subsequent measurements (RFC 6298)
		diff := re.srtt - measuredRTT
		if diff < 0 {
			diff = -diff
		}

		re.rttvar = time.Duration(float64(re.rttvar)*(1-re.beta) + float64(diff)*re.beta)
		re.srtt = time.Duration(float64(re.srtt)*(1-re.alpha) + float64(measuredRTT)*re.alpha)
	}

	// Calculate RTO = SRTT + 4 * RTTVAR
	re.rto = re.srtt + 4*re.rttvar

	// Clamp RTO to [minRTO, maxRTO]
	if re.rto < re.minRTO {
		re.rto = re.minRTO
	}
	if re.rto > re.maxRTO {
		re.rto = re.maxRTO
	}
}

// GetRTO returns the current retransmission timeout.
func (re *RTTEstimator) GetRTO() time.Duration {
	return re.rto
}

// BackoffRTO backs off the RTO (doubles it) for exponential backoff.
func (re *RTTEstimator) BackoffRTO() {
	re.rto *= 2
	if re.rto > re.maxRTO {
		re.rto = re.maxRTO
	}
}

// GetSRTT returns the smoothed RTT.
func (re *RTTEstimator) GetSRTT() time.Duration {
	return re.srtt
}
