package binproto

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/alme23/tracker/internal/models"
)

// MagicHeader identifies the data format during transmission
const MagicHeader = "TRCK1"

// Static errors
var (
	ErrStringTooLong          = errors.New("string too long")
	ErrWriteMagic             = errors.New("failed to write magic header")
	ErrWriteTimestamp         = errors.New("failed to write timestamp")
	ErrArchitectureOutOfRange = errors.New("architecture value out of range")
	ErrInvalidDataLength      = errors.New("invalid data length")
	ErrBufferTooSmall         = errors.New("buffer too small")
)

// Encoder encodes SystemSnapshot into a compact binary format
type Encoder struct {
	buf *bytes.Buffer
}

// NewEncoder creates a new binary encoder
func NewEncoder() *Encoder {
	return &Encoder{
		buf: &bytes.Buffer{},
	}
}

// Encode serializes SystemSnapshot into binary format
func (e *Encoder) Encode(s *models.SystemSnapshot) ([]byte, error) {
	e.buf.Reset()

	// Magic header for versioning
	if _, err := e.buf.WriteString(MagicHeader); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWriteMagic, err)
	}

	// Timestamp (int64 Unix timestamp)
	if err := binary.Write(e.buf, binary.LittleEndian, s.Timestamp); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWriteTimestamp, err)
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

// writeString writes a string with 2-byte length prefix
func (e *Encoder) writeString(s string) error {
	if len(s) > math.MaxUint16 {
		return fmt.Errorf("%w: %d bytes (max %d)", ErrStringTooLong, len(s), math.MaxUint16)
	}

	// #nosec G115 -- len(s) is already checked to be < math.MaxUint16
	length := uint16(len(s))
	if err := binary.Write(e.buf, binary.LittleEndian, length); err != nil {
		return err
	}
	_, err := e.buf.WriteString(s)
	return err
}

// writeFlags writes boolean flags as a single byte
func (e *Encoder) writeFlags(flags ...bool) error {
	var b byte
	for i, f := range flags {
		if f {
			b |= 1 << i
		}
	}
	return e.buf.WriteByte(b)
}

// encodeUser encodes UserInfo
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

// encodeOS encodes OSInfo
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

	// Safe conversion with bounds check
	arch := o.Architecture
	if arch < 0 || arch > math.MaxUint8 {
		return fmt.Errorf("%w: %d", ErrArchitectureOutOfRange, arch)
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

	// MachineGUID as 16 bytes
	guidBytes := [16]byte(o.MachineGUID)
	if _, err := e.buf.Write(guidBytes[:]); err != nil {
		return err
	}

	return nil
}

// encodeProcessor encodes ProcessorInfo
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

// encodeRAM encodes RAMInfo
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
	// #nosec G115 -- len(r.Sticks) is always < math.MaxUint16
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

// encodeStick encodes RAMStick
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

// encodeDrives encodes DiskStatuses
func (e *Encoder) encodeDrives(drives models.DiskStatuses) error {
	// #nosec G115 -- len(drives) is always < math.MaxUint16
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

// encodeDrive encodes DriveInfo
func (e *Encoder) encodeDrive(d *models.DriveInfo) error {
	if err := e.writeString(d.Letter); err != nil {
		return err
	}
	// #nosec G115 -- DriveType values are always within uint8 range (0-6)
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

// encodeServices encodes ServicesStatuses
func (e *Encoder) encodeServices(services models.ServicesStatuses) error {
	// #nosec G115 -- len(services) is always < math.MaxUint16
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

// encodeService encodes ServiceStatus
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

// encodeNetwork encodes NetworkStatuses
func (e *Encoder) encodeNetwork(network models.NetworkStatuses) error {
	// #nosec G115 -- len(network) is always < math.MaxUint16
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

// encodeInterface encodes InterfaceInfo
func (e *Encoder) encodeInterface(i *models.InterfaceInfo) error {
	// #nosec G115 -- i.Index is always < math.MaxInt32
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
	// #nosec G115 -- InterfaceType values are always within uint8 range (0-6)
	if err := e.buf.WriteByte(byte(i.Type)); err != nil {
		return err
	}
	if err := e.writeFlags(i.Operational); err != nil {
		return err
	}
	// #nosec G115 -- IPAssignment values are always within uint8 range (0-3)
	if err := e.buf.WriteByte(byte(i.IPAssignment)); err != nil {
		return err
	}

	// IP Addresses
	// #nosec G115 -- IP address count is always small
	if err := binary.Write(e.buf, binary.LittleEndian, uint8(len(i.IPAddresses))); err != nil {
		return err
	}
	for _, ip := range i.IPAddresses {
		ipBytes := ip.To4()
		if ipBytes == nil {
			ipBytes = ip.To16()
		}
		// #nosec G115 -- IP address length is 4 or 16 bytes
		if err := e.buf.WriteByte(byte(len(ipBytes))); err != nil {
			return err
		}
		if _, err := e.buf.Write(ipBytes); err != nil {
			return err
		}
	}
	return nil
}

// encodeHost encodes HostInfo
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
