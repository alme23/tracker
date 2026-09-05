package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// OSInfo содержит детальную информацию об операционной системе
type OSInfo struct {
	Name             string         `json:"name"`
	Edition          string         `json:"edition"` // НОВОЕ: Редакция ОС (например, "ENTERPRISE", "PRO")
	BuildNumber      string         `json:"build_number"`
	KernelVersion    string         `json:"kernel_version"`
	Architecture     ArchFamilyType `json:"architecture"`
	Locale           string         `json:"locale"`
	InstallDate      int64          `json:"install_date"`
	InstallationType string         `json:"installation_type"`
	PowerShellVer    string         `json:"powershell_version"`
	SecureBootLines  bool           `json:"secure_boot"`
	MachineGUID      BinaryUUID     `json:"machine_guid"`
	ProductID        string         `json:"product_id"`
	RegisteredOwner  string         `json:"registered_owner"`
	RegisteredOrg    string         `json:"registered_organization"`
	IsVirtual        bool           `json:"is_virtual"`
	IsHypervisor     bool           `json:"is_hypervisor"`
}

// GetInstallDateString возвращает дату установки в формате "YYYY-MM-DD"
func (o *OSInfo) GetInstallDateString() string {
	if o.InstallDate == 0 {
		return "UNKNOWN"
	}
	return time.Unix(o.InstallDate, 0).Format("2006-01-02")
}

// GetInstallDateTime возвращает дату установки как time.Time
func (o *OSInfo) GetInstallDateTime() time.Time {
	if o.InstallDate == 0 {
		return time.Time{}
	}
	return time.Unix(o.InstallDate, 0)
}

// BinaryUUID представляет собой ультра-легкие 16 байт памяти для хранения UUID
type BinaryUUID uuid.UUID

// String возвращает стандартное строковое представление UUID (с дефисами)
func (b BinaryUUID) String() string {
	return uuid.UUID(b).String()
}

// MarshalJSON превращает 16 байт в красивую строку UUID при генерации JSON
func (b BinaryUUID) MarshalJSON() ([]byte, error) {
	return json.Marshal(b.String())
}

// UnmarshalJSON позволяет парсить текстовую строку UUID обратно в 16 байт
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

// ArchFamilyType представляет собой семейство архитектур процессоров.
// Полностью дублирует внутренний системный тип рантайма Go (internal/goarch).
type ArchFamilyType int

const (
	AMD64       ArchFamilyType = iota // 0
	ARM                               // 1
	ARM64                             // 2
	I386                              // 3
	LOONG64                           // 4
	MIPS                              // 5
	MIPS64                            // 6
	PPC64                             // 7
	RISCV64                           // 8
	S390X                             // 9
	WASM                              // 10
	UnknownArch                       // Наш фолбек на случай непредвиденных систем
)

func (a ArchFamilyType) String() string {
	switch a {
	case UnknownArch:
		return "UNKNOWN"
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
		return "UNKNOWN"
	}
}

func (a ArchFamilyType) MarshalJSON() ([]byte, error) { return json.Marshal(a.String()) }

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
