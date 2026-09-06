package models

// ProcessorInfo contains detailed characteristics of the central processor
type ProcessorInfo struct {
	Model             string `json:"model"`                   // CPU model name (e.g., "Intel Core i7-10700K")
	VendorID          string `json:"vendor_id"`               // CPU vendor (e.g., "GenuineIntel")
	ProcessorID       string `json:"processor_id"`            // Processor identifier string
	PhysicalCores     uint32 `json:"physical_cores"`          // Number of physical cores
	LogicalProcessors uint32 `json:"logical_processors"`      // Number of logical processors (threads)
	BaseSpeedMHz      uint16 `json:"base_speed_mhz"`          // Base clock speed in MHz
	HardwareVirtAvail bool   `json:"hardware_virt_available"` // Hardware virtualization support (VT-x/AMD-V)
	NXBitSupported    bool   `json:"nx_bit_supported"`        // NX/XD bit support (DEP)
	SMTEnabled        bool   `json:"smt_enabled"`             // Simultaneous Multithreading (Hyper-Threading)
	NUMAEnabled       bool   `json:"numa_enabled"`            // NUMA architecture support
	L1CacheBytes      uint64 `json:"l1_cache_bytes"`          // L1 cache size in bytes (typically 32-64 KB per core)
	L2CacheBytes      uint64 `json:"l2_cache_bytes"`          // L2 cache size in bytes (typically 256 KB - 1 MB per core)
	L3CacheBytes      uint64 `json:"l3_cache_bytes"`          // L3 cache size in bytes (typically 8-32 MB shared)
}
