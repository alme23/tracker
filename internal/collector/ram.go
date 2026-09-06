// tracker/internal/collector/ram.go

//go:build windows

package collector

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unsafe"

	"github.com/alme23/tracker/internal/models"
)

// Системные константы для работы с таблицами прошивки
const (
	providerRSMB = 0x52534D42 // "RSMB" в формате BigEndian/DWORD для SMBIOS
)

// memoryStatusEx — точная копия структуры MEMORYSTATUSEX из Win32 API
type memoryStatusEx struct {
	dwLength                uint32
	dwMemoryLoad            uint32
	ullTotalPhys            uint64
	ullAvailPhys            uint64
	ullTotalPageFile        uint64
	ullAvailPageFile        uint64
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

// Заголовок таблицы SMBIOS
type smbiosTableStructure struct {
	Type   uint8
	Length uint8
	Handle uint16
}

type RAMCollector struct{}

func NewRAMCollector() *RAMCollector {
	return &RAMCollector{}
}

// Collect собирает информацию о памяти и планках нативно без использования WMI
func (c *RAMCollector) Collect() (models.RAMInfo, error) {
	var info models.RAMInfo

	// 1. Статистика использования памяти (WinAPI)
	var memStatus memoryStatusEx
	memStatus.dwLength = uint32(unsafe.Sizeof(memStatus))

	ret, _, err := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memStatus)))
	if ret == 0 {
		return info, fmt.Errorf("ошибка вызова WinAPI GlobalMemoryStatusEx: %w", err)
	}

	info.TotalBytes = memStatus.ullTotalPhys
	info.AvailableBytes = memStatus.ullAvailPhys
	info.TotalPageFile = memStatus.ullTotalPageFile
	info.AvailablePageFile = memStatus.ullAvailPageFile

	// 2. Сбор физических планок напрямую из SMBIOS таблицы Type 17
	sticks, err := c.getPhysicalSticksFromSMBIOS()
	if err == nil {
		info.Sticks = sticks
	}

	return info, nil
}

// getPhysicalSticksFromSMBIOS считывает прошивку и вытаскивает информацию о слотах памяти
func (c *RAMCollector) getPhysicalSticksFromSMBIOS() ([]models.RAMStick, error) {
	// Делаем первый вызов, чтобы узнать точный размер таблицы SMBIOS в байтах
	ret, _, _ := procGetSystemFirmwareTable.Call(
		uintptr(providerRSMB),
		0,
		0,
		0,
	)
	if ret == 0 {
		return nil, fmt.Errorf("SMBIOS таблицы недоступны")
	}

	buffer := make([]byte, ret)
	ret, _, err := procGetSystemFirmwareTable.Call(
		uintptr(providerRSMB),
		0,
		uintptr(unsafe.Pointer(unsafe.SliceData(buffer))),
		uintptr(ret),
	)
	if ret == 0 {
		return nil, fmt.Errorf("ошибка чтения таблицы GetSystemFirmwareTable: %w", err)
	}

	// Пропускаем 8 байт заголовка вывода RSMB Windows
	if len(buffer) < 8 {
		return nil, fmt.Errorf("невалидный формат буфера SMBIOS")
	}
	smbiosData := buffer[8:]

	var sticks []models.RAMStick
	offset := 0

	for offset+4 <= len(smbiosData) {
		// Читаем заголовок текущей структуры SMBIOS
		header := smbiosTableStructure{
			Type:   smbiosData[offset],
			Length: smbiosData[offset+1],
			Handle: binary.LittleEndian.Uint16(smbiosData[offset+2 : offset+4]),
		}

		if int(header.Length) < 4 || offset+int(header.Length) > len(smbiosData) {
			break
		}

		// Выделяем данные структуры и блок текстовых строк, идущих сразу за ней
		structBytes := smbiosData[offset : offset+int(header.Length)]

		// Каждая структура завершается двойным нулем (0x00 0x00), ищем конец блока строк
		stringOffset := offset + int(header.Length)
		endStrings := stringOffset
		for endStrings+1 < len(smbiosData) {
			if smbiosData[endStrings] == 0 && smbiosData[endStrings+1] == 0 {
				endStrings += 2
				break
			}
			endStrings++
		}

		stringBytes := smbiosData[stringOffset:endStrings]
		stringsList := c.parseSMBIOSStrings(stringBytes)

		// Type 17 — это Memory Device (структура планки памяти)
		if header.Type == 17 && len(structBytes) >= 28 {
			stick := c.parseType17Structure(structBytes, stringsList)
			// Добавляем только реально установленные планки (размер > 0)
			if stick.Capacity > 0 {
				sticks = append(sticks, stick)
			}
		}

		// Смещаемся к следующей структуре SMBIOS
		offset = endStrings
	}

	return sticks, nil
}

// parseType17Structure парсит сырые байты структуры Memory Device
func (c *RAMCollector) parseType17Structure(data []byte, textStrings []string) models.RAMStick {
	var stick models.RAMStick

	// Смещение 0x0C: Размер планки (2 байта)
	rawSize := binary.LittleEndian.Uint16(data[12:14])
	if rawSize == 0 || rawSize == 0xFFFF {
		return stick // Слот пустой
	}

	// Если старший бит равен 1, размер указан в мегабайтах, иначе в килобайтах (зависит от ревизии SMBIOS)
	// Но обычно значение меньше 0x7FFF — это чистые Мегабайты.
	if (rawSize & 0x8000) == 0 {
		stick.Capacity = uint64(rawSize) * 1024 * 1024
	} else {
		stick.Capacity = uint64(rawSize&0x7FFF) * 1024 // в КБ
	}

	// Смещение 0x15: Скорость памяти в МГц (2 байта)
	if len(data) >= 23 {
		stick.SpeedMHz = uint32(binary.LittleEndian.Uint16(data[21:23]))
	}

	// Извлекаем текстовые индексы строк (индексация в SMBIOS начинается с 1)
	getString := func(indexByte byte) string {
		idx := int(indexByte)
		if idx > 0 && idx <= len(textStrings) {
			return textStrings[idx-1]
		}
		return "Unknown"
	}

	// Индексы строк внутри структуры Type 17
	if len(data) >= 8 {
		stick.Slot = getString(data[8]) // Device Locator (например, DIMM 0)
	}
	if len(data) >= 24 {
		stick.Manufacturer = getString(data[23]) // Производитель
	}
	if len(data) >= 25 {
		stick.SerialNumber = getString(data[24]) // Серийный номер
	}
	if len(data) >= 27 {
		stick.PartNumber = getString(data[26]) // Номер партии (Part Number)
	}

	return stick
}

// parseSMBIOSStrings разбивает блок строк, разделенных байтом 0x00, в срез строк Go
func (c *RAMCollector) parseSMBIOSStrings(data []byte) []string {
	var res []string
	parts := bytes.Split(data, []byte{0})
	for _, p := range parts {
		s := string(bytes.TrimSpace(p))
		if s != "" {
			res = append(res, s)
		}
	}
	return res
}
