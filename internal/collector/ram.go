// tracker/internal/collector/ram.go

//go:build windows

package collector

import (
	"fmt"
	"unsafe"

	"github.com/alme23/tracker/internal/models"
)

// memoryStatusEx — точная копия структуры MEMORYSTATUSEX из Win32 API.
type memoryStatusEx struct {
	dwLength                uint32 // Размер структуры в байтах
	dwMemoryLoad            uint32 // Процент использования памяти
	ullTotalPhys            uint64 // Общий объем физической памяти в байтах
	ullAvailPhys            uint64 // Свободная физическая память в байтах
	ullTotalPageFile        uint64 // Максимальный лимит коммита памяти в байтах
	ullAvailPageFile        uint64 // Свободный лимит коммита памяти в байтах
	ullTotalVirtual         uint64
	ullAvailVirtual         uint64
	ullAvailExtendedVirtual uint64
}

type RAMCollector struct{}

func NewRAMCollector() *RAMCollector {
	return &RAMCollector{}
}

// Collect собирает информацию о состоянии оперативной памяти в сырых байтах напрямую из ядра ОС
func (c *RAMCollector) Collect() (models.RAMInfo, error) {
	var info models.RAMInfo

	// 1. Выделяем структуру в памяти
	var memStatus memoryStatusEx

	// ВАЖНО: Windows требует, чтобы размер структуры (64 байта) был записан в её первое поле
	memStatus.dwLength = uint32(unsafe.Sizeof(memStatus))

	// 2. Делаем прямой низкоуровневый вызов к ядру ОС (работает мгновенно)
	ret, _, err := procGlobalMemoryStatusEx.Call(uintptr(unsafe.Pointer(&memStatus)))
	if ret == 0 {
		return info, fmt.Errorf("ошибка вызова WinAPI GlobalMemoryStatusEx: %w", err)
	}

	// 3. Записываем чистые сырые данные в байтах напрямую в модель без деления и округлений
	info.TotalBytes = memStatus.ullTotalPhys
	info.AvailableBytes = memStatus.ullAvailPhys
	info.TotalPageFile = memStatus.ullTotalPageFile
	info.AvailablePageFile = memStatus.ullAvailPageFile

	return info, nil
}
