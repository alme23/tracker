//go:build windows

package collector

import (
	"errors"
	"fmt"
	"net"
	"syscall"
	"unsafe"

	"github.com/alme23/tracker/internal/models"
	"golang.org/x/sys/windows"
)

// NetworkCollector collects information about network adapters
type NetworkCollector struct{}

// NewNetworkCollector creates a new NetworkCollector
func NewNetworkCollector() *NetworkCollector {
	return &NetworkCollector{}
}

// Collect gathers information about all network adapters
func (c *NetworkCollector) Collect() (models.NetworkStatuses, error) {
	flags := uint32(windows.GAA_FLAG_INCLUDE_PREFIX)

	size := uint32(16384)
	var buf []byte

	const maxAttempts = 3
	var err error

	for range maxAttempts {
		buf = make([]byte, size)

		// #nosec G103 -- safe use of unsafe.SliceData with local buffer
		err = windows.GetAdaptersAddresses(
			syscall.AF_UNSPEC,
			flags,
			0,
			(*windows.IpAdapterAddresses)(unsafe.Pointer(unsafe.SliceData(buf))),
			&size,
		)
		if err == nil {
			break
		}
		if !errors.Is(err, windows.ERROR_BUFFER_OVERFLOW) {
			return nil, fmt.Errorf("WinAPI GetAdaptersAddresses error: %w", err)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("network buffer allocation attempts exceeded: %w", err)
	}

	// Count adapters for precise allocation
	adapterCount := 0
	// #nosec G103 -- safe use of unsafe.SliceData with local buffer
	curr := (*windows.IpAdapterAddresses)(unsafe.Pointer(unsafe.SliceData(buf)))
	for curr != nil {
		adapterCount++
		curr = curr.Next
	}

	result := make(models.NetworkStatuses, 0, adapterCount)

	// #nosec G103 -- safe use of unsafe.SliceData with local buffer
	adapter := (*windows.IpAdapterAddresses)(unsafe.Pointer(unsafe.SliceData(buf)))
	for adapter != nil {
		info := models.InterfaceInfo{
			Index:       int(adapter.IfIndex),
			Name:        windows.BytePtrToString(adapter.AdapterName),
			Description: windows.UTF16PtrToString(adapter.Description),
			Operational: adapter.OperStatus == windows.IfOperStatusUp,
		}

		// Interface type
		switch adapter.IfType {
		case windows.IF_TYPE_ETHERNET_CSMACD:
			info.Type = models.TypeEthernet
		case windows.IF_TYPE_IEEE80211:
			info.Type = models.TypeWireless
		case windows.IF_TYPE_SOFTWARE_LOOPBACK:
			info.Type = models.TypeLoopback
		case windows.IF_TYPE_TUNNEL:
			info.Type = models.TypeTunnel
		case windows.IF_TYPE_PPP:
			info.Type = models.TypePPP
		default:
			info.Type = models.TypeUnknown
		}

		// MAC address
		if adapter.PhysicalAddressLength > 0 && adapter.PhysicalAddressLength <= uint32(len(adapter.PhysicalAddress)) {
			info.MAC = net.HardwareAddr(adapter.PhysicalAddress[:adapter.PhysicalAddressLength]).String()
		}

		// IP assignment
		switch {
		case info.Type == models.TypeLoopback:
			info.IPAssignment = models.AssignmentNotApps
		case (adapter.Flags & 0x0004) != 0:
			info.IPAssignment = models.AssignmentDHCP
		default:
			info.IPAssignment = models.AssignmentStatic
		}

		// IP addresses
		if adapter.FirstUnicastAddress != nil {
			info.IPAddresses = make([]net.IP, 0, 2)
			unicastAddr := adapter.FirstUnicastAddress

			for unicastAddr != nil {
				sa, err := unicastAddr.Address.Sockaddr.Sockaddr()
				if err == nil {
					switch v := sa.(type) {
					case *syscall.SockaddrInet4:
						ip := make(net.IP, 4)
						copy(ip, v.Addr[:])
						info.IPAddresses = append(info.IPAddresses, ip)
					case *syscall.SockaddrInet6:
						ip := make(net.IP, 16)
						copy(ip, v.Addr[:])
						info.IPAddresses = append(info.IPAddresses, ip)
					}
				}
				unicastAddr = unicastAddr.Next
			}
		}

		result = append(result, info)
		adapter = adapter.Next
	}

	return result, nil
}
