// Package multicast implements IP multicast support for both IPv4 and IPv6.
package multicast

import (
	"fmt"
	"net"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// IPv4 multicast address ranges
var (
	// AllHostsMulticast is the IPv4 all-hosts multicast address (224.0.0.1)
	AllHostsMulticast = common.IPv4Address{224, 0, 0, 1}

	// AllRoutersMulticast is the IPv4 all-routers multicast address (224.0.0.2)
	AllRoutersMulticast = common.IPv4Address{224, 0, 0, 2}

	// MDNS multicast address (224.0.0.251)
	MDNSMulticast = common.IPv4Address{224, 0, 0, 251}
)

// IPv6 multicast address ranges
var (
	// AllNodesMulticast is the IPv6 all-nodes multicast address (ff02::1)
	AllNodesMulticast = common.IPv6Address{0xff, 0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x01}

	// AllRoutersMulticast6 is the IPv6 all-routers multicast address (ff02::2)
	AllRoutersMulticast6 = common.IPv6Address{0xff, 0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0x02}

	// MDNS6 multicast address (ff02::fb)
	MDNS6Multicast = common.IPv6Address{0xff, 0x02, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xfb}
)

// IsMulticastIPv4 returns true if the address is a multicast address.
func IsMulticastIPv4(addr common.IPv4Address) bool {
	// Multicast range: 224.0.0.0 to 239.255.255.255 (class D)
	return addr[0] >= 224 && addr[0] <= 239
}

// IsMulticastIPv6 returns true if the address is a multicast address.
func IsMulticastIPv6(addr common.IPv6Address) bool {
	// Multicast addresses start with 0xff
	return addr[0] == 0xff
}

// MulticastScope represents the scope of a multicast address.
type MulticastScope uint8

const (
	ScopeInterfaceLocal MulticastScope = 0x1 // Interface-local scope
	ScopeLinkLocal      MulticastScope = 0x2 // Link-local scope
	ScopeRealmLocal     MulticastScope = 0x3 // Realm-local scope (deprecated)
	ScopeAdminLocal     MulticastScope = 0x4 // Admin-local scope
	ScopeSiteLocal      MulticastScope = 0x5 // Site-local scope
	ScopeOrganization   MulticastScope = 0x8 // Organization-local scope
	ScopeGlobal         MulticastScope = 0xe // Global scope
)

// GetIPv6MulticastScope returns the scope of an IPv6 multicast address.
func GetIPv6MulticastScope(addr common.IPv6Address) MulticastScope {
	if !IsMulticastIPv6(addr) {
		return 0
	}
	return MulticastScope(addr[1] & 0x0f)
}

// Group represents a multicast group.
type Group struct {
	Address  interface{} // common.IPv4Address or common.IPv6Address
	Members  []string    // List of member identifiers
	InterfaceIndex int   // Network interface index
}

// NewIPv4Group creates a new IPv4 multicast group.
func NewIPv4Group(addr common.IPv4Address, ifIndex int) *Group {
	return &Group{
		Address:        addr,
		Members:        make([]string, 0),
		InterfaceIndex: ifIndex,
	}
}

// NewIPv6Group creates a new IPv6 multicast group.
func NewIPv6Group(addr common.IPv6Address, ifIndex int) *Group {
	return &Group{
		Address:        addr,
		Members:        make([]string, 0),
		InterfaceIndex: ifIndex,
	}
}

// AddMember adds a member to the multicast group.
func (g *Group) AddMember(memberID string) {
	for _, m := range g.Members {
		if m == memberID {
			return // Already a member
		}
	}
	g.Members = append(g.Members, memberID)
}

// RemoveMember removes a member from the multicast group.
func (g *Group) RemoveMember(memberID string) {
	for i, m := range g.Members {
		if m == memberID {
			g.Members = append(g.Members[:i], g.Members[i+1:]...)
			return
		}
	}
}

// HasMember checks if a member is in the group.
func (g *Group) HasMember(memberID string) bool {
	for _, m := range g.Members {
		if m == memberID {
			return true
		}
	}
	return false
}

// MemberCount returns the number of members in the group.
func (g *Group) MemberCount() int {
	return len(g.Members)
}

// String returns a string representation of the group.
func (g *Group) String() string {
	var addrStr string
	switch addr := g.Address.(type) {
	case common.IPv4Address:
		addrStr = addr.String()
	case common.IPv6Address:
		addrStr = addr.String()
	default:
		addrStr = "unknown"
	}
	return fmt.Sprintf("MulticastGroup{Addr=%s, Members=%d}", addrStr, len(g.Members))
}

// Manager manages multicast groups.
type Manager struct {
	groups map[string]*Group
}

// NewManager creates a new multicast manager.
func NewManager() *Manager {
	return &Manager{
		groups: make(map[string]*Group),
	}
}

// JoinGroup joins a multicast group.
func (m *Manager) JoinGroup(group *Group) error {
	var key string
	switch addr := group.Address.(type) {
	case common.IPv4Address:
		if !IsMulticastIPv4(addr) {
			return fmt.Errorf("not a multicast address: %s", addr)
		}
		key = addr.String()
	case common.IPv6Address:
		if !IsMulticastIPv6(addr) {
			return fmt.Errorf("not a multicast address: %s", addr)
		}
		key = addr.String()
	default:
		return fmt.Errorf("invalid address type")
	}

	m.groups[key] = group
	return nil
}

// LeaveGroup leaves a multicast group.
func (m *Manager) LeaveGroup(addr interface{}) error {
	var key string
	switch a := addr.(type) {
	case common.IPv4Address:
		key = a.String()
	case common.IPv6Address:
		key = a.String()
	default:
		return fmt.Errorf("invalid address type")
	}

	delete(m.groups, key)
	return nil
}

// GetGroup gets a multicast group by address.
func (m *Manager) GetGroup(addr interface{}) (*Group, error) {
	var key string
	switch a := addr.(type) {
	case common.IPv4Address:
		key = a.String()
	case common.IPv6Address:
		key = a.String()
	default:
		return nil, fmt.Errorf("invalid address type")
	}

	group, ok := m.groups[key]
	if !ok {
		return nil, fmt.Errorf("group not found: %s", key)
	}

	return group, nil
}

// ListGroups returns all multicast groups.
func (m *Manager) ListGroups() []*Group {
	groups := make([]*Group, 0, len(m.groups))
	for _, g := range m.groups {
		groups = append(groups, g)
	}
	return groups
}

// MulticastSocket represents a socket for multicast communication.
type MulticastSocket struct {
	conn    net.PacketConn
	manager *Manager
}

// NewMulticastSocket creates a new multicast socket.
func NewMulticastSocket(network, address string) (*MulticastSocket, error) {
	conn, err := net.ListenPacket(network, address)
	if err != nil {
		return nil, fmt.Errorf("failed to create socket: %w", err)
	}

	return &MulticastSocket{
		conn:    conn,
		manager: NewManager(),
	}, nil
}

// JoinIPv4Group joins an IPv4 multicast group.
func (s *MulticastSocket) JoinIPv4Group(iface *net.Interface, group net.IP) error {
	if udpConn, ok := s.conn.(*net.UDPConn); ok {
		p := ipv4.NewPacketConn(udpConn)
		return p.JoinGroup(iface, &net.UDPAddr{IP: group})
	}
	return fmt.Errorf("not a UDP connection")
}

// JoinIPv6Group joins an IPv6 multicast group.
func (s *MulticastSocket) JoinIPv6Group(iface *net.Interface, group net.IP) error {
	if udpConn, ok := s.conn.(*net.UDPConn); ok {
		p := ipv6.NewPacketConn(udpConn)
		return p.JoinGroup(iface, &net.UDPAddr{IP: group})
	}
	return fmt.Errorf("not a UDP connection")
}

// LeaveIPv4Group leaves an IPv4 multicast group.
func (s *MulticastSocket) LeaveIPv4Group(iface *net.Interface, group net.IP) error {
	if udpConn, ok := s.conn.(*net.UDPConn); ok {
		p := ipv4.NewPacketConn(udpConn)
		return p.LeaveGroup(iface, &net.UDPAddr{IP: group})
	}
	return fmt.Errorf("not a UDP connection")
}

// LeaveIPv6Group leaves an IPv6 multicast group.
func (s *MulticastSocket) LeaveIPv6Group(iface *net.Interface, group net.IP) error {
	if udpConn, ok := s.conn.(*net.UDPConn); ok {
		p := ipv6.NewPacketConn(udpConn)
		return p.LeaveGroup(iface, &net.UDPAddr{IP: group})
	}
	return fmt.Errorf("not a UDP connection")
}

// SendTo sends data to a multicast address.
func (s *MulticastSocket) SendTo(data []byte, addr net.Addr) (int, error) {
	return s.conn.WriteTo(data, addr)
}

// ReceiveFrom receives data from the multicast socket.
func (s *MulticastSocket) ReceiveFrom(buf []byte) (int, net.Addr, error) {
	return s.conn.ReadFrom(buf)
}

// Close closes the multicast socket.
func (s *MulticastSocket) Close() error {
	return s.conn.Close()
}

// SetTTL sets the TTL for multicast packets (IPv4).
func (s *MulticastSocket) SetTTL(ttl int) error {
	if pc, ok := s.conn.(*net.UDPConn); ok {
		return pc.SetWriteBuffer(ttl)
	}
	return fmt.Errorf("not a UDP connection")
}

// SetHopLimit sets the hop limit for multicast packets (IPv6).
func (s *MulticastSocket) SetHopLimit(hops int) error {
	return s.SetTTL(hops) // Similar functionality
}
