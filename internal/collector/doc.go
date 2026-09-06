// Package collector contains system information collectors for Windows.
//
// The collectors gather data about:
//   - Operating system (version, edition, locale)
//   - Processor (model, cores, cache)
//   - Memory (total, available, physical modules)
//   - Drives (type, file system, capacity)
//   - Services (RDP, VNC status and ports)
//   - Network adapters (type, MAC, IP addresses)
//   - Host (hostname, domain, BIOS, motherboard)
//   - User (username, domain, admin rights)
//
// The SystemCollector coordinates parallel collection from all child
// collectors for optimal performance.
//
// Example:
//
//	sc := collector.NewSystemCollector(2 * time.Second)
//	snapshot, err := sc.CollectAll()
//	if err != nil {
//		log.Fatal(err)
//	}
package collector
