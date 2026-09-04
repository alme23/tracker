package models

// ProcessorInfo содержит детальные характеристики центрального процессора
type ProcessorInfo struct {
	Model             string `json:"model"`
	VendorID          string `json:"vendor_id"`
	ProcessorID       string `json:"processor_id"`
	PhysicalCores     uint32 `json:"physical_cores"`
	LogicalProcessors uint32 `json:"logical_processors"`
	BaseSpeedMHz      uint16 `json:"base_speed_mhz"`
	HardwareVirtAvail bool   `json:"hardware_virt_available"`
	NXBitSupported    bool   `json:"nx_bit_supported"`
	SMTEnabled        bool   `json:"smt_enabled"`
	NUMAEnabled       bool   `json:"numa_enabled"`
	L1CacheBytes      uint64 `json:"l1_cache_bytes"` // L1 кэш (обычно 32-64 KB на ядро)
	L2CacheBytes      uint64 `json:"l2_cache_bytes"` // L2 кэш (обычно 256 KB - 1 MB на ядро)
	L3CacheBytes      uint64 `json:"l3_cache_bytes"` // L3 кэш (обычно 8-32 MB общий)
}
