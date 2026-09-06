//go:build windows

package collector

import (
	"context"
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

// vncSignature describes parameters of a specific VNC implementation
type vncSignature struct {
	BrandName   string // Display name (e.g., "UltraVNC")
	ServiceName string // Windows service name
	RegistryKey string // Registry path in HKLM
	ValueName   string // Port value name
	DefaultPort string // Fallback port
}

// ServiceCollector collects information about services (RDP, VNC)
type ServiceCollector struct {
	dialTimeout time.Duration
}

// NewServiceCollector creates a new ServiceCollector
func NewServiceCollector(timeout time.Duration) *ServiceCollector {
	return &ServiceCollector{
		dialTimeout: timeout,
	}
}

// Collect gathers service status information
func (c *ServiceCollector) Collect() (models.ServicesStatuses, error) {
	targetIP := c.getLocalIP()
	vncKnownServers := getVncSignatures()

	// Use errgroup for parallel port checks
	var g errgroup.Group
	var mu sync.Mutex

	// Pre-allocate results slice
	results := make(models.ServicesStatuses, 0, 1+len(vncKnownServers))

	// 1. Check RDP in parallel
	g.Go(func() error {
		status := models.ServiceStatus{Name: "RDP", ServiceName: "TermService"}
		if err := c.processRdpService(&status, targetIP); err != nil {
			return err
		}

		mu.Lock()
		results = append(results, status)
		mu.Unlock()
		return nil
	})

	// 2. Check each VNC signature in parallel
	for _, vnc := range vncKnownServers {
		g.Go(func() error {
			installed, running, err := c.checkWindowsService(vnc.ServiceName)
			if err != nil {
				// Return the error instead of nil
				return fmt.Errorf("VNC service check (%s): %w", vnc.BrandName, err)
			}
			if !installed {
				return nil // Service not installed, skip
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

	// Wait for all port checks to complete
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return results, nil
}

// processRdpService processes the RDP service and returns error instead of writing to channel
func (c *ServiceCollector) processRdpService(status *models.ServiceStatus, targetIP string) error {
	installed, running, err := c.checkWindowsService(status.ServiceName)
	if err != nil {
		return fmt.Errorf("RDP service check error: %w", err)
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

// readRegistryPortGeneric reads a port from the registry, handling both DWORD and string types
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

// checkWindowsService checks if a Windows service exists and is running
func (c *ServiceCollector) checkWindowsService(serviceName string) (installed, running bool, err error) {
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

// checkFirewallPort checks if a TCP port is accessible
func (c *ServiceCollector) checkFirewallPort(host, port string) bool {
	address := net.JoinHostPort(host, port)

	// Use Dialer with context and timeout
	dialer := &net.Dialer{
		Timeout: c.dialTimeout,
	}

	conn, err := dialer.DialContext(context.Background(), "tcp", address)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

// getLocalIP returns the local IP address
func (c *ServiceCollector) getLocalIP() string {
	dialer := &net.Dialer{
		Timeout: 3 * time.Second,
	}

	conn, err := dialer.DialContext(context.Background(), "udp", "77.88.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer func() {
		_ = conn.Close()
	}()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// getVncSignatures returns the list of known VNC implementations
func getVncSignatures() []vncSignature {
	return []vncSignature{
		{BrandName: "TightVNC", ServiceName: "tvnserver", RegistryKey: `SOFTWARE\TightVNC\Server`, ValueName: "RfbPort", DefaultPort: "5900"},
		{BrandName: "UltraVNC", ServiceName: "uvnc_service", RegistryKey: `SOFTWARE\ORL\WinVNC3`, ValueName: "PortNumber", DefaultPort: "5900"},
		{BrandName: "RealVNC", ServiceName: "vncserver", RegistryKey: `SOFTWARE\RealVNC\vncserver`, ValueName: "Port", DefaultPort: "5900"},
		{BrandName: "TigerVNC", ServiceName: "TigerVNC Server", RegistryKey: `SOFTWARE\TigerVNC\Server`, ValueName: "Port", DefaultPort: "5900"},
	}
}
