package ip

import (
	"fmt"
	"sort"
	"sync"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

// OptimizedRoutingTable uses a sorted list for faster lookups
type OptimizedRoutingTable struct {
	mu              sync.RWMutex
	routes          []*Route
	sortedRoutes    []*routeEntry // Sorted by prefix length (longest first)
	defaultGateway  *Route
	localInterfaces map[string]common.IPv4Address
	dirty           bool // Indicates routes need resorting
}

type routeEntry struct {
	route      *Route
	prefixLen  int
	networkInt uint32 // Network address as uint32 for faster comparison
	maskInt    uint32 // Netmask as uint32
}

// NewOptimizedRoutingTable creates a new optimized routing table
func NewOptimizedRoutingTable() *OptimizedRoutingTable {
	return &OptimizedRoutingTable{
		routes:          make([]*Route, 0),
		sortedRoutes:    make([]*routeEntry, 0),
		localInterfaces: make(map[string]common.IPv4Address),
		dirty:           false,
	}
}

// AddRoute adds a route to the routing table
func (rt *OptimizedRoutingTable) AddRoute(route *Route) error {
	if route == nil {
		return fmt.Errorf("route is nil")
	}

	rt.mu.Lock()
	defer rt.mu.Unlock()

	// Check if this is a default route
	if route.Destination == (common.IPv4Address{0, 0, 0, 0}) &&
		route.Netmask == (common.IPv4Address{0, 0, 0, 0}) {
		rt.defaultGateway = route
	}

	rt.routes = append(rt.routes, route)
	rt.dirty = true

	return nil
}

// RemoveRoute removes a route from the routing table
func (rt *OptimizedRoutingTable) RemoveRoute(destination, netmask common.IPv4Address) bool {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	for i, route := range rt.routes {
		if route.Destination == destination && route.Netmask == netmask {
			rt.routes = append(rt.routes[:i], rt.routes[i+1:]...)

			if rt.defaultGateway == route {
				rt.defaultGateway = nil
			}

			rt.dirty = true
			return true
		}
	}

	return false
}

// Lookup finds the best route using optimized longest prefix match
func (rt *OptimizedRoutingTable) Lookup(dst common.IPv4Address) (*Route, common.IPv4Address, error) {
	rt.mu.RLock()

	// Rebuild sorted routes if dirty
	if rt.dirty {
		rt.mu.RUnlock()
		rt.mu.Lock()
		if rt.dirty { // Double-check after acquiring write lock
			rt.rebuildSortedRoutes()
			rt.dirty = false
		}
		rt.mu.Unlock()
		rt.mu.RLock()
	}

	defer rt.mu.RUnlock()

	dstInt := ipToUint32(dst)

	// Search sorted routes (longest prefix first)
	for _, entry := range rt.sortedRoutes {
		// Check if destination matches this route's network
		if (dstInt & entry.maskInt) == entry.networkInt {
			// Found a match
			nextHop := entry.route.Gateway
			if nextHop == (common.IPv4Address{0, 0, 0, 0}) {
				// Direct delivery
				nextHop = dst
			}
			return entry.route, nextHop, nil
		}
	}

	return nil, common.IPv4Address{}, fmt.Errorf("no route to host: %s", dst)
}

// rebuildSortedRoutes rebuilds the sorted route list (must hold write lock)
func (rt *OptimizedRoutingTable) rebuildSortedRoutes() {
	rt.sortedRoutes = make([]*routeEntry, 0, len(rt.routes))

	for _, route := range rt.routes {
		entry := &routeEntry{
			route:      route,
			prefixLen:  countOnes(route.Netmask),
			networkInt: ipToUint32(route.Destination) & ipToUint32(route.Netmask),
			maskInt:    ipToUint32(route.Netmask),
		}
		rt.sortedRoutes = append(rt.sortedRoutes, entry)
	}

	// Sort by prefix length (longest first), then by metric (lower first)
	sort.Slice(rt.sortedRoutes, func(i, j int) bool {
		if rt.sortedRoutes[i].prefixLen != rt.sortedRoutes[j].prefixLen {
			return rt.sortedRoutes[i].prefixLen > rt.sortedRoutes[j].prefixLen
		}
		return rt.sortedRoutes[i].route.Metric < rt.sortedRoutes[j].route.Metric
	})
}

// AddLocalInterface adds a local interface to the routing table
func (rt *OptimizedRoutingTable) AddLocalInterface(name string, addr common.IPv4Address) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.localInterfaces[name] = addr
}

// RemoveLocalInterface removes a local interface from the routing table
func (rt *OptimizedRoutingTable) RemoveLocalInterface(name string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	delete(rt.localInterfaces, name)
}

// GetRoutes returns a copy of all routes
func (rt *OptimizedRoutingTable) GetRoutes() []*Route {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	routes := make([]*Route, len(rt.routes))
	copy(routes, rt.routes)
	return routes
}

// Helper functions

func ipToUint32(addr common.IPv4Address) uint32 {
	return uint32(addr[0])<<24 | uint32(addr[1])<<16 | uint32(addr[2])<<8 | uint32(addr[3])
}

func countOnes(mask common.IPv4Address) int {
	count := 0
	for _, b := range mask {
		for b != 0 {
			count += int(b & 1)
			b >>= 1
		}
	}
	return count
}

// CachedRoutingTable adds a cache layer on top of OptimizedRoutingTable
type CachedRoutingTable struct {
	table *OptimizedRoutingTable
	cache sync.Map // map[common.IPv4Address]*Route
}

// NewCachedRoutingTable creates a new routing table with caching
func NewCachedRoutingTable() *CachedRoutingTable {
	return &CachedRoutingTable{
		table: NewOptimizedRoutingTable(),
	}
}

// AddRoute adds a route and invalidates cache
func (crt *CachedRoutingTable) AddRoute(route *Route) error {
	err := crt.table.AddRoute(route)
	if err == nil {
		crt.cache = sync.Map{} // Clear cache
	}
	return err
}

// RemoveRoute removes a route and invalidates cache
func (crt *CachedRoutingTable) RemoveRoute(destination, netmask common.IPv4Address) bool {
	removed := crt.table.RemoveRoute(destination, netmask)
	if removed {
		crt.cache = sync.Map{} // Clear cache
	}
	return removed
}

// Lookup finds the best route with caching
func (crt *CachedRoutingTable) Lookup(dst common.IPv4Address) (*Route, common.IPv4Address, error) {
	// Check cache first
	if cached, ok := crt.cache.Load(dst); ok {
		if entry, ok := cached.(*cacheEntry); ok {
			return entry.route, entry.nextHop, nil
		}
	}

	// Cache miss, do actual lookup
	route, nextHop, err := crt.table.Lookup(dst)
	if err == nil {
		// Store in cache
		crt.cache.Store(dst, &cacheEntry{
			route:   route,
			nextHop: nextHop,
		})
	}

	return route, nextHop, err
}

// AddLocalInterface adds a local interface
func (crt *CachedRoutingTable) AddLocalInterface(name string, addr common.IPv4Address) {
	crt.table.AddLocalInterface(name, addr)
}

// RemoveLocalInterface removes a local interface
func (crt *CachedRoutingTable) RemoveLocalInterface(name string) {
	crt.table.RemoveLocalInterface(name)
}

// GetRoutes returns all routes
func (crt *CachedRoutingTable) GetRoutes() []*Route {
	return crt.table.GetRoutes()
}

type cacheEntry struct {
	route   *Route
	nextHop common.IPv4Address
}

// ClearCache clears the routing cache
func (crt *CachedRoutingTable) ClearCache() {
	crt.cache = sync.Map{}
}
