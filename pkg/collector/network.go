//go:build windows

package collector

import (
	"fmt"
	"net"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// GetNetworkInterfaces собирает абсолютно ВСЕ сетевые адаптеры компьютера
// напрямую через системное API Windows без риска сдвига указателей в памяти.
func GetNetworkInterfaces() ([]NetworkInterface, error) {
	var interfaces []NetworkInterface

	// Выделяем первоначальный буфер в памяти (15 КБ)
	size := uint32(15000)
	buf := make([]byte, size)

	// AF_UNSPEC (0) — запрашиваем и IPv4, и IPv6 адреса
	// Флаги 0x0006 — пропускаем Anycast и Multicast адреса для максимальной скорости опроса
	err := windows.GetAdaptersAddresses(windows.AF_UNSPEC, windows.GAA_FLAG_SKIP_ANYCAST|windows.GAA_FLAG_SKIP_MULTICAST, 0, (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0])), &size)

	// Если буфер оказался мал, выделяем память заново по требованию ядра Windows
	if err == windows.ERROR_BUFFER_OVERFLOW {
		buf = make([]byte, size)
		err = windows.GetAdaptersAddresses(windows.AF_UNSPEC, windows.GAA_FLAG_SKIP_ANYCAST|windows.GAA_FLAG_SKIP_MULTICAST, 0, (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0])), &size)
	}

	if err != nil {
		return nil, fmt.Errorf("GetAdaptersAddresses failed: %w", err)
	}

	// Читаем связанный список структур в памяти
	curr := (*windows.IpAdapterAddresses)(unsafe.Pointer(&buf[0]))
	for curr != nil {
		// Игнорируем loopback-заглушки (IF_TYPE_SOFTWARE_LOOPBACK = 24)
		if curr.IfType == 24 {
			curr = curr.Next
			continue
		}

		var ni NetworkInterface

		// Конвертируем UTF16 имя адаптера в понятную строку Go
		ni.Name = windows.UTF16PtrToString(curr.FriendlyName)

		// Извлекаем чистые байты аппаратного MAC-адреса хоста
		if curr.PhysicalAddressLength > 0 && curr.PhysicalAddressLength <= 8 {
			macBytes := curr.PhysicalAddress[:curr.PhysicalAddressLength]
			m := net.HardwareAddr(macBytes)
			ni.MAC = MACAddress(m)
		}

		// Вычисляем сетевой статус (IfOperStatusUp = 1)
		if curr.OperStatus == windows.IfOperStatusUp {
			ni.Status = InterfaceStatusUp
		} else {
			ni.Status = InterfaceStatusDown
		}

		// Проверяем статус DHCP (Флаг IP_ADAPTER_DHCP_ENABLED = 0x0004)
		if (curr.Flags & 0x0004) != 0 {
			ni.IPType = IPAssignmentTypeDHCP
		} else {
			ni.IPType = IPAssignmentTypeStatic
		}

		// Собираем все привязанные юникаст IP-адреса (IPv4 / IPv6)
		unicast := curr.FirstUnicastAddress
		for unicast != nil {
			if unicast.Address.Sockaddr != nil {
				sa, err := unicast.Address.Sockaddr.Sockaddr()
				if err == nil {
					var ip net.IP
					switch sock := sa.(type) {
					case *syscall.SockaddrInet4:
						ip = net.IPv4(sock.Addr[0], sock.Addr[1], sock.Addr[2], sock.Addr[3])
					case *syscall.SockaddrInet6:
						ip = make(net.IP, 16)
						copy(ip, sock.Addr[:])
					}
					if ip != nil && !ip.IsLoopback() {
						ni.IPAddresses = append(ni.IPAddresses, IPAddress(ip))
					}
				}
			}
			unicast = unicast.Next
		}

		// Записываем адаптер, только если у него считался валидный физический MAC
		if len(ni.MAC) > 0 {
			interfaces = append(interfaces, ni)
		}
		curr = curr.Next
	}

	return interfaces, nil
}
