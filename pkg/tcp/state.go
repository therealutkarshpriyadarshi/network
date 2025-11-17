// Package tcp implements TCP state machine as defined in RFC 793.
package tcp

import "fmt"

// State represents the TCP connection state.
type State int

const (
	// CLOSED represents a connection that doesn't exist.
	StateClosed State = iota

	// LISTEN represents waiting for a connection request from any remote TCP.
	StateListen

	// SYN_SENT represents waiting for a matching connection request
	// after having sent a connection request.
	StateSynSent

	// SYN_RECEIVED represents waiting for a confirming connection
	// request acknowledgment after having both received and sent a
	// connection request.
	StateSynReceived

	// ESTABLISHED represents an open connection, data received can be
	// delivered to the user. This is the normal state for data transfer.
	StateEstablished

	// FIN_WAIT_1 represents waiting for a connection termination request
	// from the remote TCP, or an acknowledgment of the connection
	// termination request previously sent.
	StateFinWait1

	// FIN_WAIT_2 represents waiting for a connection termination request
	// from the remote TCP.
	StateFinWait2

	// CLOSE_WAIT represents waiting for a connection termination request
	// from the local user.
	StateCloseWait

	// CLOSING represents waiting for a connection termination request
	// acknowledgment from the remote TCP.
	StateClosing

	// LAST_ACK represents waiting for an acknowledgment of the connection
	// termination request previously sent to the remote TCP.
	StateLastAck

	// TIME_WAIT represents waiting for enough time to pass to be sure
	// the remote TCP received the acknowledgment of its connection
	// termination request.
	StateTimeWait
)

// String returns the string representation of the state.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "CLOSED"
	case StateListen:
		return "LISTEN"
	case StateSynSent:
		return "SYN_SENT"
	case StateSynReceived:
		return "SYN_RECEIVED"
	case StateEstablished:
		return "ESTABLISHED"
	case StateFinWait1:
		return "FIN_WAIT_1"
	case StateFinWait2:
		return "FIN_WAIT_2"
	case StateCloseWait:
		return "CLOSE_WAIT"
	case StateClosing:
		return "CLOSING"
	case StateLastAck:
		return "LAST_ACK"
	case StateTimeWait:
		return "TIME_WAIT"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", int(s))
	}
}

// IsConnectionEstablished returns true if the state represents an established connection.
func (s State) IsConnectionEstablished() bool {
	return s == StateEstablished || s == StateFinWait1 || s == StateFinWait2 ||
		s == StateCloseWait || s == StateClosing || s == StateLastAck
}

// CanSendData returns true if the state allows sending data.
func (s State) CanSendData() bool {
	return s == StateEstablished || s == StateCloseWait
}

// CanReceiveData returns true if the state allows receiving data.
func (s State) CanReceiveData() bool {
	return s == StateEstablished || s == StateFinWait1 || s == StateFinWait2
}

// Event represents an event that can trigger a state transition.
type Event int

const (
	// EventPassiveOpen represents a passive open (server).
	EventPassiveOpen Event = iota

	// EventActiveOpen represents an active open (client).
	EventActiveOpen

	// EventSend represents sending data.
	EventSend

	// EventReceiveSyn represents receiving a SYN segment.
	EventReceiveSyn

	// EventReceiveSynAck represents receiving a SYN+ACK segment.
	EventReceiveSynAck

	// EventReceiveAck represents receiving an ACK segment.
	EventReceiveAck

	// EventReceiveFin represents receiving a FIN segment.
	EventReceiveFin

	// EventReceiveFinAck represents receiving a FIN+ACK segment.
	EventReceiveFinAck

	// EventClose represents a close request from the application.
	EventClose

	// EventTimeout represents a timeout.
	EventTimeout
)

// String returns the string representation of the event.
func (e Event) String() string {
	switch e {
	case EventPassiveOpen:
		return "PASSIVE_OPEN"
	case EventActiveOpen:
		return "ACTIVE_OPEN"
	case EventSend:
		return "SEND"
	case EventReceiveSyn:
		return "RECEIVE_SYN"
	case EventReceiveSynAck:
		return "RECEIVE_SYN_ACK"
	case EventReceiveAck:
		return "RECEIVE_ACK"
	case EventReceiveFin:
		return "RECEIVE_FIN"
	case EventReceiveFinAck:
		return "RECEIVE_FIN_ACK"
	case EventClose:
		return "CLOSE"
	case EventTimeout:
		return "TIMEOUT"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", int(e))
	}
}

// StateMachine manages TCP state transitions.
type StateMachine struct {
	state State
}

// NewStateMachine creates a new TCP state machine.
func NewStateMachine() *StateMachine {
	return &StateMachine{
		state: StateClosed,
	}
}

// GetState returns the current state.
func (sm *StateMachine) GetState() State {
	return sm.state
}

// Transition attempts to transition to a new state based on an event.
// Returns an error if the transition is not valid.
func (sm *StateMachine) Transition(event Event) error {
	newState, err := sm.nextState(event)
	if err != nil {
		return err
	}

	sm.state = newState
	return nil
}

// SetState directly sets the state (use with caution).
func (sm *StateMachine) SetState(state State) {
	sm.state = state
}

// nextState determines the next state based on current state and event.
func (sm *StateMachine) nextState(event Event) (State, error) {
	switch sm.state {
	case StateClosed:
		switch event {
		case EventPassiveOpen:
			return StateListen, nil
		case EventActiveOpen:
			return StateSynSent, nil
		default:
			return sm.state, fmt.Errorf("invalid event %s for state %s", event, sm.state)
		}

	case StateListen:
		switch event {
		case EventReceiveSyn:
			return StateSynReceived, nil
		case EventActiveOpen:
			return StateSynSent, nil
		case EventClose:
			return StateClosed, nil
		default:
			return sm.state, fmt.Errorf("invalid event %s for state %s", event, sm.state)
		}

	case StateSynSent:
		switch event {
		case EventReceiveSynAck:
			return StateEstablished, nil
		case EventReceiveSyn:
			return StateSynReceived, nil
		case EventClose:
			return StateClosed, nil
		default:
			return sm.state, fmt.Errorf("invalid event %s for state %s", event, sm.state)
		}

	case StateSynReceived:
		switch event {
		case EventReceiveAck:
			return StateEstablished, nil
		case EventClose:
			return StateFinWait1, nil
		case EventReceiveFin:
			return StateCloseWait, nil
		default:
			return sm.state, fmt.Errorf("invalid event %s for state %s", event, sm.state)
		}

	case StateEstablished:
		switch event {
		case EventClose:
			return StateFinWait1, nil
		case EventReceiveFin:
			return StateCloseWait, nil
		default:
			// Sending and receiving data doesn't change state
			return sm.state, nil
		}

	case StateFinWait1:
		switch event {
		case EventReceiveAck:
			return StateFinWait2, nil
		case EventReceiveFin:
			return StateClosing, nil
		case EventReceiveFinAck:
			return StateTimeWait, nil
		default:
			return sm.state, fmt.Errorf("invalid event %s for state %s", event, sm.state)
		}

	case StateFinWait2:
		switch event {
		case EventReceiveFin:
			return StateTimeWait, nil
		default:
			return sm.state, fmt.Errorf("invalid event %s for state %s", event, sm.state)
		}

	case StateCloseWait:
		switch event {
		case EventClose:
			return StateLastAck, nil
		default:
			// Can still send data
			return sm.state, nil
		}

	case StateClosing:
		switch event {
		case EventReceiveAck:
			return StateTimeWait, nil
		default:
			return sm.state, fmt.Errorf("invalid event %s for state %s", event, sm.state)
		}

	case StateLastAck:
		switch event {
		case EventReceiveAck:
			return StateClosed, nil
		default:
			return sm.state, fmt.Errorf("invalid event %s for state %s", event, sm.state)
		}

	case StateTimeWait:
		switch event {
		case EventTimeout:
			return StateClosed, nil
		default:
			// Stay in TIME_WAIT until timeout
			return sm.state, nil
		}

	default:
		return sm.state, fmt.Errorf("unknown state %s", sm.state)
	}
}
