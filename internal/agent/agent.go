// internal/agent/agent.go
package agent

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/alme23/tracker/internal/binproto"
	"github.com/alme23/tracker/internal/collector"
	"github.com/alme23/tracker/internal/models"
	"github.com/alme23/tracker/internal/secproto"
)

// Config содержит настройки агента
type Config struct {
	ServerAddr   string        // Адрес сервера (host:port)
	SharedSecret string        // Секретный ключ
	Timeout      time.Duration // Таймаут соединения
}

// Agent отвечает за сбор и отправку данных
type Agent struct {
	config    Config
	collector *collector.SystemCollector
	encoder   *binproto.Encoder
}

// New создает нового агента
func New(cfg Config) *Agent {
	return &Agent{
		config:    cfg,
		collector: collector.NewSystemCollector(2 * time.Second),
		encoder:   binproto.NewEncoder(),
	}
}

// RunOnce собирает данные и отправляет их один раз
func (a *Agent) RunOnce(ctx context.Context) error {
	// 1. Сбор данных
	log.Println("Сбор данных...")
	snapshot, err := a.collector.CollectAll()
	if err != nil {
		return fmt.Errorf("сбор данных: %w", err)
	}

	log.Printf("Данные собраны: hostname=%s, user=%s",
		snapshot.Host.Hostname, snapshot.User.Username)

	// 2. Отправка
	if err := a.Send(ctx, snapshot); err != nil {
		return fmt.Errorf("отправка данных: %w", err)
	}

	log.Printf("Данные отправлены на %s", a.config.ServerAddr)

	return nil
}

// Send отправляет snapshot на сервер
func (a *Agent) Send(ctx context.Context, snapshot *models.SystemSnapshot) error {
	// Бинарная сериализация
	data, err := a.encoder.Encode(snapshot)
	if err != nil {
		return fmt.Errorf("сериализация: %w", err)
	}

	log.Printf("Размер пакета: %d байт", len(data))

	// Отправка через secproto
	if err := secproto.Send(ctx, a.config.ServerAddr, a.config.SharedSecret, data); err != nil {
		return fmt.Errorf("secproto: %w", err)
	}

	return nil
}
