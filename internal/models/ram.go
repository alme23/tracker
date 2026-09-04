package models

// RAMInfo содержит динамические данные из ОС и физическую топологию железа
type RAMInfo struct {
	TotalBytes        uint64 `json:"total_bytes"`
	AvailableBytes    uint64 `json:"available_bytes"`
	TotalPageFile     uint64 `json:"total_pagefile"`
	AvailablePageFile uint64 `json:"available_pagefile"`
}
