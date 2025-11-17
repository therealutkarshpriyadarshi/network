// Package ip implements IP routing table functionality.
package ip

import (
	"fmt"
	"net"
	"sync"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

// Route represents a routing table entry.
type Route struct {
	Destination common.IPv4Address // Destination network
	Netmask     common.IPv4Address // Network mask
	Gateway     common.IPv4Address // Next hop gateway (0.0.0.0 for direct)
	Interface   string             // Network interface name
	Metric      int                // Route metric (lower is better)
}

// RoutingTable manages IP routes.
type RoutingTable struct {
	mu              sync.RWMutex
	routes          []*Route
	defaultGateway  *Route
	localInterfaces map[string]common.IPv4Address // interface name -> IP address
}

// NewRoutingTable creates a new routing table.
func NewRoutingTable() *RoutingTable {
	return &RoutingTable{
		routes:          make([]*Route, 0),
		localInterfaces: make(map[string]common.IPv4Address),
	}
}

// AddRoute adds a route to the routing table.
func (rt *RoutingTable) AddRoute(route *Route) error {
	if route == nil {
		return fmt.Errorf("route is nil")
	}

	rt.mu.Lock()
	defer rt.mu.Unlock()

	// Check if this is a default route (0.0.0.0/0)
	if route.Destination == (common.IPv4Address{0, 0, 0, 0}) &&
		route.Netmask == (common.IPv4Address{0, 0, 0, 0}) {
		rt.defaultGateway = route
	}

	rt.routes = append(rt.routes, route)
	return nil
}

// RemoveRoute removes a route from the routing table.
func (rt *RoutingTable) RemoveRoute(destination, netmask common.IPv4Address) bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	for i, route := range rt.routes {
		if route.Destination == destination && route.Netmask == netmask {
			// Remove route
			rt.routes = append(rt.routes[:i], rt.routes[i+1:]...)

			// Clear default gateway if this was it
			if rt.defaultGateway == route {
				rt.defaultGateway = nil
			}

			return true
		}
	}

	return false
}

// Lookup finds the best route for a destination IP address.
// Returns the route and next hop IP address.
func (rt *RoutingTable) Lookup(dst common.IPv4Address) (*Route, common.IPv4Address, error) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	var bestRoute *Route
	var bestPrefixLen int = -1

	// Find the most specific route (longest prefix match)
	for _, route := range rt.routes {
		if rt.matches(dst, route.Destination, route.Netmask) {
			prefixLen := rt.countOnes(route.Netmask)
			if prefixLen > bestPrefixLen {
				bestRoute = route
				bestPrefixLen = prefixLen
			}
		}
	}

	if bestRoute == nil {
		return nil, common.IPv4Address{}, fmt.Errorf("no route to host: %s", dst)
	}

	// Determine next hop
	nextHop := dst
	if bestRoute.Gateway != (common.IPv4Address{0, 0, 0, 0}) {
		// Use gateway
		nextHop = bestRoute.Gateway
	}

	return bestRoute, nextHop, nil
}

// matches checks if an IP address matches a network (destination & netmask).
func (rt *RoutingTable) matches(ip, network, netmask common.IPv4Address) bool {
	for i := 0; i < 4; i++ {
		if (ip[i] & netmask[i]) != (network[i] & netmask[i]) {
			return false
		}
	}
	return true
}

// countOnes counts the number of 1 bits in a netmask (prefix length).
func (rt *RoutingTable) countOnes(netmask common.IPv4Address) int {
	count := 0
	for i := 0; i < 4; i++ {
		b := netmask[i]
		for b != 0 {
			count += int(b & 1)
			b >>= 1
		}
	}
	return count
}

// SetDefaultGateway sets the default gateway.
func (rt *RoutingTable) SetDefaultGateway(gateway common.IPv4Address, iface string) error {
	route := &Route{
		Destination: common.IPv4Address{0, 0, 0, 0},
		Netmask:     common.IPv4Address{0, 0, 0, 0},
		Gateway:     gateway,
		Interface:   iface,
		Metric:      0,
	}
	return rt.AddRoute(route)
}

// GetDefaultGateway returns the default gateway route.
func (rt *RoutingTable) GetDefaultGateway() *Route {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.defaultGateway
}

// AddLocalInterface registers a local network interface.
func (rt *RoutingTable) AddLocalInterface(iface string, ip common.IPv4Address) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.localInterfaces[iface] = ip
}

// GetLocalInterface returns the IP address for a local interface.
func (rt *RoutingTable) GetLocalInterface(iface string) (common.IPv4Address, bool) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	ip, exists := rt.localInterfaces[iface]
	return ip, exists
}

// IsLocalAddress checks if an IP address belongs to a local interface.
func (rt *RoutingTable) IsLocalAddress(ip common.IPv4Address) bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	for _, localIP := range rt.localInterfaces {
		if localIP == ip {
			return true
		}
	}
	return false
}

// GetRoutes returns all routes in the routing table.
func (rt *RoutingTable) GetRoutes() []*Route {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	routes := make([]*Route, len(rt.routes))
	copy(routes, rt.routes)
	return routes
}

// String returns a human-readable representation of the routing table.
func (rt *RoutingTable) String() string {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	s := "Routing Table:\n"
	s += "Destination     Netmask         Gateway         Interface  Metric\n"
	s += "-------------------------------------------------------------------\n"

	for _, route := range rt.routes {
		gateway := route.Gateway.String()
		if route.Gateway == (common.IPv4Address{0, 0, 0, 0}) {
			gateway = "direct"
		}

		s += fmt.Sprintf("%-15s %-15s %-15s %-10s %d\n",
			route.Destination, route.Netmask, gateway, route.Interface, route.Metric)
	}

	return s
}

// LoadSystemRoutes loads routes from the system routing table (Linux).
func (rt *RoutingTable) LoadSystemRoutes() error {
	// This is a simplified version that uses the net package
	// In a real implementation, you might parse /proc/net/route directly

	ifaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("failed to get interfaces: %w", err)
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}

			ipv4 := ipNet.IP.To4()
			if ipv4 == nil {
				continue // Skip non-IPv4 addresses
			}

			// Register local interface
			var localIP common.IPv4Address
			copy(localIP[:], ipv4)
			rt.AddLocalInterface(iface.Name, localIP)

			// Add route for local network
			var network, netmask common.IPv4Address
			copy(network[:], ipNet.IP.To4())
			copy(netmask[:], ipNet.Mask)

			// Zero out host bits in network address
			for i := 0; i < 4; i++ {
				network[i] = network[i] & netmask[i]
			}

			route := &Route{
				Destination: network,
				Netmask:     netmask,
				Gateway:     common.IPv4Address{0, 0, 0, 0}, // Direct route
				Interface:   iface.Name,
				Metric:      0,
			}
			rt.AddRoute(route)
		}
	}

	return nil
}
