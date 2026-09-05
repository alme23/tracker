// tracker/internal/collector/network.go

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

type NetworkCollector struct{}

func NewNetworkCollector() *NetworkCollector {
	return &NetworkCollector{}
}

func (c *NetworkCollector) Collect() (models.NetworkStatuses, error) {
	flags := uint32(windows.GAA_FLAG_INCLUDE_PREFIX)

	size := uint32(15000)
	var buf []byte

	const maxAttempts = 5
	var err error

	for range maxAttempts {
		buf = make([]byte, size)

		err = windows.GetAdaptersAddresses(
			syscall.AF_UNSPEC,
			flags,
			0,
			// Безопасный способ получить указатель на базовый массив слайса без риска panic
			(*windows.IpAdapterAddresses)(unsafe.Pointer(unsafe.SliceData(buf))),
			&size,
		)
		if err == nil {
			break
		}
		if !errors.Is(err, windows.ERROR_BUFFER_OVERFLOW) {
			return nil, fmt.Errorf("ошибка WinAPI GetAdaptersAddresses: %w", err)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("превышено количество попыток выделения буфера GetAdaptersAddresses: %w", err)
	}

	// Если буфер по какой-то причине пуст, прерываемся во избежание паники
	if len(buf) == 0 {
		return nil, fmt.Errorf("получен пустой буфер адресов адаптеров")
	}

	result := make(models.NetworkStatuses, 0, 8)

	// Используем безопасный указатель на данные слайса
	adapter := (*windows.IpAdapterAddresses)(unsafe.Pointer(unsafe.SliceData(buf)))
	for adapter != nil {
		info := models.InterfaceInfo{
			Index:       int(adapter.IfIndex),
			Name:        windows.BytePtrToString(adapter.AdapterName),
			Description: windows.UTF16PtrToString(adapter.Description),
			Operational: adapter.OperStatus == windows.IfOperStatusUp,
		}

		// Тип интерфейса
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

		// MAC-адрес
		if adapter.PhysicalAddressLength > 0 && adapter.PhysicalAddressLength <= uint32(len(adapter.PhysicalAddress)) {
			info.MAC = net.HardwareAddr(adapter.PhysicalAddress[:adapter.PhysicalAddressLength]).String()
		}

		// IP Assignment
		if info.Type == models.TypeLoopback {
			info.IPAssignment = models.AssignmentNotApps
		} else if (adapter.Flags & 0x0004) != 0 {
			info.IPAssignment = models.AssignmentDHCP
		} else {
			info.IPAssignment = models.AssignmentStatic
		}

		// IP-адреса
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
