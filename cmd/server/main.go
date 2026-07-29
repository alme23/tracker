package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/gob"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/alme23/tracker/internal/crypto"
	"github.com/alme23/tracker/internal/database"
	"github.com/alme23/tracker/pkg/collector"
)

var (
	repo *database.Repository
	wg   sync.WaitGroup // Отслеживает активные сессии агентов
)

func loadEnv() {
	execPath, err := os.Executable()
	if err != nil {
		return
	}
	envPath := filepath.Join(filepath.Dir(execPath), ".env")
	file, err := os.Open(envPath)
	if err != nil {
		file, err = os.Open(".env")
		if err != nil {
			return
		}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			_ = os.Setenv(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}
}

func main() {
	loadEnv()

	dbFile := os.Getenv("DB_FILE")
	if dbFile == "" {
		dbFile = "tracker.db"
	}

	var err error
	repo, err = database.NewRepository(dbFile)
	if err != nil {
		log.Fatalf("[Критическая ошибка] Не удалось запустить репозиторий SQLite: %v", err)
	}

	host := os.Getenv("SERVER_HOST")
	if host == "" {
		host = "0.0.0.0"
	}
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "9000"
	}
	listenAddress := net.JoinHostPort(host, port)

	listener, err := net.Listen("tcp", listenAddress)
	if err != nil {
		_ = repo.Close()
		log.Fatalf("[Критическая ошибка] Не удалось занять адрес %s: %v", listenAddress, err)
	}

	fmt.Println("====================================================")
	fmt.Printf("🚀 Tracker-Сервер успешно запущен на %s\n", listenAddress)
	fmt.Println("🔐 Контекст: Математический ECDH Диффи-Хеллман (P-256) + AES-GCM")
	fmt.Println("📊 Режим: Умный Time-Series аудит изменений (Delta-фильтрация)")
	fmt.Println("🛑 Включен безопасный режим завершения работы (Graceful Shutdown)")
	fmt.Println("====================================================")

	// Создаем канал для перехвата системных сигналов завершения работы
	shutdownCtx, cancelShutdown := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Горутина для асинхронного ожидания сигнала остановки
	go func() {
		sig := <-sigChan
		fmt.Printf("\n🛑 Получен сигнал %v. Инициирован Graceful Shutdown...\n", sig)

		// Закрываем сетевой слушатель, чтобы сервер перестал принимать новые TCP-подключения
		_ = listener.Close()

		// Сигнализируем главному потоку о начале остановки
		cancelShutdown()
	}()

	// Главный цикл обработки сетевых подключений
	for {
		conn, err := listener.Accept()
		if err != nil {
			// Проверяем, вызвана ли ошибка закрытием слушателя во время shutdown
			select {
			case <-shutdownCtx.Done():
				break
			default:
				log.Printf("[Сервер] Ошибка Accept(): %v", err)
				continue
			}
			break
		}

		// Увеличиваем счетчик активных сессий перед запуском горутины
		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done() // Уменьшаем счетчик по завершении обработки пакета
			handleAgentConnection(c)
		}(conn)
	}

	// Ожидаем завершения обработки всех текущих пакетов, которые уже находятся в памяти
	fmt.Println("[Сервер] Ожидание завершения обработки текущих сессий клиентов...")

	// Настраиваем жесткий таймаут ожидания (например, 10 секунд), чтобы сервер не завис вечно
	waitChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
		fmt.Println("[Сервер] Все сессии успешно обработаны и записаны.")
	case <-time.After(10 * time.Second):
		log.Println("[Предупреждение] Время ожидания истекло, принудительное завершение незакрытых сессий.")
	}

	// Безопасно закрываем базу данных, сбрасывая все WAL-логи и транзакции на жесткий диск
	fmt.Println("[Сервер] Закрытие пула базы данных SQLite...")
	if err := repo.Close(); err != nil {
		log.Printf("[Ошибка] Ошибка при закрытии SQLite: %v", err)
	} else {
		fmt.Println("[БД] База данных успешно закрыта без повреждения файлов.")
	}

	fmt.Println("👋 Tracker-Сервер полностью остановлен.")
}

func handleAgentConnection(conn net.Conn) {
	defer conn.Close()
	remoteIP := conn.RemoteAddr().(*net.TCPAddr).IP.String()

	serverPrivKey, serverPubKeyBytes, err := crypto.GenerateECDHKeypair()
	if err != nil {
		log.Printf("[Сервер] Критическая ошибка криптографии: %v", err)
		return
	}

	agentPubKeyBytes := make([]byte, 65)
	if _, err := io.ReadFull(conn, agentPubKeyBytes); err != nil {
		if os.Getenv("SHOW_DEBUG_LOGS") == "true" {
			log.Printf("[Сервер] Ошибка получения ключа от агента %s: %v", remoteIP, err)
		}
		return
	}

	if _, err := conn.Write(serverPubKeyBytes); err != nil {
		return
	}

	sessionKey, err := crypto.DeriveSharedSecret(serverPrivKey, agentPubKeyBytes)
	if err != nil {
		log.Printf("[Сервер] Ошибка вычисления секрета сессии с %s: %v", remoteIP, err)
		return
	}

	encryptedData, err := io.ReadAll(conn)
	if err != nil || len(encryptedData) == 0 {
		return
	}

	decryptedGzip, err := crypto.Decrypt(sessionKey, encryptedData)
	if err != nil {
		log.Printf("[Сервер] Ошибка дешифрования от %s. Взлом сессии или рассинхронизация.", remoteIP)
		return
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(decryptedGzip))
	if err != nil {
		log.Printf("[Сервер] Ошибка декомпрессии Gzip от %s: %v", remoteIP, err)
		return
	}
	defer gzipReader.Close()

	var metrics collector.Metrics
	decoder := gob.NewDecoder(gzipReader)
	if err := decoder.Decode(&metrics); err != nil {
		log.Printf("[Сервер] Ошибка бинарной десериализации GOB от %s: %v", remoteIP, err)
		return
	}

	lastMetrics, exists, err := repo.GetLastMetrics(metrics.Host.DeviceID)
	if err != nil {
		log.Printf("[Ошибка БД] Не удалось проверить историю хоста: %v", err)
		_ = repo.SaveMetrics(metrics, remoteIP)
		return
	}

	if exists && isHardwareEqual(lastMetrics, metrics) {
		if err := repo.UpdateTimestamp(metrics.Host.DeviceID, metrics.Timestamp, remoteIP); err != nil {
			log.Printf("[Ошибка БД] Не удалось обновить таймстамп хоста %s: %v", metrics.Host.ComputerName, err)
		} else if os.Getenv("SHOW_DEBUG_LOGS") == "true" {
			log.Printf("[БД] Конфигурация хоста %s не изменилась. Таймстамп обновлен.", metrics.Host.ComputerName)
		}
	} else {
		if err := repo.SaveMetrics(metrics, remoteIP); err != nil {
			log.Printf("[Ошибка БД] Не удалось сохранить комплексный аудит хоста %s: %v", metrics.Host.ComputerName, err)
		} else {
			log.Printf("[БД] 👀 Зафиксировано изменение конфигурации или новый хост: %s. Создана новая реляционная запись.", metrics.Host.ComputerName)
		}
	}

	printMetrics(metrics, remoteIP)
}

// isHardwareEqual проверяет неизменность железных узлов.
// Полностью защищен от динамической смены букв дисков и пустых серийных номеров.
func isHardwareEqual(old collector.Metrics, new collector.Metrics) bool {
	// 1. Проверяем процессор
	if old.CPU.Name != new.CPU.Name ||
		old.CPU.PhysicalCores != new.CPU.PhysicalCores ||
		old.CPU.LogicalProcessors != new.CPU.LogicalProcessors {
		return false
	}

	// 2. Проверяем общий объем RAM
	if old.Memory.TotalRAMBytes != new.Memory.TotalRAMBytes {
		return false
	}

	// 3. Сравниваем количество физических дисков
	if len(old.Disks) != len(new.Disks) {
		return false
	}

	// Строим карту старых дисков.
	// Чтобы защититься от пустых/одинаковых серийников и смены букв разделов,
	// делаем составной ключ: "Модель_Объем" или "Серийник" (если он валидный).
	oldDisksMap := make(map[string]collector.DiskInfo)
	for _, d := range old.Disks {
		sn := strings.TrimSpace(d.SerialNumber)
		var key string
		if sn == "" || sn == "Unknown" || strings.Contains(strings.ToLower(sn), "unknown") {
			// Если серийник не валидный, создаем уникальный отпечаток самого железа: модель + размер в байтах
			key = fmt.Sprintf("%s_%d", strings.TrimSpace(d.Vendor), d.TotalBytes)
		} else {
			key = sn
		}
		oldDisksMap[key] = d
	}

	// Проверяем новые диски по такой же схеме составных ключей
	for _, nDisk := range new.Disks {
		sn := strings.TrimSpace(nDisk.SerialNumber)
		var key string
		if sn == "" || sn == "Unknown" || strings.Contains(strings.ToLower(sn), "unknown") {
			key = fmt.Sprintf("%s_%d", strings.TrimSpace(nDisk.Vendor), nDisk.TotalBytes)
		} else {
			key = sn
		}

		oDisk, exists := oldDisksMap[key]
		if !exists {
			log.Printf("[Diff-Trace] Найден новый накопитель (ключ: %s), которого не было в БД", key)
			return false
		}

		// ИСПРАВЛЕНО: Мы БОЛЬШЕ НЕ сравниваем oDisk.Drive != nDisk.Drive!
		// Если диск поменял букву в Windows (был G:\, стал C:\) — это всё еще то же самое железо.
		if oDisk.TotalBytes != nDisk.TotalBytes {
			log.Printf("[Diff-Trace] Накопитель %s изменил физический размер", nDisk.Vendor)
			return false
		}
	}

	// 4. Сравниваем видеокарты
	if len(old.Video) != len(new.Video) {
		return false
	}

	oldGpusMap := make(map[string]collector.VideoInfo)
	for _, v := range old.Video {
		nameKey := strings.ToLower(strings.TrimSpace(v.Name))
		oldGpusMap[nameKey] = v
	}

	for _, nGpu := range new.Video {
		nameKey := strings.ToLower(strings.TrimSpace(nGpu.Name))
		oGpu, exists := oldGpusMap[nameKey]
		if !exists {
			return false
		}
		if oGpu.VendorID != nGpu.VendorID || oGpu.DeviceID != nGpu.DeviceID || oGpu.DedicatedMemory != nGpu.DedicatedMemory {
			return false
		}
	}

	return true // Железо совпало на 100%
}

func printMetrics(m collector.Metrics, ip string) {
	fmt.Printf("\n📥 [ВЕРИФИЦИРОВАНО ECDH] Время приема: %s (IP: %s)\n", time.Unix(m.Timestamp, 0).Format("2006-01-02 15:04:05"), ip)
	fmt.Printf("💻 Хост: %s | FQDN: %s | Пользователь: %s [%s]\n", m.Host.ComputerName, m.Host.FQDN, m.User.DisplayName, m.User.UserName)
	fmt.Printf("📊 Хранилище: дисков зафиксировано (%d), сетей (%d), GPU (%d)\n", len(m.Disks), len(m.Network), len(m.Video))
	fmt.Println("====================================================")
}
