package udp

import (
	"testing"
	"time"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

func TestNewSocket(t *testing.T) {
	s := NewSocket()
	if s == nil {
		t.Fatal("NewSocket() returned nil")
	}
	if s.bound {
		t.Error("New socket should not be bound")
	}
	if s.closed {
		t.Error("New socket should not be closed")
	}
}

func TestSocketBind(t *testing.T) {
	tests := []struct {
		name    string
		addr    Address
		wantErr bool
	}{
		{
			name: "bind to specific port",
			addr: Address{
				IP:   common.IPv4Address{192, 168, 1, 100},
				Port: 8080,
			},
			wantErr: false,
		},
		{
			name: "bind to ephemeral port",
			addr: Address{
				IP:   common.IPv4Address{192, 168, 1, 100},
				Port: 0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSocket()
			err := s.Bind(tt.addr)
			if (err != nil) != tt.wantErr {
				t.Errorf("Bind() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if !s.IsBound() {
					t.Error("Socket should be bound after Bind()")
				}

				localAddr, err := s.LocalAddr()
				if err != nil {
					t.Errorf("LocalAddr() error = %v", err)
					return
				}

				if localAddr.IP != tt.addr.IP {
					t.Errorf("LocalAddr() IP = %v, want %v", localAddr.IP, tt.addr.IP)
				}
				if localAddr.Port != tt.addr.Port {
					t.Errorf("LocalAddr() Port = %v, want %v", localAddr.Port, tt.addr.Port)
				}
			}
		})
	}
}

func TestSocketBindTwice(t *testing.T) {
	s := NewSocket()
	addr := Address{
		IP:   common.IPv4Address{192, 168, 1, 100},
		Port: 8080,
	}

	// First bind should succeed
	err := s.Bind(addr)
	if err != nil {
		t.Fatalf("First Bind() error = %v", err)
	}

	// Second bind should fail
	err = s.Bind(addr)
	if err == nil {
		t.Error("Second Bind() should fail")
	}
}

func TestSocketSendTo(t *testing.T) {
	s := NewSocket()
	localAddr := Address{
		IP:   common.IPv4Address{192, 168, 1, 100},
		Port: 8080,
	}
	remoteAddr := Address{
		IP:   common.IPv4Address{192, 168, 1, 1},
		Port: 80,
	}
	data := []byte("Hello, UDP!")

	// SendTo should fail before binding
	_, err := s.SendTo(data, remoteAddr)
	if err == nil {
		t.Error("SendTo() should fail before binding")
	}

	// Bind socket
	err = s.Bind(localAddr)
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}

	// SendTo should succeed after binding
	pkt, err := s.SendTo(data, remoteAddr)
	if err != nil {
		t.Errorf("SendTo() error = %v", err)
		return
	}

	if pkt.SourcePort != localAddr.Port {
		t.Errorf("SendTo() SourcePort = %v, want %v", pkt.SourcePort, localAddr.Port)
	}
	if pkt.DestinationPort != remoteAddr.Port {
		t.Errorf("SendTo() DestinationPort = %v, want %v", pkt.DestinationPort, remoteAddr.Port)
	}
	if string(pkt.Data) != string(data) {
		t.Errorf("SendTo() Data = %v, want %v", string(pkt.Data), string(data))
	}
}

func TestSocketReceive(t *testing.T) {
	s := NewSocket()
	localAddr := Address{
		IP:   common.IPv4Address{192, 168, 1, 100},
		Port: 8080,
	}
	remoteAddr := Address{
		IP:   common.IPv4Address{192, 168, 1, 1},
		Port: 80,
	}
	data := []byte("Test message")

	// Bind socket
	err := s.Bind(localAddr)
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}

	// Simulate receiving a packet
	err = s.Receive(data, remoteAddr)
	if err != nil {
		t.Errorf("Receive() error = %v", err)
	}

	// Try to receive it
	recvData, from, err := s.RecvFrom(100 * time.Millisecond)
	if err != nil {
		t.Errorf("RecvFrom() error = %v", err)
		return
	}

	if string(recvData) != string(data) {
		t.Errorf("RecvFrom() data = %v, want %v", string(recvData), string(data))
	}
	if from.IP != remoteAddr.IP {
		t.Errorf("RecvFrom() from.IP = %v, want %v", from.IP, remoteAddr.IP)
	}
	if from.Port != remoteAddr.Port {
		t.Errorf("RecvFrom() from.Port = %v, want %v", from.Port, remoteAddr.Port)
	}
}

func TestSocketRecvFromTimeout(t *testing.T) {
	s := NewSocket()
	localAddr := Address{
		IP:   common.IPv4Address{192, 168, 1, 100},
		Port: 8080,
	}

	// Bind socket
	err := s.Bind(localAddr)
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}

	// Try to receive without any packets - should timeout
	_, _, err = s.RecvFrom(100 * time.Millisecond)
	if err == nil {
		t.Error("RecvFrom() should timeout when no packets available")
	}
}

func TestSocketClose(t *testing.T) {
	s := NewSocket()
	localAddr := Address{
		IP:   common.IPv4Address{192, 168, 1, 100},
		Port: 8080,
	}

	// Bind socket
	err := s.Bind(localAddr)
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}

	// Close socket
	err = s.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	if !s.IsClosed() {
		t.Error("Socket should be closed after Close()")
	}

	// Second close should fail
	err = s.Close()
	if err == nil {
		t.Error("Second Close() should fail")
	}

	// Operations should fail on closed socket
	_, err = s.SendTo([]byte("test"), Address{Port: 80})
	if err == nil {
		t.Error("SendTo() should fail on closed socket")
	}

	_, _, err = s.RecvFrom(100 * time.Millisecond)
	if err == nil {
		t.Error("RecvFrom() should fail on closed socket")
	}
}

func TestAddressString(t *testing.T) {
	addr := Address{
		IP:   common.IPv4Address{192, 168, 1, 100},
		Port: 8080,
	}
	expected := "192.168.1.100:8080"
	if addr.String() != expected {
		t.Errorf("Address.String() = %v, want %v", addr.String(), expected)
	}
}

func TestNewDemultiplexer(t *testing.T) {
	d := NewDemultiplexer()
	if d == nil {
		t.Fatal("NewDemultiplexer() returned nil")
	}
	if d.sockets == nil {
		t.Error("Demultiplexer sockets map should be initialized")
	}
}

func TestDemultiplexerBind(t *testing.T) {
	d := NewDemultiplexer()
	s := NewSocket()

	// Bind to specific port
	port, err := d.Bind(s, 8080)
	if err != nil {
		t.Errorf("Bind() error = %v", err)
	}
	if port != 8080 {
		t.Errorf("Bind() returned port %d, want 8080", port)
	}

	// Try to bind another socket to same port - should fail
	s2 := NewSocket()
	_, err = d.Bind(s2, 8080)
	if err == nil {
		t.Error("Bind() to already-bound port should fail")
	}

	// Bind to ephemeral port
	s3 := NewSocket()
	port, err = d.Bind(s3, 0)
	if err != nil {
		t.Errorf("Bind() to ephemeral port error = %v", err)
	}
	if port < EphemeralPortStart || port > EphemeralPortEnd {
		t.Errorf("Bind() returned port %d, want ephemeral port in range [%d, %d]",
			port, EphemeralPortStart, EphemeralPortEnd)
	}
}

func TestDemultiplexerUnbind(t *testing.T) {
	d := NewDemultiplexer()
	s := NewSocket()

	// Bind socket
	port, err := d.Bind(s, 8080)
	if err != nil {
		t.Fatalf("Bind() error = %v", err)
	}

	// Unbind socket
	err = d.Unbind(port)
	if err != nil {
		t.Errorf("Unbind() error = %v", err)
	}

	// Unbind again should fail
	err = d.Unbind(port)
	if err == nil {
		t.Error("Unbind() on unbound port should fail")
	}

	// Should be able to bind to the port again
	s2 := NewSocket()
	_, err = d.Bind(s2, 8080)
	if err != nil {
		t.Errorf("Bind() after Unbind() error = %v", err)
	}
}

func TestDemultiplexerDeliver(t *testing.T) {
	d := NewDemultiplexer()
	s := NewSocket()

	// Setup socket
	localAddr := Address{
		IP:   common.IPv4Address{192, 168, 1, 100},
		Port: 8080,
	}
	err := s.Bind(localAddr)
	if err != nil {
		t.Fatalf("Socket.Bind() error = %v", err)
	}

	// Bind to demultiplexer
	_, err = d.Bind(s, localAddr.Port)
	if err != nil {
		t.Fatalf("Demultiplexer.Bind() error = %v", err)
	}

	// Create packet
	remoteAddr := Address{
		IP:   common.IPv4Address{192, 168, 1, 1},
		Port: 12345,
	}
	data := []byte("Test data")
	pkt := NewPacket(remoteAddr.Port, localAddr.Port, data)

	// Deliver packet
	err = d.Deliver(pkt, remoteAddr)
	if err != nil {
		t.Errorf("Deliver() error = %v", err)
	}

	// Receive from socket
	recvData, from, err := s.RecvFrom(100 * time.Millisecond)
	if err != nil {
		t.Errorf("RecvFrom() error = %v", err)
		return
	}

	if string(recvData) != string(data) {
		t.Errorf("RecvFrom() data = %v, want %v", string(recvData), string(data))
	}
	if from != remoteAddr {
		t.Errorf("RecvFrom() from = %v, want %v", from, remoteAddr)
	}
}

func TestDemultiplexerDeliverNoSocket(t *testing.T) {
	d := NewDemultiplexer()

	// Create packet for port with no socket bound
	remoteAddr := Address{
		IP:   common.IPv4Address{192, 168, 1, 1},
		Port: 12345,
	}
	pkt := NewPacket(remoteAddr.Port, 9999, []byte("Test"))

	// Deliver should fail
	err := d.Deliver(pkt, remoteAddr)
	if err == nil {
		t.Error("Deliver() to unbound port should fail")
	}
}

func TestDemultiplexerEphemeralPortAllocation(t *testing.T) {
	d := NewDemultiplexer()

	// Allocate multiple ephemeral ports
	ports := make(map[uint16]bool)
	for i := 0; i < 100; i++ {
		s := NewSocket()
		port, err := d.Bind(s, 0)
		if err != nil {
			t.Fatalf("Bind() iteration %d error = %v", i, err)
		}

		// Check port is in ephemeral range
		if port < EphemeralPortStart || port > EphemeralPortEnd {
			t.Errorf("Bind() returned port %d, want ephemeral port in range [%d, %d]",
				port, EphemeralPortStart, EphemeralPortEnd)
		}

		// Check port is not duplicate
		if ports[port] {
			t.Errorf("Bind() returned duplicate port %d", port)
		}
		ports[port] = true
	}
}
