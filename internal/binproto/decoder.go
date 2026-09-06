package binproto

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/alme23/tracker/internal/models"
)

// Static errors
var (
	ErrInvalidMagicHeader = errors.New("invalid magic header")
	ErrReadMagicHeader    = errors.New("failed to read magic header")
	ErrReadTimestamp      = errors.New("failed to read timestamp")
)

// Decoder decodes binary data back into SystemSnapshot
type Decoder struct {
	buf *bytes.Reader
}

// NewDecoder creates a decoder from binary data
func NewDecoder(data []byte) *Decoder {
	return &Decoder{
		buf: bytes.NewReader(data),
	}
}

// Decode deserializes binary data into SystemSnapshot
func (d *Decoder) Decode() (*models.SystemSnapshot, error) {
	// Check magic header
	magic := make([]byte, 5)
	if _, err := io.ReadFull(d.buf, magic); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReadMagicHeader, err)
	}
	if string(magic) != MagicHeader {
		return nil, fmt.Errorf("%w: got %q, want %q", ErrInvalidMagicHeader, magic, MagicHeader)
	}

	snapshot := &models.SystemSnapshot{}

	// Timestamp (int64)
	if err := binary.Read(d.buf, binary.LittleEndian, &snapshot.Timestamp); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrReadTimestamp, err)
	}

	// User
	if err := d.decodeUser(&snapshot.User); err != nil {
		return nil, fmt.Errorf("decode user: %w", err)
	}

	// OS
	if err := d.decodeOS(&snapshot.OS); err != nil {
		return nil, fmt.Errorf("decode os: %w", err)
	}

	// Processor
	if err := d.decodeProcessor(&snapshot.Processor); err != nil {
		return nil, fmt.Errorf("decode processor: %w", err)
	}

	// RAM
	if err := d.decodeRAM(&snapshot.RAM); err != nil {
		return nil, fmt.Errorf("decode ram: %w", err)
	}

	// Drives
	if err := d.decodeDrives(&snapshot.Drives); err != nil {
		return nil, fmt.Errorf("decode drives: %w", err)
	}

	// Services
	if err := d.decodeServices(&snapshot.Services); err != nil {
		return nil, fmt.Errorf("decode services: %w", err)
	}

	// Network
	if err := d.decodeNetwork(&snapshot.Network); err != nil {
		return nil, fmt.Errorf("decode network: %w", err)
	}

	// Host
	if err := d.decodeHost(&snapshot.Host); err != nil {
		return nil, fmt.Errorf("decode host: %w", err)
	}

	return snapshot, nil
}

// readString reads a string with 2-byte length prefix
func (d *Decoder) readString() (string, error) {
	var length uint16
	if err := binary.Read(d.buf, binary.LittleEndian, &length); err != nil {
		return "", err
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(d.buf, buf); err != nil {
		return "", err
	}

	return string(buf), nil
}

// decodeUser decodes UserInfo
func (d *Decoder) decodeUser(u *models.UserInfo) error {
	var err error
	if u.Username, err = d.readString(); err != nil {
		return err
	}
	if u.FullName, err = d.readString(); err != nil {
		return err
	}
	if u.Domain, err = d.readString(); err != nil {
		return err
	}
	if u.DomainFull, err = d.readString(); err != nil {
		return err
	}
	if u.Workgroup, err = d.readString(); err != nil {
		return err
	}
	if u.ProfilePath, err = d.readString(); err != nil {
		return err
	}

	flags, err := d.buf.ReadByte()
	if err != nil {
		return err
	}
	u.IsAdmin = flags&0x01 != 0
	u.IsDomainUser = flags&0x02 != 0
	u.IsLocalUser = flags&0x04 != 0

	return nil
}

// decodeOS decodes OSInfo
func (d *Decoder) decodeOS(o *models.OSInfo) error {
	var err error
	if o.Name, err = d.readString(); err != nil {
		return err
	}
	if o.Edition, err = d.readString(); err != nil {
		return err
	}
	if o.BuildNumber, err = d.readString(); err != nil {
		return err
	}
	if o.KernelVersion, err = d.readString(); err != nil {
		return err
	}
	if o.Locale, err = d.readString(); err != nil {
		return err
	}
	if o.InstallationType, err = d.readString(); err != nil {
		return err
	}
	if o.PowerShellVer, err = d.readString(); err != nil {
		return err
	}
	if o.ProductID, err = d.readString(); err != nil {
		return err
	}
	if o.RegisteredOwner, err = d.readString(); err != nil {
		return err
	}
	if o.RegisteredOrg, err = d.readString(); err != nil {
		return err
	}

	var arch uint8
	if err := binary.Read(d.buf, binary.LittleEndian, &arch); err != nil {
		return err
	}
	o.Architecture = models.ArchFamilyType(arch)

	if err := binary.Read(d.buf, binary.LittleEndian, &o.InstallDate); err != nil {
		return err
	}

	flags, err := d.buf.ReadByte()
	if err != nil {
		return err
	}
	o.SecureBootLines = flags&0x01 != 0
	o.IsVirtual = flags&0x02 != 0
	o.IsHypervisor = flags&0x04 != 0

	// MachineGUID (16 bytes)
	guidBytes := make([]byte, 16)
	if _, err := io.ReadFull(d.buf, guidBytes); err != nil {
		return err
	}
	copy(o.MachineGUID[:], guidBytes)

	return nil
}

// decodeProcessor decodes ProcessorInfo
func (d *Decoder) decodeProcessor(p *models.ProcessorInfo) error {
	var err error
	if p.Model, err = d.readString(); err != nil {
		return err
	}
	if p.VendorID, err = d.readString(); err != nil {
		return err
	}
	if p.ProcessorID, err = d.readString(); err != nil {
		return err
	}

	if err := binary.Read(d.buf, binary.LittleEndian, &p.PhysicalCores); err != nil {
		return err
	}
	if err := binary.Read(d.buf, binary.LittleEndian, &p.LogicalProcessors); err != nil {
		return err
	}
	if err := binary.Read(d.buf, binary.LittleEndian, &p.BaseSpeedMHz); err != nil {
		return err
	}

	flags, err := d.buf.ReadByte()
	if err != nil {
		return err
	}
	p.HardwareVirtAvail = flags&0x01 != 0
	p.NXBitSupported = flags&0x02 != 0
	p.SMTEnabled = flags&0x04 != 0
	p.NUMAEnabled = flags&0x08 != 0

	if err := binary.Read(d.buf, binary.LittleEndian, &p.L1CacheBytes); err != nil {
		return err
	}
	if err := binary.Read(d.buf, binary.LittleEndian, &p.L2CacheBytes); err != nil {
		return err
	}
	return binary.Read(d.buf, binary.LittleEndian, &p.L3CacheBytes)
}

// decodeRAM decodes RAMInfo
func (d *Decoder) decodeRAM(r *models.RAMInfo) error {
	if err := binary.Read(d.buf, binary.LittleEndian, &r.TotalBytes); err != nil {
		return err
	}
	if err := binary.Read(d.buf, binary.LittleEndian, &r.AvailableBytes); err != nil {
		return err
	}
	if err := binary.Read(d.buf, binary.LittleEndian, &r.TotalPageFile); err != nil {
		return err
	}
	if err := binary.Read(d.buf, binary.LittleEndian, &r.AvailablePageFile); err != nil {
		return err
	}

	var count uint16
	if err := binary.Read(d.buf, binary.LittleEndian, &count); err != nil {
		return err
	}

	r.Sticks = make([]models.RAMStick, count)
	for i := range r.Sticks {
		if err := d.decodeStick(&r.Sticks[i]); err != nil {
			return err
		}
	}

	return nil
}

// decodeStick decodes RAMStick
func (d *Decoder) decodeStick(s *models.RAMStick) error {
	var err error
	if s.Slot, err = d.readString(); err != nil {
		return err
	}
	if err := binary.Read(d.buf, binary.LittleEndian, &s.Capacity); err != nil {
		return err
	}
	if err := binary.Read(d.buf, binary.LittleEndian, &s.SpeedMHz); err != nil {
		return err
	}
	if s.Manufacturer, err = d.readString(); err != nil {
		return err
	}
	if s.SerialNumber, err = d.readString(); err != nil {
		return err
	}
	if s.PartNumber, err = d.readString(); err != nil {
		return err
	}
	return nil
}

// decodeDrives decodes DiskStatuses
func (d *Decoder) decodeDrives(drives *models.DiskStatuses) error {
	var count uint16
	if err := binary.Read(d.buf, binary.LittleEndian, &count); err != nil {
		return err
	}

	*drives = make(models.DiskStatuses, count)
	for i := range *drives {
		if err := d.decodeDrive(&(*drives)[i]); err != nil {
			return err
		}
	}
	return nil
}

// decodeDrive decodes DriveInfo
func (d *Decoder) decodeDrive(drive *models.DriveInfo) error {
	var err error
	if drive.Letter, err = d.readString(); err != nil {
		return err
	}

	typeByte, err := d.buf.ReadByte()
	if err != nil {
		return err
	}
	drive.Type = models.DriveType(typeByte)

	if drive.FSType, err = d.readString(); err != nil {
		return err
	}
	if err := binary.Read(d.buf, binary.LittleEndian, &drive.TotalBytes); err != nil {
		return err
	}
	if err := binary.Read(d.buf, binary.LittleEndian, &drive.FreeBytes); err != nil {
		return err
	}
	if err := binary.Read(d.buf, binary.LittleEndian, &drive.UsedBytes); err != nil {
		return err
	}
	if drive.VolumeName, err = d.readString(); err != nil {
		return err
	}
	if err := binary.Read(d.buf, binary.LittleEndian, &drive.SerialNumber); err != nil {
		return err
	}

	readyByte, err := d.buf.ReadByte()
	if err != nil {
		return err
	}
	drive.IsReady = readyByte == 1

	return nil
}

// decodeServices decodes ServicesStatuses
func (d *Decoder) decodeServices(services *models.ServicesStatuses) error {
	var count uint16
	if err := binary.Read(d.buf, binary.LittleEndian, &count); err != nil {
		return err
	}

	*services = make(models.ServicesStatuses, count)
	for i := range *services {
		if err := d.decodeService(&(*services)[i]); err != nil {
			return err
		}
	}
	return nil
}

// decodeService decodes ServiceStatus
func (d *Decoder) decodeService(s *models.ServiceStatus) error {
	var err error
	if s.Name, err = d.readString(); err != nil {
		return err
	}
	if s.ServiceName, err = d.readString(); err != nil {
		return err
	}

	flags, err := d.buf.ReadByte()
	if err != nil {
		return err
	}
	s.Installed = flags&0x01 != 0
	s.Running = flags&0x02 != 0
	s.PortOpen = flags&0x04 != 0

	return binary.Read(d.buf, binary.LittleEndian, &s.Port)
}

// decodeNetwork decodes NetworkStatuses
func (d *Decoder) decodeNetwork(network *models.NetworkStatuses) error {
	var count uint16
	if err := binary.Read(d.buf, binary.LittleEndian, &count); err != nil {
		return err
	}

	*network = make(models.NetworkStatuses, count)
	for i := range *network {
		if err := d.decodeInterface(&(*network)[i]); err != nil {
			return err
		}
	}
	return nil
}

// decodeInterface decodes InterfaceInfo
func (d *Decoder) decodeInterface(i *models.InterfaceInfo) error {
	var index int32
	if err := binary.Read(d.buf, binary.LittleEndian, &index); err != nil {
		return fmt.Errorf("read index: %w", err)
	}
	i.Index = int(index)

	var err error
	if i.Name, err = d.readString(); err != nil {
		return err
	}
	if i.Description, err = d.readString(); err != nil {
		return err
	}
	if i.MAC, err = d.readString(); err != nil {
		return err
	}

	typeByte, err := d.buf.ReadByte()
	if err != nil {
		return err
	}
	i.Type = models.InterfaceType(typeByte)

	opByte, err := d.buf.ReadByte()
	if err != nil {
		return err
	}
	i.Operational = opByte == 1

	assignByte, err := d.buf.ReadByte()
	if err != nil {
		return err
	}
	i.IPAssignment = models.IPAssignment(assignByte)

	var ipCount uint8
	if err := binary.Read(d.buf, binary.LittleEndian, &ipCount); err != nil {
		return err
	}

	i.IPAddresses = make([]net.IP, ipCount)
	for j := range i.IPAddresses {
		ipLen, err := d.buf.ReadByte()
		if err != nil {
			return err
		}
		ipBytes := make([]byte, ipLen)
		if _, err := io.ReadFull(d.buf, ipBytes); err != nil {
			return err
		}
		i.IPAddresses[j] = net.IP(ipBytes)
	}

	return nil
}

// decodeHost decodes HostInfo
func (d *Decoder) decodeHost(h *models.HostInfo) error {
	strings := []*string{
		&h.Hostname, &h.FQDN, &h.PhysicalHostname, &h.PhysicalFQDN,
		&h.Domain, &h.Workgroup, &h.TimeZone,
		&h.Manufacturer, &h.Model, &h.SKU, &h.Family, &h.Version,
		&h.SerialNumber, &h.BIOSVendor, &h.BIOSVersion, &h.BIOSDate,
		&h.BaseBoardManufacturer, &h.BaseBoardProduct, &h.BaseBoardVersion,
	}

	for _, s := range strings {
		val, err := d.readString()
		if err != nil {
			return err
		}
		*s = val
	}

	if err := binary.Read(d.buf, binary.LittleEndian, &h.UpTimeSeconds); err != nil {
		return err
	}
	if err := binary.Read(d.buf, binary.LittleEndian, &h.BootTime); err != nil {
		return err
	}
	if err := binary.Read(d.buf, binary.LittleEndian, &h.TimeZoneOffset); err != nil {
		return err
	}
	if err := binary.Read(d.buf, binary.LittleEndian, &h.BIOSMajorRelease); err != nil {
		return err
	}
	return binary.Read(d.buf, binary.LittleEndian, &h.BIOSMinorRelease)
}
