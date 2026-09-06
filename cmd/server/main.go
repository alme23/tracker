package main

import (
	"log"

	"github.com/alme23/tracker/internal/server"
)

func main() {
	cfg := server.Config{
		ListenAddr:   ":8443",
		SharedSecret: "test-secret-key",
		DBPath:       "tracker.db",
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatalf("Ошибка сервера: %v", err)
	}

	if err := srv.RunWithSignals(); err != nil {
		log.Fatalf("Ошибка запуска: %v", err)
	}
}
