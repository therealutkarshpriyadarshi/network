// Package tcp implements TCP receive buffer management.
package tcp

import (
	"sync"
)

// ReceiveBuffer manages the receive buffer for a TCP connection.
type ReceiveBuffer struct {
	buffer   []byte
	capacity int
	mu       sync.Mutex
}

// NewReceiveBuffer creates a new receive buffer with the given capacity.
func NewReceiveBuffer(capacity int) *ReceiveBuffer {
	return &ReceiveBuffer{
		buffer:   make([]byte, 0, capacity),
		capacity: capacity,
	}
}

// Write adds data to the receive buffer.
// Returns the number of bytes written (may be less than len(data) if buffer is full).
func (rb *ReceiveBuffer) Write(data []byte) int {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	available := rb.capacity - len(rb.buffer)
	if available <= 0 {
		return 0
	}

	toWrite := len(data)
	if toWrite > available {
		toWrite = available
	}

	rb.buffer = append(rb.buffer, data[:toWrite]...)
	return toWrite
}

// Read reads up to n bytes from the receive buffer.
// Returns the data read (may be less than n if buffer has less data).
func (rb *ReceiveBuffer) Read(n int) []byte {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if len(rb.buffer) == 0 {
		return nil
	}

	if n > len(rb.buffer) {
		n = len(rb.buffer)
	}

	data := make([]byte, n)
	copy(data, rb.buffer[:n])
	rb.buffer = rb.buffer[n:]

	return data
}

// Peek reads up to n bytes from the receive buffer without removing them.
func (rb *ReceiveBuffer) Peek(n int) []byte {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	if len(rb.buffer) == 0 {
		return nil
	}

	if n > len(rb.buffer) {
		n = len(rb.buffer)
	}

	data := make([]byte, n)
	copy(data, rb.buffer[:n])

	return data
}

// Len returns the number of bytes in the receive buffer.
func (rb *ReceiveBuffer) Len() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	return len(rb.buffer)
}

// Available returns the number of bytes available in the receive buffer.
func (rb *ReceiveBuffer) Available() int {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	return rb.capacity - len(rb.buffer)
}

// Clear clears the receive buffer.
func (rb *ReceiveBuffer) Clear() {
	rb.mu.Lock()
	defer rb.mu.Unlock()

	rb.buffer = rb.buffer[:0]
}
