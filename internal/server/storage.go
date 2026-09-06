package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/alme23/tracker/internal/models"
	_ "modernc.org/sqlite" // SQLite driver (pure Go)
)

// Storage handles data persistence in SQLite
type Storage struct {
	db *sql.DB
}

// NewStorage creates a new storage instance
func NewStorage(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("database open: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	// Use context with timeout for ping
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("database ping: %w", err)
	}

	s := &Storage{db: db}

	if err := s.enableWAL(); err != nil {
		return nil, fmt.Errorf("WAL enable: %w", err)
	}

	if err := s.createSchema(); err != nil {
		return nil, fmt.Errorf("schema creation: %w", err)
	}

	s.startCleanupRoutine()

	log.Printf("Database opened: %s (WAL mode, 7-day retention)", dbPath)

	return s, nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	return s.db.Close()
}

// CleanupOldData removes data older than retention periods
func (s *Storage) CleanupOldData() error {
	const (
		metricsRetentionDays  = 7  // Metrics retention period
		sessionsRetentionDays = 90 // Sessions retention period
		alertsRetentionDays   = 30 // Alerts retention period (after resolution)
	)

	metricsThreshold := time.Now().AddDate(0, 0, -metricsRetentionDays).Unix()
	sessionsThreshold := time.Now().AddDate(0, 0, -sessionsRetentionDays).Unix()

	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("transaction start: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// Delete old RAM metrics
	result, err := tx.ExecContext(context.Background(), `DELETE FROM ram_metrics WHERE timestamp < ?`, metricsThreshold)
	if err != nil {
		return fmt.Errorf("ram_metrics cleanup: %w", err)
	}
	ramDeleted, _ := result.RowsAffected()

	// Delete old disk metrics
	result, err = tx.ExecContext(context.Background(), `DELETE FROM disk_metrics WHERE timestamp < ?`, metricsThreshold)
	if err != nil {
		return fmt.Errorf("disk_metrics cleanup: %w", err)
	}
	diskDeleted, _ := result.RowsAffected()

	// Delete old inactive sessions
	result, err = tx.ExecContext(context.Background(), `DELETE FROM sessions WHERE is_active = 0 AND login_time < ?`, sessionsThreshold)
	if err != nil {
		return fmt.Errorf("sessions cleanup: %w", err)
	}
	sessionsDeleted, _ := result.RowsAffected()

	// Delete old resolved alerts
	result, err = tx.ExecContext(context.Background(), `
		DELETE FROM alerts
		WHERE resolved_at IS NOT NULL
		AND resolved_at < datetime('now', ?)`,
		fmt.Sprintf("-%d days", alertsRetentionDays),
	)
	if err != nil {
		return fmt.Errorf("alerts cleanup: %w", err)
	}
	alertsDeleted, _ := result.RowsAffected()

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("cleanup commit: %w", err)
	}

	if ramDeleted+diskDeleted+sessionsDeleted+alertsDeleted > 0 {
		log.Printf("Cleanup: RAM=%d, Disks=%d, Sessions=%d, Alerts=%d",
			ramDeleted, diskDeleted, sessionsDeleted, alertsDeleted)
	}

	return nil
}

// SaveSnapshot saves a snapshot to the database
func (s *Storage) SaveSnapshot(snapshot *models.SystemSnapshot) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("transaction start: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	// 1. Computer
	computerID, err := s.getOrCreateComputer(tx, &snapshot.Host)
	if err != nil {
		return err
	}

	// 2. User
	userID, err := s.getOrCreateUser(tx, &snapshot.User)
	if err != nil {
		return err
	}

	// 3. Session
	if err := s.handleSession(tx, userID, computerID, snapshot.Timestamp, snapshot); err != nil {
		return err
	}

	// 4. Connection config
	if err := s.updateConnectionConfig(tx, computerID, snapshot.Services); err != nil {
		return err
	}

	// 5. Inventory
	if err := s.updateInventory(tx, computerID, snapshot); err != nil {
		return err
	}

	// 6. RAM metrics
	if err := s.saveRAMMetrics(tx, computerID, snapshot.Timestamp, &snapshot.RAM); err != nil {
		return err
	}

	// 7. Disk metrics
	if err := s.saveDiskMetrics(tx, computerID, snapshot.Timestamp, snapshot.Drives); err != nil {
		return err
	}

	// 8. Alerts
	if err := s.checkAlerts(tx, computerID, &snapshot.RAM, snapshot.Drives); err != nil {
		return err
	}

	return tx.Commit()
}

// GetDatabaseSize returns the database size in bytes
func (s *Storage) GetDatabaseSize() (int64, error) {
	var pageCount, pageSize int64

	if err := s.db.QueryRowContext(context.Background(), "PRAGMA page_count").Scan(&pageCount); err != nil {
		return 0, err
	}

	if err := s.db.QueryRowContext(context.Background(), "PRAGMA page_size").Scan(&pageSize); err != nil {
		return 0, err
	}

	return pageCount * pageSize, nil
}

// enableWAL enables WAL mode for better performance
func (s *Storage) enableWAL() error {
	pragmas := []string{
		"PRAGMA journal_mode=WAL",
		"PRAGMA synchronous=NORMAL",
		"PRAGMA busy_timeout=5000",
		"PRAGMA cache_size=-65536",
		"PRAGMA temp_store=MEMORY",
		"PRAGMA foreign_keys=ON",
	}

	for _, pragma := range pragmas {
		if _, err := s.db.ExecContext(context.Background(), pragma); err != nil {
			return fmt.Errorf("PRAGMA %s: %w", pragma, err)
		}
	}

	return nil
}

// createSchema creates database tables
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

	_, err := s.db.ExecContext(context.Background(), schema)
	return err
}

// startCleanupRoutine starts periodic data cleanup
func (s *Storage) startCleanupRoutine() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			if err := s.CleanupOldData(); err != nil {
				log.Printf("Cleanup error: %v", err)
			}
		}
	}()
}

// getOrCreateComputer finds or creates a computer record
func (s *Storage) getOrCreateComputer(tx *sql.Tx, host *models.HostInfo) (int64, error) {
	var id int64
	err := tx.QueryRowContext(context.Background(), `
		SELECT id FROM computers
		WHERE hostname = ? AND COALESCE(domain, '') = COALESCE(?, '')`,
		host.Hostname, host.Domain,
	).Scan(&id)

	if errors.Is(err, sql.ErrNoRows) {
		result, err := tx.ExecContext(context.Background(), `
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
		log.Printf("New computer created: %s (id=%d)", host.Hostname, id)
	} else if err != nil {
		return 0, err
	}

	_, err = tx.ExecContext(context.Background(), `
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

// getOrCreateUser finds or creates a user record
func (s *Storage) getOrCreateUser(tx *sql.Tx, u *models.UserInfo) (int64, error) {
	var id int64
	err := tx.QueryRowContext(context.Background(), `SELECT id FROM users WHERE username = ?`, u.Username).Scan(&id)

	if errors.Is(err, sql.ErrNoRows) {
		result, err := tx.ExecContext(context.Background(), `
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
		log.Printf("New user created: %s (id=%d)", u.Username, id)
	} else if err != nil {
		return 0, err
	}

	_, err = tx.ExecContext(context.Background(), `
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

// handleSession processes user session
func (s *Storage) handleSession(tx *sql.Tx, userID, computerID, timestamp int64, snapshot *models.SystemSnapshot) error {
	sessionType := s.determineSessionType(snapshot)

	var activeSessionID int64
	var activeComputerID int64

	err := tx.QueryRowContext(context.Background(), `
		SELECT id, computer_id FROM sessions
		WHERE user_id = ? AND is_active = 1
		ORDER BY login_time DESC LIMIT 1`,
		userID,
	).Scan(&activeSessionID, &activeComputerID)

	if errors.Is(err, sql.ErrNoRows) {
		_, err := tx.ExecContext(context.Background(), `
			INSERT INTO sessions (user_id, computer_id, login_time, last_seen, session_type, is_active)
			VALUES (?, ?, ?, ?, ?, 1)`,
			userID, computerID, timestamp, timestamp, sessionType,
		)
		if err != nil {
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	if activeComputerID != computerID {
		_, err := tx.ExecContext(context.Background(), `
			UPDATE sessions SET is_active = 0, logout_time = ?
			WHERE id = ?`,
			timestamp, activeSessionID,
		)
		if err != nil {
			return err
		}

		_, err = tx.ExecContext(context.Background(), `
			INSERT INTO sessions (user_id, computer_id, login_time, last_seen, session_type, is_active)
			VALUES (?, ?, ?, ?, ?, 1)`,
			userID, computerID, timestamp, timestamp, sessionType,
		)
		return err
	}

	_, err = tx.ExecContext(context.Background(), `
		UPDATE sessions SET last_seen = ?, session_type = ?, logout_time = NULL
		WHERE id = ?`,
		timestamp, sessionType, activeSessionID,
	)
	return err
}

// determineSessionType determines the session type
// TODO: use snapshot to determine session type (Console/RDP)
func (s *Storage) determineSessionType(_ *models.SystemSnapshot) string {
	return "console"
}

// updateConnectionConfig updates RDP/VNC connection settings
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

	_, err := tx.ExecContext(context.Background(), `
		INSERT OR REPLACE INTO connection_config (
			computer_id, rdp_enabled, rdp_port, rdp_port_open,
			vnc_enabled, vnc_type, vnc_port, vnc_port_open, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)`,
		computerID, rdpEnabled, rdpPort, rdpOpen,
		vncEnabled, vncType, vncPort, vncOpen,
	)
	return err
}

// updateInventory updates inventory when changes are detected
func (s *Storage) updateInventory(tx *sql.Tx, computerID int64, snapshot *models.SystemSnapshot) error {
	// Serialize drives with error check
	disksJSON, err := json.Marshal(snapshot.Drives)
	if err != nil {
		return fmt.Errorf("drives serialization: %w", err)
	}

	// Serialize network with error check
	networkJSON, err := json.Marshal(snapshot.Network)
	if err != nil {
		return fmt.Errorf("network serialization: %w", err)
	}

	// Get last version
	var lastVersion int
	var lastRAMTotal uint64 // Changed from int64 to uint64
	var lastCPUModel string

	err = tx.QueryRowContext(context.Background(), `
		SELECT version, COALESCE(ram_total, 0), COALESCE(cpu_model, '')
		FROM inventory
		WHERE computer_id = ?
		ORDER BY version DESC LIMIT 1`,
		computerID,
	).Scan(&lastVersion, &lastRAMTotal, &lastCPUModel)

	if !errors.Is(err, sql.ErrNoRows) && err != nil {
		return err
	}

	changed := false
	if errors.Is(err, sql.ErrNoRows) {
		changed = true // First record
	} else {
		if lastRAMTotal != snapshot.RAM.TotalBytes {
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
	_, err = tx.ExecContext(context.Background(), `
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
	if err != nil {
		return fmt.Errorf("inventory creation: %w", err)
	}

	return nil
}

// saveRAMMetrics saves RAM usage metrics
func (s *Storage) saveRAMMetrics(tx *sql.Tx, computerID, timestamp int64, ram *models.RAMInfo) error {
	if ram.TotalBytes == 0 {
		return nil
	}

	usedBytes := ram.TotalBytes - ram.AvailableBytes
	usedPercent := float64(usedBytes) / float64(ram.TotalBytes) * 100

	_, err := tx.ExecContext(context.Background(), `
		INSERT INTO ram_metrics (
			computer_id, timestamp, total_bytes, available_bytes, used_bytes, used_percent
		) VALUES (?, ?, ?, ?, ?, ?)`,
		computerID, timestamp, ram.TotalBytes, ram.AvailableBytes, usedBytes, usedPercent,
	)
	return err
}

// saveDiskMetrics saves disk space metrics
func (s *Storage) saveDiskMetrics(tx *sql.Tx, computerID, timestamp int64, drives models.DiskStatuses) error {
	for _, d := range drives {
		if d.TotalBytes == 0 || !d.IsReady {
			continue
		}

		usedBytes := d.TotalBytes - d.FreeBytes
		freePercent := float64(d.FreeBytes) / float64(d.TotalBytes) * 100

		_, err := tx.ExecContext(context.Background(), `
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

// checkAlerts checks threshold values and creates alerts
func (s *Storage) checkAlerts(tx *sql.Tx, computerID int64, ram *models.RAMInfo, drives models.DiskStatuses) error {
	// RAM check
	if ram.TotalBytes > 0 {
		usedPercent := float64(ram.TotalBytes-ram.AvailableBytes) / float64(ram.TotalBytes) * 100

		switch {
		case usedPercent >= 95:
			if err := s.createAlert(tx, computerID, "ram_high_usage", "critical",
				fmt.Sprintf("RAM usage: %.1f%%", usedPercent), usedPercent, 95); err != nil {
				return err
			}
		case usedPercent >= 80:
			if err := s.createAlert(tx, computerID, "ram_high_usage", "warning",
				fmt.Sprintf("RAM usage: %.1f%%", usedPercent), usedPercent, 80); err != nil {
				return err
			}
		default:
			if err := s.resolveAlert(tx, computerID, "ram_high_usage"); err != nil {
				return err
			}
		}
	}

	// Disk check
	for _, d := range drives {
		if d.TotalBytes == 0 || !d.IsReady {
			continue
		}

		freePercent := float64(d.FreeBytes) / float64(d.TotalBytes) * 100

		switch {
		case freePercent <= 5:
			if err := s.createAlert(tx, computerID, "disk_low_space", "critical",
				fmt.Sprintf("Disk %s: %.1f%% free", d.Letter, freePercent), freePercent, 5); err != nil {
				return err
			}
		case freePercent <= 20:
			if err := s.createAlert(tx, computerID, "disk_low_space", "warning",
				fmt.Sprintf("Disk %s: %.1f%% free", d.Letter, freePercent), freePercent, 20); err != nil {
				return err
			}
		default:
			if err := s.resolveAlert(tx, computerID, "disk_low_space"); err != nil {
				return err
			}
		}
	}

	return nil
}

// createAlert creates a new alert or updates an existing one
func (s *Storage) createAlert(tx *sql.Tx, computerID int64, alertType, severity, message string, value, threshold float64) error {
	var existingID int64
	err := tx.QueryRowContext(context.Background(), `
		SELECT id FROM alerts
		WHERE computer_id = ? AND alert_type = ? AND resolved_at IS NULL`,
		computerID, alertType,
	).Scan(&existingID)

	if errors.Is(err, sql.ErrNoRows) {
		_, err := tx.ExecContext(context.Background(), `
			INSERT INTO alerts (computer_id, alert_type, severity, message, value, threshold)
			VALUES (?, ?, ?, ?, ?, ?)`,
			computerID, alertType, severity, message, value, threshold,
		)
		return err
	} else if err != nil {
		return err
	}

	_, err = tx.ExecContext(context.Background(), `
		UPDATE alerts SET severity = ?, value = ?, message = ?
		WHERE id = ?`,
		severity, value, message, existingID,
	)
	return err
}

// resolveAlert closes an active alert
func (s *Storage) resolveAlert(tx *sql.Tx, computerID int64, alertType string) error {
	_, err := tx.ExecContext(context.Background(), `
		UPDATE alerts SET resolved_at = CURRENT_TIMESTAMP
		WHERE computer_id = ? AND alert_type = ? AND resolved_at IS NULL`,
		computerID, alertType,
	)
	return err
}
