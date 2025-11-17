package common

import (
	"io"
	"testing"
)

func TestNewPacketBuffer(t *testing.T) {
	size := 1500
	pb := NewPacketBuffer(size)

	if pb.Len() != size {
		t.Errorf("NewPacketBuffer length = %d, want %d", pb.Len(), size)
	}

	if pb.Position() != 0 {
		t.Errorf("Initial position = %d, want 0", pb.Position())
	}

	if pb.Remaining() != size {
		t.Errorf("Remaining = %d, want %d", pb.Remaining(), size)
	}
}

func TestNewPacketBufferFromBytes(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04}
	pb := NewPacketBufferFromBytes(data)

	if pb.Len() != len(data) {
		t.Errorf("Length = %d, want %d", pb.Len(), len(data))
	}

	// Verify data matches
	for i, b := range data {
		if pb.Bytes()[i] != b {
			t.Errorf("Byte %d = 0x%02X, want 0x%02X", i, pb.Bytes()[i], b)
		}
	}
}

func TestPacketBufferReadByte(t *testing.T) {
	data := []byte{0x12, 0x34, 0x56}
	pb := NewPacketBufferFromBytes(data)

	// Read first byte
	b, err := pb.ReadByte()
	if err != nil {
		t.Fatalf("ReadByte() error = %v", err)
	}
	if b != 0x12 {
		t.Errorf("ReadByte() = 0x%02X, want 0x12", b)
	}
	if pb.Position() != 1 {
		t.Errorf("Position = %d, want 1", pb.Position())
	}

	// Read second byte
	b, err = pb.ReadByte()
	if err != nil {
		t.Fatalf("ReadByte() error = %v", err)
	}
	if b != 0x34 {
		t.Errorf("ReadByte() = 0x%02X, want 0x34", b)
	}

	// Read third byte
	b, err = pb.ReadByte()
	if err != nil {
		t.Fatalf("ReadByte() error = %v", err)
	}
	if b != 0x56 {
		t.Errorf("ReadByte() = 0x%02X, want 0x56", b)
	}

	// Try to read beyond end
	_, err = pb.ReadByte()
	if err != io.EOF {
		t.Errorf("ReadByte() error = %v, want EOF", err)
	}
}

func TestPacketBufferReadBytes(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04, 0x05}
	pb := NewPacketBufferFromBytes(data)

	// Read 3 bytes
	bytes, err := pb.ReadBytes(3)
	if err != nil {
		t.Fatalf("ReadBytes() error = %v", err)
	}
	if len(bytes) != 3 {
		t.Errorf("ReadBytes() length = %d, want 3", len(bytes))
	}
	for i, expected := range []byte{0x01, 0x02, 0x03} {
		if bytes[i] != expected {
			t.Errorf("ReadBytes()[%d] = 0x%02X, want 0x%02X", i, bytes[i], expected)
		}
	}

	// Try to read beyond end
	_, err = pb.ReadBytes(10)
	if err != io.EOF {
		t.Errorf("ReadBytes() error = %v, want EOF", err)
	}
}

func TestPacketBufferReadUint16(t *testing.T) {
	data := []byte{0x12, 0x34, 0x56, 0x78}
	pb := NewPacketBufferFromBytes(data)

	// Read first uint16 (big endian)
	val, err := pb.ReadUint16()
	if err != nil {
		t.Fatalf("ReadUint16() error = %v", err)
	}
	if val != 0x1234 {
		t.Errorf("ReadUint16() = 0x%04X, want 0x1234", val)
	}

	// Read second uint16
	val, err = pb.ReadUint16()
	if err != nil {
		t.Fatalf("ReadUint16() error = %v", err)
	}
	if val != 0x5678 {
		t.Errorf("ReadUint16() = 0x%04X, want 0x5678", val)
	}
}

func TestPacketBufferReadUint32(t *testing.T) {
	data := []byte{0x12, 0x34, 0x56, 0x78}
	pb := NewPacketBufferFromBytes(data)

	val, err := pb.ReadUint32()
	if err != nil {
		t.Fatalf("ReadUint32() error = %v", err)
	}
	if val != 0x12345678 {
		t.Errorf("ReadUint32() = 0x%08X, want 0x12345678", val)
	}
}

func TestPacketBufferReadMAC(t *testing.T) {
	data := []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	pb := NewPacketBufferFromBytes(data)

	mac, err := pb.ReadMAC()
	if err != nil {
		t.Fatalf("ReadMAC() error = %v", err)
	}

	expected := MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	if mac != expected {
		t.Errorf("ReadMAC() = %v, want %v", mac, expected)
	}
}

func TestPacketBufferReadIPv4(t *testing.T) {
	data := []byte{192, 168, 1, 1}
	pb := NewPacketBufferFromBytes(data)

	ip, err := pb.ReadIPv4()
	if err != nil {
		t.Fatalf("ReadIPv4() error = %v", err)
	}

	expected := IPv4Address{192, 168, 1, 1}
	if ip != expected {
		t.Errorf("ReadIPv4() = %v, want %v", ip, expected)
	}
}

func TestPacketBufferWrite(t *testing.T) {
	pb := NewPacketBuffer(10)

	// Write byte
	if err := pb.WriteByte(0x12); err != nil {
		t.Fatalf("WriteByte() error = %v", err)
	}

	// Write uint16
	if err := pb.WriteUint16(0x3456); err != nil {
		t.Fatalf("WriteUint16() error = %v", err)
	}

	// Write uint32
	if err := pb.WriteUint32(0x789ABCDE); err != nil {
		t.Fatalf("WriteUint32() error = %v", err)
	}

	// Verify data
	expected := []byte{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0x00, 0x00, 0x00}
	pb.Reset()
	for i := 0; i < 7; i++ {
		b, _ := pb.ReadByte()
		if b != expected[i] {
			t.Errorf("Byte %d = 0x%02X, want 0x%02X", i, b, expected[i])
		}
	}
}

func TestPacketBufferWriteMAC(t *testing.T) {
	pb := NewPacketBuffer(6)
	mac := MACAddress{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}

	if err := pb.WriteMAC(mac); err != nil {
		t.Fatalf("WriteMAC() error = %v", err)
	}

	pb.Reset()
	readMAC, err := pb.ReadMAC()
	if err != nil {
		t.Fatalf("ReadMAC() error = %v", err)
	}

	if readMAC != mac {
		t.Errorf("MAC = %v, want %v", readMAC, mac)
	}
}

func TestPacketBufferWriteIPv4(t *testing.T) {
	pb := NewPacketBuffer(4)
	ip := IPv4Address{192, 168, 1, 1}

	if err := pb.WriteIPv4(ip); err != nil {
		t.Fatalf("WriteIPv4() error = %v", err)
	}

	pb.Reset()
	readIP, err := pb.ReadIPv4()
	if err != nil {
		t.Fatalf("ReadIPv4() error = %v", err)
	}

	if readIP != ip {
		t.Errorf("IP = %v, want %v", readIP, ip)
	}
}

func TestPacketBufferPosition(t *testing.T) {
	pb := NewPacketBuffer(10)

	// Test SetPosition
	if err := pb.SetPosition(5); err != nil {
		t.Fatalf("SetPosition() error = %v", err)
	}
	if pb.Position() != 5 {
		t.Errorf("Position = %d, want 5", pb.Position())
	}

	// Test invalid position
	if err := pb.SetPosition(100); err == nil {
		t.Error("SetPosition(100) should return error")
	}

	// Test Reset
	pb.Reset()
	if pb.Position() != 0 {
		t.Errorf("Position after Reset = %d, want 0", pb.Position())
	}
}

func TestPacketBufferSkip(t *testing.T) {
	pb := NewPacketBuffer(10)

	if err := pb.Skip(5); err != nil {
		t.Fatalf("Skip() error = %v", err)
	}
	if pb.Position() != 5 {
		t.Errorf("Position = %d, want 5", pb.Position())
	}

	// Try to skip beyond end
	if err := pb.Skip(10); err != io.EOF {
		t.Errorf("Skip() error = %v, want EOF", err)
	}
}

func TestPacketBufferRemaining(t *testing.T) {
	pb := NewPacketBuffer(10)

	if pb.Remaining() != 10 {
		t.Errorf("Remaining = %d, want 10", pb.Remaining())
	}

	pb.Skip(3)
	if pb.Remaining() != 7 {
		t.Errorf("Remaining = %d, want 7", pb.Remaining())
	}

	pb.Skip(7)
	if pb.Remaining() != 0 {
		t.Errorf("Remaining = %d, want 0", pb.Remaining())
	}
}

func TestHexDump(t *testing.T) {
	data := []byte{
		0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77,
		0x88, 0x99, 0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF,
		0x48, 0x65, 0x6C, 0x6C, 0x6F, // "Hello"
	}

	dump := HexDump(data)

	// Just verify it produces output
	if len(dump) == 0 {
		t.Error("HexDump() returned empty string")
	}

	// Should contain hex representation
	if len(dump) < len(data)*3 {
		t.Error("HexDump() output seems too short")
	}
}

func TestPacketBufferHexDump(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03, 0x04}
	pb := NewPacketBufferFromBytes(data)

	dump := pb.HexDump()
	if len(dump) == 0 {
		t.Error("PacketBuffer.HexDump() returned empty string")
	}
}

// Benchmark tests
func BenchmarkPacketBufferRead(b *testing.B) {
	data := make([]byte, 1500)
	for i := range data {
		data[i] = byte(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pb := NewPacketBufferFromBytes(data)
		for pb.Remaining() >= 4 {
			pb.ReadUint32()
		}
	}
}

func BenchmarkPacketBufferWrite(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pb := NewPacketBuffer(1500)
		for pb.Remaining() >= 4 {
			pb.WriteUint32(0x12345678)
		}
	}
}
