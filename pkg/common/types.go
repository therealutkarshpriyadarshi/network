// Package common provides shared types and utilities used across the network stack.
package common

import (
	"encoding/binary"
	"fmt"
	"net"
)

// MACAddress represents a 48-bit hardware address.
type MACAddress [6]byte

// String returns the MAC address in standard format (e.g., "00:11:22:33:44:55").
func (m MACAddress) String() string {
	return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x",
		m[0], m[1], m[2], m[3], m[4], m[5])
}

// IsBroadcast returns true if this is a broadcast MAC address (FF:FF:FF:FF:FF:FF).
func (m MACAddress) IsBroadcast() bool {
	return m[0] == 0xFF && m[1] == 0xFF && m[2] == 0xFF &&
		m[3] == 0xFF && m[4] == 0xFF && m[5] == 0xFF
}

// IsMulticast returns true if the least significant bit of the first byte is 1.
func (m MACAddress) IsMulticast() bool {
	return m[0]&0x01 != 0
}

// ParseMAC parses a string MAC address (e.g., "00:11:22:33:44:55").
func ParseMAC(s string) (MACAddress, error) {
	hw, err := net.ParseMAC(s)
	if err != nil {
		return MACAddress{}, err
	}
	if len(hw) != 6 {
		return MACAddress{}, fmt.Errorf("invalid MAC address length: %d", len(hw))
	}
	var mac MACAddress
	copy(mac[:], hw)
	return mac, nil
}

// BroadcastMAC is the broadcast MAC address (FF:FF:FF:FF:FF:FF).
var BroadcastMAC = MACAddress{0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF}

// IPv4Address represents a 32-bit IPv4 address.
type IPv4Address [4]byte

// String returns the IP address in dotted decimal format (e.g., "192.168.1.1").
func (ip IPv4Address) String() string {
	return fmt.Sprintf("%d.%d.%d.%d", ip[0], ip[1], ip[2], ip[3])
}

// ToUint32 converts the IPv4 address to a uint32 in network byte order.
func (ip IPv4Address) ToUint32() uint32 {
	return binary.BigEndian.Uint32(ip[:])
}

// ParseIPv4 parses a string IPv4 address (e.g., "192.168.1.1").
func ParseIPv4(s string) (IPv4Address, error) {
	ip := net.ParseIP(s)
	if ip == nil {
		return IPv4Address{}, fmt.Errorf("invalid IP address: %s", s)
	}
	ip = ip.To4()
	if ip == nil {
		return IPv4Address{}, fmt.Errorf("not an IPv4 address: %s", s)
	}
	var addr IPv4Address
	copy(addr[:], ip)
	return addr, nil
}

// IPv4FromUint32 converts a uint32 to an IPv4 address.
func IPv4FromUint32(v uint32) IPv4Address {
	var addr IPv4Address
	binary.BigEndian.PutUint32(addr[:], v)
	return addr
}

// EtherType represents the protocol type in an Ethernet frame.
type EtherType uint16

// Common EtherType values.
const (
	EtherTypeIPv4 EtherType = 0x0800 // Internet Protocol version 4
	EtherTypeARP  EtherType = 0x0806 // Address Resolution Protocol
	EtherTypeIPv6 EtherType = 0x86DD // Internet Protocol version 6
)

// String returns a human-readable name for the EtherType.
func (et EtherType) String() string {
	switch et {
	case EtherTypeIPv4:
		return "IPv4"
	case EtherTypeARP:
		return "ARP"
	case EtherTypeIPv6:
		return "IPv6"
	default:
		return fmt.Sprintf("Unknown(0x%04x)", uint16(et))
	}
}

// Protocol represents the protocol number in an IP header.
type Protocol uint8

// Common protocol numbers.
const (
	ProtocolICMP Protocol = 1  // Internet Control Message Protocol
	ProtocolTCP  Protocol = 6  // Transmission Control Protocol
	ProtocolUDP  Protocol = 17 // User Datagram Protocol
)

// String returns a human-readable name for the protocol.
func (p Protocol) String() string {
	switch p {
	case ProtocolICMP:
		return "ICMP"
	case ProtocolTCP:
		return "TCP"
	case ProtocolUDP:
		return "UDP"
	default:
		return fmt.Sprintf("Unknown(%d)", uint8(p))
	}
}
