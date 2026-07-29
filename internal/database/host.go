package database

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/alme23/tracker/pkg/collector"
)

func (r *Repository) SaveMetrics(m collector.Metrics, ip string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	success := false
	defer func() {
		if !success {
			_ = tx.Rollback()
		}
	}()

	hostQuery := `
	INSERT INTO host_metrics (
		timestamp, computer_name, fqdn, domain_name, os_name, os_edition, os_version, os_build,
		install_date_str, architecture, product_id, machine_guid, user_login, user_fullname,
		user_domain, user_join_type, bios_vendor, bios_version, bios_release_date,
		baseboard_vendor, baseboard_product, baseboard_version, baseboard_serial,
		cpu_name, cpu_cores, cpu_threads, cpu_speed_mhz, cpu_l1_bytes, cpu_l2_bytes, cpu_l3_bytes,
		ram_total_bytes, ram_free_bytes, client_ip
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);`

	// Внутри функции SaveMetrics в файле internal/database/host.go замените блок tx.Exec на этот:
	res, err := tx.Exec(hostQuery,
		m.Timestamp, m.Host.ComputerName, m.Host.FQDN, m.Host.DomainName, m.Host.OSName, m.Host.OSEdition, m.Host.OSVersion,
		m.RemoteAccess.RDPStatus, // ИСПРАВЛЕНО: Пишем статус RDP в поле os_build
		m.Host.InstallDateStr,
		m.RemoteAccess.VNCStatus, // ИСПРАВЛЕНО: Пишем статус VNC в поле architecture
		m.Host.ProductID, m.Host.DeviceID, m.User.UserName, m.User.DisplayName,
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

	if len(m.Video) > 0 {
		gpuQuery := `INSERT INTO gpu_metrics (host_id, name, vendor_id, device_id, dedicated_memory, shared_memory) VALUES (?, ?, ?, ?, ?, ?);`
		for _, v := range m.Video {
			if _, err = tx.Exec(gpuQuery, hostID, v.Name, v.VendorID, v.DeviceID, v.DedicatedMemory, v.SharedMemory); err != nil {
				return fmt.Errorf("failed to insert GPU: %w", err)
			}
		}
	}

	if len(m.Disks) > 0 {
		diskQuery := `INSERT INTO disk_metrics (host_id, drive, type, vendor, serial_number, total_bytes, free_bytes) VALUES (?, ?, ?, ?, ?, ?, ?);`
		for _, d := range m.Disks {
			if _, err = tx.Exec(diskQuery, hostID, d.Drive, d.Type, d.Vendor, d.SerialNumber, d.TotalBytes, d.FreeBytes); err != nil {
				return fmt.Errorf("failed to insert disk: %w", err)
			}
		}
	}

	if len(m.Network) > 0 {
		netQuery := `INSERT INTO network_metrics (host_id, name, mac, status, ip_type, ip_addresses) VALUES (?, ?, ?, ?, ?, ?);`
		for _, n := range m.Network {
			var ips []string
			for _, ip := range n.IPAddresses {
				ips = append(ips, ip.String())
			}
			ipListStr := strings.Join(ips, ", ")

			if _, err = tx.Exec(netQuery, hostID, n.Name, n.MAC.String(), n.Status.String(), n.IPType.String(), ipListStr); err != nil {
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

func (r *Repository) GetLastMetrics(machineGUID string) (collector.Metrics, bool, error) {
	var m collector.Metrics
	var hostID int64

	query := `
	SELECT id, timestamp, cpu_name, cpu_cores, cpu_threads, ram_total_bytes
	FROM host_metrics
	WHERE machine_guid = ?
	ORDER BY timestamp DESC LIMIT 1;`

	err := r.db.QueryRow(query, machineGUID).Scan(
		&hostID, &m.Timestamp, &m.CPU.Name, &m.CPU.PhysicalCores, &m.CPU.LogicalProcessors, &m.Memory.TotalRAMBytes,
	)

	if err == sql.ErrNoRows {
		return m, false, nil
	}
	if err != nil {
		return m, false, fmt.Errorf("failed to query last host metrics: %w", err)
	}

	m.Host.DeviceID = machineGUID

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
