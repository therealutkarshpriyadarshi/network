package tcp

import (
	"testing"
)

func TestStateMachineTransitions(t *testing.T) {
	tests := []struct {
		name        string
		initialState State
		event       Event
		expectedState State
		expectError bool
	}{
		// CLOSED state transitions
		{
			name:          "CLOSED -> LISTEN (passive open)",
			initialState:  StateClosed,
			event:         EventPassiveOpen,
			expectedState: StateListen,
			expectError:   false,
		},
		{
			name:          "CLOSED -> SYN_SENT (active open)",
			initialState:  StateClosed,
			event:         EventActiveOpen,
			expectedState: StateSynSent,
			expectError:   false,
		},
		// LISTEN state transitions
		{
			name:          "LISTEN -> SYN_RECEIVED (receive SYN)",
			initialState:  StateListen,
			event:         EventReceiveSyn,
			expectedState: StateSynReceived,
			expectError:   false,
		},
		{
			name:          "LISTEN -> CLOSED (close)",
			initialState:  StateListen,
			event:         EventClose,
			expectedState: StateClosed,
			expectError:   false,
		},
		// SYN_SENT state transitions
		{
			name:          "SYN_SENT -> ESTABLISHED (receive SYN+ACK)",
			initialState:  StateSynSent,
			event:         EventReceiveSynAck,
			expectedState: StateEstablished,
			expectError:   false,
		},
		// SYN_RECEIVED state transitions
		{
			name:          "SYN_RECEIVED -> ESTABLISHED (receive ACK)",
			initialState:  StateSynReceived,
			event:         EventReceiveAck,
			expectedState: StateEstablished,
			expectError:   false,
		},
		// ESTABLISHED state transitions
		{
			name:          "ESTABLISHED -> FIN_WAIT_1 (close)",
			initialState:  StateEstablished,
			event:         EventClose,
			expectedState: StateFinWait1,
			expectError:   false,
		},
		{
			name:          "ESTABLISHED -> CLOSE_WAIT (receive FIN)",
			initialState:  StateEstablished,
			event:         EventReceiveFin,
			expectedState: StateCloseWait,
			expectError:   false,
		},
		// FIN_WAIT_1 state transitions
		{
			name:          "FIN_WAIT_1 -> FIN_WAIT_2 (receive ACK)",
			initialState:  StateFinWait1,
			event:         EventReceiveAck,
			expectedState: StateFinWait2,
			expectError:   false,
		},
		{
			name:          "FIN_WAIT_1 -> CLOSING (receive FIN)",
			initialState:  StateFinWait1,
			event:         EventReceiveFin,
			expectedState: StateClosing,
			expectError:   false,
		},
		{
			name:          "FIN_WAIT_1 -> TIME_WAIT (receive FIN+ACK)",
			initialState:  StateFinWait1,
			event:         EventReceiveFinAck,
			expectedState: StateTimeWait,
			expectError:   false,
		},
		// FIN_WAIT_2 state transitions
		{
			name:          "FIN_WAIT_2 -> TIME_WAIT (receive FIN)",
			initialState:  StateFinWait2,
			event:         EventReceiveFin,
			expectedState: StateTimeWait,
			expectError:   false,
		},
		// CLOSE_WAIT state transitions
		{
			name:          "CLOSE_WAIT -> LAST_ACK (close)",
			initialState:  StateCloseWait,
			event:         EventClose,
			expectedState: StateLastAck,
			expectError:   false,
		},
		// CLOSING state transitions
		{
			name:          "CLOSING -> TIME_WAIT (receive ACK)",
			initialState:  StateClosing,
			event:         EventReceiveAck,
			expectedState: StateTimeWait,
			expectError:   false,
		},
		// LAST_ACK state transitions
		{
			name:          "LAST_ACK -> CLOSED (receive ACK)",
			initialState:  StateLastAck,
			event:         EventReceiveAck,
			expectedState: StateClosed,
			expectError:   false,
		},
		// TIME_WAIT state transitions
		{
			name:          "TIME_WAIT -> CLOSED (timeout)",
			initialState:  StateTimeWait,
			event:         EventTimeout,
			expectedState: StateClosed,
			expectError:   false,
		},
		// Invalid transitions
		{
			name:          "CLOSED -> invalid event",
			initialState:  StateClosed,
			event:         EventReceiveFin,
			expectedState: StateClosed,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := NewStateMachine()
			sm.SetState(tt.initialState)

			err := sm.Transition(tt.event)

			if (err != nil) != tt.expectError {
				t.Fatalf("Transition() error = %v, expectError %v", err, tt.expectError)
			}

			if !tt.expectError {
				if sm.GetState() != tt.expectedState {
					t.Errorf("State = %s, want %s", sm.GetState(), tt.expectedState)
				}
			}
		})
	}
}

func TestStateHelpers(t *testing.T) {
	tests := []struct {
		state               State
		isEstablished       bool
		canSendData         bool
		canReceiveData      bool
	}{
		{StateClosed, false, false, false},
		{StateListen, false, false, false},
		{StateSynSent, false, false, false},
		{StateSynReceived, false, false, false},
		{StateEstablished, true, true, true},
		{StateFinWait1, true, false, true},
		{StateFinWait2, true, false, true},
		{StateCloseWait, true, true, false},
		{StateClosing, true, false, false},
		{StateLastAck, true, false, false},
		{StateTimeWait, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.state.String(), func(t *testing.T) {
			if tt.state.IsConnectionEstablished() != tt.isEstablished {
				t.Errorf("IsConnectionEstablished() = %v, want %v", tt.state.IsConnectionEstablished(), tt.isEstablished)
			}
			if tt.state.CanSendData() != tt.canSendData {
				t.Errorf("CanSendData() = %v, want %v", tt.state.CanSendData(), tt.canSendData)
			}
			if tt.state.CanReceiveData() != tt.canReceiveData {
				t.Errorf("CanReceiveData() = %v, want %v", tt.state.CanReceiveData(), tt.canReceiveData)
			}
		})
	}
}

func TestStateString(t *testing.T) {
	states := []State{
		StateClosed, StateListen, StateSynSent, StateSynReceived,
		StateEstablished, StateFinWait1, StateFinWait2, StateCloseWait,
		StateClosing, StateLastAck, StateTimeWait,
	}

	for _, state := range states {
		str := state.String()
		if str == "" {
			t.Errorf("String() for state %d returned empty string", state)
		}
	}
}

func TestEventString(t *testing.T) {
	events := []Event{
		EventPassiveOpen, EventActiveOpen, EventSend, EventReceiveSyn,
		EventReceiveSynAck, EventReceiveAck, EventReceiveFin,
		EventReceiveFinAck, EventClose, EventTimeout,
	}

	for _, event := range events {
		str := event.String()
		if str == "" {
			t.Errorf("String() for event %d returned empty string", event)
		}
	}
}
