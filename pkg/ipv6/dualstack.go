// Package ipv6 provides dual-stack networking support.
package ipv6

import (
	"fmt"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
	"github.com/therealutkarshpriyadarshi/network/pkg/ip"
)

// IPAddress represents either an IPv4 or IPv6 address.
type IPAddress struct {
	IPv4 *common.IPv4Address
	IPv6 *common.IPv6Address
}

// NewIPv4Address creates a dual-stack address from an IPv4 address.
func NewIPv4Address(addr common.IPv4Address) IPAddress {
	return IPAddress{IPv4: &addr}
}

// NewIPv6Address creates a dual-stack address from an IPv6 address.
func NewIPv6Address(addr common.IPv6Address) IPAddress {
	return IPAddress{IPv6: &addr}
}

// IsIPv4 returns true if this is an IPv4 address.
func (ip IPAddress) IsIPv4() bool {
	return ip.IPv4 != nil
}

// IsIPv6 returns true if this is an IPv6 address.
func (ip IPAddress) IsIPv6() bool {
	return ip.IPv6 != nil
}

// String returns a string representation of the address.
func (ip IPAddress) String() string {
	if ip.IPv4 != nil {
		return ip.IPv4.String()
	}
	if ip.IPv6 != nil {
		return ip.IPv6.String()
	}
	return "<invalid>"
}

// DualStackPacket represents either an IPv4 or IPv6 packet.
type DualStackPacket struct {
	IPv4 *ip.Packet
	IPv6 *Packet
}

// NewIPv4Packet creates a dual-stack packet from an IPv4 packet.
func NewIPv4Packet(pkt *ip.Packet) *DualStackPacket {
	return &DualStackPacket{IPv4: pkt}
}

// NewIPv6Packet creates a dual-stack packet from an IPv6 packet.
func NewIPv6Packet(pkt *Packet) *DualStackPacket {
	return &DualStackPacket{IPv6: pkt}
}

// IsIPv4 returns true if this is an IPv4 packet.
func (p *DualStackPacket) IsIPv4() bool {
	return p.IPv4 != nil
}

// IsIPv6 returns true if this is an IPv6 packet.
func (p *DualStackPacket) IsIPv6() bool {
	return p.IPv6 != nil
}

// Serialize serializes the packet.
func (p *DualStackPacket) Serialize() ([]byte, error) {
	if p.IPv4 != nil {
		return p.IPv4.Serialize()
	}
	if p.IPv6 != nil {
		return p.IPv6.Serialize()
	}
	return nil, fmt.Errorf("invalid dual-stack packet")
}

// GetProtocol returns the next protocol.
func (p *DualStackPacket) GetProtocol() common.Protocol {
	if p.IPv4 != nil {
		return p.IPv4.Protocol
	}
	if p.IPv6 != nil {
		return p.IPv6.NextHeader
	}
	return 0
}

// GetPayload returns the packet payload.
func (p *DualStackPacket) GetPayload() []byte {
	if p.IPv4 != nil {
		return p.IPv4.Payload
	}
	if p.IPv6 != nil {
		return p.IPv6.Payload
	}
	return nil
}

// String returns a string representation of the packet.
func (p *DualStackPacket) String() string {
	if p.IPv4 != nil {
		return p.IPv4.String()
	}
	if p.IPv6 != nil {
		return p.IPv6.String()
	}
	return "<invalid>"
}
