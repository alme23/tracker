//go:build windows

package collector

import (
	"fmt"
	"net"
	"strings"

	"golang.org/x/sys/windows/registry"
)

// GetNetworkInterfaces собирает абсолютно ВСЕ сетевые адаптеры компьютера напрямую из реестра Windows.
// Этот метод гарантированно находит даже отключенные сетевые карты без воткнутого кабеля.
func GetNetworkInterfaces() ([]NetworkInterface, error) {
	var interfaces []NetworkInterface

	// Главная ветка, где Windows регистрирует все сетевые подключения
	basePath := `SYSTEM\CurrentControlSet\Control\Network\{4D36E972-E325-11CE-BFC1-08002BE10318}`
	baseKey, err := registry.OpenKey(registry.LOCAL_MACHINE, basePath, registry.READ)
	if err != nil {
		return nil, fmt.Errorf("failed to open network registry key: %w", err)
	}
	defer func() { _ = baseKey.Close() }()

	// Получаем список GUID всех сетевых адаптеров системы
	guids, err := baseKey.ReadSubKeyNames(-1)
	if err != nil {
		return nil, err
	}

	for _, guid := range guids {
		// Нам нужны только папки с GUID интерфейсов, игнорируем служебные ключи
		if !strings.HasPrefix(guid, "{") {
			continue
		}

		// Открываем подпапку Connection для каждого адаптера, чтобы узнать его понятное имя
		connKey, err := registry.OpenKey(registry.LOCAL_MACHINE, basePath+`\`+guid+`\Connection`, registry.QUERY_VALUE)
		if err != nil {
			continue
		}

		adapterName, _, err := connKey.GetStringValue("Name")
		_ = connKey.Close()
		if err != nil || adapterName == "" {
			continue
		}

		var netIf NetworkInterface
		netIf.Name = adapterName
		netIf.Status = InterfaceStatusDown // По умолчанию ставим Down

		// Обращаемся к параллельной ветке настроек TCP/IP для этого конкретного GUID адаптера
		tcpipPath := `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces\` + guid
		tcpKey, err := registry.OpenKey(registry.LOCAL_MACHINE, tcpipPath, registry.QUERY_VALUE)

		if err == nil {
			// 1. Определяем тип назначения IP (DHCP или Static) строго для этого GUID
			enableDHCP, _, err := tcpKey.GetIntegerValue("EnableDHCP")
			if err == nil && enableDHCP == 1 {
				netIf.IPType = IPAssignmentTypeDHCP
			} else {
				netIf.IPType = IPAssignmentTypeStatic
			}

			// 2. Проверяем наличие назначенных IP-адресов
			var rawIPs []string
			if netIf.IPType == IPAssignmentTypeDHCP {
				dhcpIP, _, err := tcpKey.GetStringValue("DhcpIPAddress")
				if err == nil && dhcpIP != "" && dhcpIP != "0.0.0.0" {
					rawIPs = append(rawIPs, dhcpIP)
					netIf.Status = InterfaceStatusUp // Раз есть живой DHCP IP — адаптер активен
				}
			} else {
				staticIPs, _, err := tcpKey.GetStringsValue("IPAddress")
				if err == nil {
					for _, ip := range staticIPs {
						if ip != "" && ip != "0.0.0.0" {
							rawIPs = append(rawIPs, ip)
							netIf.Status = InterfaceStatusUp // Есть статический IP — адаптер активен
						}
					}
				}
			}

			// Преобразуем строковые IP в ваш кастомный тип IPAddress
			for _, ipStr := range rawIPs {
				parsedIP := net.ParseIP(ipStr)
				if parsedIP != nil {
					netIf.IPAddresses = append(netIf.IPAddresses, IPAddress(parsedIP))
				}
			}
			_ = tcpKey.Close()
		} else {
			// Если ветки Tcpip для GUID нет, значит адаптер никогда не настраивался (всегда Static)
			netIf.IPType = IPAssignmentTypeStatic
		}

		// 3. Получаем реальный MAC-адрес устройства
		// Так как в реестре MAC лежит не всегда, мы ищем его через стандартную библиотеку по GUID имени,
		// а если не находим (адаптер полностью выключен) — оставляем пустым или ищем в ветке Hardware
		netIf.MAC = MACAddress(getMacByInterfaceGUID(guid))

		interfaces = append(interfaces, netIf)
	}

	return interfaces, nil
}

// getMacByInterfaceGUID пытается найти MAC-адрес через API Go, сопоставляя GUID
func getMacByInterfaceGUID(guid string) []byte {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}

	// В Windows net.Interface.Name для скрытых/низкоуровневых интерфейсов часто совпадает с GUID
	// или содержит его внутри системного описания.
	for _, iface := range ifaces {
		if strings.Contains(strings.ToLower(iface.Name), strings.ToLower(guid)) {
			return iface.HardwareAddr
		}
	}

	// Попробуем альтернативный поиск по реестру в свойствах железа, если интерфейс Down
	cardPath := `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters\Interfaces\` + guid
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, cardPath, registry.QUERY_VALUE)
	if err == nil {
		defer func() { _ = key.Close() }()
		// Иногда Windows кэширует физический адрес адаптера здесь
		if macStr, _, err := key.GetStringValue("NetworkAddress"); err == nil && macStr != "" {
			hw, _ := net.ParseMAC(macStr)
			return hw
		}
	}
	return nil
}
