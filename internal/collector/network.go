//go:build windows

package collector

import (
	"fmt"
	"net"
	"syscall"
	"unsafe"

	"github.com/alme23/tracker/internal/models"
	"golang.org/x/sys/windows"
)

type NetworkCollector struct{}

func NewNetworkCollector() *NetworkCollector {
	return &NetworkCollector{}
}

func (c *NetworkCollector) Collect() (models.NetworkStatuses, error) {
	var result models.NetworkStatuses

	flags := uint32(windows.GAA_FLAG_INCLUDE_PREFIX)
	var size uint32 = 15000

	var buf []byte
	var err error

	for {
		buf = make([]byte, size)

		// ИСПРАВЛЕНО: Передаем указатель на ПЕРВЫЙ ЭЛЕМЕНТ данных (&buf[0])
		err = windows.GetAdaptersAddresses(
			syscall.AF_UNSPEC,
			flags,
			0,
			(*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0])), // Исправлено здесь
			&size,
		)
		if err == nil {
			break
		}
		if err != windows.ERROR_BUFFER_OVERFLOW {
			return nil, fmt.Errorf("ошибка WinAPI GetAdaptersAddresses: %v", err)
		}
	}

	adapter := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
	for adapter != nil {
		name := windows.BytePtrToString(adapter.AdapterName)
		description := windows.UTF16PtrToString(adapter.Description)

		var macStr string
		if adapter.PhysicalAddressLength > 0 {
			// Отрезаем от массива байт только реальную длину MAC-адреса
			macStr = net.HardwareAddr(adapter.PhysicalAddress[:adapter.PhysicalAddressLength]).String()
		}

		var netType models.InterfaceType
		switch adapter.IfType {
		case windows.IF_TYPE_ETHERNET_CSMACD:
			netType = models.TypeEthernet
		case windows.IF_TYPE_IEEE80211:
			netType = models.TypeWireless
		case windows.IF_TYPE_SOFTWARE_LOOPBACK:
			netType = models.TypeLoopback
		case windows.IF_TYPE_TUNNEL:
			netType = models.TypeTunnel
		case windows.IF_TYPE_PPP:
			netType = models.TypePPP
		case windows.IF_TYPE_OTHER:
			netType = models.TypeOther
		default:
			netType = models.TypeUnknown
		}

		// --- ОПРЕДЕЛЕНИЕ СПОСОБА ПОЛУЧЕНИЯ IP (DHCP или СТАТИКА) ---
		var ipAssignment models.IPAssignment
		if netType == models.TypeLoopback { // ИСПРАВЛЕНО: сравнение типов происходит мгновенно
			ipAssignment = models.AssignmentNotApps
		} else {
			if (adapter.Flags & 0x0004) != 0 {
				ipAssignment = models.AssignmentDHCP
			} else {
				ipAssignment = models.AssignmentStatic
			}
		}

		var ipAddresses []net.IP
		unicastAddr := adapter.FirstUnicastAddress
		for unicastAddr != nil {
			sa, err := unicastAddr.Address.Sockaddr.Sockaddr()
			if err == nil {
				switch v := sa.(type) {
				case *syscall.SockaddrInet4:
					// Создаем net.IP из 4 байт IPv4.
					// v.Addr[:] копирует байты напрямую без выделения тяжелых строк
					ip := net.IP(v.Addr[:])
					ipAddresses = append(ipAddresses, ip)

				case *syscall.SockaddrInet6:
					// Создаем net.IP из 16 байт IPv6
					ip := net.IP(v.Addr[:])
					ipAddresses = append(ipAddresses, ip)
				}
			}
			unicastAddr = unicastAddr.Next
		}

		isOperational := adapter.OperStatus == windows.IfOperStatusUp

		result = append(result, models.InterfaceInfo{
			Index:        int(adapter.IfIndex),
			Name:         name,
			Description:  description,
			MAC:          macStr,
			Type:         netType,
			Operational:  isOperational,
			IPAssignment: ipAssignment,
			IPAddresses:  ipAddresses,
		})

		adapter = adapter.Next
	}

	return result, nil
}
