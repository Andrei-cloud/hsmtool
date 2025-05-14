package hsm

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/andrei-cloud/anet"
)

// ConnectionState represents the current state of the HSM connection.
const (
	Disconnected ConnectionState = iota
	Connected
	Reconnecting
)

type ConnectionState int32

// Connection manages the HSM connection using anet broker.
type Connection struct {
	mu             sync.RWMutex
	state          atomic.Int32
	host           string
	port           string
	broker         anet.Broker
	pool           anet.Pool
	stateChanged   func(ConnectionState)
	stateCallbacks []func(state ConnectionState, lastError error)
	poolCap        uint32
	workerCount    int
	stopChan       chan struct{}
	lastError      error
	defaultConfig  *anet.PoolConfig
	reconnecting   atomic.Bool
	sendMu         sync.Mutex // serialize command sends
}

// NewConnection creates a new HSM connection manager.
func NewConnection(stateChanged func(ConnectionState)) *Connection {
	return &Connection{
		state:        atomic.Int32{},
		workerCount:  3,
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
func (c *Connection) Connect(
	host, port string,
	numConns uint32,
) error {
	if ConnectionState(c.state.Load()) == Connected {
		return errors.New("already connected")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	// Reset reconnection flag.
	c.reconnecting.Store(false)

	// Clean up any existing broker and pool.
	if c.broker != nil {
		c.broker.Close()
		c.broker = nil
	}
	if c.pool != nil {
		c.pool.Close()
		c.pool = nil
	}

	if numConns < 1 {
		numConns = 1
	}
	c.poolCap = numConns
	c.host = host
	c.port = port

	broker, pool, err := c.createBroker()
	if err != nil {
		c.lastError = err
		return err
	}
	c.broker = broker
	c.pool = pool

	// Start broker in a goroutine with proper error handling.
	go func() {
		brokerToRun := c.broker
		if brokerToRun == nil {
			c.setState(Disconnected)
			return
		}

		startErr := brokerToRun.Start()

		c.mu.Lock()
		// Only handle errors if this is still the active broker.
		if c.broker == brokerToRun && (startErr != nil && !errors.Is(startErr, anet.ErrQuit)) {
			c.lastError = fmt.Errorf("broker stopped unexpectedly: %w", startErr)
			// Only attempt reconnection if not deliberately disconnecting.
			if !c.reconnecting.Load() {
				go c.handleReconnection()
			}
		} else if c.broker == brokerToRun {
			// Only clear error if this is still the active broker.
			c.lastError = nil
		}
		c.mu.Unlock()

		// Only change state if this is still the active broker.
		c.mu.RLock()
		if c.broker == brokerToRun {
			c.setState(Disconnected)
		}
		c.mu.RUnlock()
	}()

	// Wait a short time to ensure broker starts.
	time.Sleep(100 * time.Millisecond)

	c.setState(Connected)
	c.lastError = nil

	return nil
}

// Disconnect closes the HSM connection.
func (c *Connection) Disconnect() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ConnectionState(c.state.Load()) == Disconnected {
		return errors.New("already disconnected")
	}

	c.setState(Disconnected)

	if c.pool != nil {
		c.pool.Close()
	}

	if c.broker != nil {
		c.broker.Close()
	}

	c.broker = nil
	c.pool = nil

	return nil
}

// GetState returns the current connection state.
func (c *Connection) GetState() ConnectionState {
	return ConnectionState(c.state.Load())
}

// GetPoolCapacity returns the configured capacity of the connection pool.
// This indicates how many concurrent connections the pool can manage.
func (c *Connection) GetPoolCapacity() uint32 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Ensure poolCap is returned even if broker is not yet initialized or is nil,
	// as poolCap is set during Connect before broker creation.
	return c.poolCap
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
	c.notifyStateChange()
}

// RegisterStateCallback registers a callback function to be called when connection state changes.
func (c *Connection) RegisterStateCallback(callback func(state ConnectionState, lastError error)) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stateCallbacks = append(c.stateCallbacks, callback)
}

func (c *Connection) notifyStateChange() {
	state := ConnectionState(c.state.Load())
	var err error
	if c.lastError != nil {
		err = c.lastError
	}
	for _, callback := range c.stateCallbacks {
		if callback != nil {
			go callback(state, err) // Non-blocking notifications
		}
	}
}

// ExecuteCommand sends a command to the HSM and returns the response.
func (c *Connection) ExecuteCommand(command []byte, timeout time.Duration) ([]byte, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.broker == nil {
		return nil, errors.New("broker is not initialized")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	response, err := c.broker.SendContext(ctx, &command)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// hasIsClosed checks if the broker implements IsClosed().
func hasIsClosed(b any) bool {
	type isClosed interface {
		IsClosed() bool
	}
	_, ok := b.(isClosed)

	return ok
}

// createBroker initializes the anet broker.
func (c *Connection) createBroker() (anet.Broker, anet.Pool, error) {
	addr := fmt.Sprintf("%s:%s", c.host, c.port)

	factory := func(address string) (anet.PoolItem, error) {
		conn, err := net.DialTimeout("tcp", address, c.defaultConfig.DialTimeout)
		if err != nil {
			return nil, fmt.Errorf("failed to dial %s: %w", address, err)
		}
		if tcpConn, ok := conn.(*net.TCPConn); ok {
			_ = tcpConn.SetKeepAlive(true)
			_ = tcpConn.SetKeepAlivePeriod(c.defaultConfig.KeepAliveInterval)
		}

		return conn, nil
	}

	pool := anet.NewPool(c.poolCap, factory, addr, c.defaultConfig)

	broker := anet.NewBroker([]anet.Pool{pool}, c.workerCount, nil, nil)
	if broker == nil {
		pool.Close()
		return nil, nil, errors.New("failed to create anet broker")
	}

	return broker, pool, nil
}

// handleReconnection attempts to reconnect to the HSM.
func (c *Connection) handleReconnection() {
	// Ensure only one reconnection attempt runs at a time
	if !c.reconnecting.CompareAndSwap(false, true) {
		return
	}
	defer c.reconnecting.Store(false)

	c.mu.Lock()
	c.state.Store(int32(Reconnecting))
	c.notifyStateChange()
	c.mu.Unlock()

	// Initialize reconnection parameters
	maxAttempts := 5
	backoffBase := time.Second
	maxBackoff := 30 * time.Second
	attempt := 0

	for attempt < maxAttempts {
		// Calculate backoff duration with exponential increase
		backoff := time.Duration(
			math.Min(float64(backoffBase)*math.Pow(2, float64(attempt)), float64(maxBackoff)),
		)
		time.Sleep(backoff)
		attempt++

		// Clean up existing connection
		c.mu.Lock()
		if c.broker != nil {
			if b, ok := any(c.broker).(interface{ Stop() error }); ok {
				_ = b.Stop() // Ignore error as we're replacing it anyway
			}
			c.broker = nil
		}
		if c.pool != nil {
			if p, ok := any(c.pool).(interface{ Stop() error }); ok {
				_ = p.Stop()
			}
			c.pool = nil
		}
		c.mu.Unlock()

		// Create new connection
		broker, pool, err := c.createBroker()
		if err != nil {
			c.mu.Lock()
			c.lastError = fmt.Errorf("reconnection attempt %d failed: %w", attempt, err)
			c.mu.Unlock()

			continue
		}

		err = broker.Start()
		if err != nil {
			if b, ok := any(broker).(interface{ Stop() error }); ok {
				_ = b.Stop()
			}
			if p, ok := any(pool).(interface{ Stop() error }); ok {
				_ = p.Stop()
			}
			c.mu.Lock()
			c.lastError = fmt.Errorf("broker start failed on attempt %d: %w", attempt, err)
			c.mu.Unlock()

			continue
		}

		// Connection successful
		c.mu.Lock()
		c.pool = pool
		c.broker = broker
		c.state.Store(int32(Connected))
		c.lastError = nil
		c.notifyStateChange()
		c.mu.Unlock()

		return // Successful reconnection
	}

	// All attempts failed
	c.mu.Lock()
	c.state.Store(int32(Disconnected))
	if c.lastError == nil {
		c.lastError = fmt.Errorf("failed to reconnect after %d attempts", maxAttempts)
	}
	c.notifyStateChange()
	c.mu.Unlock()
}
