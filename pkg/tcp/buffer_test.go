package tcp

import (
	"testing"
)

func TestSendBuffer(t *testing.T) {
	sb := NewSendBuffer()

	// Test write
	data := []byte("Hello, World!")
	n := sb.Write(data)
	if n != len(data) {
		t.Errorf("Write() = %d, want %d", n, len(data))
	}

	// Test length
	if sb.Len() != len(data) {
		t.Errorf("Len() = %d, want %d", sb.Len(), len(data))
	}

	// Test read
	read := sb.Read(5)
	if string(read) != "Hello" {
		t.Errorf("Read() = %s, want Hello", read)
	}

	if sb.Len() != len(data)-5 {
		t.Errorf("Len() after read = %d, want %d", sb.Len(), len(data)-5)
	}

	// Test peek
	peeked := sb.Peek(3)
	if string(peeked) != ", W" {
		t.Errorf("Peek() = %s, want ', W'", peeked)
	}

	// Peek should not remove data
	if sb.Len() != len(data)-5 {
		t.Errorf("Len() after peek = %d, want %d", sb.Len(), len(data)-5)
	}

	// Test clear
	sb.Clear()
	if sb.Len() != 0 {
		t.Errorf("Len() after clear = %d, want 0", sb.Len())
	}
}

func TestReceiveBuffer(t *testing.T) {
	capacity := 1024
	rb := NewReceiveBuffer(capacity)

	// Test write
	data := []byte("Hello, World!")
	n := rb.Write(data)
	if n != len(data) {
		t.Errorf("Write() = %d, want %d", n, len(data))
	}

	// Test length
	if rb.Len() != len(data) {
		t.Errorf("Len() = %d, want %d", rb.Len(), len(data))
	}

	// Test available
	if rb.Available() != capacity-len(data) {
		t.Errorf("Available() = %d, want %d", rb.Available(), capacity-len(data))
	}

	// Test read
	read := rb.Read(5)
	if string(read) != "Hello" {
		t.Errorf("Read() = %s, want Hello", read)
	}

	if rb.Len() != len(data)-5 {
		t.Errorf("Len() after read = %d, want %d", rb.Len(), len(data)-5)
	}

	// Test peek
	peeked := rb.Peek(3)
	if string(peeked) != ", W" {
		t.Errorf("Peek() = %s, want ', W'", peeked)
	}

	// Peek should not remove data
	if rb.Len() != len(data)-5 {
		t.Errorf("Len() after peek = %d, want %d", rb.Len(), len(data)-5)
	}

	// Test clear
	rb.Clear()
	if rb.Len() != 0 {
		t.Errorf("Len() after clear = %d, want 0", rb.Len())
	}

	if rb.Available() != capacity {
		t.Errorf("Available() after clear = %d, want %d", rb.Available(), capacity)
	}
}

func TestReceiveBufferOverflow(t *testing.T) {
	capacity := 10
	rb := NewReceiveBuffer(capacity)

	// Write more data than capacity
	data := []byte("This is more than 10 bytes")
	n := rb.Write(data)

	if n != capacity {
		t.Errorf("Write() = %d, want %d (capacity)", n, capacity)
	}

	if rb.Len() != capacity {
		t.Errorf("Len() = %d, want %d", rb.Len(), capacity)
	}

	if rb.Available() != 0 {
		t.Errorf("Available() = %d, want 0", rb.Available())
	}
}
