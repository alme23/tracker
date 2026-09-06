package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// OSInfo contains detailed information about the operating system
type OSInfo struct {
	Name             string         `json:"name"`                    // OS name (e.g., "Windows 10 Pro")
	Edition          string         `json:"edition"`                 // OS edition (e.g., "Professional")
	BuildNumber      string         `json:"build_number"`            // Build number (e.g., "19045")
	KernelVersion    string         `json:"kernel_version"`          // Kernel version (e.g., "10.0.19045")
	Architecture     ArchFamilyType `json:"architecture"`            // CPU architecture
	Locale           string         `json:"locale"`                  // System locale (e.g., "en-US")
	InstallDate      int64          `json:"install_date"`            // Install date as Unix timestamp
	InstallationType string         `json:"installation_type"`       // "Client" or "Server"
	PowerShellVer    string         `json:"powershell_version"`      // PowerShell version
	SecureBootLines  bool           `json:"secure_boot"`             // Whether Secure Boot is enabled
	MachineGUID      BinaryUUID     `json:"machine_guid"`            // Machine GUID
	ProductID        string         `json:"product_id"`              // Windows product ID
	RegisteredOwner  string         `json:"registered_owner"`        // Registered owner name
	RegisteredOrg    string         `json:"registered_organization"` // Registered organization
	IsVirtual        bool           `json:"is_virtual"`              // Whether running on a VM
	IsHypervisor     bool           `json:"is_hypervisor"`           // Whether running as hypervisor host
}

// GetInstallDateString returns the install date in "YYYY-MM-DD" format
func (o *OSInfo) GetInstallDateString() string {
	if o.InstallDate == 0 {
		return UNKNOWN
	}
	return time.Unix(o.InstallDate, 0).Format("2006-01-02")
}

// GetInstallDateTime returns the install date as time.Time
func (o *OSInfo) GetInstallDateTime() time.Time {
	if o.InstallDate == 0 {
		return time.Time{}
	}
	return time.Unix(o.InstallDate, 0)
}

// BinaryUUID is an ultra-lightweight 16-byte UUID representation
type BinaryUUID uuid.UUID

// String returns the standard string representation of the UUID (with hyphens)
func (b BinaryUUID) String() string {
	return uuid.UUID(b).String()
}

// MarshalJSON converts 16 bytes to a UUID string during JSON generation
func (b BinaryUUID) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.String())
}

// UnmarshalJSON parses a UUID string back to 16 bytes
func (b *BinaryUUID) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	parsed, err := uuid.Parse(s)
	if err != nil {
		return err
	}

	*b = BinaryUUID(parsed)
	return nil
}

// ArchFamilyType represents a processor architecture family
type ArchFamilyType int

// Processor architecture families
const (
	AMD64       ArchFamilyType = iota // AMD64 is the x86-64 architecture
	ARM                               // ARM is the 32-bit ARM architecture
	ARM64                             // ARM64 is the 64-bit ARM architecture
	I386                              // I386 is the 32-bit x86 architecture
	LOONG64                           // LOONG64 is the LoongArch 64-bit architecture
	MIPS                              // MIPS is the MIPS 32-bit architecture
	MIPS64                            // MIPS64 is the MIPS 64-bit architecture
	PPC64                             // PPC64 is the PowerPC 64-bit architecture
	RISCV64                           // RISCV64 is the RISC-V 64-bit architecture
	S390X                             // S390X is the IBM z/Architecture
	WASM                              // WASM is the WebAssembly architecture
	UnknownArch                       // UnknownArch is a fallback for unexpected systems
)

// String returns the string representation of ArchFamilyType
func (a ArchFamilyType) String() string {
	switch a {
	case UnknownArch:
		return UNKNOWN
	case AMD64:
		return "AMD64"
	case ARM:
		return "ARM"
	case ARM64:
		return "ARM64"
	case I386:
		return "I386"
	case LOONG64:
		return "LOONG64"
	case MIPS:
		return "MIPS"
	case MIPS64:
		return "MIPS64"
	case PPC64:
		return "PPC64"
	case RISCV64:
		return "RISCV64"
	case S390X:
		return "S390X"
	case WASM:
		return "WASM"
	default:
		return UNKNOWN
	}
}

// MarshalJSON serializes ArchFamilyType to JSON
func (a ArchFamilyType) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

// UnmarshalJSON deserializes ArchFamilyType from JSON
func (a *ArchFamilyType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	switch s {
	case "AMD64":
		*a = AMD64
	case "ARM":
		*a = ARM
	case "ARM64":
		*a = ARM64
	case "I386":
		*a = I386
	case "LOONG64":
		*a = LOONG64
	case "MIPS":
		*a = MIPS
	case "MIPS64":
		*a = MIPS64
	case "PPC64":
		*a = PPC64
	case "RISCV64":
		*a = RISCV64
	case "S390X":
		*a = S390X
	case "WASM":
		*a = WASM
	default:
		*a = UnknownArch
	}
	return nil
}
