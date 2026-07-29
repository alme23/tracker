package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/gob"
	"fmt"
	"html/template"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/alme23/tracker/internal/crypto"
	"github.com/alme23/tracker/internal/database"
	"github.com/alme23/tracker/pkg/collector"
	// Импортируем крипто-модуль, репозиторий БД и пакет коллектора
)

var (
	repo      *database.Repository
	wg        sync.WaitGroup
	clientsMu sync.Mutex
	clients   = make(map[chan string]bool)

	// Флаг остановки (0 - работает, 1 - тушится) для предотвращения гонки базы данных
	isShuttingDown int32
)

// broadcastNewRow отправляет JSON-пакет данных во все открытые веб-браузеры для SSE
func broadcastNewRow(m collector.Metrics, ip string) {
	// Если сервер тушится, прекращаем рассылку
	if atomic.LoadInt32(&isShuttingDown) == 1 {
		return
	}

	logTime := time.Now().Format("2006-01-02 15:04:05")
	fqdn := m.Host.FQDN
	if fqdn == "" {
		fqdn = m.Host.ComputerName
	}
	rdpStat := m.RemoteAccess.RDPStatus
	if rdpStat == "" {
		rdpStat = "Stopped"
	}
	vncStat := m.RemoteAccess.VNCStatus
	if vncStat == "" {
		vncStat = "Not Installed"
	}

	jsonData := fmt.Sprintf(
		`{"guid":"%s","time":"%s","host":"%s","fqdn":"%s","login":"%s","name":"%s","ip":"%s","rdp":"%s","vnc":"%s"}`,
		template.JSEscapeString(m.Host.DeviceID),
		template.JSEscapeString(logTime),
		template.JSEscapeString(m.Host.ComputerName),
		template.JSEscapeString(fqdn),
		template.JSEscapeString(m.User.UserName),
		template.JSEscapeString(m.User.DisplayName),
		template.JSEscapeString(ip),
		template.JSEscapeString(rdpStat),
		template.JSEscapeString(vncStat),
	)

	clientsMu.Lock()
	defer clientsMu.Unlock()
	for ch := range clients {
		select {
		case ch <- jsonData:
		default:
		}
	}
}

// handleSSE удерживает постоянный HTTP-канал с браузерами для трансляции событий онлайн
func handleSSE(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	messageChan := make(chan string, 10)
	clientsMu.Lock()
	clients[messageChan] = true
	clientsMu.Unlock()

	defer func() {
		clientsMu.Lock()
		delete(clients, messageChan)
		clientsMu.Unlock()
		close(messageChan)
	}()

	_, _ = fmt.Fprintf(w, "retry: 2000\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case msg := <-messageChan:
			_, err := fmt.Fprintf(w, "data: %s\n\n", msg)
			if err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// handleWebPanel динамически подгружает и рендерит внешний шаблон index.html
func handleWebPanel(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/events" {
		handleSSE(w, r)
		return
	}
	if r.URL.Path == "/details" {
		guid := r.URL.Query().Get("guid")
		w.Header().Set("Content-Type", "application/json")

		// Защита: если база закрывается, отдаем пустой ответ
		if atomic.LoadInt32(&isShuttingDown) == 1 {
			_, _ = w.Write([]byte(`{"error":"Сервер останавливается"}`))
			return
		}

		jsonBytes, err := repo.GetHostDetailsJSON(guid)
		if err != nil {
			_, _ = w.Write([]byte(`{"error":"Данные хоста еще не собраны дельта-фильтром"}`))
			return
		}
		_, _ = w.Write(jsonBytes)
		return
	}
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Защита рендеринга при shutdown
	if atomic.LoadInt32(&isShuttingDown) == 1 {
		http.Error(w, "Сервер останавливается", http.StatusServiceUnavailable)
		return
	}

	searchQuery := r.URL.Query().Get("q")
	logs, err := repo.SearchSessions(searchQuery)
	if err != nil {
		http.Error(w, "Ошибка чтения БД: "+err.Error(), http.StatusInternalServerError)
		return
	}

	execPath, err := os.Executable()
	if err != nil {
		http.Error(w, "Внутренняя ошибка", http.StatusInternalServerError)
		return
	}
	templatePath := filepath.Join(filepath.Dir(execPath), "templates", "index.html")

	funcMap := template.FuncMap{
		"textContains": func(s, substr string) bool {
			return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
		},
	}

	tmpl := template.New(filepath.Base(templatePath)).Funcs(funcMap)
	tmpl, err = tmpl.ParseFiles(templatePath)
	if err != nil {
		tmpl, err = template.New("index.html").Funcs(funcMap).ParseFiles("templates/index.html")
		if err != nil {
			http.Error(w, "Ошибка загрузки index.html: "+err.Error(), http.StatusInternalServerError)
			return
		}
	}

	data := struct {
		Query string
		Logs  []database.WebSessionLog
	}{
		Query: searchQuery,
		Logs:  logs,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.Execute(w, data)
}

// handleAgentConnection координирует зашифрованную сессию передачи метрик от Windows-агента
func handleAgentConnection(conn net.Conn) {
	defer conn.Close()
	remoteIP := conn.RemoteAddr().(*net.TCPAddr).IP.String()

	// АТОМАРНАЯ ЗАЗАЩИТА: Если база данных уже закрывается, мгновенно рвем сокет
	if atomic.LoadInt32(&isShuttingDown) == 1 {
		return
	}

	serverPrivKey, serverPubKeyBytes, err := crypto.GenerateECDHKeypair()
	if err != nil {
		return
	}

	agentPubKeyBytes := make([]byte, 65)
	if _, err := io.ReadFull(conn, agentPubKeyBytes); err != nil {
		return
	}

	if _, err := conn.Write(serverPubKeyBytes); err != nil {
		return
	}

	sessionKey, err := crypto.DeriveSharedSecret(serverPrivKey, agentPubKeyBytes)
	if err != nil {
		return
	}

	encryptedData, err := io.ReadAll(conn)
	if err != nil || len(encryptedData) == 0 {
		return
	}

	decryptedGzip, err := crypto.Decrypt(sessionKey, encryptedData)
	if err != nil {
		log.Printf("[Сервер] Ошибка дешифрования от %s.", remoteIP)
		return
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(decryptedGzip))
	if err != nil {
		return
	}
	defer gzipReader.Close()

	var metrics collector.Metrics
	if err := gob.NewDecoder(gzipReader).Decode(&metrics); err != nil {
		log.Printf("[Сервер] Ошибка бинарной десериализации GOB от %s: %v", remoteIP, err)
		return
	}

	// ФИНАЛЬНАЯ ПРОВЕРКА ПЕРЕД ЗАПИСЬЮ В СУБД
	if atomic.LoadInt32(&isShuttingDown) == 1 {
		return
	}

	// 1. АУДИТ ВХОДОВ: Фиксируем сессию в базу session_logs
	if err := repo.LogSession(metrics, remoteIP); err != nil {
		log.Printf("[Ошибка БД] Не удалось зафиксировать сессию пользователя: %v", err)
	}

	// 2. РЕАЛТАЙМ-ВЕБ: Транслируем сессию в браузеры
	go broadcastNewRow(metrics, remoteIP)

	// 3. ДЕЛЬТА-ФИЛЬТРАЦИЯ ЖЕЛЕЗА: Пишем в host_metrics только физические изменения
	lastMetrics, exists, err := repo.GetLastMetrics(metrics.Host.DeviceID)
	if err != nil {
		_ = repo.SaveMetrics(metrics, remoteIP)
		return
	}

	if exists && isHardwareEqual(lastMetrics, metrics) {
		if err := repo.UpdateTimestamp(metrics.Host.DeviceID, metrics.Timestamp, remoteIP); err != nil {
			log.Printf("[Ошибка БД] Не удалось обновить таймстамп хоста %s: %v", metrics.Host.ComputerName, err)
		}
	} else {
		if err := repo.SaveMetrics(metrics, remoteIP); err != nil {
			log.Printf("[Ошибка БД] Не удалось сохранить аудит хоста %s: %v", metrics.Host.ComputerName, err)
		}
	}
}

// isHardwareEqual сопоставляет статические железные параметры хоста
func isHardwareEqual(old collector.Metrics, new collector.Metrics) bool {
	if old.CPU.Name != new.CPU.Name ||
		old.CPU.PhysicalCores != new.CPU.PhysicalCores ||
		old.CPU.LogicalProcessors != new.CPU.LogicalProcessors ||
		old.Memory.TotalRAMBytes != new.Memory.TotalRAMBytes {
		return false
	}
	if len(old.Disks) != len(new.Disks) {
		return false
	}
	oldDisksMap := make(map[string]collector.DiskInfo)
	for _, d := range old.Disks {
		sn := strings.TrimSpace(d.SerialNumber)
		var key string
		if sn == "" || sn == "Unknown" || strings.Contains(strings.ToLower(sn), "unknown") {
			key = fmt.Sprintf("%s_%d", strings.TrimSpace(d.Vendor), d.TotalBytes)
		} else {
			key = sn
		}
		oldDisksMap[key] = d
	}
	for _, nDisk := range new.Disks {
		sn := strings.TrimSpace(nDisk.SerialNumber)
		var key string
		if sn == "" || sn == "Unknown" || strings.Contains(strings.ToLower(sn), "unknown") {
			key = fmt.Sprintf("%s_%d", strings.TrimSpace(nDisk.Vendor), nDisk.TotalBytes)
		} else {
			key = sn
		}
		oDisk, exists := oldDisksMap[key]
		if !exists || oDisk.TotalBytes != nDisk.TotalBytes {
			return false
		}
	}
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
		if !exists || oGpu.VendorID != nGpu.VendorID || oGpu.DeviceID != nGpu.DeviceID || oGpu.DedicatedMemory != nGpu.DedicatedMemory {
			return false
		}
	}
	return true
}

// loadEnv построчно читает файл .env
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
			// ИСПРАВЛЕНО: Явно передаем индексы элементов слайса [0] и [1]
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			_ = os.Setenv(key, value)
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

	webPort := os.Getenv("WEB_PORT")
	if webPort == "" {
		webPort = "8080"
	}

	mux := http.NewServeMux()
	execPath, _ := os.Executable()
	baseDir := filepath.Dir(execPath)

	templatesRootDir := filepath.Join(baseDir, "templates")
	if _, err := os.Stat(templatesRootDir); os.IsNotExist(err) {
		templatesRootDir = "templates"
		if _, err := os.Stat(templatesRootDir); os.IsNotExist(err) {
			templatesRootDir = "cmd/server/templates"
		}
	}

	fileServer := http.FileServer(http.Dir(templatesRootDir))
	mux.Handle("/static/", fileServer)
	mux.HandleFunc("/", handleWebPanel)

	webServer := &http.Server{
		Addr:    net.JoinHostPort(host, webPort),
		Handler: mux,
	}

	go func() {
		log.Printf("[ВЕБ] Панель Real-Time мониторинга: http://localhost:%s", webPort)
		if err := webServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("[ВЕБ] Ошибка HTTP сервера: %v", err)
		}
	}()

	fmt.Println("====================================================")
	fmt.Printf("🚀 Tracker-Сервер успешно запущен на %s\n", listenAddress)
	fmt.Printf("🖥️ Веб-интерфейс активен на порту :%s\n", webPort)
	fmt.Println("🔐 Контекст: Математический ECDH Диффи-Хеллман (P-256) + AES-GCM")
	fmt.Println("📊 Режим: Умный Time-Series аудит изменений (Delta-фильтрация)")
	fmt.Println("📡 Потоковый апдейт: Автообновление строк через SSE-JSON")
	fmt.Println("🛑 Включен безопасный режим завершения работы (Graceful Shutdown)")
	fmt.Println("====================================================")

	_, cancelShutdown := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-sigChan
		fmt.Printf("\n🛑 Получен сигнал %v. Инициирован Graceful Shutdown...\n", sig)

		// АТОМАРНО ВЫСТАВЛЯЕМ ФЛАГ ОСТАНОВКИ — блокирует новые транзакции к СУБД
		atomic.StoreInt32(&isShuttingDown, 1)

		_ = listener.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = webServer.Shutdown(ctx)

		cancelShutdown()
	}()

	// Главный цикл обработки сетевых подключений агентов
	for {
		conn, err := listener.Accept()
		if err != nil {
			// ИСПРАВЛЕНО: Если атомарный флаг равен 1, значит идет штатное тушение сервера.
			// Мы мгновенно и бесшумно выходим из цикла, полностью игнорируя техническую ошибку сокета.
			if atomic.LoadInt32(&isShuttingDown) == 1 {
				break
			}

			// Выводим ошибку в лог ТОЛЬКО если она произошла во время обычной работы сервера
			log.Printf("[Сервер] Ошибка Accept(): %v", err)
			continue
		}

		// wg.Add(1) вынесен СТРОГО ДО запуска горутины, что гарантирует точность ожидания wg.Wait()
		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done()
			handleAgentConnection(c)
		}(conn)
	}

	fmt.Println("[Сервер] Ожидание завершения обработки текущих сессий клиентов...")
	waitChan := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
		fmt.Println("[Сервер] Все сессии успешно обработаны.")
	case <-time.After(10 * time.Second):
		log.Println("[Предупреждение] Время ожидания истекло, принудительное завершение.")
	}

	_ = repo.Close()
	fmt.Println("👋 Tracker-Сервер полностью остановлен без повреждения SQLite.")
}
