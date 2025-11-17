// Package tcp implements TCP Fast Open (TFO) as defined in RFC 7413.
package tcp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"net"
)

const (
	// TFOCookieLen is the length of a TFO cookie (16 bytes).
	TFOCookieLen = 16

	// TFOMaxCookieLen is the maximum length of a TFO cookie option.
	TFOMaxCookieLen = 18 // Kind(1) + Length(1) + Cookie(up to 16)
)

// TFOCookie represents a TCP Fast Open cookie.
type TFOCookie [TFOCookieLen]byte

// TFOState manages TCP Fast Open state.
type TFOState struct {
	// Server-side state
	key cipher.Block // AES key for generating cookies

	// Client-side state
	cookieCache map[string]TFOCookie // IP -> Cookie mapping
}

// NewTFOState creates a new TCP Fast Open state manager.
func NewTFOState() (*TFOState, error) {
	// Generate random AES key for cookie generation
	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate TFO key: %w", err)
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	return &TFOState{
		key:         block,
		cookieCache: make(map[string]TFOCookie),
	}, nil
}

// GenerateCookie generates a TFO cookie for the given client IP.
func (tfo *TFOState) GenerateCookie(clientIP net.IP) (TFOCookie, error) {
	var cookie TFOCookie

	// Use client IP as input to AES
	input := make([]byte, 16)
	copy(input, clientIP.To16())

	// Encrypt to generate cookie
	tfo.key.Encrypt(cookie[:], input)

	return cookie, nil
}

// ValidateCookie validates a TFO cookie for the given client IP.
func (tfo *TFOState) ValidateCookie(clientIP net.IP, cookie TFOCookie) bool {
	expected, err := tfo.GenerateCookie(clientIP)
	if err != nil {
		return false
	}

	// Constant-time comparison
	return subtle_constantTimeCompare(cookie[:], expected[:])
}

// subtle_constantTimeCompare performs constant-time comparison of two byte slices.
func subtle_constantTimeCompare(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}

	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}

	return v == 0
}

// CacheCookie caches a TFO cookie for the given server IP.
func (tfo *TFOState) CacheCookie(serverIP string, cookie TFOCookie) {
	tfo.cookieCache[serverIP] = cookie
}

// GetCachedCookie retrieves a cached TFO cookie for the given server IP.
func (tfo *TFOState) GetCachedCookie(serverIP string) (TFOCookie, bool) {
	cookie, ok := tfo.cookieCache[serverIP]
	return cookie, ok
}

// BuildTFOOption builds a TCP Fast Open option.
// If cookie is nil, this is a cookie request (SYN without cookie).
// If cookie is provided, this is a SYN with cookie.
func BuildTFOOption(cookie []byte) []byte {
	if len(cookie) == 0 {
		// Cookie request (empty TFO option)
		return []byte{OptionKindTFO, 2}
	}

	if len(cookie) > TFOCookieLen {
		return nil
	}

	length := 2 + len(cookie)
	opt := make([]byte, length)
	opt[0] = OptionKindTFO
	opt[1] = uint8(length)
	copy(opt[2:], cookie)

	return opt
}

// GetTFOCookie extracts a TFO cookie from segment options.
func (s *Segment) GetTFOCookie() ([]byte, error) {
	opts, err := s.ParseOptions()
	if err != nil {
		return nil, err
	}

	if tfoData, ok := opts[OptionKindTFO]; ok {
		// Empty TFO option means cookie request
		if len(tfoData) == 0 {
			return []byte{}, nil
		}

		// Return the cookie
		return tfoData, nil
	}

	return nil, fmt.Errorf("TFO option not found")
}

// HasTFO checks if the TFO option is present.
func (s *Segment) HasTFO() bool {
	opts, err := s.ParseOptions()
	if err != nil {
		return false
	}
	_, ok := opts[OptionKindTFO]
	return ok
}

// TFOConnection represents a TCP Fast Open connection state.
type TFOConnection struct {
	state      *TFOState
	cookie     *TFOCookie
	dataQueued []byte // Data to send with SYN
}

// NewTFOConnection creates a new TFO connection.
func NewTFOConnection(state *TFOState) *TFOConnection {
	return &TFOConnection{
		state: state,
	}
}

// SetCookie sets the TFO cookie for this connection.
func (c *TFOConnection) SetCookie(cookie TFOCookie) {
	c.cookie = &cookie
}

// QueueData queues data to be sent with the SYN packet.
func (c *TFOConnection) QueueData(data []byte) {
	c.dataQueued = append(c.dataQueued, data...)
}

// GetQueuedData returns and clears the queued data.
func (c *TFOConnection) GetQueuedData() []byte {
	data := c.dataQueued
	c.dataQueued = nil
	return data
}

// HasCookie returns true if this connection has a TFO cookie.
func (c *TFOConnection) HasCookie() bool {
	return c.cookie != nil
}
