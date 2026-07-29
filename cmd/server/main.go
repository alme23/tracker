package main

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	// Обновленные пути импорта
	"github.com/alme23/tracker/internal/crypto"
	"github.com/alme23/tracker/pkg/collector"
)

func main() {
	// 1. Запуск слушателя на сыром TCP-порту
	listener, err := net.Listen("tcp", ":9000")
	if err != nil {
		log.Fatalf("[Сервер] Ошибка запуска: %v", err)
	}
	defer listener.Close()
	fmt.Println("====================================================")
	fmt.Println("🚀 Tracker-Сервер успешно запущен на порту :9000")
	fmt.Println("🔒 Режим: AES-256-GCM + Gzip + GOB (Без сертификатов)")
	fmt.Println("====================================================")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("[Сервер] Ошибка подключения клиента: %v", err)
			continue
		}
		// Обрабатываем каждую одноразовую сессию агента асинхронно
		go handleAgentConnection(conn)
	}
}

func handleAgentConnection(conn net.Conn) {
	defer conn.Close()

	// 1. Считываем все зашифрованные байты из сокета
	encryptedData, err := io.ReadAll(conn)
	if err != nil {
		log.Printf("[Сервер] Ошибка чтения данных из сокета: %v", err)
		return
	}

	if len(encryptedData) == 0 {
		return
	}

	// 2. Симметричное дешифрование AES-GCM
	decryptedGzip, err := crypto.Decrypt(encryptedData)
	if err != nil {
		log.Printf("[Сервер] Критическая ошибка дешифрования: %v (Данные подменены или неверный ключ)", err)
		return
	}

	// 3. Распаковка потока Gzip
	gzipReader, err := gzip.NewReader(bytes.NewReader(decryptedGzip))
	if err != nil {
		log.Printf("[Сервер] Ошибка декомпрессии Gzip: %v", err)
		return
	}
	defer gzipReader.Close()

	// 4. Декодирование GOB напрямую в структуру из вашего пакета collector
	var metrics collector.Metrics
	decoder := gob.NewDecoder(gzipReader)
	if err := decoder.Decode(&metrics); err != nil {
		log.Printf("[Сервер] Ошибка бинарной десериализации GOB: %v", err)
		return
	}

	// 5. Красивый вывод полученных системных метрик в консоль
	printMetrics(metrics)
}

func printMetrics(m collector.Metrics) {
	fmt.Printf("\n📥 [ПАКЕТ ПРИНЯТ] Время: %s\n", time.Unix(m.Timestamp, 0).Format("2006-01-02 15:04:05"))
	fmt.Println("----------------------------------------------------")
	fmt.Printf("💻 Имя хоста (NetBIOS): %s\n", m.Host.ComputerName)
	fmt.Printf("🌐 Полное имя (FQDN):   %s\n", m.Host.FQDN)       // Выведет: pc01.corp.local
	fmt.Printf("🏢 Домен системы:       %s\n", m.Host.DomainName) // Выведет: corp.local

	fmt.Printf("🪟 Система:      %s (%s, %s)\n", m.Host.OSName, m.Host.OSEdition, m.Host.OSVersion)
	fmt.Printf("🔧 Сборка ОС:    %s\n", m.Host.OSBuild)
	fmt.Printf("📅 Установка ОС: %s\n", m.Host.InstallDateStr)
	fmt.Printf("🆔 Machine GUID: %s\n", m.Host.DeviceID)
	fmt.Printf("👤 Логин:        %s\n", m.User.UserName)
	fmt.Printf("📛 Полное имя:   %s\n", m.User.DisplayName) // Выведет: "Иван Иванов"
	fmt.Printf("🌐 Сеть хоста:  %s (%s)\n", m.User.Domain, m.User.JoinType)
	fmt.Printf("🧱 Плата:        %s (Модель: %s, С/Н: %s)\n", m.BaseBoard.Vendor, m.BaseBoard.Product, m.BaseBoard.SerialNumber)
	fmt.Printf("📟 BIOS:         %s (Версия: %s, Дата: %s)\n", m.BIOS.Vendor, m.BIOS.Version, m.BIOS.ReleaseDate)
	fmt.Printf("🧠 Процессор:    %s\n", m.CPU.Name)
	fmt.Printf("   ├─ Ядра/Потоки: %d физ. ядер / %d лог. процессоров\n", m.CPU.PhysicalCores, m.CPU.LogicalProcessors)
	fmt.Printf("   ├─ Частота:     %d MHz\n", m.CPU.MaxSpeedMHz)
	fmt.Printf("   └─ Кэш память:  L1: %d KB | L2: %d KB | L3: %d MB\n", m.CPU.CacheL1Bytes/1024, m.CPU.CacheL2Bytes/1024, m.CPU.CacheL3Bytes/1024/1024)

	// Вывод оперативной памяти
	totalGB := float64(m.Memory.TotalRAMBytes) / 1024 / 1024 / 1024
	freeGB := float64(m.Memory.FreeRAMBytes) / 1024 / 1024 / 1024
	fmt.Printf("📊 Память RAM:   Total: %.2f GB | Free: %.2f GB\n", totalGB, freeGB)

	// Вывод видеокарт
	if len(m.Video) > 0 {
		fmt.Println("📺 Видеокарты:")
		for _, v := range m.Video {
			var memInfo string
			if v.DedicatedMemory > 0 {
				memInfo = fmt.Sprintf("Dedicated: %.2f GB", float64(v.DedicatedMemory)/1024/1024/1024)
			} else {
				memInfo = fmt.Sprintf("Shared: %.2f GB", float64(v.SharedMemory)/1024/1024/1024)
			}
			fmt.Printf("   └─ %s [VEN_0x%X & DEV_0x%X] (%s)\n", v.Name, v.VendorID, v.DeviceID, memInfo)
		}
	}

	// Вывод дисков
	if len(m.Disks) > 0 {
		fmt.Println("💾 Накопители:")
		for _, d := range m.Disks {
			fmt.Printf("   └─ %s [%s] %s (С/Н: %s) | Объем: %d GB | Свободно: %d GB\n",
				d.Drive, d.Type, d.Vendor, d.SerialNumber, d.TotalBytes/1024/1024/1024, d.FreeBytes/1024/1024/1024)
		}
	}

	// Вывод сетевых интерфейсов
	if len(m.Network) > 0 {
		fmt.Println("🌐 Сетевые интерфейсы:")
		for _, n := range m.Network {
			statusSymbol := "🔴"
			if n.Status == collector.InterfaceStatusUp {
				statusSymbol = "🟢"
			}
			// Важно: вызываем метод .String() у кастомного типа MACAddress
			fmt.Printf("   ├─ %s %s MAC: %s [%s]\n", statusSymbol, n.Name, n.MAC.String(), n.IPType)
			for _, ip := range n.IPAddresses {
				// Важно: вызываем метод .String() у кастомного типа IPAddress
				fmt.Printf("   │  └─ IP: %s\n", ip.String())
			}
		}
	}
	fmt.Println("====================================================")
}
