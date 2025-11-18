package tcp

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ConnectionProfiler tracks performance metrics for TCP connections
type ConnectionProfiler struct {
	mu sync.RWMutex

	// Timing metrics (in nanoseconds)
	segmentProcessingTime atomic.Uint64
	checksumTime          atomic.Uint64
	stateTransitionTime   atomic.Uint64
	bufferOperationTime   atomic.Uint64
	retransmitTime        atomic.Uint64

	// Operation counters
	segmentsProcessed   atomic.Uint64
	checksumsCalculated atomic.Uint64
	stateTransitions    atomic.Uint64
	bufferOperations    atomic.Uint64
	retransmissions     atomic.Uint64

	// Performance metrics
	minLatency atomic.Uint64
	maxLatency atomic.Uint64
	avgLatency atomic.Uint64

	// Throughput tracking
	bytesProcessed atomic.Uint64
	packetsDropped atomic.Uint64

	// Histogram bins for latency distribution (in microseconds)
	latencyHistogram [10]atomic.Uint64 // 0-1, 1-2, 2-5, 5-10, 10-20, 20-50, 50-100, 100-200, 200-500, 500+
}

// Global profiler instance
var globalProfiler = NewConnectionProfiler()

// NewConnectionProfiler creates a new connection profiler
func NewConnectionProfiler() *ConnectionProfiler {
	return &ConnectionProfiler{
		minLatency: atomic.Uint64{},
		maxLatency: atomic.Uint64{},
	}
}

// GetGlobalProfiler returns the global profiler instance
func GetGlobalProfiler() *ConnectionProfiler {
	return globalProfiler
}

// RecordSegmentProcessing records time taken to process a segment
func (cp *ConnectionProfiler) RecordSegmentProcessing(start time.Time, size int) {
	duration := time.Since(start).Nanoseconds()
	cp.segmentProcessingTime.Add(uint64(duration))
	cp.segmentsProcessed.Add(1)
	cp.bytesProcessed.Add(uint64(size))

	// Update latency stats
	latencyUs := uint64(duration / 1000)
	cp.updateLatency(latencyUs)
	cp.updateHistogram(latencyUs)
}

// RecordChecksum records checksum calculation time
func (cp *ConnectionProfiler) RecordChecksum(start time.Time) {
	duration := time.Since(start).Nanoseconds()
	cp.checksumTime.Add(uint64(duration))
	cp.checksumsCalculated.Add(1)
}

// RecordStateTransition records state transition time
func (cp *ConnectionProfiler) RecordStateTransition(start time.Time) {
	duration := time.Since(start).Nanoseconds()
	cp.stateTransitionTime.Add(uint64(duration))
	cp.stateTransitions.Add(1)
}

// RecordBufferOperation records buffer operation time
func (cp *ConnectionProfiler) RecordBufferOperation(start time.Time) {
	duration := time.Since(start).Nanoseconds()
	cp.bufferOperationTime.Add(uint64(duration))
	cp.bufferOperations.Add(1)
}

// RecordRetransmit records retransmission time
func (cp *ConnectionProfiler) RecordRetransmit(start time.Time) {
	duration := time.Since(start).Nanoseconds()
	cp.retransmitTime.Add(uint64(duration))
	cp.retransmissions.Add(1)
}

// RecordPacketDrop records a dropped packet
func (cp *ConnectionProfiler) RecordPacketDrop() {
	cp.packetsDropped.Add(1)
}

// updateLatency updates min/max/avg latency
func (cp *ConnectionProfiler) updateLatency(latencyUs uint64) {
	// Update min latency
	for {
		current := cp.minLatency.Load()
		if current != 0 && current <= latencyUs {
			break
		}
		if cp.minLatency.CompareAndSwap(current, latencyUs) {
			break
		}
	}

	// Update max latency
	for {
		current := cp.maxLatency.Load()
		if current >= latencyUs {
			break
		}
		if cp.maxLatency.CompareAndSwap(current, latencyUs) {
			break
		}
	}

	// Update average (using exponential moving average)
	current := cp.avgLatency.Load()
	newAvg := (current*15 + latencyUs) / 16 // EMA with alpha=1/16
	cp.avgLatency.Store(newAvg)
}

// updateHistogram updates the latency histogram
func (cp *ConnectionProfiler) updateHistogram(latencyUs uint64) {
	var bin int
	switch {
	case latencyUs < 1:
		bin = 0
	case latencyUs < 2:
		bin = 1
	case latencyUs < 5:
		bin = 2
	case latencyUs < 10:
		bin = 3
	case latencyUs < 20:
		bin = 4
	case latencyUs < 50:
		bin = 5
	case latencyUs < 100:
		bin = 6
	case latencyUs < 200:
		bin = 7
	case latencyUs < 500:
		bin = 8
	default:
		bin = 9
	}
	cp.latencyHistogram[bin].Add(1)
}

// GetStats returns current profiling statistics
func (cp *ConnectionProfiler) GetStats() *ProfileStats {
	segmentsProcessed := cp.segmentsProcessed.Load()
	if segmentsProcessed == 0 {
		return &ProfileStats{}
	}

	return &ProfileStats{
		AvgSegmentProcessingTime: time.Duration(cp.segmentProcessingTime.Load() / segmentsProcessed),
		AvgChecksumTime:          time.Duration(cp.checksumTime.Load() / max(cp.checksumsCalculated.Load(), 1)),
		AvgStateTransitionTime:   time.Duration(cp.stateTransitionTime.Load() / max(cp.stateTransitions.Load(), 1)),
		AvgBufferOperationTime:   time.Duration(cp.bufferOperationTime.Load() / max(cp.bufferOperations.Load(), 1)),
		AvgRetransmitTime:        time.Duration(cp.retransmitTime.Load() / max(cp.retransmissions.Load(), 1)),

		SegmentsProcessed:   segmentsProcessed,
		ChecksumsCalculated: cp.checksumsCalculated.Load(),
		StateTransitions:    cp.stateTransitions.Load(),
		BufferOperations:    cp.bufferOperations.Load(),
		Retransmissions:     cp.retransmissions.Load(),

		MinLatency: time.Duration(cp.minLatency.Load() * 1000),
		MaxLatency: time.Duration(cp.maxLatency.Load() * 1000),
		AvgLatency: time.Duration(cp.avgLatency.Load() * 1000),

		BytesProcessed: cp.bytesProcessed.Load(),
		PacketsDropped: cp.packetsDropped.Load(),

		LatencyHistogram: cp.getHistogram(),
	}
}

// getHistogram returns the latency histogram
func (cp *ConnectionProfiler) getHistogram() [10]uint64 {
	var hist [10]uint64
	for i := 0; i < 10; i++ {
		hist[i] = cp.latencyHistogram[i].Load()
	}
	return hist
}

// ProfileStats contains profiling statistics
type ProfileStats struct {
	AvgSegmentProcessingTime time.Duration
	AvgChecksumTime          time.Duration
	AvgStateTransitionTime   time.Duration
	AvgBufferOperationTime   time.Duration
	AvgRetransmitTime        time.Duration

	SegmentsProcessed   uint64
	ChecksumsCalculated uint64
	StateTransitions    uint64
	BufferOperations    uint64
	Retransmissions     uint64

	MinLatency time.Duration
	MaxLatency time.Duration
	AvgLatency time.Duration

	BytesProcessed uint64
	PacketsDropped uint64

	LatencyHistogram [10]uint64
}

// String returns a formatted string of the stats
func (ps *ProfileStats) String() string {
	throughputGbps := float64(ps.BytesProcessed*8) / float64(ps.AvgSegmentProcessingTime.Nanoseconds()) * float64(ps.SegmentsProcessed)

	return fmt.Sprintf(`TCP Connection Profile:
  Segments Processed:    %d
  Checksums Calculated:  %d
  State Transitions:     %d
  Buffer Operations:     %d
  Retransmissions:       %d
  Packets Dropped:       %d

  Average Times:
    Segment Processing:  %v (%.2f µs)
    Checksum:            %v (%.2f µs)
    State Transition:    %v (%.2f µs)
    Buffer Operation:    %v (%.2f µs)
    Retransmit:          %v (%.2f µs)

  Latency Stats:
    Min:  %v (%.2f µs)
    Avg:  %v (%.2f µs)
    Max:  %v (%.2f µs)

  Throughput:
    Bytes Processed:     %d
    Estimated Throughput: %.2f Gbps

  Latency Distribution (µs):
    <1:      %d (%.1f%%)
    1-2:     %d (%.1f%%)
    2-5:     %d (%.1f%%)
    5-10:    %d (%.1f%%)
    10-20:   %d (%.1f%%)
    20-50:   %d (%.1f%%)
    50-100:  %d (%.1f%%)
    100-200: %d (%.1f%%)
    200-500: %d (%.1f%%)
    500+:    %d (%.1f%%)`,
		ps.SegmentsProcessed,
		ps.ChecksumsCalculated,
		ps.StateTransitions,
		ps.BufferOperations,
		ps.Retransmissions,
		ps.PacketsDropped,

		ps.AvgSegmentProcessingTime, float64(ps.AvgSegmentProcessingTime.Nanoseconds())/1000.0,
		ps.AvgChecksumTime, float64(ps.AvgChecksumTime.Nanoseconds())/1000.0,
		ps.AvgStateTransitionTime, float64(ps.AvgStateTransitionTime.Nanoseconds())/1000.0,
		ps.AvgBufferOperationTime, float64(ps.AvgBufferOperationTime.Nanoseconds())/1000.0,
		ps.AvgRetransmitTime, float64(ps.AvgRetransmitTime.Nanoseconds())/1000.0,

		ps.MinLatency, float64(ps.MinLatency.Nanoseconds())/1000.0,
		ps.AvgLatency, float64(ps.AvgLatency.Nanoseconds())/1000.0,
		ps.MaxLatency, float64(ps.MaxLatency.Nanoseconds())/1000.0,

		ps.BytesProcessed,
		throughputGbps,

		ps.LatencyHistogram[0], percentage(ps.LatencyHistogram[0], ps.SegmentsProcessed),
		ps.LatencyHistogram[1], percentage(ps.LatencyHistogram[1], ps.SegmentsProcessed),
		ps.LatencyHistogram[2], percentage(ps.LatencyHistogram[2], ps.SegmentsProcessed),
		ps.LatencyHistogram[3], percentage(ps.LatencyHistogram[3], ps.SegmentsProcessed),
		ps.LatencyHistogram[4], percentage(ps.LatencyHistogram[4], ps.SegmentsProcessed),
		ps.LatencyHistogram[5], percentage(ps.LatencyHistogram[5], ps.SegmentsProcessed),
		ps.LatencyHistogram[6], percentage(ps.LatencyHistogram[6], ps.SegmentsProcessed),
		ps.LatencyHistogram[7], percentage(ps.LatencyHistogram[7], ps.SegmentsProcessed),
		ps.LatencyHistogram[8], percentage(ps.LatencyHistogram[8], ps.SegmentsProcessed),
		ps.LatencyHistogram[9], percentage(ps.LatencyHistogram[9], ps.SegmentsProcessed),
	)
}

// Reset resets all profiling statistics
func (cp *ConnectionProfiler) Reset() {
	cp.segmentProcessingTime.Store(0)
	cp.checksumTime.Store(0)
	cp.stateTransitionTime.Store(0)
	cp.bufferOperationTime.Store(0)
	cp.retransmitTime.Store(0)

	cp.segmentsProcessed.Store(0)
	cp.checksumsCalculated.Store(0)
	cp.stateTransitions.Store(0)
	cp.bufferOperations.Store(0)
	cp.retransmissions.Store(0)

	cp.minLatency.Store(0)
	cp.maxLatency.Store(0)
	cp.avgLatency.Store(0)

	cp.bytesProcessed.Store(0)
	cp.packetsDropped.Store(0)

	for i := 0; i < 10; i++ {
		cp.latencyHistogram[i].Store(0)
	}
}

func percentage(value, total uint64) float64 {
	if total == 0 {
		return 0.0
	}
	return float64(value) / float64(total) * 100.0
}

func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
