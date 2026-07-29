package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

type WebSessionLog struct {
	Timestamp    string
	ComputerName string
	FQDN         string
	UserLogin    string
	UserFullname string
	ClientIP     string
	MachineGUID  string
	RDPStatus    string
	VNCStatus    string
}

// SearchSessions осуществляет гибкий поиск по журналу входов сотрудников
func (r *Repository) SearchSessions(query string) ([]WebSessionLog, error) {
	var logs []WebSessionLog

	sqlQuery := `
	SELECT
		datetime(COALESCE(h.timestamp, s.timestamp), 'unixepoch', 'localtime') AS log_time,
		s.computer_name,
		COALESCE(h.fqdn, s.computer_name) AS fqdn,
		s.user_login,
		s.user_fullname,
		s.client_ip,
		s.machine_guid,
		COALESCE(h.os_build, 'Stopped') AS rdp_status,
		COALESCE(h.architecture, 'Not Installed') AS vnc_status
	FROM session_logs s
	LEFT JOIN host_metrics h ON s.machine_guid = h.machine_guid
	`

	query = strings.TrimSpace(query)
	var err error
	var rows *sql.Rows

	if query != "" {
		sqlQuery += `
		WHERE s.computer_name LIKE ?
		   OR h.fqdn LIKE ?
		   OR s.user_login LIKE ?
		   OR s.user_fullname LIKE ?
		   OR s.client_ip LIKE ?
		`
		searchTerm := "%" + query + "%"
		sqlQuery += " ORDER BY COALESCE(h.timestamp, s.timestamp) DESC LIMIT 100;"
		rows, err = r.db.Query(sqlQuery, searchTerm, searchTerm, searchTerm, searchTerm, searchTerm)
	} else {
		sqlQuery += " ORDER BY COALESCE(h.timestamp, s.timestamp) DESC LIMIT 100;"
		rows, err = r.db.Query(sqlQuery)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to execute search query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var l WebSessionLog
		err := rows.Scan(&l.Timestamp, &l.ComputerName, &l.FQDN, &l.UserLogin, &l.UserFullname, &l.ClientIP, &l.MachineGUID, &l.RDPStatus, &l.VNCStatus)
		if err == nil {
			if l.RDPStatus == "" {
				l.RDPStatus = "Stopped"
			}
			if l.VNCStatus == "" {
				l.VNCStatus = "Not Installed"
			}
			logs = append(logs, l)
		}
	}

	return logs, nil
}

// GetHostDetailsJSON собирает полную конфигурацию железа по GUID и возвращает как JSON байты
func (r *Repository) GetHostDetailsJSON(machineGUID string) ([]byte, error) {
	data := make(map[string]interface{})

	// ИСПРАВЛЕНО: Объявляем все переменные строго ОДИН РАЗ, исключая дублирование (redeclared)
	var (
		hostID                                      int64
		osName, osEdition, osVersion, osBuild, arch string
		cpuName, baseboardVendor, baseboardProduct  string
		biosVendor, biosVersion                     string
		cpuCores, cpuThreads, cpuSpeed              int
		cpuL1, cpuL2, cpuL3                         int64
	)

	// 1. Извлекаем полный паспорт хоста и характеристики процессора
	query := `
	SELECT
		id, os_name, os_edition, os_version, os_build, architecture,
		cpu_name, cpu_cores, cpu_threads, cpu_speed_mhz,
		cpu_l1_bytes, cpu_l2_bytes, cpu_l3_bytes,
		baseboard_vendor, baseboard_product, bios_vendor, bios_version
	FROM host_metrics
	WHERE machine_guid = ?
	ORDER BY timestamp DESC LIMIT 1;`

	err := r.db.QueryRow(query, machineGUID).Scan(
		&hostID, &osName, &osEdition, &osVersion, &osBuild, &arch,
		&cpuName, &cpuCores, &cpuThreads, &cpuSpeed,
		&cpuL1, &cpuL2, &cpuL3,
		&baseboardVendor, &baseboardProduct, &biosVendor, &biosVersion,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query host passport: %w", err)
	}

	// Пакуем базовые строки спецификации
	data["os"] = fmt.Sprintf("%s (%s) %s Build %s [%s]", osName, osEdition, osVersion, osBuild, arch)
	data["motherboard"] = fmt.Sprintf("%s %s", baseboardVendor, baseboardProduct)
	data["bios"] = fmt.Sprintf("%s Версия: %s", biosVendor, biosVersion)

	// Пакуем все характеристики процессора в JSON
	data["cpu_name"] = cpuName
	data["cpu_cores"] = cpuCores
	data["cpu_threads"] = cpuThreads
	data["cpu_speed"] = cpuSpeed
	data["cpu_l1"] = cpuL1
	data["cpu_l2"] = cpuL2
	data["cpu_l3"] = cpuL3

	// 2. Извлекаем диски в виде массива структур со всеми байтами
	type WebDiskInfo struct {
		Drive      string `json:"drive"`
		Type       string `json:"type"`
		Vendor     string `json:"vendor"`
		TotalBytes int64  `json:"total_bytes"`
		FreeBytes  int64  `json:"free_bytes"`
	}
	var disks []WebDiskInfo

	rowsD, err := r.db.Query("SELECT drive, type, vendor, total_bytes, free_bytes FROM disk_metrics WHERE host_id = ?", hostID)
	if err == nil {
		defer rowsD.Close()
		for rowsD.Next() {
			var d WebDiskInfo
			if rErr := rowsD.Scan(&d.Drive, &d.Type, &d.Vendor, &d.TotalBytes, &d.FreeBytes); rErr == nil {
				disks = append(disks, d)
			}
		}
	}
	data["disks"] = disks

	// 3. Извлекаем видеокарты
	var gpus []string
	rowsG, err := r.db.Query("SELECT name, dedicated_memory FROM gpu_metrics WHERE host_id = ?", hostID)
	if err == nil {
		defer rowsG.Close()
		for rowsG.Next() {
			var name string
			var mem int64
			if rErr := rowsG.Scan(&name, &mem); rErr == nil {
				gpus = append(gpus, fmt.Sprintf("%s (Память: %.1f GB)", name, float64(mem)/1024/1024/1024))
			}
		}
	}
	data["gpus"] = gpus

	// 4. Извлекаем сетевые интерфейсы в виде массива структур (ФИНАЛЬНОЕ ВЫРАВНИВАНИЕ)
	type WebNetInfo struct {
		Name        string `json:"name"`
		Mac         string `json:"mac"` // Тег строго совпадает с полем json:"mac"
		IPAddresses string `json:"ips"`
		Status      string `json:"status"`
		IPType      string `json:"ip_type"`
	}
	var networks []WebNetInfo

	// Извлекаем чистую колонку 'mac' без псевдонимов
	queryNet := `
		SELECT name, mac, status, ip_type, ip_addresses
		FROM network_metrics
		WHERE host_id = ?
		ORDER BY status DESC, name ASC;`

	rowsN, err := r.db.Query(queryNet, hostID)
	if err == nil {
		defer rowsN.Close()
		for rowsN.Next() {
			var n WebNetInfo
			// Сканируем колонку mac строго в переменную n.Mac
			if rErr := rowsN.Scan(&n.Name, &n.Mac, &n.Status, &n.IPType, &n.IPAddresses); rErr == nil {
				networks = append(networks, n)
			}
		}
	}
	data["networks"] = networks

	return json.Marshal(data)
}
