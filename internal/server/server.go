// internal/server/server.go
package server

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/alme23/tracker/internal/binproto"
	"github.com/alme23/tracker/internal/secproto"
)

// Config содержит настройки сервера
type Config struct {
	ListenAddr   string // Адрес для прослушивания
	SharedSecret string // Секретный ключ
	DBPath       string // Путь к базе данных SQLite
}

// Server принимает данные от агентов
type Server struct {
	config   Config
	storage  *Storage
	listener net.Listener

	mu      sync.RWMutex
	running bool
	wg      sync.WaitGroup
	conns   map[net.Conn]struct{}
	done    chan struct{}
}

// New создает новый сервер
func New(cfg Config) (*Server, error) {
	storage, err := NewStorage(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("создание хранилища: %w", err)
	}

	return &Server{
		config:  cfg,
		storage: storage,
		conns:   make(map[net.Conn]struct{}),
		done:    make(chan struct{}),
	}, nil
}

// Start запускает сервер
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.config.ListenAddr)
	if err != nil {
		return fmt.Errorf("запуск listener: %w", err)
	}
	s.listener = listener

	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	log.Printf("Сервер запущен на %s", s.config.ListenAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			s.mu.RLock()
			running := s.running
			s.mu.RUnlock()

			if !running {
				return nil // Сервер остановлен
			}

			log.Printf("Ошибка accept: %v", err)
			continue
		}

		// Регистрируем соединение
		s.mu.Lock()
		s.conns[conn] = struct{}{}
		s.mu.Unlock()

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// Stop останавливает сервер с graceful shutdown
func (s *Server) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	log.Println("Остановка сервера...")

	// 1. Закрываем listener (новые соединения не принимаются)
	if s.listener != nil {
		s.listener.Close()
		log.Println("Listener закрыт")
	}

	// 2. Закрываем все активные соединения
	s.mu.Lock()
	for conn := range s.conns {
		conn.Close()
	}
	s.mu.Unlock()
	log.Println("Активные соединения закрыты")

	// 3. Ждем завершения обработчиков с таймаутом
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("Все обработчики завершены")
	case <-time.After(10 * time.Second):
		log.Println("Таймаут ожидания обработчиков")
	}

	// 4. Закрываем хранилище
	if s.storage != nil {
		s.storage.Close()
		log.Println("База данных закрыта")
	}

	close(s.done)
	log.Println("Сервер остановлен")
}

// RunWithSignals запускает сервер с обработкой системных сигналов
func (s *Server) RunWithSignals() error {
	// Канал для сигналов
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Запускаем сервер в горутине
	errChan := make(chan error, 1)
	go func() {
		errChan <- s.Start()
	}()

	// Ждем сигнал или ошибку
	select {
	case sig := <-sigChan:
		log.Printf("Получен сигнал %v", sig)
		s.Stop()
		return nil

	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("ошибка сервера: %w", err)
		}
		return nil
	}
}

// handleConnection обрабатывает одно соединение
func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		// Удаляем соединение из мапы
		s.mu.Lock()
		delete(s.conns, conn)
		s.mu.Unlock()

		s.wg.Done()
		conn.Close()
	}()

	remoteAddr := conn.RemoteAddr().String()
	log.Printf("Подключение от %s", remoteAddr)

	// Проверяем, не остановлен ли сервер
	s.mu.RLock()
	running := s.running
	s.mu.RUnlock()

	if !running {
		return
	}

	// Принимаем и расшифровываем данные
	data, err := secproto.HandleConnection(conn, s.config.SharedSecret)
	if err != nil {
		log.Printf("Ошибка secproto от %s: %v", remoteAddr, err)
		return
	}

	// Декодируем бинарные данные
	decoder := binproto.NewDecoder(data)
	snapshot, err := decoder.Decode()
	if err != nil {
		log.Printf("Ошибка декодирования от %s: %v", remoteAddr, err)
		return
	}

	// Сохраняем в БД
	if err := s.storage.SaveSnapshot(snapshot); err != nil {
		log.Printf("Ошибка сохранения от %s: %v", remoteAddr, err)
		return
	}

	log.Printf("Данные сохранены: hostname=%s, user=%s",
		snapshot.Host.Hostname, snapshot.User.Username)
}
