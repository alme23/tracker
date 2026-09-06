// tracker/internal/binproto/encoder.go

// Package binproto provides a lightweight, high-performance binary serialization
// protocol for SystemSnapshot, optimized for network transmission.
package binproto

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/alme23/tracker/internal/models"
)

// MagicHeader идентифицирует формат данных при передаче
const MagicHeader = "TRCK1"

// Encoder кодирует SystemSnapshot в компактный бинарный формат
type Encoder struct {
	buf *bytes.Buffer
}

// NewEncoder создает новый бинарный кодер
func NewEncoder() *Encoder {
	return &Encoder{
		buf: bytes.NewBuffer(make([]byte, 0, 4096)), // 4KB предварительно
	}
}

// Encode сериализует SystemSnapshot в бинарный формат
func (e *Encoder) Encode(s *models.SystemSnapshot) ([]byte, error) {
	e.buf.Reset()

	// Magic header для версионирования
	if _, err := e.buf.WriteString(MagicHeader); err != nil {
		return nil, fmt.Errorf("write magic header: %w", err)
	}

	// Timestamp (int64 Unix timestamp)
	if err := binary.Write(e.buf, binary.LittleEndian, s.Timestamp); err != nil {
		return nil, fmt.Errorf("write timestamp: %w", err)
	}

	// User
	if err := e.encodeUser(&s.User); err != nil {
		return nil, fmt.Errorf("encode user: %w", err)
	}

	// OS
	if err := e.encodeOS(&s.OS); err != nil {
		return nil, fmt.Errorf("encode os: %w", err)
	}

	// Processor
	if err := e.encodeProcessor(&s.Processor); err != nil {
		return nil, fmt.Errorf("encode processor: %w", err)
	}

	// RAM
	if err := e.encodeRAM(&s.RAM); err != nil {
		return nil, fmt.Errorf("encode ram: %w", err)
	}

	// Drives
	if err := e.encodeDrives(s.Drives); err != nil {
		return nil, fmt.Errorf("encode drives: %w", err)
	}

	// Services
	if err := e.encodeServices(s.Services); err != nil {
		return nil, fmt.Errorf("encode services: %w", err)
	}

	// Network
	if err := e.encodeNetwork(s.Network); err != nil {
		return nil, fmt.Errorf("encode network: %w", err)
	}

	// Host
	if err := e.encodeHost(&s.Host); err != nil {
		return nil, fmt.Errorf("encode host: %w", err)
	}

	return e.buf.Bytes(), nil
}

// writeString записывает строку с 2-байтовой длиной
func (e *Encoder) writeString(s string) error {
	if len(s) > 65535 {
		return fmt.Errorf("string too long: %d bytes", len(s))
	}
	if err := binary.Write(e.buf, binary.LittleEndian, uint16(len(s))); err != nil {
		return err
	}
	_, err := e.buf.WriteString(s)
	return err
}

// writeFlags записывает булевы флаги одним байтом
func (e *Encoder) writeFlags(flags ...bool) error {
	var b byte
	for i, f := range flags {
		if f {
			b |= 1 << i
		}
	}
	return e.buf.WriteByte(b)
}

// encodeUser кодирует UserInfo
func (e *Encoder) encodeUser(u *models.UserInfo) error {
	strings := []string{
		u.Username, u.FullName, u.Domain, u.DomainFull,
		u.Workgroup, u.ProfilePath,
	}
	for _, s := range strings {
		if err := e.writeString(s); err != nil {
			return err
		}
	}

	return e.writeFlags(u.IsAdmin, u.IsDomainUser, u.IsLocalUser)
}

// encodeOS кодирует OSInfo
func (e *Encoder) encodeOS(o *models.OSInfo) error {
	strings := []string{
		o.Name, o.Edition, o.BuildNumber, o.KernelVersion,
		o.Locale, o.InstallationType, o.PowerShellVer,
		o.ProductID, o.RegisteredOwner, o.RegisteredOrg,
	}
	for _, s := range strings {
		if err := e.writeString(s); err != nil {
			return err
		}
	}

	if err := binary.Write(e.buf, binary.LittleEndian, uint8(o.Architecture)); err != nil {
		return err
	}
	if err := binary.Write(e.buf, binary.LittleEndian, o.InstallDate); err != nil {
		return err
	}

	if err := e.writeFlags(o.SecureBootLines, o.IsVirtual, o.IsHypervisor); err != nil {
		return err
	}

	// MachineGUID как 16 байт
	guidBytes := [16]byte(o.MachineGUID)
	if _, err := e.buf.Write(guidBytes[:]); err != nil {
		return err
	}

	return nil
}

// encodeProcessor кодирует ProcessorInfo
func (e *Encoder) encodeProcessor(p *models.ProcessorInfo) error {
	if err := e.writeString(p.Model); err != nil {
		return err
	}
	if err := e.writeString(p.VendorID); err != nil {
		return err
	}
	if err := e.writeString(p.ProcessorID); err != nil {
		return err
	}

	if err := binary.Write(e.buf, binary.LittleEndian, p.PhysicalCores); err != nil {
		return err
	}
	if err := binary.Write(e.buf, binary.LittleEndian, p.LogicalProcessors); err != nil {
		return err
	}
	if err := binary.Write(e.buf, binary.LittleEndian, p.BaseSpeedMHz); err != nil {
		return err
	}

	if err := e.writeFlags(p.HardwareVirtAvail, p.NXBitSupported, p.SMTEnabled, p.NUMAEnabled); err != nil {
		return err
	}

	if err := binary.Write(e.buf, binary.LittleEndian, p.L1CacheBytes); err != nil {
		return err
	}
	if err := binary.Write(e.buf, binary.LittleEndian, p.L2CacheBytes); err != nil {
		return err
	}
	return binary.Write(e.buf, binary.LittleEndian, p.L3CacheBytes)
}

// encodeRAM кодирует RAMInfo
func (e *Encoder) encodeRAM(r *models.RAMInfo) error {
	if err := binary.Write(e.buf, binary.LittleEndian, r.TotalBytes); err != nil {
		return err
	}
	if err := binary.Write(e.buf, binary.LittleEndian, r.AvailableBytes); err != nil {
		return err
	}
	if err := binary.Write(e.buf, binary.LittleEndian, r.TotalPageFile); err != nil {
		return err
	}
	if err := binary.Write(e.buf, binary.LittleEndian, r.AvailablePageFile); err != nil {
		return err
	}

	// Sticks
	if err := binary.Write(e.buf, binary.LittleEndian, uint16(len(r.Sticks))); err != nil {
		return err
	}
	for i := range r.Sticks {
		if err := e.encodeStick(&r.Sticks[i]); err != nil {
			return err
		}
	}
	return nil
}

// encodeStick кодирует RAMStick
func (e *Encoder) encodeStick(s *models.RAMStick) error {
	if err := e.writeString(s.Slot); err != nil {
		return err
	}
	if err := binary.Write(e.buf, binary.LittleEndian, s.Capacity); err != nil {
		return err
	}
	if err := binary.Write(e.buf, binary.LittleEndian, s.SpeedMHz); err != nil {
		return err
	}
	if err := e.writeString(s.Manufacturer); err != nil {
		return err
	}
	if err := e.writeString(s.SerialNumber); err != nil {
		return err
	}
	return e.writeString(s.PartNumber)
}

// encodeDrives кодирует DiskStatuses
func (e *Encoder) encodeDrives(drives models.DiskStatuses) error {
	if err := binary.Write(e.buf, binary.LittleEndian, uint16(len(drives))); err != nil {
		return err
	}
	for i := range drives {
		if err := e.encodeDrive(&drives[i]); err != nil {
			return err
		}
	}
	return nil
}

// encodeDrive кодирует DriveInfo
func (e *Encoder) encodeDrive(d *models.DriveInfo) error {
	if err := e.writeString(d.Letter); err != nil {
		return err
	}
	if err := e.buf.WriteByte(byte(d.Type)); err != nil {
		return err
	}
	if err := e.writeString(d.FSType); err != nil {
		return err
	}
	if err := binary.Write(e.buf, binary.LittleEndian, d.TotalBytes); err != nil {
		return err
	}
	if err := binary.Write(e.buf, binary.LittleEndian, d.FreeBytes); err != nil {
		return err
	}
	if err := binary.Write(e.buf, binary.LittleEndian, d.UsedBytes); err != nil {
		return err
	}
	if err := e.writeString(d.VolumeName); err != nil {
		return err
	}
	if err := binary.Write(e.buf, binary.LittleEndian, d.SerialNumber); err != nil {
		return err
	}
	return e.writeFlags(d.IsReady)
}

// encodeServices кодирует ServicesStatuses
func (e *Encoder) encodeServices(services models.ServicesStatuses) error {
	if err := binary.Write(e.buf, binary.LittleEndian, uint16(len(services))); err != nil {
		return err
	}
	for i := range services {
		if err := e.encodeService(&services[i]); err != nil {
			return err
		}
	}
	return nil
}

// encodeService кодирует ServiceStatus
func (e *Encoder) encodeService(s *models.ServiceStatus) error {
	if err := e.writeString(s.Name); err != nil {
		return err
	}
	if err := e.writeString(s.ServiceName); err != nil {
		return err
	}

	if err := e.writeFlags(s.Installed, s.Running, s.PortOpen); err != nil {
		return err
	}

	return binary.Write(e.buf, binary.LittleEndian, s.Port)
}

// encodeNetwork кодирует NetworkStatuses
func (e *Encoder) encodeNetwork(network models.NetworkStatuses) error {
	if err := binary.Write(e.buf, binary.LittleEndian, uint16(len(network))); err != nil {
		return err
	}
	for i := range network {
		if err := e.encodeInterface(&network[i]); err != nil {
			return err
		}
	}
	return nil
}

// encodeInterface кодирует InterfaceInfo
func (e *Encoder) encodeInterface(i *models.InterfaceInfo) error {
	if err := binary.Write(e.buf, binary.LittleEndian, int32(i.Index)); err != nil {
		return err
	}
	if err := e.writeString(i.Name); err != nil {
		return err
	}
	if err := e.writeString(i.Description); err != nil {
		return err
	}
	if err := e.writeString(i.MAC); err != nil {
		return err
	}
	if err := e.buf.WriteByte(byte(i.Type)); err != nil {
		return err
	}
	if err := e.writeFlags(i.Operational); err != nil {
		return err
	}
	if err := e.buf.WriteByte(byte(i.IPAssignment)); err != nil {
		return err
	}

	// IP Addresses
	if err := binary.Write(e.buf, binary.LittleEndian, uint8(len(i.IPAddresses))); err != nil {
		return err
	}
	for _, ip := range i.IPAddresses {
		ipBytes := ip.To4()
		if ipBytes == nil {
			ipBytes = ip.To16()
		}
		if err := e.buf.WriteByte(byte(len(ipBytes))); err != nil {
			return err
		}
		if _, err := e.buf.Write(ipBytes); err != nil {
			return err
		}
	}
	return nil
}

// encodeHost кодирует HostInfo
func (e *Encoder) encodeHost(h *models.HostInfo) error {
	strings := []string{
		h.Hostname, h.FQDN, h.PhysicalHostname, h.PhysicalFQDN,
		h.Domain, h.Workgroup, h.TimeZone,
		h.Manufacturer, h.Model, h.SKU, h.Family, h.Version,
		h.SerialNumber, h.BIOSVendor, h.BIOSVersion, h.BIOSDate,
		h.BaseBoardManufacturer, h.BaseBoardProduct, h.BaseBoardVersion,
	}
	for _, s := range strings {
		if err := e.writeString(s); err != nil {
			return err
		}
	}

	if err := binary.Write(e.buf, binary.LittleEndian, h.UpTimeSeconds); err != nil {
		return err
	}
	if err := binary.Write(e.buf, binary.LittleEndian, h.BootTime); err != nil {
		return err
	}
	if err := binary.Write(e.buf, binary.LittleEndian, h.TimeZoneOffset); err != nil {
		return err
	}
	if err := binary.Write(e.buf, binary.LittleEndian, h.BIOSMajorRelease); err != nil {
		return err
	}
	return binary.Write(e.buf, binary.LittleEndian, h.BIOSMinorRelease)
}
