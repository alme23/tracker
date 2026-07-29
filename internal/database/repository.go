package database

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	// Драйвер modernc sqlite на чистом Go
	"github.com/alme23/tracker/pkg/collector"
	_ "modernc.org/sqlite"
)

// Repository инкапсулирует пул подключений к БД
type Repository struct {
	db *sql.DB
}

// NewRepository открывает SQLite и жестко включает реляционный режим через DSN-параметры
func NewRepository(dbPath string) (*Repository, error) {
	// Формируем DSN строку подключения.
	// Параметр _pragma=foreign_keys(1) принудительно включает поддержку реляционных связей на уровне ядра SQLite!
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite with dsn: %w", err)
	}

	// Защита от блокировок: изолируем пул в 1 поток для работы со встроенной БД
	db.SetMaxOpenConns(1)

	repo := &Repository{db: db}
	if err := repo.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return repo, nil
}

// Close безопасно закрывает пул соединений с БД
func (r *Repository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

// migrate создает структуру таблиц и внешние связи
func (r *Repository) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS host_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp INTEGER,
			computer_name TEXT,
			fqdn TEXT,
			domain_name TEXT,
			os_name TEXT,
			os_edition TEXT,
			os_version TEXT,
			os_build TEXT,
			install_date_str TEXT,
			architecture TEXT,
			product_id TEXT,
			machine_guid TEXT,
			user_login TEXT,
			user_fullname TEXT,
			user_domain TEXT,
			user_join_type TEXT,
			bios_vendor TEXT,
			bios_version TEXT,
			bios_release_date TEXT,
			baseboard_vendor TEXT,
			baseboard_product TEXT,
			baseboard_version TEXT,
			baseboard_serial TEXT,
			cpu_name TEXT,
			cpu_cores INTEGER,
			cpu_threads INTEGER,
			cpu_speed_mhz INTEGER,
			cpu_l1_bytes INTEGER,
			cpu_l2_bytes INTEGER,
			cpu_l3_bytes INTEGER,
			ram_total_bytes INTEGER,
			ram_free_bytes INTEGER,
			client_ip TEXT
		);`,

		`CREATE TABLE IF NOT EXISTS gpu_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			host_id INTEGER,
			name TEXT,
			vendor_id INTEGER,
			device_id INTEGER,
			dedicated_memory INTEGER,
			shared_memory INTEGER,
			FOREIGN KEY(host_id) REFERENCES host_metrics(id) ON DELETE CASCADE
		);`,

		`CREATE TABLE IF NOT EXISTS disk_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			host_id INTEGER,
			drive TEXT,
			type TEXT,
			vendor TEXT,
			serial_number TEXT,
			total_bytes INTEGER,
			free_bytes INTEGER,
			FOREIGN KEY(host_id) REFERENCES host_metrics(id) ON DELETE CASCADE
		);`,

		`CREATE TABLE IF NOT EXISTS network_metrics (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			host_id INTEGER,
			name TEXT,
			mac TEXT,
			status TEXT,
			ip_type TEXT,
			ip_addresses TEXT,
			FOREIGN KEY(host_id) REFERENCES host_metrics(id) ON DELETE CASCADE
		);`,
	}

	for _, q := range queries {
		if _, err := r.db.Exec(q); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	return nil
}

// SaveMetrics атомарно записывает все метрики хоста в реляционную структуру
func (r *Repository) SaveMetrics(m collector.Metrics, ip string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Переменная для отслеживания успешности транзакции
	success := false
	defer func() {
		if !success {
			_ = tx.Rollback()
		}
	}()

	// 1. Запись в главную таблицу хостов
	hostQuery := `
	INSERT INTO host_metrics (
		timestamp, computer_name, fqdn, domain_name, os_name, os_edition, os_version, os_build,
		install_date_str, architecture, product_id, machine_guid, user_login, user_fullname,
		user_domain, user_join_type, bios_vendor, bios_version, bios_release_date,
		baseboard_vendor, baseboard_product, baseboard_version, baseboard_serial,
		cpu_name, cpu_cores, cpu_threads, cpu_speed_mhz, cpu_l1_bytes, cpu_l2_bytes, cpu_l3_bytes,
		ram_total_bytes, ram_free_bytes, client_ip
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

	res, err := tx.Exec(hostQuery,
		m.Timestamp, m.Host.ComputerName, m.Host.FQDN, m.Host.DomainName, m.Host.OSName, m.Host.OSEdition, m.Host.OSVersion, m.Host.OSBuild,
		m.Host.InstallDateStr, m.Host.Architecture, m.Host.ProductID, m.Host.DeviceID, m.User.UserName, m.User.DisplayName,
		m.User.Domain, m.User.JoinType, m.BIOS.Vendor, m.BIOS.Version, m.BIOS.ReleaseDate,
		m.BaseBoard.Vendor, m.BaseBoard.Product, m.BaseBoard.Version, m.BaseBoard.SerialNumber,
		m.CPU.Name, m.CPU.PhysicalCores, m.CPU.LogicalProcessors, m.CPU.MaxSpeedMHz, m.CPU.CacheL1Bytes, m.CPU.CacheL2Bytes, m.CPU.CacheL3Bytes,
		m.Memory.TotalRAMBytes, m.Memory.FreeRAMBytes, ip,
	)
	if err != nil {
		return fmt.Errorf("failed to insert host: %w", err)
	}

	hostID, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get insert ID: %w", err)
	}

	// 2. Запись массива видеокарт
	if len(m.Video) > 0 {
		gpuQuery := `INSERT INTO gpu_metrics (host_id, name, vendor_id, device_id, dedicated_memory, shared_memory) VALUES (?, ?, ?, ?, ?, ?);`
		for _, v := range m.Video {
			if _, err = tx.Exec(gpuQuery, hostID, v.Name, v.VendorID, v.DeviceID, v.DedicatedMemory, v.SharedMemory); err != nil {
				return fmt.Errorf("failed to insert GPU: %w", err)
			}
		}
	}

	// 3. Запись массива дисков
	if len(m.Disks) > 0 {
		diskQuery := `INSERT INTO disk_metrics (host_id, drive, type, vendor, serial_number, total_bytes, free_bytes) VALUES (?, ?, ?, ?, ?, ?, ?);`
		for _, d := range m.Disks {
			if _, err = tx.Exec(diskQuery, hostID, d.Drive, d.Type, d.Vendor, d.SerialNumber, d.TotalBytes, d.FreeBytes); err != nil {
				return fmt.Errorf("failed to insert disk: %w", err)
			}
		}
	}

	// 4. Запись массива сетевых интерфейсов
	if len(m.Network) > 0 {
		netQuery := `INSERT INTO network_metrics (host_id, name, mac, status, ip_type, ip_addresses) VALUES (?, ?, ?, ?, ?, ?);`
		for _, n := range m.Network {
			var ips []string
			for _, ip := range n.IPAddresses {
				ips = append(ips, ip.String())
			}
			ipListStr := strings.Join(ips, ", ")

			if _, err = tx.Exec(netQuery, hostID, n.Name, n.MAC.String(), n.Status.String(), n.IPType.String(), ipListStr); err != nil {
				log.Printf("[Предупреждение БД] Ошибка парсинга интерфейса %s, пропускаем: %v", n.Name, err)
				continue
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	success = true
	return nil
}

// GetLastMetrics ищет последнюю сохраненную конфигурацию ПК по его MachineGuid.
func (r *Repository) GetLastMetrics(machineGUID string) (collector.Metrics, bool, error) {
	var m collector.Metrics
	var hostID int64

	// ИСПРАВЛЕНО: Добавлено считывание cpu_threads, чтобы данные не смещались
	query := `
	SELECT id, timestamp, cpu_name, cpu_cores, cpu_threads, ram_total_bytes
	FROM host_metrics
	WHERE machine_guid = ?
	ORDER BY timestamp DESC LIMIT 1;`

	// ИСПРАВЛЕНО: Сканируем ядра в PhysicalCores, а потоки в LogicalProcessors
	err := r.db.QueryRow(query, machineGUID).Scan(
		&hostID, &m.Timestamp, &m.CPU.Name, &m.CPU.PhysicalCores, &m.CPU.LogicalProcessors, &m.Memory.TotalRAMBytes,
	)

	if err == sql.ErrNoRows {
		return m, false, nil // Хост новый
	}
	if err != nil {
		return m, false, fmt.Errorf("failed to query last host metrics: %w", err)
	}

	m.Host.DeviceID = machineGUID

	// Диски (оставляем без изменений)
	diskQuery := `SELECT drive, vendor, serial_number, total_bytes FROM disk_metrics WHERE host_id = ?;`
	rows, err := r.db.Query(diskQuery, hostID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var d collector.DiskInfo
			if err := rows.Scan(&d.Drive, &d.Vendor, &d.SerialNumber, &d.TotalBytes); err == nil {
				m.Disks = append(m.Disks, d)
			}
		}
	}

	// Видеокарты (оставляем без изменений)
	gpuQuery := `SELECT name, vendor_id, device_id, dedicated_memory FROM gpu_metrics WHERE host_id = ?;`
	gpuRows, err := r.db.Query(gpuQuery, hostID)
	if err == nil {
		defer gpuRows.Close()
		for gpuRows.Next() {
			var v collector.VideoInfo
			if err := gpuRows.Scan(&v.Name, &v.VendorID, &v.DeviceID, &v.DedicatedMemory); err == nil {
				m.Video = append(m.Video, v)
			}
		}
	}

	return m, true, nil
}

// UpdateTimestamp обновляет время последнего обнаружения хоста в его самой свежей записи.
func (r *Repository) UpdateTimestamp(machineGUID string, newTimestamp int64, clientIP string) error {
	query := `
	UPDATE host_metrics
	SET timestamp = ?, client_ip = ?
	WHERE id = (
		SELECT id FROM host_metrics
		WHERE machine_guid = ?
		ORDER BY timestamp DESC LIMIT 1
	);`

	_, err := r.db.Exec(query, newTimestamp, clientIP, machineGUID)
	if err != nil {
		return fmt.Errorf("failed to update timestamp: %w", err)
	}
	return nil
}
