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

// Config contains agent settings
type Config struct {
	ServerAddr   string        // Server address (host:port)
	SharedSecret string        // Shared secret key for encryption
	Timeout      time.Duration // Connection timeout
}

// Agent is responsible for collecting and sending data
type Agent struct {
	config    Config
	collector *collector.SystemCollector
	encoder   *binproto.Encoder
}

// New creates a new agent
func New(cfg Config) *Agent {
	return &Agent{
		config:    cfg,
		collector: collector.NewSystemCollector(2 * time.Second),
		encoder:   binproto.NewEncoder(),
	}
}

// RunOnce collects data and sends it once
func (a *Agent) RunOnce(ctx context.Context) error {
	// 1. Collect data
	log.Println("Collecting data...")
	snapshot, err := a.collector.CollectAll()
	if err != nil {
		return fmt.Errorf("data collection: %w", err)
	}

	log.Printf("Data collected: hostname=%s, user=%s",
		snapshot.Host.Hostname, snapshot.User.Username)

	// 2. Send data
	if err := a.Send(ctx, snapshot); err != nil {
		return fmt.Errorf("data sending: %w", err)
	}

	log.Printf("Data sent to %s", a.config.ServerAddr)

	return nil
}

// Send sends a snapshot to the server
func (a *Agent) Send(ctx context.Context, snapshot *models.SystemSnapshot) error {
	// Binary serialization
	data, err := a.encoder.Encode(snapshot)
	if err != nil {
		return fmt.Errorf("serialization: %w", err)
	}

	log.Printf("Packet size: %d bytes", len(data))

	// Send via secproto
	if err := secproto.Send(ctx, a.config.ServerAddr, a.config.SharedSecret, data); err != nil {
		return fmt.Errorf("secproto: %w", err)
	}

	return nil
}
