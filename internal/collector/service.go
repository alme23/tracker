// tracker/internal/collector/service.go

//go:build windows

package collector

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/alme23/tracker/internal/models"
	"golang.org/x/sync/errgroup"
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

	// Используем errgroup прямо внутри коллектора служб для параллельной проверки портов
	var g errgroup.Group
	var mu sync.Mutex

	// Заранее выделяем слайс результатов
	results := make(models.ServicesStatuses, 0, 1+len(vncKnownServers))

	// 1. Параллельно проверяем RDP
	g.Go(func() error {
		status := models.ServiceStatus{Name: "RDP", ServiceName: "TermService"}
		// Исправлено: передаем ошибку в errgroup, если проверка WinAPI завершилась сбоем
		if err := c.processRdpService(&status, targetIP); err != nil {
			return err
		}

		mu.Lock()
		results = append(results, status)
		mu.Unlock()
		return nil
	})

	// 2. Параллельно проверяем КАЖДУЮ сигнатуру VNC, а не обходим их в цикле!
	for _, vnc := range vncKnownServers {
		g.Go(func() error {
			installed, running, err := c.checkWindowsService(vnc.ServiceName)
			if err != nil || !installed {
				return nil // Пропускаем, если не установлена (ошибки логируем локально/игнорируем)
			}

			status := models.ServiceStatus{
				Name:        fmt.Sprintf("VNC (%s)", vnc.BrandName),
				ServiceName: vnc.ServiceName,
				Installed:   true,
				Running:     running,
			}

			portStr := c.readRegistryPortGeneric(vnc.RegistryKey, vnc.ValueName, vnc.DefaultPort)
			if parsedPort, err := strconv.ParseUint(portStr, 10, 16); err == nil {
				status.Port = uint16(parsedPort)
			} else {
				status.Port = 5900
			}

			if status.Running && status.Port > 0 {
				status.PortOpen = c.checkFirewallPort(targetIP, strconv.FormatUint(uint64(status.Port), 10))
			}

			mu.Lock()
			results = append(results, status)
			mu.Unlock()
			return nil
		})
	}

	// Ждем выполнения всех проверок портов (если RDP вернет критическую ошибку, мы её зафиксируем)
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

// processRdpService обрабатывает обычную службу (RDP) и возвращает error вместо записи в канал
func (c *ServiceCollector) processRdpService(status *models.ServiceStatus, targetIP string) error {
	installed, running, err := c.checkWindowsService(status.ServiceName)
	if err != nil {
		return fmt.Errorf("ошибка проверки службы RDP: %w", err)
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

	return nil
}

func (c *ServiceCollector) readRegistryPortGeneric(keyPath, valueName, defaultPort string) string {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, keyPath, registry.QUERY_VALUE)
	if err != nil {
		return defaultPort
	}
	defer func() {
		_ = k.Close()
	}()

	_, valType, err := k.GetValue(valueName, nil)
	if err != nil {
		return defaultPort
	}

	switch valType {
	case registry.DWORD:
		val, _, err := k.GetIntegerValue(valueName)
		if err == nil && val > 0 && val <= 65535 {
			return strconv.FormatUint(val, 10)
		}

	case registry.SZ, registry.EXPAND_SZ:
		val, _, err := k.GetStringValue(valueName)
		if err == nil && val != "" {
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
	defer func() {
		_ = windows.CloseServiceHandle(scmHandle)
	}()

	namePtr, err := windows.UTF16PtrFromString(serviceName)
	if err != nil {
		return false, false, err
	}

	serviceHandle, err := windows.OpenService(scmHandle, namePtr, windows.SERVICE_QUERY_STATUS)
	if err != nil {
		if errors.Is(err, windows.ERROR_SERVICE_DOES_NOT_EXIST) {
			return false, false, nil
		}
		return false, false, err
	}
	defer func() {
		_ = windows.CloseServiceHandle(serviceHandle)
	}()

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
	_ = conn.Close()
	return true
}

func (c *ServiceCollector) getLocalIP() string {
	conn, err := net.Dial("udp", "77.88.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer func() {
		_ = conn.Close()
	}()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

func getVncSignatures() []vncSignature {
	return []vncSignature{
		{BrandName: "TightVNC", ServiceName: "tvnserver", RegistryKey: `SOFTWARE\TightVNC\Server`, ValueName: "RfbPort", DefaultPort: "5900"},
		{BrandName: "UltraVNC", ServiceName: "uvnc_service", RegistryKey: `SOFTWARE\ORL\WinVNC3`, ValueName: "PortNumber", DefaultPort: "5900"},
		{BrandName: "RealVNC", ServiceName: "vncserver", RegistryKey: `SOFTWARE\RealVNC\vncserver`, ValueName: "Port", DefaultPort: "5900"},
		{BrandName: "TigerVNC", ServiceName: "TigerVNC Server", RegistryKey: `SOFTWARE\TigerVNC\Server`, ValueName: "Port", DefaultPort: "5900"},
	}
}
