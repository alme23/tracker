// tracker/internal/collector/network.go

//go:build windows

package collector

import (
	"fmt"
	"net"
	"sync"
	"syscall"
	"unsafe"

	"github.com/alme23/tracker/internal/models"
	"golang.org/x/sys/windows"
)

type NetworkCollector struct {
	// Пул буферов для переиспользования памяти
	bufferPool sync.Pool
}

func NewNetworkCollector() *NetworkCollector {
	return &NetworkCollector{
		bufferPool: sync.Pool{
			New: func() interface{} {
				buf := make([]byte, 15000)
				return &buf
			},
		},
	}
}

func (c *NetworkCollector) Collect() (models.NetworkStatuses, error) {
	flags := uint32(windows.GAA_FLAG_INCLUDE_PREFIX)

	// Получаем буфер из пула
	bufPtr := c.bufferPool.Get().(*[]byte)
	buf := *bufPtr
	defer func() {
		*bufPtr = buf[:0]
		c.bufferPool.Put(bufPtr)
	}()

	// Убеждаемся, что буфер достаточно большой
	if cap(buf) < 15000 {
		buf = make([]byte, 15000)
	} else {
		buf = buf[:cap(buf)]
	}

	size := uint32(len(buf))

	var err error
	for {
		err = windows.GetAdaptersAddresses(
			syscall.AF_UNSPEC,
			flags,
			0,
			(*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0])),
			&size,
		)
		if err == nil {
			break
		}
		if err != windows.ERROR_BUFFER_OVERFLOW {
			return nil, fmt.Errorf("ошибка WinAPI GetAdaptersAddresses: %v", err)
		}
		// Увеличиваем буфер
		buf = make([]byte, size)
	}

	// Предварительно выделяем результат (обычно 4-8 адаптеров)
	result := make(models.NetworkStatuses, 0, 8)

	adapter := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
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
		if adapter.PhysicalAddressLength > 0 {
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
