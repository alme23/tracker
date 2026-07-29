//go:build windows

package collector

import (
	"golang.org/x/sys/windows/registry" // Подключаем работу с реестром
	"golang.org/x/sys/windows/svc"
	"golang.org/x/sys/windows/svc/mgr"
)

// GetRemoteAccessServices опрашивает менеджер служб и реестр Windows
func GetRemoteAccessServices() (RemoteAccessInfo, error) {
	var info RemoteAccessInfo
	info.RDPStatus = "Stopped"
	info.VNCStatus = "Not Installed"

	// 1. ДВОЙНАЯ ПРОВЕРКА RDP (Служба + Реестр)
	// Проверяем глобальный порт и разрешение RDP в реестре Windows
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Terminal Server`, registry.QUERY_VALUE)
	if err == nil {
		// Значение fDenyTSConnections = 0 означает, что RDP ВКЛЮЧЕН в системе
		denyConn, _, errDeny := k.GetIntegerValue("fDenyTSConnections")
		if errDeny == nil && denyConn == 0 {
			info.RDPStatus = "Running"
			info.RDPAvailable = true
		}
		_ = k.Close()
	}

	// Если реестр подтвердил работу, или пытаемся достучаться через менеджер служб (SCM)
	m, err := mgr.Connect()
	if err == nil {
		defer func() { _ = m.Disconnect() }()

		if s, err := m.OpenService("TermService"); err == nil {
			if status, err := s.Query(); err == nil {
				// Если служба реально запущена, это приоритет
				if status.State == svc.Running {
					info.RDPStatus = "Running"
					info.RDPAvailable = true
				} else if !info.RDPAvailable {
					info.RDPStatus = statusToString(status.State)
				}
			}
			_ = s.Close()
		}
	}

	// 2. ПРОВЕРКА СЛУЖБ СЕМЕЙСТВА VNC
	if err == nil {
		vncServiceNames := map[string]string{
			"uvnc_service":    "UltraVNC",
			"tvnserver":       "TightVNC",
			"winvnc":          "RealVNC",
			"TigerVNC":        "TigerVNC",
			"TightVNC Server": "TightVNC",
		}

		for sName, commercialName := range vncServiceNames {
			if s, err := m.OpenService(sName); err == nil {
				if status, err := s.Query(); err == nil {
					sStateStr := statusToString(status.State)
					info.VNCStatus = commercialName + " (" + sStateStr + ")"
					if status.State == svc.Running {
						info.VNCAvailable = true
					}
				}
				_ = s.Close()
				break
			}
		}
	}

	return info, nil
}

func statusToString(state svc.State) string {
	switch state {
	case svc.Stopped:
		return "Stopped"
	case svc.StartPending:
		return "Starting"
	case svc.StopPending:
		return "Stopping"
	case svc.Running:
		return "Running"
	default:
		return "Stopped"
	}
}
