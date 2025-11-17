package tcp

import (
	"testing"
	"time"
)

func TestRetransmitQueue(t *testing.T) {
	rq := NewRetransmitQueue()

	// Test empty queue
	if rq.Len() != 0 {
		t.Errorf("Len() = %d, want 0", rq.Len())
	}

	if rq.GetFirst() != nil {
		t.Error("GetFirst() should return nil for empty queue")
	}

	// Add segments
	seg1 := NewSegment(12345, 80, 1000, 0, FlagSYN, 65535, nil)
	seg2 := NewSegment(12345, 80, 1001, 0, FlagACK, 65535, []byte("data1"))
	seg3 := NewSegment(12345, 80, 1006, 0, FlagACK, 65535, []byte("data2"))

	now := time.Now()
	rq.Add(1000, seg1, now)
	rq.Add(1001, seg2, now)
	rq.Add(1006, seg3, now)

	if rq.Len() != 3 {
		t.Errorf("Len() = %d, want 3", rq.Len())
	}

	// Test GetFirst
	first := rq.GetFirst()
	if first == nil {
		t.Fatal("GetFirst() returned nil")
	}
	if first.SequenceNumber != 1000 {
		t.Errorf("GetFirst().SequenceNumber = %d, want 1000", first.SequenceNumber)
	}

	// Test Remove
	rq.Remove(1001)
	if rq.Len() != 2 {
		t.Errorf("Len() after Remove() = %d, want 2", rq.Len())
	}

	// Test RemoveBefore
	rq.RemoveBefore(1006)
	if rq.Len() != 1 {
		t.Errorf("Len() after RemoveBefore() = %d, want 1", rq.Len())
	}

	// Test Clear
	rq.Clear()
	if rq.Len() != 0 {
		t.Errorf("Len() after Clear() = %d, want 0", rq.Len())
	}
}

func TestRetransmitQueueExpired(t *testing.T) {
	rq := NewRetransmitQueue()

	seg1 := NewSegment(12345, 80, 1000, 0, FlagSYN, 65535, nil)
	seg2 := NewSegment(12345, 80, 1001, 0, FlagACK, 65535, []byte("data"))

	// Add one old segment and one new segment
	oldTime := time.Now().Add(-2 * time.Second)
	newTime := time.Now()

	rq.Add(1000, seg1, oldTime)
	rq.Add(1001, seg2, newTime)

	// Get expired with 1 second timeout
	expired := rq.GetExpired(time.Second)

	if len(expired) != 1 {
		t.Errorf("GetExpired() returned %d segments, want 1", len(expired))
	}

	if len(expired) > 0 && expired[0].SeqNum != 1000 {
		t.Errorf("Expired segment SeqNum = %d, want 1000", expired[0].SeqNum)
	}
}

func TestSeqComparison(t *testing.T) {
	tests := []struct {
		name     string
		seq1     uint32
		seq2     uint32
		expected bool
	}{
		{"before: 100 < 200", 100, 200, true},
		{"not before: 200 < 100", 200, 100, false},
		{"equal: 100 < 100", 100, 100, false},
		{"wraparound: 0xFFFFFF00 < 0x00000100", 0xFFFFFF00, 0x00000100, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := seqBefore(tt.seq1, tt.seq2)
			if result != tt.expected {
				t.Errorf("seqBefore(%d, %d) = %v, want %v", tt.seq1, tt.seq2, result, tt.expected)
			}
		})
	}
}
