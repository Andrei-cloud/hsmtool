package hsm

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/andrei-cloud/anet"
)

// Config holds HSM client configuration.
type Config struct {
	Host        string
	Port        int
	PoolSize    uint32
	LMKIndex    int
	MaxWorker   int
	DialTimeout time.Duration
	IdleTimeout time.Duration
}

// Client represents an HSM client.
type Client struct {
	broker  anet.Broker
	pool    anet.Pool
	config  *Config
	mu      sync.RWMutex
	isReady bool
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		PoolSize:    5,
		MaxWorker:   2,
		DialTimeout: 5 * time.Second,
		IdleTimeout: 60 * time.Second,
	}
}

// factory creates a function that implements anet.Factory.
func (c *Client) factory() anet.Factory {
	return func(addr string) (anet.PoolItem, error) {
		conn, err := net.DialTimeout("tcp", addr, c.config.DialTimeout)
		if err != nil {
			return nil, err
		}

		return conn, nil
	}
}

// NewClient creates a new HSM client with the given configuration.
func NewClient(cfg *Config) *Client {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	client := &Client{config: cfg}

	// Create pool configuration.
	poolCfg := &anet.PoolConfig{
		DialTimeout: cfg.DialTimeout,
		IdleTimeout: cfg.IdleTimeout,
	}

	// Initialize pool and broker.
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	client.pool = anet.NewPool(cfg.PoolSize, client.factory(), addr, poolCfg)
	client.broker = anet.NewBroker([]anet.Pool{client.pool}, cfg.MaxWorker, nil, nil)
	client.isReady = true

	return client
}

// SendCommand sends a command to the HSM and returns the response.
func (c *Client) SendCommand(cmd []byte) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if !c.isReady {
		return nil, errors.New("hsm client not ready")
	}

	resp, err := c.broker.Send(&cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to send command: %v", err)
	}

	return resp, nil
}

// Close closes the HSM client connection.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.isReady = false
	c.pool.Close()
}
