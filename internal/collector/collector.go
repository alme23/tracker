//go:build windows

package collector

import (
	"time"

	"github.com/alme23/tracker/internal/models"
	"golang.org/x/sync/errgroup"
)

// SystemCollector координирует запуск всех дочерних сборщиков
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

// NewSystemCollector инициализирует все зависимости в одном месте
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

// CollectAll запускает параллельный сбор со всех коллекторов и склеивает результаты в единый snapshot
func (sc *SystemCollector) CollectAll() (*models.SystemSnapshot, error) {
	snapshot := &models.SystemSnapshot{
		Timestamp: time.Now(),
	}

	var g errgroup.Group

	// 1. Запускаем коллектор операционной системы
	g.Go(func() error {
		osData, err := sc.osColl.Collect()
		if err != nil {
			return err
		}
		snapshot.OS = osData
		return nil
	})

	// 2. Запускаем коллектор сетевых интерфейсов
	g.Go(func() error {
		networkData, err := sc.networkColl.Collect()
		if err != nil {
			return err
		}
		snapshot.Network = networkData
		return nil
	})

	// 3. Запускаем коллектор Windows-служб (RDP/VNC)
	g.Go(func() error {
		servicesData, err := sc.serviceColl.Collect()
		if err != nil {
			return err
		}
		snapshot.Services = servicesData
		return nil
	})

	// 4. Запускаем коллектор процессора параллельно с остальными
	g.Go(func() error {
		procData, err := sc.processorColl.Collect()
		if err != nil {
			return err
		}
		snapshot.Processor = procData
		return nil
	})

	// 5. Запускаем коллектор оперативной памяти параллельно с остальными
	g.Go(func() error {
		ramData, err := sc.ramColl.Collect()
		if err != nil {
			return err
		}
		snapshot.RAM = ramData
		return nil
	})

	// 6. Запускаем коллектор дисков параллельно с остальными
	g.Go(func() error {
		diskData, err := sc.diskColl.Collect()
		if err != nil {
			return err
		}
		snapshot.Drives = diskData
		return nil
	})

	// 7. Запускаем коллектор хоста параллельно с остальными
	g.Go(func() error {
		hostData, err := sc.hostColl.Collect()
		if err != nil {
			return err
		}
		snapshot.Host = hostData
		return nil
	})

	// 7. Запускаем коллектор хоста параллельно с остальными
	g.Go(func() error {
		userData, err := sc.userColl.Collect()
		if err != nil {
			return err
		}
		snapshot.User = userData
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	return snapshot, nil
}
