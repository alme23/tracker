// internal/server/storage.go
package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/alme23/tracker/internal/models"
	_ "modernc.org/sqlite"
)

// Storage отвечает за сохранение данных в SQLite
type Storage struct {
	db *sql.DB
}

// NewStorage создает новое хранилище
func NewStorage(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("открытие БД: %w", err)
	}

	// SQLite — одно соединение на запись
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("подключение: %w", err)
	}

	s := &Storage{db: db}

	// Включаем WAL режим
	if err := s.enableWAL(); err != nil {
		return nil, fmt.Errorf("включение WAL: %w", err)
	}

	// Создаем схему
	if err := s.createSchema(); err != nil {
		return nil, fmt.Errorf("создание схемы: %w", err)
	}

	// Запускаем очистку старых метрик
	s.startCleanupRoutine()

	log.Printf("База данных открыта: %s (WAL режим, очистка 7 дней)", dbPath)

	return s, nil
}

// Close закрывает соединение
func (s *Storage) Close() error {
	return s.db.Close()
}

// enableWAL включает WAL режим для параллельных чтений
func (s *Storage) enableWAL() error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA cache_size=-65536", // 64 MB кэш
		"PRAGMA temp_store=MEMORY",
		"PRAGMA foreign_keys=ON",
	}

	for _, pragma := range pragmas {
		if _, err := s.db.Exec(pragma); err != nil {
			return fmt.Errorf("PRAGMA %s: %w", pragma, err)
		}
	}

	log.Println("WAL режим включен")
	return nil
}

// createSchema создает таблицы
func (s *Storage) createSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS computers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		hostname TEXT NOT NULL,
		domain TEXT,
		manufacturer TEXT,
		model TEXT,
		first_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(hostname, domain)
	);

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		full_name TEXT,
		domain TEXT,
		first_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		user_id INTEGER NOT NULL,
		computer_id INTEGER NOT NULL,
		login_time INTEGER NOT NULL,
		logout_time INTEGER,
		last_seen INTEGER,
		session_type TEXT,
		is_active BOOLEAN DEFAULT 1,
		FOREIGN KEY (user_id) REFERENCES users(id),
		FOREIGN KEY (computer_id) REFERENCES computers(id)
	);

	CREATE INDEX IF NOT EXISTS idx_sessions_active ON sessions(is_active, login_time DESC);
	CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id, login_time DESC);
	CREATE INDEX IF NOT EXISTS idx_sessions_computer ON sessions(computer_id, login_time DESC);

	CREATE TABLE IF NOT EXISTS connection_config (
		computer_id INTEGER PRIMARY KEY,
		rdp_enabled BOOLEAN DEFAULT 0,
		rdp_port INTEGER DEFAULT 3389,
		rdp_port_open BOOLEAN DEFAULT 0,
		vnc_enabled BOOLEAN DEFAULT 0,
		vnc_type TEXT,
		vnc_port INTEGER DEFAULT 5900,
		vnc_port_open BOOLEAN DEFAULT 0,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (computer_id) REFERENCES computers(id)
	);

	CREATE TABLE IF NOT EXISTS inventory (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		computer_id INTEGER NOT NULL,
		version INTEGER NOT NULL DEFAULT 1,
		os_name TEXT,
		os_edition TEXT,
		os_build TEXT,
		cpu_model TEXT,
		cpu_cores INTEGER,
		cpu_threads INTEGER,
		ram_total INTEGER,
		disks_json TEXT,
		network_json TEXT,
		captured_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (computer_id) REFERENCES computers(id),
		UNIQUE(computer_id, version)
	);

	CREATE INDEX IF NOT EXISTS idx_inventory_computer ON inventory(computer_id, version DESC);

	-- МЕТРИКИ: Использование RAM
	CREATE TABLE IF NOT EXISTS ram_metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		computer_id INTEGER NOT NULL,
		timestamp INTEGER NOT NULL,
		total_bytes INTEGER NOT NULL,
		available_bytes INTEGER NOT NULL,
		used_bytes INTEGER NOT NULL,
		used_percent REAL NOT NULL,
		FOREIGN KEY (computer_id) REFERENCES computers(id)
	);

	CREATE INDEX IF NOT EXISTS idx_ram_metrics ON ram_metrics(computer_id, timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_ram_metrics_time ON ram_metrics(timestamp);

	-- МЕТРИКИ: Свободное место на дисках
	CREATE TABLE IF NOT EXISTS disk_metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		computer_id INTEGER NOT NULL,
		timestamp INTEGER NOT NULL,
		letter TEXT NOT NULL,
		total_bytes INTEGER NOT NULL,
		free_bytes INTEGER NOT NULL,
		used_bytes INTEGER NOT NULL,
		free_percent REAL NOT NULL,
		FOREIGN KEY (computer_id) REFERENCES computers(id)
	);

	CREATE INDEX IF NOT EXISTS idx_disk_metrics ON disk_metrics(computer_id, timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_disk_metrics_time ON disk_metrics(timestamp);

	-- АЛЕРТЫ
	CREATE TABLE IF NOT EXISTS alerts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		computer_id INTEGER NOT NULL,
		alert_type TEXT NOT NULL,
		severity TEXT NOT NULL,
		message TEXT NOT NULL,
		value REAL NOT NULL,
		threshold REAL NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		resolved_at TIMESTAMP,
		FOREIGN KEY (computer_id) REFERENCES computers(id)
	);

	CREATE INDEX IF NOT EXISTS idx_alerts_active ON alerts(computer_id, resolved_at);
	CREATE INDEX IF NOT EXISTS idx_alerts_type ON alerts(alert_type, resolved_at);
	`

	_, err := s.db.Exec(schema)
	return err
}

// startCleanupRoutine запускает периодическую очистку
func (s *Storage) startCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			if err := s.CleanupOldData(); err != nil {
				log.Printf("Ошибка очистки: %v", err)
			}
		}
	}()

	log.Println("Запущена периодическая очистка (каждый час)")
}

// CleanupOldData удаляет старые данные
func (s *Storage) CleanupOldData() error {
	// Глубина хранения
	const (
		metricsRetentionDays  = 7  // Метрики: 7 дней
		sessionsRetentionDays = 90 // Сессии: 90 дней
		alertsRetentionDays   = 30 // Алерты: 30 дней (после решения)
	)

	// Вычисляем пороговые значения (Unix timestamp)
	metricsThreshold := time.Now().AddDate(0, 0, -metricsRetentionDays).Unix()
	sessionsThreshold := time.Now().AddDate(0, 0, -sessionsRetentionDays).Unix()
	alertsThreshold := time.Now().AddDate(0, 0, -alertsRetentionDays)

	// Начинаем транзакцию
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("начало транзакции: %w", err)
	}
	defer tx.Rollback()

	// Удаляем старые метрики RAM
	result, err := tx.Exec(`
		DELETE FROM ram_metrics WHERE timestamp < ?`,
		metricsThreshold,
	)
	if err != nil {
		return fmt.Errorf("очистка ram_metrics: %w", err)
	}
	ramDeleted, _ := result.RowsAffected()

	// Удаляем старые метрики дисков
	result, err = tx.Exec(`
		DELETE FROM disk_metrics WHERE timestamp < ?`,
		metricsThreshold,
	)
	if err != nil {
		return fmt.Errorf("очистка disk_metrics: %w", err)
	}
	diskDeleted, _ := result.RowsAffected()

	// Удаляем старые сессии (только неактивные)
	result, err = tx.Exec(`
		DELETE FROM sessions WHERE is_active = 0 AND login_time < ?`,
		sessionsThreshold,
	)
	if err != nil {
		return fmt.Errorf("очистка sessions: %w", err)
	}
	sessionsDeleted, _ := result.RowsAffected()

	// Удаляем старые решенные алерты (используем Unix timestamp)
	result, err = tx.Exec(`
		DELETE FROM alerts
		WHERE resolved_at IS NOT NULL
		AND strftime('%s', resolved_at) < ?`,
		alertsThreshold.Unix(),
	)
	if err != nil {
		return fmt.Errorf("очистка alerts: %w", err)
	}
	alertsDeleted, _ := result.RowsAffected()

	// Коммитим
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("коммит очистки: %w", err)
	}

	// Логируем результаты
	if ramDeleted+diskDeleted+sessionsDeleted+alertsDeleted > 0 {
		log.Printf("Очистка: RAM=%d, Диски=%d, Сессии=%d, Алерты=%d",
			ramDeleted, diskDeleted, sessionsDeleted, alertsDeleted)
	}

	return nil
}

// VACUUM выполняет оптимизацию базы данных
func (s *Storage) VACUUM() error {
	log.Println("Запуск VACUUM...")
	_, err := s.db.Exec("VACUUM")
	if err != nil {
		return fmt.Errorf("VACUUM: %w", err)
	}
	log.Println("VACUUM завершен")
	return nil
}

// GetDatabaseSize возвращает размер базы данных
func (s *Storage) GetDatabaseSize() (int64, error) {
	var pageCount int64
	var pageSize int64

	err := s.db.QueryRow("PRAGMA page_count").Scan(&pageCount)
	if err != nil {
		return 0, err
	}

	err = s.db.QueryRow("PRAGMA page_size").Scan(&pageSize)
	if err != nil {
		return 0, err
	}

	return pageCount * pageSize, nil
}

// SaveSnapshot сохраняет snapshot в БД
func (s *Storage) SaveSnapshot(snapshot *models.SystemSnapshot) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("начало транзакции: %w", err)
	}
	defer tx.Rollback()

	// 1. Компьютер
	computerID, err := s.getOrCreateComputer(tx, &snapshot.Host)
	if err != nil {
		return err
	}

	// 2. Пользователь
	userID, err := s.getOrCreateUser(tx, &snapshot.User)
	if err != nil {
		return err
	}

	// 3. Сессия
	if err := s.handleSession(tx, userID, computerID, snapshot.Timestamp, snapshot); err != nil {
		return err
	}

	// 4. Конфигурация подключения
	if err := s.updateConnectionConfig(tx, computerID, snapshot.Services); err != nil {
		return err
	}

	// 5. Инвентаризация
	if err := s.updateInventory(tx, computerID, snapshot); err != nil {
		return err
	}

	// 6. Метрики RAM
	if err := s.saveRAMMetrics(tx, computerID, snapshot.Timestamp, &snapshot.RAM); err != nil {
		return err
	}

	// 7. Метрики дисков
	if err := s.saveDiskMetrics(tx, computerID, snapshot.Timestamp, snapshot.Drives); err != nil {
		return err
	}

	// 8. Алерты
	if err := s.checkAlerts(tx, computerID, &snapshot.RAM, snapshot.Drives); err != nil {
		return err
	}

	return tx.Commit()
}

// getOrCreateComputer находит или создает компьютер
func (s *Storage) getOrCreateComputer(tx *sql.Tx, host *models.HostInfo) (int64, error) {
	var id int64
	err := tx.QueryRow(`
		SELECT id FROM computers
		WHERE hostname = ? AND COALESCE(domain, '') = COALESCE(?, '')`,
		host.Hostname, host.Domain,
	).Scan(&id)

	if err == sql.ErrNoRows {
		result, err := tx.Exec(`
			INSERT INTO computers (hostname, domain, manufacturer, model)
			VALUES (?, ?, ?, ?)`,
			host.Hostname, host.Domain, host.Manufacturer, host.Model,
		)
		if err != nil {
			return 0, err
		}
		id, err = result.LastInsertId()
		if err != nil {
			return 0, err
		}
		log.Printf("Создан новый компьютер: %s (id=%d)", host.Hostname, id)
	} else if err != nil {
		return 0, err
	}

	_, err = tx.Exec(`
		UPDATE computers SET
			last_seen = CURRENT_TIMESTAMP,
			manufacturer = COALESCE(?, manufacturer),
			model = COALESCE(?, model)
		WHERE id = ?`,
		host.Manufacturer, host.Model, id,
	)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// getOrCreateUser находит или создает пользователя
func (s *Storage) getOrCreateUser(tx *sql.Tx, u *models.UserInfo) (int64, error) {
	var id int64
	err := tx.QueryRow(`SELECT id FROM users WHERE username = ?`, u.Username).Scan(&id)

	if err == sql.ErrNoRows {
		result, err := tx.Exec(`
			INSERT INTO users (username, full_name, domain)
			VALUES (?, ?, ?)`,
			u.Username, u.FullName, u.Domain,
		)
		if err != nil {
			return 0, err
		}
		id, err = result.LastInsertId()
		if err != nil {
			return 0, err
		}
		log.Printf("Создан новый пользователь: %s (id=%d)", u.Username, id)
	} else if err != nil {
		return 0, err
	}

	_, err = tx.Exec(`
		UPDATE users SET
			last_seen = CURRENT_TIMESTAMP,
			full_name = COALESCE(?, full_name),
			domain = COALESCE(?, domain)
		WHERE id = ?`,
		u.FullName, u.Domain, id,
	)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// handleSession обрабатывает сессию с session_type
func (s *Storage) handleSession(tx *sql.Tx, userID, computerID int64, timestamp int64, snapshot *models.SystemSnapshot) error {
	// Определяем session_type
	sessionType := s.determineSessionType(snapshot)

	var activeSessionID int64
	var activeComputerID int64

	err := tx.QueryRow(`
		SELECT id, computer_id FROM sessions
		WHERE user_id = ? AND is_active = 1
		ORDER BY login_time DESC LIMIT 1`,
		userID,
	).Scan(&activeSessionID, &activeComputerID)

	if err == sql.ErrNoRows {
		_, err := tx.Exec(`
			INSERT INTO sessions (user_id, computer_id, login_time, last_seen, session_type, is_active)
			VALUES (?, ?, ?, ?, ?, 1)`,
			userID, computerID, timestamp, timestamp, sessionType,
		)
		if err != nil {
			return err
		}
		log.Printf("Создана новая сессия: user=%d, computer=%d, type=%s",
			userID, computerID, sessionType)
		return nil
	} else if err != nil {
		return err
	}

	if activeComputerID != computerID {
		_, err := tx.Exec(`
			UPDATE sessions SET is_active = 0, logout_time = ?
			WHERE id = ?`,
			timestamp, activeSessionID,
		)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			INSERT INTO sessions (user_id, computer_id, login_time, last_seen, session_type, is_active)
			VALUES (?, ?, ?, ?, ?, 1)`,
			userID, computerID, timestamp, timestamp, sessionType,
		)
		if err != nil {
			return err
		}
		log.Printf("Пользователь перешел на другой компьютер. Новая сессия: type=%s", sessionType)
		return nil
	}

	// Тот же компьютер — обновляем last_seen и session_type
	_, err = tx.Exec(`
		UPDATE sessions SET last_seen = ?, session_type = ?, logout_time = NULL
		WHERE id = ?`,
		timestamp, sessionType, activeSessionID,
	)
	if err != nil {
		return err
	}

	return nil
}

// determineSessionType определяет тип сессии
func (s *Storage) determineSessionType(snapshot *models.SystemSnapshot) string {
	// Если есть информация о сессии в snapshot, используем её
	// Пока определяем по наличию RDP подключения
	// В будущем можно добавить поле в snapshot

	// Проверяем, есть ли активное RDP подключение
	for _, svc := range snapshot.Services {
		if svc.Name == "RDP" && svc.Running {
			// Если RDP запущен, пользователь может быть как локально, так и удаленно
			// Определяем по uptime: если uptime маленький, вероятно, только что вошел
			return "console" // По умолчанию console
		}
	}

	return "console" // По умолчанию console
}

// updateConnectionConfig обновляет конфигурацию подключения
func (s *Storage) updateConnectionConfig(tx *sql.Tx, computerID int64, services models.ServicesStatuses) error {
	var rdpEnabled, rdpOpen bool
	var rdpPort uint16 = 3389
	var vncEnabled, vncOpen bool
	var vncType string
	var vncPort uint16 = 5900

	for _, svc := range services {
		if svc.Name == "RDP" {
			rdpEnabled = svc.Installed
			rdpOpen = svc.PortOpen
			if svc.Port > 0 {
				rdpPort = svc.Port
			}
		}

		if svc.Name == "VNC" || (len(svc.Name) > 4 && svc.Name[:4] == "VNC (") {
			vncEnabled = svc.Installed
			vncOpen = svc.PortOpen
			if len(svc.Name) > 5 {
				vncType = svc.Name[5 : len(svc.Name)-1]
			}
			if svc.Port > 0 {
				vncPort = svc.Port
			}
		}
	}

	_, err := tx.Exec(`
		INSERT OR REPLACE INTO connection_config (
			computer_id, rdp_enabled, rdp_port, rdp_port_open,
			vnc_enabled, vnc_type, vnc_port, vnc_port_open, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		computerID, rdpEnabled, rdpPort, rdpOpen,
		vncEnabled, vncType, vncPort, vncOpen,
	)
	return err
}

// updateInventory обновляет инвентаризацию
func (s *Storage) updateInventory(tx *sql.Tx, computerID int64, snapshot *models.SystemSnapshot) error {
	disksJSON, _ := json.Marshal(snapshot.Drives)
	networkJSON, _ := json.Marshal(snapshot.Network)

	var lastVersion int
	var lastRAMTotal int64
	var lastCPUModel string

	err := tx.QueryRow(`
		SELECT version, COALESCE(ram_total, 0), COALESCE(cpu_model, '')
		FROM inventory
		WHERE computer_id = ?
		ORDER BY version DESC LIMIT 1`,
		computerID,
	).Scan(&lastVersion, &lastRAMTotal, &lastCPUModel)

	if err != sql.ErrNoRows && err != nil {
		return err
	}

	changed := false
	if err == sql.ErrNoRows {
		changed = true
	} else {
		if lastRAMTotal != int64(snapshot.RAM.TotalBytes) {
			changed = true
		}
		if lastCPUModel != snapshot.Processor.Model {
			changed = true
		}
	}

	if !changed {
		return nil
	}

	newVersion := lastVersion + 1
	_, err = tx.Exec(`
		INSERT INTO inventory (
			computer_id, version, os_name, os_edition, os_build,
			cpu_model, cpu_cores, cpu_threads, ram_total,
			disks_json, network_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		computerID, newVersion,
		snapshot.OS.Name, snapshot.OS.Edition, snapshot.OS.BuildNumber,
		snapshot.Processor.Model, snapshot.Processor.PhysicalCores,
		snapshot.Processor.LogicalProcessors, snapshot.RAM.TotalBytes,
		string(disksJSON), string(networkJSON),
	)
	return err
}

// saveRAMMetrics сохраняет метрики RAM
func (s *Storage) saveRAMMetrics(tx *sql.Tx, computerID int64, timestamp int64, ram *models.RAMInfo) error {
	if ram.TotalBytes == 0 {
		return nil
	}

	usedBytes := ram.TotalBytes - ram.AvailableBytes
	usedPercent := float64(usedBytes) / float64(ram.TotalBytes) * 100

	_, err := tx.Exec(`
		INSERT INTO ram_metrics (
			computer_id, timestamp, total_bytes, available_bytes, used_bytes, used_percent
		) VALUES (?, ?, ?, ?, ?, ?)`,
		computerID, timestamp, ram.TotalBytes, ram.AvailableBytes, usedBytes, usedPercent,
	)
	return err
}

// saveDiskMetrics сохраняет метрики дисков
func (s *Storage) saveDiskMetrics(tx *sql.Tx, computerID int64, timestamp int64, drives models.DiskStatuses) error {
	for _, d := range drives {
		if d.TotalBytes == 0 || !d.IsReady {
			continue
		}

		usedBytes := d.TotalBytes - d.FreeBytes
		freePercent := float64(d.FreeBytes) / float64(d.TotalBytes) * 100

		_, err := tx.Exec(`
			INSERT INTO disk_metrics (
				computer_id, timestamp, letter, total_bytes, free_bytes, used_bytes, free_percent
			) VALUES (?, ?, ?, ?, ?, ?, ?)`,
			computerID, timestamp, d.Letter, d.TotalBytes, d.FreeBytes, usedBytes, freePercent,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// checkAlerts проверяет пороговые значения и создает алерты
func (s *Storage) checkAlerts(tx *sql.Tx, computerID int64, ram *models.RAMInfo, drives models.DiskStatuses) error {
	// Проверка RAM
	if ram.TotalBytes > 0 {
		usedPercent := float64(ram.TotalBytes-ram.AvailableBytes) / float64(ram.TotalBytes) * 100

		if usedPercent >= 95 {
			if err := s.createAlert(tx, computerID, "ram_high_usage", "critical",
				fmt.Sprintf("RAM usage: %.1f%%", usedPercent), usedPercent, 95); err != nil {
				return err
			}
		} else if usedPercent >= 80 {
			if err := s.createAlert(tx, computerID, "ram_high_usage", "warning",
				fmt.Sprintf("RAM usage: %.1f%%", usedPercent), usedPercent, 80); err != nil {
				return err
			}
		} else {
			// Проблема решена — закрываем алерт
			if err := s.resolveAlert(tx, computerID, "ram_high_usage"); err != nil {
				return err
			}
		}
	}

	// Проверка дисков
	for _, d := range drives {
		if d.TotalBytes == 0 || !d.IsReady {
			continue
		}

		freePercent := float64(d.FreeBytes) / float64(d.TotalBytes) * 100

		if freePercent <= 5 {
			if err := s.createAlert(tx, computerID, "disk_low_space", "critical",
				fmt.Sprintf("Disk %s: %.1f%% free", d.Letter, freePercent), freePercent, 5); err != nil {
				return err
			}
		} else if freePercent <= 20 {
			if err := s.createAlert(tx, computerID, "disk_low_space", "warning",
				fmt.Sprintf("Disk %s: %.1f%% free", d.Letter, freePercent), freePercent, 20); err != nil {
				return err
			}
		} else {
			// Проблема решена — закрываем алерт
			if err := s.resolveAlert(tx, computerID, "disk_low_space"); err != nil {
				return err
			}
		}
	}

	return nil
}

// createAlert создает алерт (если он еще не активен)
func (s *Storage) createAlert(tx *sql.Tx, computerID int64, alertType, severity, message string, value, threshold float64) error {
	// Проверяем, есть ли уже активный алерт такого же типа
	var existingID int64
	err := tx.QueryRow(`
		SELECT id FROM alerts
		WHERE computer_id = ? AND alert_type = ? AND resolved_at IS NULL`,
		computerID, alertType,
	).Scan(&existingID)

	if err == sql.ErrNoRows {
		// Создаем новый алерт
		_, err := tx.Exec(`
			INSERT INTO alerts (computer_id, alert_type, severity, message, value, threshold)
			VALUES (?, ?, ?, ?, ?, ?)`,
			computerID, alertType, severity, message, value, threshold,
		)
		if err != nil {
			return err
		}
		log.Printf("Создан алерт: %s (severity=%s, value=%.1f%%)",
			alertType, severity, value)
		return nil
	} else if err != nil {
		return err
	}

	// Алерт уже существует — обновляем severity и value
	_, err = tx.Exec(`
		UPDATE alerts SET severity = ?, value = ?, message = ?
		WHERE id = ?`,
		severity, value, message, existingID,
	)
	return err
}

// resolveAlert закрывает алерт (проблема решена)
func (s *Storage) resolveAlert(tx *sql.Tx, computerID int64, alertType string) error {
	_, err := tx.Exec(`
		UPDATE alerts SET resolved_at = CURRENT_TIMESTAMP
		WHERE computer_id = ? AND alert_type = ? AND resolved_at IS NULL`,
		computerID, alertType,
	)
	return err
}
