package models

// RAMInfo хранит общую информацию о памяти и массив физических планок
type RAMInfo struct {
	TotalBytes        uint64 `json:"total_bytes"`
	AvailableBytes    uint64 `json:"available_bytes"`
	TotalPageFile     uint64 `json:"total_page_file"`
	AvailablePageFile uint64 `json:"available_page_file"`

	// ДОБАВЬТЕ ЭТО ПОЛЕ:
	Sticks []RAMStick `json:"sticks,omitempty"`
}

// ДОБАВЬТЕ ЭТУ СТРУКТУРУ:
type RAMStick struct {
	Slot         string `json:"slot"`
	Capacity     uint64 `json:"capacity_bytes"`
	SpeedMHz     uint32 `json:"speed_mhz"`
	Manufacturer string `json:"manufacturer"`
	SerialNumber string `json:"serial_number"`
	PartNumber   string `json:"part_number"`
}
