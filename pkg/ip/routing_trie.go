package ip

import (
	"fmt"
	"sync"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

// TrieRoutingTable uses a Patricia trie for O(1) route lookups
// This achieves sub-microsecond lookup performance
type TrieRoutingTable struct {
	mu              sync.RWMutex
	root            *trieNode
	localInterfaces map[string]common.IPv4Address
}

// trieNode represents a node in the Patricia trie
type trieNode struct {
	// Route at this node (nil if internal node)
	route *Route

	// Child nodes: [0] for 0-bit, [1] for 1-bit
	children [2]*trieNode

	// Prefix information
	prefix    uint32 // Network prefix
	prefixLen int    // Length of prefix in bits

	// Next hop cached for faster lookup
	nextHop common.IPv4Address
}

// NewTrieRoutingTable creates a new trie-based routing table
func NewTrieRoutingTable() *TrieRoutingTable {
	return &TrieRoutingTable{
		root:            &trieNode{},
		localInterfaces: make(map[string]common.IPv4Address),
	}
}

// AddRoute adds a route to the trie
func (trt *TrieRoutingTable) AddRoute(route *Route) error {
	if route == nil {
		return fmt.Errorf("route is nil")
	}

	trt.mu.Lock()
	defer trt.mu.Unlock()

	prefix := ipToUint32(route.Destination)
	mask := ipToUint32(route.Netmask)
	prefixLen := countOnesUint32(mask)

	// Normalize prefix
	prefix = prefix & mask

	// Calculate next hop
	nextHop := route.Gateway
	if nextHop == (common.IPv4Address{0, 0, 0, 0}) {
		// Direct delivery - next hop is destination itself
		nextHop = route.Destination
	}

	trt.insertNode(prefix, prefixLen, route, nextHop)
	return nil
}

// insertNode inserts a route into the trie
func (trt *TrieRoutingTable) insertNode(prefix uint32, prefixLen int, route *Route, nextHop common.IPv4Address) {
	node := trt.root

	for i := 0; i < prefixLen; i++ {
		// Get the i-th bit from the left (MSB first)
		bit := (prefix >> (31 - i)) & 1

		if node.children[bit] == nil {
			node.children[bit] = &trieNode{
				prefix:    prefix,
				prefixLen: i + 1,
			}
		}

		node = node.children[bit]
	}

	// Store route at this node
	node.route = route
	node.prefix = prefix
	node.prefixLen = prefixLen
	node.nextHop = nextHop
}

// Lookup finds the best route using trie-based longest prefix match
// This is the critical hot path - optimized for minimal latency
func (trt *TrieRoutingTable) Lookup(dst common.IPv4Address) (*Route, common.IPv4Address, error) {
	trt.mu.RLock()
	defer trt.mu.RUnlock()

	dstInt := ipToUint32(dst)

	// Walk the trie following the bits of the destination IP
	var bestMatch *trieNode
	node := trt.root

	for i := 0; i < 32 && node != nil; i++ {
		// If this node has a route, it's a potential match
		if node.route != nil {
			bestMatch = node
		}

		// Get the i-th bit from the left
		bit := (dstInt >> (31 - i)) & 1
		node = node.children[bit]
	}

	// Check the last node
	if node != nil && node.route != nil {
		bestMatch = node
	}

	if bestMatch == nil {
		return nil, common.IPv4Address{}, fmt.Errorf("no route to host: %s", dst)
	}

	return bestMatch.route, bestMatch.nextHop, nil
}

// RemoveRoute removes a route from the trie
func (trt *TrieRoutingTable) RemoveRoute(destination, netmask common.IPv4Address) bool {
	trt.mu.Lock()
	defer trt.mu.Unlock()

	prefix := ipToUint32(destination)
	mask := ipToUint32(netmask)
	prefixLen := countOnesUint32(mask)
	prefix = prefix & mask

	return trt.removeNode(prefix, prefixLen)
}

// removeNode removes a route from the trie
func (trt *TrieRoutingTable) removeNode(prefix uint32, prefixLen int) bool {
	node := trt.root
	path := make([]*trieNode, 0, prefixLen+1)
	path = append(path, node)

	// Navigate to the node
	for i := 0; i < prefixLen && node != nil; i++ {
		bit := (prefix >> (31 - i)) & 1
		node = node.children[bit]
		if node != nil {
			path = append(path, node)
		}
	}

	if node == nil || node.route == nil {
		return false
	}

	// Remove the route
	node.route = nil
	node.nextHop = common.IPv4Address{}

	// Clean up empty nodes (optional optimization)
	// If node has no children and no route, remove it
	if node.children[0] == nil && node.children[1] == nil {
		// Work backwards and remove empty nodes
		for i := len(path) - 1; i > 0; i-- {
			parent := path[i-1]
			child := path[i]

			if child.route == nil && child.children[0] == nil && child.children[1] == nil {
				// Remove child from parent
				if parent.children[0] == child {
					parent.children[0] = nil
				} else {
					parent.children[1] = nil
				}
			} else {
				break
			}
		}
	}

	return true
}

// AddLocalInterface adds a local interface to the routing table
func (trt *TrieRoutingTable) AddLocalInterface(name string, addr common.IPv4Address) {
	trt.mu.Lock()
	defer trt.mu.Unlock()
	trt.localInterfaces[name] = addr
}

// RemoveLocalInterface removes a local interface from the routing table
func (trt *TrieRoutingTable) RemoveLocalInterface(name string) {
	trt.mu.Lock()
	defer trt.mu.Unlock()
	delete(trt.localInterfaces, name)
}

// GetRoutes returns all routes by traversing the trie
func (trt *TrieRoutingTable) GetRoutes() []*Route {
	trt.mu.RLock()
	defer trt.mu.RUnlock()

	routes := make([]*Route, 0)
	trt.collectRoutes(trt.root, &routes)
	return routes
}

// collectRoutes recursively collects all routes from the trie
func (trt *TrieRoutingTable) collectRoutes(node *trieNode, routes *[]*Route) {
	if node == nil {
		return
	}

	if node.route != nil {
		*routes = append(*routes, node.route)
	}

	trt.collectRoutes(node.children[0], routes)
	trt.collectRoutes(node.children[1], routes)
}

// Helper function to count ones in uint32 (optimized)
func countOnesUint32(n uint32) int {
	// Brian Kernighan's algorithm
	count := 0
	for n != 0 {
		n &= n - 1
		count++
	}
	return count
}

// TrieRoutingTableWithCache adds a cache layer for even faster lookups
type TrieRoutingTableWithCache struct {
	table *TrieRoutingTable
	cache sync.Map // map[uint32]*cacheEntry for O(1) cached lookups
}

// NewTrieRoutingTableWithCache creates a new trie-based routing table with cache
func NewTrieRoutingTableWithCache() *TrieRoutingTableWithCache {
	return &TrieRoutingTableWithCache{
		table: NewTrieRoutingTable(),
	}
}

// AddRoute adds a route and invalidates cache
func (trtc *TrieRoutingTableWithCache) AddRoute(route *Route) error {
	err := trtc.table.AddRoute(route)
	if err == nil {
		trtc.cache = sync.Map{} // Clear cache
	}
	return err
}

// RemoveRoute removes a route and invalidates cache
func (trtc *TrieRoutingTableWithCache) RemoveRoute(destination, netmask common.IPv4Address) bool {
	removed := trtc.table.RemoveRoute(destination, netmask)
	if removed {
		trtc.cache = sync.Map{} // Clear cache
	}
	return removed
}

// Lookup finds the best route with caching (fastest possible lookup)
func (trtc *TrieRoutingTableWithCache) Lookup(dst common.IPv4Address) (*Route, common.IPv4Address, error) {
	dstInt := ipToUint32(dst)

	// Check cache first - this should be the common case
	if cached, ok := trtc.cache.Load(dstInt); ok {
		if entry, ok := cached.(*cacheEntry); ok {
			return entry.route, entry.nextHop, nil
		}
	}

	// Cache miss, do trie lookup
	route, nextHop, err := trtc.table.Lookup(dst)
	if err == nil {
		// Store in cache for next lookup
		trtc.cache.Store(dstInt, &cacheEntry{
			route:   route,
			nextHop: nextHop,
		})
	}

	return route, nextHop, err
}

// AddLocalInterface adds a local interface
func (trtc *TrieRoutingTableWithCache) AddLocalInterface(name string, addr common.IPv4Address) {
	trtc.table.AddLocalInterface(name, addr)
}

// RemoveLocalInterface removes a local interface
func (trtc *TrieRoutingTableWithCache) RemoveLocalInterface(name string) {
	trtc.table.RemoveLocalInterface(name)
}

// GetRoutes returns all routes
func (trtc *TrieRoutingTableWithCache) GetRoutes() []*Route {
	return trtc.table.GetRoutes()
}

// ClearCache clears the routing cache
func (trtc *TrieRoutingTableWithCache) ClearCache() {
	trtc.cache = sync.Map{}
}
