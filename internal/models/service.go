package models

// ServiceStatus содержит собранную информацию о сервисе
type ServiceStatus struct {
	Name        string `json:"name"`         // Отображаемое имя (RDP, VNC)
	ServiceName string `json:"service_name"` // Имя службы в Windows (например, TermService)
	Installed   bool   `json:"installed"`    // Установлен ли в системе
	Running     bool   `json:"running"`      // Запущен ли сейчас
	Port        uint16 `json:"port"`         // Текущий активный порт (например, "3389" или "53389")
	PortOpen    bool   `json:"port_open"`    // Доступен ли порт извне (проверка файрвола)
}

type ServicesStatuses []ServiceStatus
