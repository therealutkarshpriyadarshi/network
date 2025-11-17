// Package tcp implements TCP send buffer management.
package tcp

import (
	"sync"
)

// SendBuffer manages the send buffer for a TCP connection.
type SendBuffer struct {
	buffer []byte
	mu     sync.Mutex
}

// NewSendBuffer creates a new send buffer.
func NewSendBuffer() *SendBuffer {
	return &SendBuffer{
		buffer: make([]byte, 0),
	}
}

// Write adds data to the send buffer.
func (sb *SendBuffer) Write(data []byte) int {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	sb.buffer = append(sb.buffer, data...)
	return len(data)
}

// Read reads up to n bytes from the send buffer.
// Returns the data read (may be less than n if buffer has less data).
func (sb *SendBuffer) Read(n int) []byte {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	if len(sb.buffer) == 0 {
		return nil
	}

	if n > len(sb.buffer) {
		n = len(sb.buffer)
	}

	data := make([]byte, n)
	copy(data, sb.buffer[:n])
	sb.buffer = sb.buffer[n:]

	return data
}

// Peek reads up to n bytes from the send buffer without removing them.
func (sb *SendBuffer) Peek(n int) []byte {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	if len(sb.buffer) == 0 {
		return nil
	}

	if n > len(sb.buffer) {
		n = len(sb.buffer)
	}

	data := make([]byte, n)
	copy(data, sb.buffer[:n])

	return data
}

// Len returns the number of bytes in the send buffer.
func (sb *SendBuffer) Len() int {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	return len(sb.buffer)
}

// Clear clears the send buffer.
func (sb *SendBuffer) Clear() {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	sb.buffer = sb.buffer[:0]
}
