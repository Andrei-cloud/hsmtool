// nolint:all // test package
package logger

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLevel_String(t *testing.T) {
	tests := []struct {
		name string
		l    Level
		want string
	}{
		{"debug", DEBUG, "DEBUG"},
		{"info", INFO, "INFO"},
		{"error", ERROR, "ERROR"},
		{"unknown", Level(99), "UNKNOWN"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.l.String(); got != tt.want {
				t.Errorf("Level.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewLogger(t *testing.T) {
	tempDir := t.TempDir()
	validLogPath := filepath.Join(tempDir, "app.log")
	invalidLogPath := filepath.Join(
		tempDir,
		"invalid_dir",
		"app.log",
	) // Directory does not exist initially.

	tests := []struct {
		name     string
		logPath  string
		level    Level
		callback func(Entry)
		wantErr  bool
	}{
		{
			name:    "valid_path_nil_callback",
			logPath: validLogPath,
			level:   INFO,
			wantErr: false,
		},
		{
			name: "valid_path_with_callback",
			logPath: filepath.Join(
				tempDir,
				"app_cb.log",
			), // Use a different file to avoid conflicts.
			level:    DEBUG,
			callback: func(e Entry) { /* dummy callback */ },
			wantErr:  false,
		},
		{
			name: "invalid_path_non_existent_parent_dir",
			// This test case is tricky because os.MkdirAll will create it.
			// To truly test a failure for os.OpenFile due to path, we might need to make the path unwriteable.
			// For now, we assume MkdirAll works, so OpenFile is the main point of failure if perms are wrong.
			// Let's simulate a failure by making the logPath a directory itself after MkdirAll.
			logPath: invalidLogPath, // This will be created by MkdirAll.
			level:   INFO,
			wantErr: false, // MkdirAll will create it.
		},
		{
			name:    "fail_to_open_file_if_path_is_dir",
			logPath: tempDir, // Using the tempDir itself as logPath, which is a directory.
			level:   INFO,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Special handling for the case where we want OpenFile to fail because the path is a directory.
			if tt.name == "fail_to_open_file_if_path_is_dir" {
				// MkdirAll on an existing dir is fine, but OpenFile with O_WRONLY on a dir will fail.
			} else if strings.Contains(tt.logPath, "invalid_dir") {
				// For this specific setup, ensure the parent doesn't exist before NewLogger tries to create it.
				// os.RemoveAll(filepath.Dir(tt.logPath)) // This is not needed as MkdirAll handles it.
			}

			l, err := NewLogger(tt.logPath, tt.level, tt.callback)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewLogger() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if l == nil {
					t.Error("NewLogger() returned nil logger for non-error case")
					return
				}
				if l.file == nil {
					t.Error("NewLogger() logger.file is nil")
				}
				if l.level != tt.level {
					t.Errorf("NewLogger() logger.level = %v, want %v", l.level, tt.level)
				}
				// Check callback presence (not its exact value, as funcs are hard to compare).
				if (l.callback == nil) != (tt.callback == nil) {
					t.Errorf("NewLogger() logger.callback presence mismatch")
				}
				l.Close() // Clean up the opened file.
			}
		})
	}
}

func TestLogger_Log(t *testing.T) {
	tempDir := t.TempDir()
	logFilePath := filepath.Join(tempDir, "test.log")

	tests := []struct {
		name       string
		logLevel   Level // Logger's configured level.
		entryLevel Level // Level of the message being logged.
		event      string
		status     string
		details    string
		wantLogged bool // Whether the message is expected to be written to file/callback.
		callback   func(Entry)
		checkFile  bool
		checkCb    bool
		cbPayload  *Entry // Expected payload for callback.
	}{
		{
			name:       "log_info_when_level_is_debug",
			logLevel:   DEBUG,
			entryLevel: INFO,
			event:      "TestEvent", status: "Success", details: "Info details",
			wantLogged: true,
			checkFile:  true,
		},
		{
			name:       "log_debug_when_level_is_info_not_logged",
			logLevel:   INFO,
			entryLevel: DEBUG,
			event:      "TestEventDebug", status: "Attempt", details: "Debug details",
			wantLogged: false,
			checkFile:  true,
		},
		{
			name:       "log_error_when_level_is_info",
			logLevel:   INFO,
			entryLevel: ERROR,
			event:      "TestError", status: "Failure", details: "Error details",
			wantLogged: true,
			checkFile:  true,
		},
		{
			name:       "log_with_callback",
			logLevel:   DEBUG,
			entryLevel: INFO,
			event:      "CallbackTest", status: "Triggered", details: "Callback details",
			wantLogged: true,
			checkCb:    true,
			cbPayload: &Entry{
				Level:   INFO,
				Event:   "CallbackTest",
				Status:  "Triggered",
				Details: "Callback details",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset log file for each test if checking file.
			if tt.checkFile {
				os.Remove(logFilePath) // Ensure clean file for reading.
			}

			var cbReceivedEntry Entry
			var cbCalled bool
			actualCallback := tt.callback
			if tt.checkCb {
				actualCallback = func(e Entry) {
					cbCalled = true
					cbReceivedEntry = e
				}
			}

			l, err := NewLogger(logFilePath, tt.logLevel, actualCallback)
			if err != nil {
				t.Fatalf("NewLogger() failed: %v", err)
			}
			defer l.Close()

			switch tt.entryLevel { // Use specific methods to also test them indirectly.
			case DEBUG:
				l.Debug(tt.event, tt.status, tt.details)
			case INFO:
				l.Info(tt.event, tt.status, tt.details)
			case ERROR:
				l.Error(tt.event, tt.status, tt.details)
			default:
				l.Log(tt.entryLevel, tt.event, tt.status, tt.details)
			}

			if tt.checkFile {
				file, err := os.Open(logFilePath)
				if err != nil {
					if tt.wantLogged {
						t.Fatalf("os.Open() for log file failed: %v", err)
					} else if !os.IsNotExist(err) { // If not expected and file doesn't exist, that's fine.
						t.Fatalf("os.Open() for log file failed unexpectedly: %v", err)
					}
					// If wantLogged is false and file doesn't exist, it's the correct outcome.
					if os.IsNotExist(err) && !tt.wantLogged {
						return // Test passed for this aspect.
					}
				}
				defer file.Close()

				scanner := bufio.NewScanner(file)
				foundLog := false
				for scanner.Scan() {
					var loggedEntry Entry
					if err := json.Unmarshal(scanner.Bytes(), &loggedEntry); err != nil {
						t.Errorf("json.Unmarshal() failed for log line: %v", err)
						continue
					}
					if loggedEntry.Event == tt.event && loggedEntry.Status == tt.status &&
						loggedEntry.Details == tt.details &&
						loggedEntry.Level == tt.entryLevel {
						foundLog = true
						break
					}
				}

				if foundLog != tt.wantLogged {
					t.Errorf(
						"Log entry found in file = %v, wantLogged = %v for event %s",
						foundLog,
						tt.wantLogged,
						tt.event,
					)
				}
				if !tt.wantLogged && foundLog {
					// If we didn't want it logged, but it was, that's an error.
					// If we didn't want it logged, and it wasn't, that's correct (covered by foundLog != tt.wantLogged).
				} else if tt.wantLogged && !foundLog {
					// If we wanted it logged, but it wasn't, that's an error.
					// If we wanted it logged, and it was, that's correct.
				}
			}

			if tt.checkCb {
				if cbCalled != tt.wantLogged {
					t.Errorf("Callback called = %v, wantLogged = %v", cbCalled, tt.wantLogged)
				}
				if cbCalled && tt.cbPayload != nil {
					// Compare relevant fields, ignoring timestamp.
					if cbReceivedEntry.Level != tt.cbPayload.Level ||
						cbReceivedEntry.Event != tt.cbPayload.Event ||
						cbReceivedEntry.Status != tt.cbPayload.Status ||
						cbReceivedEntry.Details != tt.cbPayload.Details {
						t.Errorf(
							"Callback received entry mismatch. Got %+v, want (approx) %+v",
							cbReceivedEntry,
							*tt.cbPayload,
						)
					}
				}
			}
		})
	}
}

func TestLogger_Close(t *testing.T) {
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "close.log")

	l, err := NewLogger(logPath, INFO, nil)
	if err != nil {
		t.Fatalf("NewLogger() failed: %v", err)
	}

	// First close.
	if err := l.Close(); err != nil {
		t.Errorf("Logger.Close() first call error = %v, want nil", err)
	}

	// Second close should ideally return an error (e.g., os.ErrClosed).
	if err := l.Close(); err == nil {
		t.Error(
			"Logger.Close() second call error = nil, want an error (e.g., os.ErrClosed or similar)",
		)
	} else {
		// Check if the error is the expected one for closing an already closed file.
		// This can be platform-dependent or depend on Go's os package internals.
		// A common error is *os.PathError with Op "close" and Err containing "file already closed".
		var pathErr *os.PathError
		if errors.As(err, &pathErr) {
			if pathErr.Op != "close" {
				t.Errorf("Logger.Close() second call PathError.Op = %q, want 'close'", pathErr.Op)
			}
			// The exact error message for Err can vary.
			// e.g. "file already closed" or "invalid argument"
			if !strings.Contains(pathErr.Err.Error(), "file already closed") && !strings.Contains(pathErr.Err.Error(), "invalid argument") {
				t.Logf("Logger.Close() second call PathError.Err = %q. This might be acceptable.", pathErr.Err.Error())
			}
		} else {
			t.Logf("Logger.Close() second call error = %v. Type: %T. This might be acceptable.", err, err)
		}
	}
}

func TestLogger_SetLevel(t *testing.T) {
	l := &Logger{mu: sync.Mutex{}, level: INFO} // Simplified logger for this test.

	tests := []struct {
		name     string
		newLevel Level
		want     Level
	}{
		{"set_to_debug", DEBUG, DEBUG},
		{"set_to_error", ERROR, ERROR},
		{"set_to_info", INFO, INFO},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l.SetLevel(tt.newLevel)
			l.mu.Lock() // Lock to safely read level, simulating internal access.
			got := l.level
			l.mu.Unlock()
			if got != tt.want {
				t.Errorf("Logger.level after SetLevel() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLogger_GetEntries(t *testing.T) {
	// Setup: Create a logger and log some entries.
	tempDir := t.TempDir()
	logPath := filepath.Join(tempDir, "get_entries.log")
	l, err := NewLogger(logPath, DEBUG, nil)
	if err != nil {
		t.Fatalf("NewLogger() failed: %v", err)
	}
	defer l.Close()

	// Log some entries - this part is more for future when GetEntries is implemented.
	// For now, we just test the "not implemented" error.
	l.Info("Event1", "Status1", "Details1")
	time.Sleep(10 * time.Millisecond) // Ensure distinct timestamps if implementation relies on it.
	l.Debug("Event2", "Status2", "Details2")
	time.Sleep(10 * time.Millisecond)
	l.Error("Event3", "Status3", "Details3")

	tests := []struct {
		name      string
		start     time.Time
		end       time.Time
		filter    string
		wantErr   bool
		expErrStr string
	}{
		{
			name:      "not_implemented_error",
			start:     time.Now().Add(-1 * time.Hour),
			end:       time.Now(),
			filter:    "",
			wantErr:   true,
			expErrStr: "not implemented",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := l.GetEntries(tt.start, tt.end, tt.filter)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetEntries() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.expErrStr {
				t.Errorf("GetEntries() error string = %q, want %q", err.Error(), tt.expErrStr)
			}
		})
	}
}
