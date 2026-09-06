package main

import (
	"context"
	"log"
	"time"

	"github.com/alme23/tracker/internal/agent"
)

func main() {
	cfg := agent.Config{
		ServerAddr:   "localhost:8443",
		SharedSecret: "test-secret-key",
		Timeout:      10 * time.Second,
	}

	a := agent.New(cfg)

	ctx := context.Background()

	if err := a.RunOnce(ctx); err != nil {
		log.Fatalf("Ошибка: %v", err)
	}

	log.Println("Готово")
}
