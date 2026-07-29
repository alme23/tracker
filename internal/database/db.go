package database

import (
	"database/sql"
	"fmt"

	// Драйвер modernc sqlite на чистом Go
	_ "modernc.org/sqlite"
)

type Repository struct {
	db *sql.DB
}

// NewRepository открывает SQLite и настраивает пул без дедлоков
func NewRepository(dbPath string) (*Repository, error) {
	// Включаем foreign keys и WAL режим
	dsn := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", dbPath)

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite with dsn: %w", err)
	}

	// ИСПРАВЛЕНО: Разрешаем параллельное чтение из базы (до 100 одновременных запросов)!
	// Это полностью ликвидирует дедлоки и таймауты между веб-панелью, поиском и SSE
	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(1)

	repo := &Repository{db: db}
	if err := repo.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return repo, nil
}

func (r *Repository) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}

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

		`CREATE TABLE IF NOT EXISTS session_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp INTEGER,
			computer_name TEXT,
			machine_guid TEXT,
			user_login TEXT,
			user_fullname TEXT,
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
