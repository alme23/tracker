package protocol

// DataPacket описывает метрики, отправляемые на сервер
type DataPacket struct {
	Hostname string  `gob:"hostname"`
	OS       string  `gob:"os"`
	CPUUsage float64 `gob:"cpu_usage"`
	RAMTotal uint64  `gob:"ram_total"`
	RAMFree  uint64  `gob:"ram_free"`
}
