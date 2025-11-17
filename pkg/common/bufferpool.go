package common

import (
	"sync"
)

// BufferPool provides a pool of reusable byte buffers
// to reduce garbage collector pressure and improve performance.
type BufferPool struct {
	pool sync.Pool
}

// Standard buffer sizes
const (
	SmallBufferSize  = 512    // For headers and small packets
	MediumBufferSize = 1500   // MTU size
	LargeBufferSize  = 65536  // Max IP packet size
)

// Global buffer pools for common sizes
var (
	SmallBufferPool  = NewBufferPool(SmallBufferSize)
	MediumBufferPool = NewBufferPool(MediumBufferSize)
	LargeBufferPool  = NewBufferPool(LargeBufferSize)
)

// NewBufferPool creates a new buffer pool with the specified buffer size.
func NewBufferPool(size int) *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				buf := make([]byte, size)
				return &buf
			},
		},
	}
}

// Get retrieves a buffer from the pool.
// The buffer should be returned to the pool using Put() when done.
func (bp *BufferPool) Get() []byte {
	bufPtr := bp.pool.Get().(*[]byte)
	return (*bufPtr)[:cap(*bufPtr)]
}

// Put returns a buffer to the pool.
// The buffer may be reused by future Get() calls.
func (bp *BufferPool) Put(buf []byte) {
	// Clear the buffer to avoid retaining references
	for i := range buf {
		buf[i] = 0
	}
	bp.pool.Put(&buf)
}

// GetBuffer returns a buffer of appropriate size from the global pools.
// Returns nil if size is larger than LargeBufferSize.
func GetBuffer(size int) []byte {
	if size <= SmallBufferSize {
		buf := SmallBufferPool.Get()
		return buf[:size]
	} else if size <= MediumBufferSize {
		buf := MediumBufferPool.Get()
		return buf[:size]
	} else if size <= LargeBufferSize {
		buf := LargeBufferPool.Get()
		return buf[:size]
	}
	// For very large buffers, allocate directly
	return make([]byte, size)
}

// PutBuffer returns a buffer to the appropriate global pool.
func PutBuffer(buf []byte) {
	if buf == nil {
		return
	}

	capacity := cap(buf)
	if capacity == SmallBufferSize {
		SmallBufferPool.Put(buf[:SmallBufferSize])
	} else if capacity == MediumBufferSize {
		MediumBufferPool.Put(buf[:MediumBufferSize])
	} else if capacity == LargeBufferSize {
		LargeBufferPool.Put(buf[:LargeBufferSize])
	}
	// For other sizes, let GC handle it
}

// BufferPoolStats holds statistics about buffer pool usage
type BufferPoolStats struct {
	Gets      uint64
	Puts      uint64
	Allocated uint64
	Reused    uint64
}

// StatefulBufferPool is a buffer pool with statistics tracking
type StatefulBufferPool struct {
	pool  sync.Pool
	size  int
	stats BufferPoolStats
	mu    sync.Mutex
}

// NewStatefulBufferPool creates a new buffer pool with statistics
func NewStatefulBufferPool(size int) *StatefulBufferPool {
	sbp := &StatefulBufferPool{
		size: size,
	}
	sbp.pool.New = func() interface{} {
		sbp.mu.Lock()
		sbp.stats.Allocated++
		sbp.mu.Unlock()
		buf := make([]byte, size)
		return &buf
	}
	return sbp
}

// Get retrieves a buffer from the pool and updates statistics
func (sbp *StatefulBufferPool) Get() []byte {
	sbp.mu.Lock()
	sbp.stats.Gets++
	sbp.mu.Unlock()

	bufPtr := sbp.pool.Get().(*[]byte)
	return (*bufPtr)[:cap(*bufPtr)]
}

// Put returns a buffer to the pool and updates statistics
func (sbp *StatefulBufferPool) Put(buf []byte) {
	sbp.mu.Lock()
	sbp.stats.Puts++
	if sbp.stats.Puts > sbp.stats.Allocated {
		sbp.stats.Reused = sbp.stats.Puts - sbp.stats.Allocated
	}
	sbp.mu.Unlock()

	// Clear the buffer
	for i := range buf {
		buf[i] = 0
	}
	sbp.pool.Put(&buf)
}

// Stats returns the current pool statistics
func (sbp *StatefulBufferPool) Stats() BufferPoolStats {
	sbp.mu.Lock()
	defer sbp.mu.Unlock()
	return sbp.stats
}

// Reset resets the pool statistics
func (sbp *StatefulBufferPool) Reset() {
	sbp.mu.Lock()
	defer sbp.mu.Unlock()
	sbp.stats = BufferPoolStats{}
}
