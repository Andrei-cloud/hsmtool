// nolint:all // test package
package hsm

import (
	"context"
	"errors"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/andrei-cloud/anet"
)

// MockBroker is a mock implementation of anet.Broker.
// It allows us to control the behavior of Send and Close methods for testing.
type MockBroker struct {
	SendFunc        func(request *[]byte) ([]byte, error)
	SendContextFunc func(ctx context.Context, request *[]byte) ([]byte, error)
	CloseFunc       func()
	StartFunc       func() error
}

// Send calls the mock SendFunc.
func (m *MockBroker) Send(request *[]byte) ([]byte, error) {
	if m.SendFunc != nil {
		return m.SendFunc(request)
	}
	return nil, errors.New("SendFunc not implemented")
}

// SendContext calls the mock SendContextFunc.
func (m *MockBroker) SendContext(ctx context.Context, request *[]byte) ([]byte, error) {
	if m.SendContextFunc != nil {
		return m.SendContextFunc(ctx, request)
	}
	// Fallback to Send if SendContextFunc is not set.
	return m.Send(request)
}

// Close calls the mock CloseFunc.
func (m *MockBroker) Close() {
	if m.CloseFunc != nil {
		m.CloseFunc()
	}
}

// Start calls the mock StartFunc.
func (m *MockBroker) Start() error {
	if m.StartFunc != nil {
		return m.StartFunc()
	}
	return nil
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name       string
		config     *Config
		wantClient *Client
		wantErr    bool
	}{
		{
			name:   "nil_config",
			config: nil,
			wantClient: &Client{
				config: DefaultConfig(),
			},
			wantErr: false,
		},
		{
			name: "custom_config",
			config: &Config{
				Host:        "localhost",
				Port:        9000,
				PoolSize:    5,
				MaxWorker:   3,
				DialTimeout: 10 * time.Second,
				IdleTimeout: 2 * time.Minute,
			},
			wantClient: &Client{
				config: &Config{
					Host:        "localhost",
					Port:        9000,
					PoolSize:    5,
					MaxWorker:   3,
					DialTimeout: 10 * time.Second,
					IdleTimeout: 2 * time.Minute,
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotClient := NewClient(tt.config)
			if !reflect.DeepEqual(gotClient.config, tt.wantClient.config) {
				t.Errorf(
					"NewClient() gotClient.config = %v, want %v",
					gotClient.config,
					tt.wantClient.config,
				)
			}
			if gotClient.pool == nil {
				t.Errorf("NewClient() gotClient.pool is nil, want not nil")
			}
			if gotClient.broker == nil {
				t.Errorf("NewClient() gotClient.broker is nil, want not nil")
			}
			if !gotClient.isReady {
				t.Errorf("NewClient() gotClient.isReady is false, want true")
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	want := &Config{
		PoolSize:    5,
		MaxWorker:   2,
		DialTimeout: 5 * time.Second,
		IdleTimeout: 60 * time.Second,
	}
	got := DefaultConfig()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("DefaultConfig() = %v, want %v", got, want)
	}
}

func TestClient_SendCommand(t *testing.T) {
	defaultCfg := DefaultConfig()
	dummyFactory := func(addr string) (anet.PoolItem, error) {
		return &net.TCPConn{}, nil
	}
	dummyPool := anet.NewPool(
		1,
		dummyFactory,
		"localhost:1234",
		&anet.PoolConfig{DialTimeout: 1 * time.Second, IdleTimeout: 1 * time.Second},
	)

	tests := []struct {
		name        string
		client      *Client
		command     []byte
		mockBroker  anet.Broker
		wantResp    []byte
		wantErr     bool
		expectedErr string
	}{
		{
			name: "success",
			client: &Client{
				config: defaultCfg,
				broker: &MockBroker{
					SendFunc: func(request *[]byte) ([]byte, error) { return []byte("ND00"), nil },
				},
				pool:    dummyPool,
				isReady: true,
			},
			command:  []byte("NC"),
			wantResp: []byte("ND00"),
			wantErr:  false,
		},
		{
			name: "send_error",
			client: &Client{
				config: defaultCfg,
				broker: &MockBroker{
					SendFunc: func(request *[]byte) ([]byte, error) { return nil, errors.New("broker send failed") },
				},
				pool:    dummyPool,
				isReady: true,
			},
			command:     []byte("NC"),
			wantResp:    nil,
			wantErr:     true,
			expectedErr: "failed to send command: broker send failed",
		},
		{
			name: "client_not_ready",
			client: &Client{
				config:  defaultCfg,
				broker:  &MockBroker{},
				pool:    dummyPool,
				isReady: false,
			},
			command:     []byte("NC"),
			wantResp:    nil,
			wantErr:     true,
			expectedErr: "hsm client not ready",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotResp, err := tt.client.SendCommand(tt.command)
			if (err != nil) != tt.wantErr {
				t.Errorf("Client.SendCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && err.Error() != tt.expectedErr {
				t.Errorf(
					"Client.SendCommand() error = %q, expectedErr %q",
					err.Error(),
					tt.expectedErr,
				)
			}
			if !reflect.DeepEqual(gotResp, tt.wantResp) {
				t.Errorf("Client.SendCommand() gotResp = %v, want %v", gotResp, tt.wantResp)
			}
		})
	}
}

func TestClient_Close(t *testing.T) {
	defaultCfg := DefaultConfig()
	closedCalledOnPool := false
	dummyPool := &MockPool{
		CloseFunc: func() {
			closedCalledOnPool = true
		},
		GetFunc:            func() (anet.PoolItem, error) { return &net.TCPConn{}, nil },
		GetWithContextFunc: func(ctx context.Context) (anet.PoolItem, error) { return &net.TCPConn{}, nil },
		PutFunc:            func(anet.PoolItem) {},
		RemoveFunc:         func(anet.PoolItem) {},
		ReleaseFunc:        func(anet.PoolItem) {},
		IsEmptyFunc:        func() bool { return true },
		LenFunc:            func() int { return 0 },
		CapFunc:            func() int { return 1 },
		AddrFunc:           func() string { return "dummy" },
		IsClosedFunc:       func() bool { return false },
	}

	tests := []struct {
		name             string
		client           *Client
		wantErr          bool
		expectPoolClosed bool
	}{
		{
			name: "success_close_ready_client",
			client: &Client{
				config:  defaultCfg,
				broker:  &MockBroker{},
				pool:    dummyPool,
				isReady: true,
			},
			wantErr:          false,
			expectPoolClosed: true,
		},
		{
			name: "close_already_not_ready_client",
			client: &Client{
				config:  defaultCfg,
				broker:  &MockBroker{},
				pool:    dummyPool,
				isReady: false,
			},
			wantErr:          false,
			expectPoolClosed: true,
		},
		{
			name:             "nil_client_pool_and_broker",
			client:           &Client{config: defaultCfg, broker: nil, pool: nil, isReady: true},
			wantErr:          false,
			expectPoolClosed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			closedCalledOnPool = false

			tt.client.Close()

			if tt.client.isReady {
				t.Errorf("Client.Close() client.isReady = true, want false")
			}

			if tt.expectPoolClosed && (tt.client.pool == nil || !closedCalledOnPool) {
				if tt.client.pool == nil {
					t.Errorf("Client.Close() expected pool.Close() to be called, but pool was nil")
				} else {
					t.Errorf("Client.Close() expected pool.Close() to be called, but it wasn't")
				}
			}
			if !tt.expectPoolClosed && closedCalledOnPool {
				t.Errorf("Client.Close() did not expect pool.Close() to be called, but it was")
			}
		})
	}
}

type MockPool struct {
	CloseFunc          func()
	GetFunc            func() (anet.PoolItem, error)
	GetWithContextFunc func(ctx context.Context) (anet.PoolItem, error)
	PutFunc            func(anet.PoolItem)
	RemoveFunc         func(anet.PoolItem)
	ReleaseFunc        func(anet.PoolItem)
	IsEmptyFunc        func() bool
	LenFunc            func() int
	CapFunc            func() int
	AddrFunc           func() string
	IsClosedFunc       func() bool
}

func (mp *MockPool) Close()                      { mp.CloseFunc() }
func (mp *MockPool) Get() (anet.PoolItem, error) { return mp.GetFunc() }
func (mp *MockPool) GetWithContext(ctx context.Context) (anet.PoolItem, error) {
	return mp.GetWithContextFunc(ctx)
}
func (mp *MockPool) Put(item anet.PoolItem)     { mp.PutFunc(item) }
func (mp *MockPool) Remove(item anet.PoolItem)  { mp.RemoveFunc(item) }
func (mp *MockPool) Release(item anet.PoolItem) { mp.ReleaseFunc(item) }
func (mp *MockPool) IsEmpty() bool              { return mp.IsEmptyFunc() }
func (mp *MockPool) Len() int                   { return mp.LenFunc() }
func (mp *MockPool) Cap() int                   { return mp.CapFunc() }
func (mp *MockPool) Addr() string               { return mp.AddrFunc() }
func (mp *MockPool) IsClosed() bool             { return mp.IsClosedFunc() }

var (
	_ anet.Broker = &MockBroker{}
	_ anet.Pool   = &MockPool{}
)
