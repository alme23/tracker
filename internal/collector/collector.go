//go:build windows

package collector

import (
	"fmt"
	"sync"
	"time"

	"github.com/alme23/tracker/internal/models"
	"golang.org/x/sync/errgroup"
)

// SystemCollector coordinates the execution of all child collectors
type SystemCollector struct {
	serviceColl   *ServiceCollector
	networkColl   *NetworkCollector
	osColl        *OSCollector
	processorColl *ProcessorCollector
	ramColl       *RAMCollector
	diskColl      *DiskCollector
	hostColl      *HostCollector
	userColl      *UserCollector
}

// NewSystemCollector initializes all dependencies in one place
func NewSystemCollector(dialTimeout time.Duration) *SystemCollector {
	return &SystemCollector{
		serviceColl:   NewServiceCollector(dialTimeout),
		networkColl:   NewNetworkCollector(),
		osColl:        NewOSCollector(),
		processorColl: NewProcessorCollector(),
		ramColl:       NewRAMCollector(),
		diskColl:      NewDiskCollector(),
		hostColl:      NewHostCollector(),
		userColl:      NewUserCollector(),
	}
}

// CollectAll runs parallel collection from all collectors and merges results into a single snapshot
func (sc *SystemCollector) CollectAll() (*models.SystemSnapshot, error) {
	snapshot := &models.SystemSnapshot{
		Timestamp: time.Now().Unix(),
	}

	var g errgroup.Group
	var mu sync.Mutex

	// PRIORITY 1: Start network and port collectors first.
	// While they wait for socket responses, the CPU can process the registry.
	g.Go(func() error {
		networkData, err := sc.networkColl.Collect()
		if err != nil {
			return fmt.Errorf("network collector: %w", err)
		}
		mu.Lock()
		snapshot.Network = networkData
		mu.Unlock()
		return nil
	})

	g.Go(func() error {
		servicesData, err := sc.serviceColl.Collect()
		if err != nil {
			return fmt.Errorf("service collector: %w", err)
		}
		mu.Lock()
		snapshot.Services = servicesData
		mu.Unlock()
		return nil
	})

	// PRIORITY 2: Instant collectors (WinAPI/SMBIOS/Registry)
	g.Go(func() error {
		osData, err := sc.osColl.Collect()
		if err != nil {
			return fmt.Errorf("OS collector: %w", err)
		}
		mu.Lock()
		snapshot.OS = osData
		mu.Unlock()
		return nil
	})

	g.Go(func() error {
		procData, err := sc.processorColl.Collect()
		if err != nil {
			return fmt.Errorf("processor collector: %w", err)
		}
		mu.Lock()
		snapshot.Processor = procData
		mu.Unlock()
		return nil
	})

	// RAM collector
	g.Go(func() error {
		ramData, err := sc.ramColl.Collect()
		if err != nil {
			return fmt.Errorf("RAM collector: %w", err)
		}
		mu.Lock()
		snapshot.RAM = ramData
		mu.Unlock()
		return nil
	})

	// Disk collector
	g.Go(func() error {
		diskData, err := sc.diskColl.Collect()
		if err != nil {
			return fmt.Errorf("disk collector: %w", err)
		}
		mu.Lock()
		snapshot.Drives = diskData
		mu.Unlock()
		return nil
	})

	// Host collector
	g.Go(func() error {
		hostData, err := sc.hostColl.Collect()
		if err != nil {
			return fmt.Errorf("host collector: %w", err)
		}
		mu.Lock()
		snapshot.Host = hostData
		mu.Unlock()
		return nil
	})

	// User collector
	g.Go(func() error {
		userData, err := sc.userColl.Collect()
		if err != nil {
			return fmt.Errorf("user collector: %w", err)
		}
		mu.Lock()
		snapshot.User = userData
		mu.Unlock()
		return nil
	})

	// Wait for all goroutines to complete
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return snapshot, nil
}
