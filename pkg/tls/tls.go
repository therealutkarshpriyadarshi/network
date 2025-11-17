// Package tls implements a TLS/SSL wrapper layer for secure communication.
package tls

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

// TLSVersion represents TLS protocol versions.
type TLSVersion uint16

const (
	// TLS versions
	VersionTLS10 TLSVersion = 0x0301
	VersionTLS11 TLSVersion = 0x0302
	VersionTLS12 TLSVersion = 0x0303
	VersionTLS13 TLSVersion = 0x0304
)

// CipherSuite represents TLS cipher suites.
type CipherSuite uint16

const (
	// TLS 1.3 cipher suites
	TLS_AES_128_GCM_SHA256       CipherSuite = 0x1301
	TLS_AES_256_GCM_SHA384       CipherSuite = 0x1302
	TLS_CHACHA20_POLY1305_SHA256 CipherSuite = 0x1303

	// TLS 1.2 cipher suites
	TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256   CipherSuite = 0xc02f
	TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384   CipherSuite = 0xc030
	TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256 CipherSuite = 0xc02b
	TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384 CipherSuite = 0xc02c
)

// Config represents TLS configuration.
type Config struct {
	MinVersion         TLSVersion
	MaxVersion         TLSVersion
	CipherSuites       []CipherSuite
	InsecureSkipVerify bool
	Certificates       []tls.Certificate
	RootCAs            *x509.CertPool
	ServerName         string
}

// DefaultConfig returns a secure default TLS configuration.
func DefaultConfig() *Config {
	return &Config{
		MinVersion: VersionTLS12,
		MaxVersion: VersionTLS13,
		CipherSuites: []CipherSuite{
			TLS_AES_128_GCM_SHA256,
			TLS_AES_256_GCM_SHA384,
			TLS_CHACHA20_POLY1305_SHA256,
			TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		},
		InsecureSkipVerify: false,
	}
}

// ToStdTLSConfig converts our TLS config to crypto/tls.Config.
func (c *Config) ToStdTLSConfig() *tls.Config {
	stdConfig := &tls.Config{
		MinVersion:         uint16(c.MinVersion),
		MaxVersion:         uint16(c.MaxVersion),
		InsecureSkipVerify: c.InsecureSkipVerify,
		Certificates:       c.Certificates,
		RootCAs:            c.RootCAs,
		ServerName:         c.ServerName,
	}

	if len(c.CipherSuites) > 0 {
		stdConfig.CipherSuites = make([]uint16, len(c.CipherSuites))
		for i, cs := range c.CipherSuites {
			stdConfig.CipherSuites[i] = uint16(cs)
		}
	}

	return stdConfig
}

// GenerateSelfSignedCert generates a self-signed certificate for testing.
func GenerateSelfSignedCert(commonName string) (certPEM, keyPEM []byte, err error) {
	// Generate RSA key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"Network Stack"},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // Valid for 1 year
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{commonName},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode certificate to PEM
	certPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})

	// Encode private key to PEM
	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return certPEM, keyPEM, nil
}

// LoadCertificate loads a certificate from PEM-encoded data.
func LoadCertificate(certPEM, keyPEM []byte) (tls.Certificate, error) {
	return tls.X509KeyPair(certPEM, keyPEM)
}

// String returns the name of the TLS version.
func (v TLSVersion) String() string {
	switch v {
	case VersionTLS10:
		return "TLS 1.0"
	case VersionTLS11:
		return "TLS 1.1"
	case VersionTLS12:
		return "TLS 1.2"
	case VersionTLS13:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown(0x%04x)", uint16(v))
	}
}

// String returns the name of the cipher suite.
func (cs CipherSuite) String() string {
	switch cs {
	case TLS_AES_128_GCM_SHA256:
		return "TLS_AES_128_GCM_SHA256"
	case TLS_AES_256_GCM_SHA384:
		return "TLS_AES_256_GCM_SHA384"
	case TLS_CHACHA20_POLY1305_SHA256:
		return "TLS_CHACHA20_POLY1305_SHA256"
	case TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256:
		return "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256"
	case TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384:
		return "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
	case TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256:
		return "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256"
	case TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384:
		return "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384"
	default:
		return fmt.Sprintf("Unknown(0x%04x)", uint16(cs))
	}
}
