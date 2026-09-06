package server

import (
	"context"
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

// Config contains server settings
type Config struct {
	ListenAddr   string // Address to listen on (e.g., ":8443")
	SharedSecret string // Shared secret key for decryption
	DBPath       string // Path to SQLite database
}

// Server accepts data from agents
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

// New creates a new server
func New(cfg Config) (*Server, error) {
	storage, err := NewStorage(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("storage creation: %w", err)
	}

	return &Server{
		config:  cfg,
		storage: storage,
		conns:   make(map[net.Conn]struct{}),
		done:    make(chan struct{}),
	}, nil
}

// Start starts the server
func (s *Server) Start() error {
	// Create listen config with context
	lc := &net.ListenConfig{}

	listener, err := lc.Listen(context.Background(), "tcp", s.config.ListenAddr)
	if err != nil {
		return fmt.Errorf("listener startup: %w", err)
	}
	s.listener = listener

	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	log.Printf("Server started on %s", s.config.ListenAddr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			s.mu.RLock()
			running := s.running
			s.mu.RUnlock()

			if !running {
				return nil // Server stopped
			}

			log.Printf("Accept error: %v", err)
			continue
		}

		// Register connection
		s.mu.Lock()
		s.conns[conn] = struct{}{}
		s.mu.Unlock()

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// RunWithSignals starts the server with system signal handling
func (s *Server) RunWithSignals() error {
	// Signal channel
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Error channel
	errChan := make(chan error, 1)

	// Start server in goroutine
	go func() {
		errChan <- s.Start()
	}()

	log.Printf("Server started. Press Ctrl+C to stop")

	// Wait for signal or error
	select {
	case sig := <-sigChan:
		log.Printf("Received signal %v", sig)
		s.Stop()
		return nil

	case err := <-errChan:
		if err != nil {
			return fmt.Errorf("server error: %w", err)
		}
		return nil
	}
}

// Stop stops the server with graceful shutdown
func (s *Server) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	log.Println("Stopping server...")

	// 1. Close listener
	if s.listener != nil {
		_ = s.listener.Close()
		log.Println("Listener closed")
	}

	// 2. Close active connections
	s.mu.Lock()
	for conn := range s.conns {
		_ = conn.Close()
	}
	s.mu.Unlock()
	log.Println("Active connections closed")

	// 3. Wait for handlers to complete
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All handlers completed")
	case <-time.After(10 * time.Second):
		log.Println("Handler wait timeout")
	}

	// 4. Final cleanup
	if err := s.storage.CleanupOldData(); err != nil {
		log.Printf("Final cleanup error: %v", err)
	}

	// 5. Close storage
	if s.storage != nil {
		_ = s.storage.Close()
		log.Println("Database closed")
	}

	close(s.done)
	log.Println("Server stopped")
}

// handleConnection processes a single connection
func (s *Server) handleConnection(conn net.Conn) {
	defer func() {
		s.mu.Lock()
		delete(s.conns, conn)
		s.mu.Unlock()

		s.wg.Done()
		_ = conn.Close()
	}()

	remoteAddr := conn.RemoteAddr().String()
	log.Printf("Connection from %s", remoteAddr)

	// Receive and decrypt data
	data, err := secproto.HandleConnection(conn, s.config.SharedSecret)
	if err != nil {
		log.Printf("secproto error from %s: %v", remoteAddr, err)
		return
	}

	// Decode binary data
	decoder := binproto.NewDecoder(data)
	snapshot, err := decoder.Decode()
	if err != nil {
		log.Printf("Decode error from %s: %v", remoteAddr, err)
		return
	}

	// Save to database
	if err := s.storage.SaveSnapshot(snapshot); err != nil {
		log.Printf("Save error from %s: %v", remoteAddr, err)
		return
	}

	log.Printf("Data saved: hostname=%s, user=%s",
		snapshot.Host.Hostname, snapshot.User.Username)
}
