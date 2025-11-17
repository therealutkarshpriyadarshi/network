package ethernet

import (
	"fmt"
	"net"
	"syscall"

	"github.com/therealutkarshpriyadarshi/network/pkg/common"
)

// Interface represents a network interface for sending and receiving Ethernet frames.
type Interface struct {
	name       string
	fd         int               // Raw socket file descriptor
	macAddress common.MACAddress // Hardware address of this interface
	index      int               // Interface index
}

// OpenInterface opens a network interface for raw packet capture and transmission.
// This requires root/sudo privileges on Linux.
//
// The interface parameter is the name of the network interface (e.g., "eth0", "wlan0").
func OpenInterface(ifname string) (*Interface, error) {
	// Get interface information
	iface, err := net.InterfaceByName(ifname)
	if err != nil {
		return nil, fmt.Errorf("failed to get interface %s: %w", ifname, err)
	}

	// Parse MAC address
	if len(iface.HardwareAddr) != 6 {
		return nil, fmt.Errorf("invalid MAC address length: %d", len(iface.HardwareAddr))
	}
	var mac common.MACAddress
	copy(mac[:], iface.HardwareAddr)

	// Create raw socket
	// AF_PACKET: Packet socket for device level access
	// SOCK_RAW: Raw protocol access
	// ETH_P_ALL: Capture all protocols
	fd, err := syscall.Socket(syscall.AF_PACKET, syscall.SOCK_RAW, int(htons(syscall.ETH_P_ALL)))
	if err != nil {
		return nil, fmt.Errorf("failed to create raw socket: %w (you may need root/sudo)", err)
	}

	// Bind socket to specific interface
	addr := syscall.SockaddrLinklayer{
		Protocol: htons(syscall.ETH_P_ALL),
		Ifindex:  iface.Index,
	}

	if err := syscall.Bind(fd, &addr); err != nil {
		syscall.Close(fd)
		return nil, fmt.Errorf("failed to bind socket to interface: %w", err)
	}

	return &Interface{
		name:       ifname,
		fd:         fd,
		macAddress: mac,
		index:      iface.Index,
	}, nil
}

// Close closes the network interface.
func (i *Interface) Close() error {
	if i.fd >= 0 {
		return syscall.Close(i.fd)
	}
	return nil
}

// Name returns the interface name.
func (i *Interface) Name() string {
	return i.name
}

// MACAddress returns the hardware address of this interface.
func (i *Interface) MACAddress() common.MACAddress {
	return i.macAddress
}

// Index returns the interface index.
func (i *Interface) Index() int {
	return i.index
}

// ReadFrame reads an Ethernet frame from the interface.
// This is a blocking call that waits for incoming packets.
func (i *Interface) ReadFrame() (*Frame, error) {
	// Buffer for receiving packet (max Ethernet frame size)
	buf := make([]byte, MaxFrameSize)

	// Read from socket
	n, _, err := syscall.Recvfrom(i.fd, buf, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to receive packet: %w", err)
	}

	// Parse the frame
	frame, err := Parse(buf[:n])
	if err != nil {
		return nil, fmt.Errorf("failed to parse frame: %w", err)
	}

	return frame, nil
}

// WriteFrame sends an Ethernet frame to the interface.
func (i *Interface) WriteFrame(frame *Frame) error {
	// Serialize frame
	data := frame.Serialize()

	// Send to socket
	addr := syscall.SockaddrLinklayer{
		Protocol: htons(syscall.ETH_P_ALL),
		Ifindex:  i.index,
		Halen:    6,
	}
	copy(addr.Addr[:], frame.Destination[:])

	err := syscall.Sendto(i.fd, data, 0, &addr)
	if err != nil {
		return fmt.Errorf("failed to send frame: %w", err)
	}

	return nil
}

// SetPromiscuous enables or disables promiscuous mode on the interface.
// In promiscuous mode, the interface captures all packets on the network,
// not just those addressed to it.
func (i *Interface) SetPromiscuous(enable bool) error {
	// Get current interface flags
	iface, err := net.InterfaceByName(i.name)
	if err != nil {
		return err
	}

	// Create ifreq structure for ioctl
	// Note: This is a simplified version. Full implementation would use
	// syscall.IFF_PROMISC flag via ioctl SIOCSIFFLAGS
	// For now, we'll document that this requires additional privileges

	if enable {
		// Would set IFF_PROMISC flag
		_ = iface // Placeholder to avoid unused variable
		return fmt.Errorf("promiscuous mode not fully implemented")
	}

	return nil
}

// htons converts a 16-bit integer from host byte order to network byte order (big endian).
func htons(v uint16) uint16 {
	// On little-endian systems, we need to swap bytes
	// On big-endian systems, this is a no-op
	// Go's binary.BigEndian handles this correctly
	return (v << 8) | (v >> 8)
}

// ListInterfaces returns a list of all network interfaces on the system.
func ListInterfaces() ([]string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(ifaces))
	for _, iface := range ifaces {
		// Skip loopback and down interfaces
		if iface.Flags&net.FlagLoopback != 0 || iface.Flags&net.FlagUp == 0 {
			continue
		}
		names = append(names, iface.Name)
	}

	return names, nil
}

// GetInterfaceInfo returns detailed information about a network interface.
func GetInterfaceInfo(ifname string) (string, error) {
	iface, err := net.InterfaceByName(ifname)
	if err != nil {
		return "", err
	}

	info := fmt.Sprintf("Interface: %s\n", iface.Name)
	info += fmt.Sprintf("  Index: %d\n", iface.Index)
	info += fmt.Sprintf("  MTU: %d\n", iface.MTU)
	info += fmt.Sprintf("  Hardware Addr: %s\n", iface.HardwareAddr)
	info += fmt.Sprintf("  Flags: %s\n", iface.Flags)

	// Get IP addresses
	addrs, err := iface.Addrs()
	if err == nil && len(addrs) > 0 {
		info += "  Addresses:\n"
		for _, addr := range addrs {
			info += fmt.Sprintf("    %s\n", addr)
		}
	}

	return info, nil
}
