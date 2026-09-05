// tracker/internal/collector/service.go

//go:build windows

package collector

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/alme23/tracker/internal/models"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// vncSignature описывает параметры конкретной реализации VNC
type vncSignature struct {
	BrandName   string // Красивое имя (например, "UltraVNC")
	ServiceName string // Имя службы в Windows
	RegistryKey string // Путь в HKLM
	ValueName   string // Название параметра порта
	DefaultPort string // Фолбек порт
}

type ServiceCollector struct {
	dialTimeout time.Duration
}

func NewServiceCollector(timeout time.Duration) *ServiceCollector {
	return &ServiceCollector{
		dialTimeout: timeout,
	}
}

func (c *ServiceCollector) Collect() (models.ServicesStatuses, error) {
	targetIP := c.getLocalIP()
	vncKnownServers := getVncSignatures()

	var wg sync.WaitGroup
	errChan := make(chan error, 2)

	// Используем локальные переменные для хранения результатов
	statuses := make(models.ServicesStatuses, 2)

	// Инициализируем с правильными именами
	statuses[0] = models.ServiceStatus{Name: "RDP", ServiceName: "TermService"}
	statuses[1] = models.ServiceStatus{Name: "VNC", ServiceName: "unknown"}

	// RDP - обрабатываем на месте
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.processRdpService(&statuses[0], targetIP, errChan)
	}()

	// VNC - обрабатываем на месте
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.processVncService(&statuses[1], vncKnownServers, targetIP, errChan)
	}()

	wg.Wait()
	close(errChan)

	// Проверяем ошибки
	for err := range errChan {
		if err != nil {
			return nil, err
		}
	}

	// Порядок гарантирован: [RDP, VNC]
	return statuses, nil
}

// processRdpService обрабатывает обычную службу (RDP)
func (c *ServiceCollector) processRdpService(status *models.ServiceStatus, targetIP string, errChan chan<- error) {
	installed, running, err := c.checkWindowsService(status.ServiceName)
	if err != nil {
		errChan <- err
		return
	}

	status.Installed = installed
	status.Running = running

	portStr := c.readRegistryPortGeneric(
		`SYSTEM\CurrentControlSet\Control\Terminal Server\WinStations\RDP-Tcp`,
		"PortNumber",
		"3389",
	)

	if parsedPort, err := strconv.ParseUint(portStr, 10, 16); err == nil {
		status.Port = uint16(parsedPort)
	} else {
		status.Port = 3389
	}

	if status.Running && status.Port > 0 {
		status.PortOpen = c.checkFirewallPort(targetIP, strconv.FormatUint(uint64(status.Port), 10))
	}
}

// processVncService обрабатывает VNC службу
func (c *ServiceCollector) processVncService(status *models.ServiceStatus, vncServers []vncSignature, targetIP string, errChan chan<- error) {
	var detected bool

	for _, vnc := range vncServers {
		installed, running, err := c.checkWindowsService(vnc.ServiceName)
		if err != nil {
			errChan <- err
			return
		}

		if installed {
			status.Name = fmt.Sprintf("VNC (%s)", vnc.BrandName)
			status.ServiceName = vnc.ServiceName
			status.Installed = true
			status.Running = running

			portStr := c.readRegistryPortGeneric(vnc.RegistryKey, vnc.ValueName, vnc.DefaultPort)
			if parsedPort, err := strconv.ParseUint(portStr, 10, 16); err == nil {
				status.Port = uint16(parsedPort)
			} else {
				status.Port = 5900
			}

			detected = true
			break
		}
	}

	if !detected {
		status.Name = "VNC"
		status.Installed = false
		status.Running = false
		status.Port = 5900
	}

	if status.Running && status.Port > 0 {
		status.PortOpen = c.checkFirewallPort(targetIP, strconv.FormatUint(uint64(status.Port), 10))
	}
}

// readRegistryPortGeneric — это "всеядный" метод чтения порта.
// Он определяет тип данных в реестре (число или строка) и корректно возвращает строковое представление.
func (c *ServiceCollector) readRegistryPortGeneric(keyPath, valueName, defaultPort string) string {
	// Открываем ветку реестра только для чтения (права админа НЕ нужны)
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.QUERY_VALUE)
	if err != nil {
		return defaultPort
	}
	defer k.Close()

	// Проверяем тип значения в реестре, чтобы избежать паники рантайма
	_, valType, err := k.GetValue(valueName, nil)
	if err != nil {
		return defaultPort
	}

	switch valType {
	case registry.DWORD: // Если порт сохранен как число (RDP, TightVNC)
		val, _, err := k.GetIntegerValue(valueName)
		// Валидируем, что число укладывается в границы сетевого порта (uint16)
		if err == nil && val > 0 && val <= 65535 {
			return strconv.FormatUint(val, 10)
		}

	case registry.SZ, registry.EXPAND_SZ: // Если порт сохранен как строка (UltraVNC в некоторых версиях)
		val, _, err := k.GetStringValue(valueName)
		if err == nil && val != "" {
			// Проверяем, что в строке действительно валидный порт uint16
			if p, parseErr := strconv.ParseUint(val, 10, 16); parseErr == nil && p > 0 {
				return val
			}
		}
	}

	return defaultPort
}

func (c *ServiceCollector) checkWindowsService(serviceName string) (installed bool, running bool, err error) {
	scmHandle, err := windows.OpenSCManager(nil, nil, windows.SC_MANAGER_CONNECT)
	if err != nil {
		return false, false, err
	}
	defer windows.CloseServiceHandle(scmHandle)

	serviceHandle, err := windows.OpenService(scmHandle, windows.StringToUTF16Ptr(serviceName), windows.SERVICE_QUERY_STATUS)
	if err != nil {
		if errors.Is(err, syscall.Errno(1060)) { // ERROR_SERVICE_DOES_NOT_EXIST
			return false, false, nil
		}
		return false, false, err
	}
	defer windows.CloseServiceHandle(serviceHandle)

	installed = true

	var status windows.SERVICE_STATUS
	err = windows.QueryServiceStatus(serviceHandle, &status)
	if err != nil {
		return installed, false, err
	}

	running = status.CurrentState == windows.SERVICE_RUNNING

	return installed, running, nil
}

func (c *ServiceCollector) checkFirewallPort(host, port string) bool {
	address := net.JoinHostPort(host, port)
	conn, err := net.DialTimeout("tcp", address, c.dialTimeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func (c *ServiceCollector) getLocalIP() string {
	// Элегантный UDP-трюк для определения интерфейса по умолчанию
	conn, err := net.Dial("udp", "77.88.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// getVncSignatures возвращает список VNC сигнатур
func getVncSignatures() []vncSignature {
	return []vncSignature{
		{BrandName: "TightVNC", ServiceName: "tvnserver", RegistryKey: `SOFTWARE\TightVNC\Server`, ValueName: "RfbPort", DefaultPort: "5900"},
		{BrandName: "UltraVNC", ServiceName: "uvnc_service", RegistryKey: `SOFTWARE\ORL\WinVNC3`, ValueName: "PortNumber", DefaultPort: "5900"},
		{BrandName: "RealVNC", ServiceName: "vncserver", RegistryKey: `SOFTWARE\RealVNC\vncserver`, ValueName: "Port", DefaultPort: "5900"},
		{BrandName: "TigerVNC", ServiceName: "TigerVNC Server", RegistryKey: `SOFTWARE\TigerVNC\Server`, ValueName: "Port", DefaultPort: "5900"},
	}
}
