package common

import (
	"testing"
)

func TestBufferPool(t *testing.T) {
	pool := NewBufferPool(1024)

	// Get a buffer
	buf := pool.Get()
	if len(buf) != 1024 {
		t.Errorf("Expected buffer size 1024, got %d", len(buf))
	}

	// Use the buffer
	for i := range buf {
		buf[i] = byte(i % 256)
	}

	// Return buffer to pool
	pool.Put(buf)

	// Get another buffer (should be the same one, now cleared)
	buf2 := pool.Get()
	if len(buf2) != 1024 {
		t.Errorf("Expected buffer size 1024, got %d", len(buf2))
	}

	// Verify it was cleared
	for i := range buf2 {
		if buf2[i] != 0 {
			t.Errorf("Buffer not cleared at position %d: got %d", i, buf2[i])
			break
		}
	}

	pool.Put(buf2)
}

func TestGlobalBufferPools(t *testing.T) {
	tests := []struct {
		name string
		size int
	}{
		{"Small", 256},
		{"Medium", 1024},
		{"Large", 32768},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := GetBuffer(tt.size)
			if len(buf) != tt.size {
				t.Errorf("Expected buffer size %d, got %d", tt.size, len(buf))
			}

			// Use buffer
			for i := range buf {
				buf[i] = byte(i % 256)
			}

			// Return to pool
			PutBuffer(buf)

			// Get another buffer
			buf2 := GetBuffer(tt.size)
			if len(buf2) != tt.size {
				t.Errorf("Expected buffer size %d, got %d", tt.size, len(buf2))
			}

			PutBuffer(buf2)
		})
	}
}

func TestStatefulBufferPool(t *testing.T) {
	pool := NewStatefulBufferPool(1024)

	// Get and put buffers
	for i := 0; i < 10; i++ {
		buf := pool.Get()
		buf[0] = byte(i)
		pool.Put(buf)
	}

	stats := pool.Stats()

	if stats.Gets != 10 {
		t.Errorf("Expected 10 gets, got %d", stats.Gets)
	}

	if stats.Puts != 10 {
		t.Errorf("Expected 10 puts, got %d", stats.Puts)
	}

	if stats.Allocated == 0 {
		t.Error("Expected some allocations")
	}

	// Reset and verify
	pool.Reset()
	stats = pool.Stats()

	if stats.Gets != 0 || stats.Puts != 0 {
		t.Error("Stats not reset properly")
	}
}

func BenchmarkBufferPoolVsAlloc(b *testing.B) {
	b.Run("WithPool", func(b *testing.B) {
		pool := NewBufferPool(1500)
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := pool.Get()
			_ = buf
			pool.Put(buf)
		}
	})

	b.Run("WithoutPool", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := make([]byte, 1500)
			_ = buf
		}
	})
}

func BenchmarkGlobalBufferPools(b *testing.B) {
	b.Run("Small", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := GetBuffer(256)
			PutBuffer(buf)
		}
	})

	b.Run("Medium", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := GetBuffer(1024)
			PutBuffer(buf)
		}
	})

	b.Run("Large", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			buf := GetBuffer(32768)
			PutBuffer(buf)
		}
	})
}

func BenchmarkBufferPoolParallel(b *testing.B) {
	pool := NewBufferPool(1500)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			buf := pool.Get()
			// Simulate work
			buf[0] = 1
			pool.Put(buf)
		}
	})
}
