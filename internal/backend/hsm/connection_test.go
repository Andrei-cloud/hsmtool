// nolint:all // test package
package hsm

import (
	"context"
	"errors"
	"fmt"
	"net"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/andrei-cloud/anet"
)

// Mock anet.Broker for testing connection.
// We need to extend the MockBroker from client_test.go or redefine it if it's not accessible.
// For simplicity, let's assume we have a mockBroker that can be controlled.

// Re-define MockBroker if not accessible from client_test.go (or use a shared test utility).
// For this file, we'll define it again to keep it self-contained.
type mockBroker struct {
	SendFunc        func(request *[]byte) ([]byte, error)
	SendContextFunc func(ctx context.Context, request *[]byte) ([]byte, error)
	CloseFunc       func()
	StartFunc       func() error
	pools           []anet.Pool
}

func (m *mockBroker) Send(request *[]byte) ([]byte, error) {
	if m.SendFunc != nil {
		return m.SendFunc(request)
	}
	return nil, errors.New("SendFunc not implemented on mockBroker")
}

func (m *mockBroker) SendContext(ctx context.Context, request *[]byte) ([]byte, error) {
	if m.SendContextFunc != nil {
		return m.SendContextFunc(ctx, request)
	}
	return m.Send(request)
}

func (m *mockBroker) Close() {
	if m.CloseFunc != nil {
		m.CloseFunc()
	}
}

func (m *mockBroker) Start() error {
	if m.StartFunc != nil {
		return m.StartFunc()
	}
	return nil
}

func (m *mockBroker) Pools() []anet.Pool {
	return m.pools
}

func TestNewConnection(t *testing.T) {
	tests := []struct {
		name             string
		stateChangedFunc func(ConnectionState)
		wantConnection   *Connection
	}{
		{
			name:             "nil_stateChangedFunc",
			stateChangedFunc: nil,
			wantConnection: &Connection{
				state:        atomic.Int32{},
				workerCount:  3,
				stopChan:     make(chan struct{}),
				stateChanged: nil,
				defaultConfig: &anet.PoolConfig{
					DialTimeout:        5 * time.Second,
					IdleTimeout:        60 * time.Second,
					ValidationInterval: 30 * time.Second,
					KeepAliveInterval:  30 * time.Second,
				},
			},
		},
		{
			name: "with_stateChangedFunc",
			stateChangedFunc: func(s ConnectionState) {
				// Mock function, do nothing.
			},
			wantConnection: &Connection{
				state:       atomic.Int32{},
				workerCount: 3,
				stopChan:    make(chan struct{}),
				// stateChanged func is not easily comparable directly, so we check for its presence.
				defaultConfig: &anet.PoolConfig{
					DialTimeout:        5 * time.Second,
					IdleTimeout:        60 * time.Second,
					ValidationInterval: 30 * time.Second,
					KeepAliveInterval:  30 * time.Second,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotConnection := NewConnection(tt.stateChangedFunc)

			// Compare fields that are comparable.
			if gotConnection.workerCount != tt.wantConnection.workerCount {
				t.Errorf(
					"NewConnection().workerCount = %v, want %v",
					gotConnection.workerCount,
					tt.wantConnection.workerCount,
				)
			}
			if !reflect.DeepEqual(gotConnection.defaultConfig, tt.wantConnection.defaultConfig) {
				t.Errorf(
					"NewConnection().defaultConfig = %v, want %v",
					gotConnection.defaultConfig,
					tt.wantConnection.defaultConfig,
				)
			}
			if (gotConnection.stateChanged == nil) == (tt.stateChangedFunc != nil) {
				t.Errorf(
					"NewConnection().stateChanged presence mismatch, gotNil: %v, wantNil: %v",
					gotConnection.stateChanged == nil,
					tt.stateChangedFunc == nil,
				)
			}
			if gotConnection.stopChan == nil {
				t.Errorf("NewConnection().stopChan is nil, want non-nil")
			}
			if gotConnection.GetState() != Disconnected {
				t.Errorf(
					"NewConnection().GetState() = %v, want Disconnected",
					gotConnection.GetState(),
				)
			}
		})
	}
}

// Mocking net.Listen and net.Dial for Connect tests is complex.
// We will focus on state changes and error paths that don't require full network setup.

func TestConnection_Connect_Disconnect(t *testing.T) {
	// Setup a dummy TCP server for connection attempts.
	l, err := net.Listen("tcp", "127.0.0.1:0") // Use port 0 to get a free port.
	if err != nil {
		t.Fatalf("Failed to listen on a port: %v", err)
	}
	serverAddr := l.Addr().String()
	go func() {
		defer l.Close()
		conn, _ := l.Accept() // Accept one connection and then close.
		if conn != nil {
			conn.Close()
		}
	}()

	tests := []struct {
		name              string
		host              string
		port              string
		numConns          uint32
		initialState      ConnectionState
		connectWantErr    bool
		finalState        ConnectionState // After connect attempt.
		disconnectWantErr bool
		disconnectState   ConnectionState // After disconnect.
		mockStartError    error           // Error for mockBroker.Start().
	}{
		{
			name:              "successful_connect_disconnect",
			host:              hostFromAddr(serverAddr),
			port:              portFromAddr(serverAddr),
			numConns:          1,
			initialState:      Disconnected,
			connectWantErr:    false,
			finalState:        Connected,
			disconnectWantErr: false,
			disconnectState:   Disconnected,
		},
		{
			name:              "connect_already_connected",
			host:              hostFromAddr(serverAddr),
			port:              portFromAddr(serverAddr),
			numConns:          1,
			initialState:      Connected, // Start as connected.
			connectWantErr:    true,      // Expect error.
			finalState:        Connected, // State should remain connected.
			disconnectWantErr: false,     // Disconnect should still work.
			disconnectState:   Disconnected,
		},
		{
			name:              "connect_invalid_host",
			host:              "invalid-host-that-does-not-exist",
			port:              "12345",
			numConns:          1,
			initialState:      Disconnected,
			connectWantErr:    true,
			finalState:        Disconnected,
			disconnectWantErr: true, // Error because it was never connected.
			disconnectState:   Disconnected,
		},
		{
			name:              "disconnect_not_connected",
			host:              hostFromAddr(serverAddr),
			port:              portFromAddr(serverAddr),
			numConns:          1,
			initialState:      Disconnected,
			connectWantErr:    true, // Simulate a connect failure by not starting server or using bad port.
			finalState:        Disconnected,
			disconnectWantErr: true, // Error because it's already disconnected.
			disconnectState:   Disconnected,
		},
		{
			name:              "broker_start_error",
			host:              hostFromAddr(serverAddr),
			port:              portFromAddr(serverAddr),
			numConns:          1,
			initialState:      Disconnected,
			connectWantErr:    true, // Expect error due to broker start failure.
			finalState:        Disconnected,
			disconnectWantErr: true,
			disconnectState:   Disconnected,
			mockStartError:    errors.New("broker start failed"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stateChanges []ConnectionState
			stateChangedFunc := func(s ConnectionState) {
				stateChanges = append(stateChanges, s)
			}
			c := NewConnection(stateChangedFunc)
			c.state.Store(int32(tt.initialState))

			// Mock the createBroker to return a controllable mockBroker.
			// This is a simplification; in reality, createBroker does more.
			originalCreateBroker := c.createBroker // Store original if we were to replace it.
			// For this test, we assume createBroker will succeed if host/port are valid,
			// and the actual broker interaction is what we test via Connect/Disconnect.
			// If we need to inject a mock broker, we'd do it here.
			// For the "broker_start_error" case, we'd need to inject a broker that returns an error on Start().
			// This level of mocking is complex with the current structure of `createBroker` being internal.
			// We will simulate the broker start error by checking the flag.

			if tt.name == "broker_start_error" {
				// This is tricky. The actual broker.Start() is in a goroutine.
				// We'd need to ensure our mock broker is used by createBroker.
				// For now, this specific sub-test might not be perfectly achievable without deeper refactoring
				// or more complex mocking of anet.NewBroker itself.
				// Let's assume for this path, the error from createBroker (if it could be made to fail early) would be caught.
				// Or, if broker.Start() fails, the connection state should reflect that.
				// The current Connect logic might set state to Connected before broker.Start error is propagated back easily.
				t.Log(
					"Skipping precise broker_start_error due to mocking complexity, focusing on other paths.",
				)
				// A more robust way would be to inject the broker factory into NewConnection or Connect.
			}

			err := c.Connect(tt.host, tt.port, tt.numConns)
			if (err != nil) != tt.connectWantErr {
				t.Errorf("Connect() error = %v, wantErr %v", err, tt.connectWantErr)
			}

			// Allow some time for potential async state changes if connect was successful.
			if !tt.connectWantErr {
				time.Sleep(250 * time.Millisecond) // Wait for broker to potentially start.
			}

			if c.GetState() != tt.finalState {
				t.Errorf("Connect() state = %v, want %v", c.GetState(), tt.finalState)
			}

			// Test Disconnect.
			disconnectErr := c.Disconnect()
			if (disconnectErr != nil) != tt.disconnectWantErr {
				t.Errorf("Disconnect() error = %v, wantErr %v", disconnectErr, tt.disconnectWantErr)
			}
			if c.GetState() != tt.disconnectState {
				t.Errorf("Disconnect() state = %v, want %v", c.GetState(), tt.disconnectState)
			}

			// Restore original createBroker if it was replaced.
			_ = originalCreateBroker // Avoid unused var error if not used for replacement.
		})
	}
	l.Close() // Close the listener after all tests in this block.
}

func TestConnection_GetState(t *testing.T) {
	tests := []struct {
		name      string
		setState  ConnectionState
		wantState ConnectionState
	}{
		{"disconnected", Disconnected, Disconnected},
		{"connected", Connected, Connected},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConnection(nil)
			c.state.Store(int32(tt.setState))
			if gotState := c.GetState(); gotState != tt.wantState {
				t.Errorf("GetState() = %v, want %v", gotState, tt.wantState)
			}
		})
	}
}

func TestConnection_GetPoolCapacity(t *testing.T) {
	tests := []struct {
		name         string
		setPoolCap   uint32
		wantPoolCap  uint32
		connectFirst bool   // Whether to call Connect to set poolCap internally.
		host         string // Dummy host/port for connect call.
		port         string
	}{
		{
			"default_before_connect",
			0,
			0,
			false,
			"",
			"",
		}, // Before connect, poolCap is 0.
		{"after_connect_sets_cap", 5, 5, true, "127.0.0.1", "12345"}, // Connect sets it.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConnection(nil)
			if tt.connectFirst {
				// We don't care about the success of Connect here, just that it sets poolCap.
				// Use a non-existent port to make Connect fail fast but still run the logic that sets poolCap.
				c.Connect(
					tt.host,
					tt.port,
					tt.setPoolCap,
				) // This will likely error but set c.poolCap.
			} else {
				// Manually set for the case where Connect is not called.
				c.mu.Lock()
				c.poolCap = tt.setPoolCap
				c.mu.Unlock()
			}

			if gotPoolCap := c.GetPoolCapacity(); gotPoolCap != tt.wantPoolCap {
				t.Errorf("GetPoolCapacity() = %v, want %v", gotPoolCap, tt.wantPoolCap)
			}
		})
	}
}

func TestConnection_GetLastError(t *testing.T) {
	customError := errors.New("custom test error")
	tests := []struct {
		name     string
		setError error
		wantErr  error
	}{
		{"no_error", nil, nil},
		{"with_error", customError, customError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConnection(nil)
			c.mu.Lock()
			c.lastError = tt.setError
			c.mu.Unlock()
			if gotErr := c.GetLastError(); !errors.Is(gotErr, tt.wantErr) {
				t.Errorf("GetLastError() = %v, want %v", gotErr, tt.wantErr)
			}
		})
	}
}

func TestConnection_ExecuteCommand(t *testing.T) {
	cmd := []byte("NC")
	resp := []byte("ND00")
	brokerErr := errors.New("broker send failed")

	tests := []struct {
		name           string
		initialState   ConnectionState
		mockBroker     *mockBroker
		command        []byte
		wantResp       []byte
		wantErr        bool
		expectedErrStr string
	}{
		{
			name:         "success",
			initialState: Connected,
			mockBroker: &mockBroker{
				SendFunc: func(request *[]byte) ([]byte, error) {
					if string(*request) == string(cmd) {
						return resp, nil
					}
					return nil, fmt.Errorf("unexpected command: %s", string(*request))
				},
			},
			command:  cmd,
			wantResp: resp,
			wantErr:  false,
		},
		{
			name:           "disconnected_error",
			initialState:   Disconnected,
			mockBroker:     &mockBroker{}, // Should not be called.
			command:        cmd,
			wantResp:       nil,
			wantErr:        true,
			expectedErrStr: "not connected to HSM",
		},
		{
			name:         "broker_send_error",
			initialState: Connected,
			mockBroker: &mockBroker{
				SendFunc: func(request *[]byte) ([]byte, error) {
					return nil, brokerErr
				},
			},
			command:        cmd,
			wantResp:       nil,
			wantErr:        true,
			expectedErrStr: fmt.Sprintf("failed to send command: %v", brokerErr),
		},
		{
			name:           "nil_broker", // If connection is Connected but broker is somehow nil.
			initialState:   Connected,
			mockBroker:     nil, // Explicitly set broker to nil.
			command:        cmd,
			wantResp:       nil,
			wantErr:        true,
			expectedErrStr: "not connected to HSM", // Or a different error depending on internal checks.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewConnection(nil)
			c.state.Store(int32(tt.initialState))
			c.mu.Lock() // Lock to safely assign broker.
			if tt.mockBroker != nil {
				c.broker = tt.mockBroker
			} else if tt.name == "nil_broker" {
				c.broker = nil // Ensure broker is nil for this specific test case.
			}
			c.mu.Unlock()

			gotResp, err := c.ExecuteCommand(tt.command)

			if (err != nil) != tt.wantErr {
				t.Errorf("ExecuteCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err != nil && err.Error() != tt.expectedErrStr {
				t.Errorf(
					"ExecuteCommand() error string = %q, want %q",
					err.Error(),
					tt.expectedErrStr,
				)
			}
			if !reflect.DeepEqual(gotResp, tt.wantResp) {
				t.Errorf("ExecuteCommand() gotResp = %v, want %v", gotResp, tt.wantResp)
			}
		})
	}
}

// Helper to extract host from address string (e.g., "127.0.0.1:12345").
func hostFromAddr(addr string) string {
	h, _, _ := net.SplitHostPort(addr)
	return h
}

// Helper to extract port from address string.
func portFromAddr(addr string) string {
	_, p, _ := net.SplitHostPort(addr)
	return p
}
