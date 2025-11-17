package common

import (
	"crypto/rand"
	"testing"
)

func TestChecksumOptimizedCorrectness(t *testing.T) {
	sizes := []int{0, 1, 15, 16, 17, 31, 32, 64, 127, 128, 255, 256, 512, 1023, 1024, 1500, 4096}

	for _, size := range sizes {
		t.Run("Size_"+string(rune(size)), func(t *testing.T) {
			data := make([]byte, size)
			rand.Read(data)

			original := CalculateChecksum(data)
			optimized := CalculateChecksumOptimized(data)
			fast := CalculateChecksumFast(data)

			if original != optimized {
				t.Errorf("CalculateChecksumOptimized mismatch: got %04x, want %04x", optimized, original)
			}

			if original != fast {
				t.Errorf("CalculateChecksumFast mismatch: got %04x, want %04x", fast, original)
			}
		})
	}
}

func TestChecksumWithPseudoHeaderOptimizedCorrectness(t *testing.T) {
	pseudoHeader := PseudoHeader{
		SourceAddr:      IPv4Address{192, 168, 1, 1},
		DestinationAddr: IPv4Address{192, 168, 1, 2},
		Protocol:        ProtocolTCP,
		Length:          1460,
	}

	sizes := []int{0, 1, 20, 64, 128, 512, 1024, 1460}

	for _, size := range sizes {
		t.Run("Size_"+string(rune(size)), func(t *testing.T) {
			data := make([]byte, size)
			rand.Read(data)

			original := CalculateChecksumWithPseudoHeader(pseudoHeader, data)
			optimized := CalculateChecksumWithPseudoHeaderOptimized(pseudoHeader, data)

			if original != optimized {
				t.Errorf("Checksum mismatch: got %04x, want %04x", optimized, original)
			}
		})
	}
}

func TestUpdateChecksumOptimizedCorrectness(t *testing.T) {
	// UpdateChecksumOptimized currently wraps UpdateChecksum,
	// so we just verify they produce identical results
	oldData := []byte{0x00, 0x01, 0x02, 0x03}
	newData := []byte{0x04, 0x05, 0x06, 0x07}
	oldChecksum := uint16(0x1234)

	result1 := UpdateChecksum(oldChecksum, oldData, newData)
	result2 := UpdateChecksumOptimized(oldChecksum, oldData, newData)

	if result1 != result2 {
		t.Errorf("UpdateChecksumOptimized mismatch: got %04x, want %04x", result2, result1)
	}
}

func BenchmarkChecksumComparison(b *testing.B) {
	sizes := []int{20, 64, 128, 512, 1024, 1500, 4096, 65536}

	for _, size := range sizes {
		data := make([]byte, size)
		rand.Read(data)

		b.Run("Original_"+string(rune(size)), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = CalculateChecksum(data)
			}
		})

		b.Run("Optimized_"+string(rune(size)), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = CalculateChecksumOptimized(data)
			}
		})

		b.Run("Fast_"+string(rune(size)), func(b *testing.B) {
			b.SetBytes(int64(size))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = CalculateChecksumFast(data)
			}
		})
	}
}

func BenchmarkChecksumWithPseudoHeaderComparison(b *testing.B) {
	pseudoHeader := PseudoHeader{
		SourceAddr:      IPv4Address{192, 168, 1, 1},
		DestinationAddr: IPv4Address{192, 168, 1, 2},
		Protocol:        ProtocolTCP,
		Length:          1460,
	}

	data := make([]byte, 1460)
	rand.Read(data)

	b.Run("Original", func(b *testing.B) {
		b.SetBytes(int64(len(data)))
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = CalculateChecksumWithPseudoHeader(pseudoHeader, data)
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		b.SetBytes(int64(len(data)))
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = CalculateChecksumWithPseudoHeaderOptimized(pseudoHeader, data)
		}
	})
}

func BenchmarkUpdateChecksumComparison(b *testing.B) {
	data := make([]byte, 1500)
	rand.Read(data)

	oldPortion := make([]byte, 16)
	newPortion := make([]byte, 16)
	copy(oldPortion, data[100:116])
	rand.Read(newPortion)

	oldChecksum := CalculateChecksum(data)

	b.Run("Original", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = UpdateChecksum(oldChecksum, oldPortion, newPortion)
		}
	})

	b.Run("Optimized", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = UpdateChecksumOptimized(oldChecksum, oldPortion, newPortion)
		}
	})
}
