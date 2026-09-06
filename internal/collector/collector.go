// tracker/internal/collector/collector.go
//
//go:build windows

package collector

import (
	"fmt"
	"sync"
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
		Timestamp: time.Now().Unix(), // Теперь int64
	}

	var g errgroup.Group
	var mu sync.Mutex

	// ПРИОРИТЕТ 1: Запускаем сетевые и портовые коллекторы первыми.
	// Пока они ждут ответа от сокетов и сетевого стека, процессор успеет разобрать весь реестр.
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

	// ПРИОРИТЕТ 2: Мгновенные коллекторы (WinAPI/SMBIOS/Реестр)
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

	// 5. Коллектор оперативной памяти
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

	// 6. Коллектор дисков
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

	// 7. Коллектор хоста
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

	// 8. Коллектор пользователя
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

	// Ждем завершения всех горутин
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return snapshot, nil
}
