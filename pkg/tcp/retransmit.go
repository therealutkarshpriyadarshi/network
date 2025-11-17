// Package tcp implements TCP retransmission queue management.
package tcp

import (
	"sync"
	"time"
)

// RetransmitEntry represents an entry in the retransmit queue.
type RetransmitEntry struct {
	SeqNum    uint32
	Segment   *Segment
	SentTime  time.Time
	RetryCount int
}

// RetransmitQueue manages segments that need to be retransmitted.
type RetransmitQueue struct {
	entries []*RetransmitEntry
	mu      sync.Mutex
}

// NewRetransmitQueue creates a new retransmit queue.
func NewRetransmitQueue() *RetransmitQueue {
	return &RetransmitQueue{
		entries: make([]*RetransmitEntry, 0),
	}
}

// Add adds a segment to the retransmit queue.
func (rq *RetransmitQueue) Add(seqNum uint32, seg *Segment, sentTime time.Time) {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	entry := &RetransmitEntry{
		SeqNum:     seqNum,
		Segment:    seg,
		SentTime:   sentTime,
		RetryCount: 0,
	}

	rq.entries = append(rq.entries, entry)
}

// Remove removes a segment from the retransmit queue by sequence number.
func (rq *RetransmitQueue) Remove(seqNum uint32) {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	for i, entry := range rq.entries {
		if entry.SeqNum == seqNum {
			rq.entries = append(rq.entries[:i], rq.entries[i+1:]...)
			return
		}
	}
}

// RemoveBefore removes all segments with sequence numbers less than the given value.
func (rq *RetransmitQueue) RemoveBefore(seqNum uint32) {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	newEntries := make([]*RetransmitEntry, 0)
	for _, entry := range rq.entries {
		// Use sequence number comparison that handles wraparound
		if !seqBefore(entry.SeqNum, seqNum) {
			newEntries = append(newEntries, entry)
		}
	}

	rq.entries = newEntries
}

// GetExpired returns all segments that have exceeded the given timeout.
func (rq *RetransmitQueue) GetExpired(timeout time.Duration) []*RetransmitEntry {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	expired := make([]*RetransmitEntry, 0)
	now := time.Now()

	for _, entry := range rq.entries {
		if now.Sub(entry.SentTime) > timeout {
			expired = append(expired, entry)
		}
	}

	return expired
}

// UpdateSentTime updates the sent time for a segment.
func (rq *RetransmitQueue) UpdateSentTime(seqNum uint32, sentTime time.Time) {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	for _, entry := range rq.entries {
		if entry.SeqNum == seqNum {
			entry.SentTime = sentTime
			entry.RetryCount++
			return
		}
	}
}

// GetFirst returns the first segment in the retransmit queue.
func (rq *RetransmitQueue) GetFirst() *Segment {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	if len(rq.entries) == 0 {
		return nil
	}

	return rq.entries[0].Segment
}

// Len returns the number of entries in the retransmit queue.
func (rq *RetransmitQueue) Len() int {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	return len(rq.entries)
}

// Clear clears the retransmit queue.
func (rq *RetransmitQueue) Clear() {
	rq.mu.Lock()
	defer rq.mu.Unlock()

	rq.entries = rq.entries[:0]
}

// seqBefore returns true if seq1 is before seq2 (handling wraparound).
func seqBefore(seq1, seq2 uint32) bool {
	// Use signed comparison to handle wraparound
	return int32(seq1-seq2) < 0
}

// seqAfter returns true if seq1 is after seq2 (handling wraparound).
func seqAfter(seq1, seq2 uint32) bool {
	return int32(seq1-seq2) > 0
}

// seqBetween returns true if seq is between start and end (handling wraparound).
func seqBetween(seq, start, end uint32) bool {
	return seqAfter(seq, start) && seqBefore(seq, end)
}
