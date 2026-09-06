package models

// RAMInfo contains information about system memory.
//
// It includes overall memory usage metrics (in bytes)
// and a list of physical memory modules (sticks).
type RAMInfo struct {
	TotalBytes        uint64     `json:"total_bytes"`         // TotalBytes is the total amount of physical memory in bytes
	AvailableBytes    uint64     `json:"available_bytes"`     // AvailableBytes is the available (free) memory in bytes
	TotalPageFile     uint64     `json:"total_page_file"`     // TotalPageFile is the total page file size in bytes
	AvailablePageFile uint64     `json:"available_page_file"` // AvailablePageFile is the available page file size in bytes
	Sticks            []RAMStick `json:"sticks"`              // Sticks is the list of physical memory modules
}

// RAMStick describes a single physical memory module (stick).
//
// Information is collected via SMBIOS (System Management BIOS)
// and contains data about the slot, capacity, speed, and manufacturer.
type RAMStick struct {
	Slot         string `json:"slot"`           // Slot is the motherboard slot name (e.g., "DIMM1", "BANK 0")
	Capacity     uint64 `json:"capacity_bytes"` // Capacity is the module size in bytes (e.g., 8589934592 = 8 GB)
	SpeedMHz     uint32 `json:"speed_mhz"`      // SpeedMHz is the module operating frequency in megahertz (e.g., 1600, 2400, 3200)
	Manufacturer string `json:"manufacturer"`   // Manufacturer is the module manufacturer (Kingston, Samsung, Hynix, etc.)
	SerialNumber string `json:"serial_number"`  // SerialNumber is the module serial number (if available)
	PartNumber   string `json:"part_number"`    // PartNumber is the manufacturer's part number (e.g., "KVR16E11/8")
}
