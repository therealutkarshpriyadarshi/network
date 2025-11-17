// Package tls provides secure TLS connections over TCP.
package tls

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
)

// Conn represents a TLS connection wrapping a network connection.
type Conn struct {
	tlsConn *tls.Conn
	config  *Config
}

// Server wraps a connection with TLS as a server.
func Server(conn net.Conn, config *Config) *Conn {
	stdConfig := config.ToStdTLSConfig()
	tlsConn := tls.Server(conn, stdConfig)

	return &Conn{
		tlsConn: tlsConn,
		config:  config,
	}
}

// Client wraps a connection with TLS as a client.
func Client(conn net.Conn, config *Config) *Conn {
	stdConfig := config.ToStdTLSConfig()
	tlsConn := tls.Client(conn, stdConfig)

	return &Conn{
		tlsConn: tlsConn,
		config:  config,
	}
}

// Handshake performs the TLS handshake.
func (c *Conn) Handshake() error {
	return c.tlsConn.Handshake()
}

// Read reads data from the TLS connection.
func (c *Conn) Read(b []byte) (int, error) {
	return c.tlsConn.Read(b)
}

// Write writes data to the TLS connection.
func (c *Conn) Write(b []byte) (int, error) {
	return c.tlsConn.Write(b)
}

// Close closes the TLS connection.
func (c *Conn) Close() error {
	return c.tlsConn.Close()
}

// ConnectionState returns the TLS connection state.
func (c *Conn) ConnectionState() tls.ConnectionState {
	return c.tlsConn.ConnectionState()
}

// VerifyHostname verifies the server's hostname.
func (c *Conn) VerifyHostname(hostname string) error {
	return c.tlsConn.VerifyHostname(hostname)
}

// GetNegotiatedProtocol returns the negotiated ALPN protocol.
func (c *Conn) GetNegotiatedProtocol() string {
	return c.tlsConn.ConnectionState().NegotiatedProtocol
}

// GetCipherSuite returns the negotiated cipher suite.
func (c *Conn) GetCipherSuite() CipherSuite {
	return CipherSuite(c.tlsConn.ConnectionState().CipherSuite)
}

// GetVersion returns the negotiated TLS version.
func (c *Conn) GetVersion() TLSVersion {
	return TLSVersion(c.tlsConn.ConnectionState().Version)
}

// IsMutualTLS returns true if mutual TLS (client certificate) is used.
func (c *Conn) IsMutualTLS() bool {
	state := c.tlsConn.ConnectionState()
	return len(state.PeerCertificates) > 0
}

// GetPeerCertificates returns the peer's certificate chain.
func (c *Conn) GetPeerCertificates() []*tls.Certificate {
	state := c.tlsConn.ConnectionState()
	certs := make([]*tls.Certificate, len(state.PeerCertificates))

	for i, cert := range state.PeerCertificates {
		tlsCert := &tls.Certificate{
			Certificate: [][]byte{cert.Raw},
		}
		certs[i] = tlsCert
	}

	return certs
}

// SecureWrite writes data securely with error handling.
func (c *Conn) SecureWrite(data []byte) error {
	n, err := c.Write(data)
	if err != nil {
		return fmt.Errorf("TLS write failed: %w", err)
	}
	if n != len(data) {
		return fmt.Errorf("incomplete write: %d/%d bytes", n, len(data))
	}
	return nil
}

// SecureRead reads exactly n bytes or returns an error.
func (c *Conn) SecureRead(n int) ([]byte, error) {
	buf := make([]byte, n)
	_, err := io.ReadFull(c.tlsConn, buf)
	if err != nil {
		return nil, fmt.Errorf("TLS read failed: %w", err)
	}
	return buf, nil
}

// String returns a description of the TLS connection.
func (c *Conn) String() string {
	state := c.tlsConn.ConnectionState()
	return fmt.Sprintf("TLS{Version=%s, CipherSuite=%s, ServerName=%s}",
		TLSVersion(state.Version), CipherSuite(state.CipherSuite), state.ServerName)
}
