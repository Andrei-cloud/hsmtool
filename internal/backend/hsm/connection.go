package hsm

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/andrei-cloud/anet"
)

// ConnectionState represents the current state of the HSM connection.
type ConnectionState int32

const (
	Disconnected ConnectionState = iota
	Connected
)

// Connection manages the HSM connection using anet broker.
type Connection struct {
	mu            sync.RWMutex
	state         atomic.Int32
	host          string
	port          string
	broker        anet.Broker
	stateChanged  func(ConnectionState)
	poolCap       uint32
	workerCount   int
	stopChan      chan struct{}
	lastError     error
	defaultConfig *anet.PoolConfig
}

// NewConnection creates a new HSM connection manager.
func NewConnection(stateChanged func(ConnectionState)) *Connection {
	return &Connection{
		state:        atomic.Int32{},
		poolCap:      2, // Number of TCP connections to maintain
		workerCount:  3, // Number of worker goroutines for the broker
		stopChan:     make(chan struct{}),
		stateChanged: stateChanged,
		defaultConfig: &anet.PoolConfig{
			DialTimeout:        5 * time.Second,
			IdleTimeout:        60 * time.Second,
			ValidationInterval: 30 * time.Second,
			KeepAliveInterval:  30 * time.Second,
		},
	}
}

// Connect attempts to connect to the HSM.
func (c *Connection) Connect(host, port string) error {
	// First check connection state without lock
	if ConnectionState(c.state.Load()) == Connected {
		return errors.New("already connected")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check state after acquiring lock
	if ConnectionState(c.state.Load()) == Connected {
		return errors.New("already connected")
	}

	c.host = host
	c.port = port

	// Create the broker using anet
	broker, err := c.createBroker()
	if err != nil {
		c.lastError = err
		return err
	}

	// Start broker in background
	brokerStarted := make(chan struct{})
	go func() {
		close(brokerStarted) // Signal that broker.Start() has been called
		if err := broker.Start(); err != nil && err != anet.ErrQuit {
			c.lastError = err
			c.setState(Disconnected) // Reset state if broker fails
		}
	}()

	// Wait briefly for initialization
	select {
	case <-brokerStarted:
		// Broker has started running
		c.broker = broker
		c.setState(Connected)
		return nil
	case <-time.After(200 * time.Millisecond):
		// Something is wrong, clean up
		broker.Close()
		err := errors.New("failed to initialize broker")
		c.lastError = err
		return err
	}
}

// Disconnect closes the HSM connection.
func (c *Connection) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.GetState() == Disconnected {
		return errors.New("already disconnected")
	}

	// Set state to disconnected first to prevent new operations
	c.setState(Disconnected)

	if c.broker != nil {
		// Close the broker which will cleanup pools and connections
		c.broker.Close()
		c.broker = nil
	}

	return nil
}

// GetState returns the current connection state.
func (c *Connection) GetState() ConnectionState {
	return ConnectionState(c.state.Load())
}

// GetLastError returns the last error that occurred.
func (c *Connection) GetLastError() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastError
}

// setState updates the connection state and notifies listeners.
func (c *Connection) setState(state ConnectionState) {
	c.state.Store(int32(state))
	if c.stateChanged != nil {
		c.stateChanged(state)
	}
}

// ExecuteCommand sends a command to the HSM and returns the response.
func (c *Connection) ExecuteCommand(cmd []byte) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.GetState() == Disconnected {
		return nil, errors.New("not connected to HSM")
	}

	resp, err := c.broker.Send(&cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	return resp, nil
}

// createBroker initializes the anet broker.
func (c *Connection) createBroker() (anet.Broker, error) {
	factory := func(addr string) (anet.PoolItem, error) {
		conn, err := net.DialTimeout("tcp", addr, c.defaultConfig.DialTimeout)
		if err != nil {
			return nil, err
		}

		tcpConn, ok := conn.(*net.TCPConn)
		if !ok {
			conn.Close()
			return nil, errors.New("not a TCP connection")
		}

		if err := tcpConn.SetKeepAlive(true); err != nil {
			conn.Close()
			return nil, err
		}

		if err := tcpConn.SetKeepAlivePeriod(c.defaultConfig.KeepAliveInterval); err != nil {
			conn.Close()
			return nil, err
		}

		return conn, nil
	}

	addr := net.JoinHostPort(c.host, c.port)
	pool := anet.NewPool(c.poolCap, factory, addr, c.defaultConfig)

	// Test connection before creating broker
	ctx, cancel := context.WithTimeout(context.Background(), c.defaultConfig.DialTimeout)
	defer cancel()

	// Try to establish a test connection
	item, err := pool.GetWithContext(ctx)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to establish test connection: %w", err)
	}
	pool.Put(item) // Return the test connection to the pool

	// Configuration for the broker with reasonable timeouts
	brokerConfig := &anet.BrokerConfig{
		WriteTimeout: 5 * time.Second,
		ReadTimeout:  5 * time.Second,
		QueueSize:    1000,
	}

	return anet.NewBroker([]anet.Pool{pool}, c.workerCount, nil, brokerConfig), nil
}
