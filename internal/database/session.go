package database

import (
	"database/sql"
	"fmt"

	"github.com/alme23/tracker/pkg/collector"
)

func (r *Repository) LogSession(m collector.Metrics, ip string) error {
	var lastID int64
	var lastTimestamp int64
	var lastUser string
	var lastIP string

	queryLast := `
	SELECT id, timestamp, user_login, client_ip
	FROM session_logs
	WHERE machine_guid = ?
	ORDER BY timestamp DESC LIMIT 1;`

	err := r.db.QueryRow(queryLast, m.Host.DeviceID).Scan(&lastID, &lastTimestamp, &lastUser, &lastIP)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to query last session log: %w", err)
	}

	const throttleInterval = 3600

	if err == sql.ErrNoRows || lastUser != m.User.UserName || lastIP != ip || (m.Timestamp-lastTimestamp) > throttleInterval {
		queryInsert := `
		INSERT INTO session_logs (timestamp, computer_name, machine_guid, user_login, user_fullname, client_ip)
		VALUES (?, ?, ?, ?, ?, ?);`

		_, err = r.db.Exec(queryInsert, m.Timestamp, m.Host.ComputerName, m.Host.DeviceID, m.User.UserName, m.User.DisplayName, ip)
		if err != nil {
			return fmt.Errorf("failed to insert new session log: %w", err)
		}
	}

	return nil
}
