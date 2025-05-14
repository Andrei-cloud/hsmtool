package tabs

import (
	"errors" // Added for errors.New.
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/andrei-cloud/hsmtool/internal/backend/hsm"
)

// Response represents a single HSM request/response pair.
type Response struct {
	Timestamp time.Time
	Request   string
	Response  string
	Latency   time.Duration
}

// HSMCommandSender represents the HSM Command Sender tab.
type HSMCommandSender struct {
	widget.BaseWidget
	container *fyne.Container

	// Input fields.
	command  *widget.Entry
	reqCount *widget.Entry
	duration *widget.Entry

	// Status indicators.
	progress   *widget.ProgressBar
	counter    *widget.Label
	tpsLabel   *widget.Label
	responses  []Response
	respMutex  sync.Mutex
	connection *hsm.Connection

	// Response fields.
	commandResponseField *widget.Entry // Field for the latest command response.
	commandHistoryField  *widget.Entry // Field for the command history.

	// Control.
	sendBtn   *widget.Button
	stopBtn   *widget.Button
	isSending bool
	stopChan  chan struct{}
	sendMutex sync.Mutex
	started   sync.WaitGroup // Track if send operation is running

	// Logging flag.
	logHistory         bool // Flag to enable or disable command history logging.
	logHistoryCheckbox *widget.Check
}

// NewHSMCommandSender creates a new HSM Command Sender tab.
func NewHSMCommandSender(conn *hsm.Connection, logHistory bool) *HSMCommandSender {
	hs := &HSMCommandSender{
		connection: conn,
		responses:  make([]Response, 0),
		logHistory: logHistory, // Initialize the flag.
	}
	hs.ExtendBaseWidget(hs)

	// Initialize input fields.
	hs.command = widget.NewMultiLineEntry()
	hs.command.SetPlaceHolder("Enter command...")

	// Initialize request count spinner with up/down buttons.
	hs.reqCount = widget.NewEntry()
	hs.reqCount.SetText("0")
	hs.reqCount.OnChanged = func(s string) {
		// Validate numeric input.
		if s == "" {
			return
		}
		if num, err := strconv.Atoi(s); err != nil || num < 0 {
			hs.reqCount.SetText("0")
		}
	}

	// Create spinner buttons.
	spinUp := widget.NewButton("▲", func() {
		count, _ := strconv.Atoi(hs.reqCount.Text)
		hs.reqCount.SetText(fmt.Sprintf("%d", count+1))
	})
	spinDown := widget.NewButton("▼", func() {
		count, _ := strconv.Atoi(hs.reqCount.Text)
		if count > 0 {
			hs.reqCount.SetText(fmt.Sprintf("%d", count-1))
		}
	})
	spinUp.Resize(fyne.NewSize(30, 20))
	spinDown.Resize(fyne.NewSize(30, 20))

	reqCountContainer := container.NewBorder(nil, nil, nil,
		container.NewVBox(spinUp, spinDown),
		hs.reqCount,
	)

	// Initialize status indicators.
	hs.progress = widget.NewProgressBar()
	hs.counter = widget.NewLabel("Completed: 0")
	hs.tpsLabel = widget.NewLabel("")

	// Initialize response fields.
	hs.initializeCommandResponseUI()

	// Create control buttons.
	hs.sendBtn = widget.NewButton("Send", hs.onSend)
	hs.stopBtn = widget.NewButton("Stop", hs.onStop)
	hs.stopBtn.Disable()

	// Register for connection state changes
	if conn != nil {
		conn.RegisterStateCallback(func(state hsm.ConnectionState, _ error) {
			// Update UI based on connection state
			fyne.Do(func() {
				if state == hsm.Connected {
					hs.sendBtn.Enable()
					if hs.tpsLabel != nil {
						hs.tpsLabel.SetText("")
					}
				} else {
					if !hs.isSending {
						hs.sendBtn.Disable()
					}
					if hs.tpsLabel != nil {
						hs.tpsLabel.SetText("HSM disconnected - reconnecting...")
					}
				}
			})
		})
	}

	// Create form layout with bold section headers.
	form := container.NewVBox(
		widget.NewLabelWithStyle("Host Command", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		hs.command,
		container.NewPadded(
			widget.NewLabelWithStyle(
				"Request Count",
				fyne.TextAlignLeading,
				fyne.TextStyle{Bold: true},
			),
			reqCountContainer,
		),
	)

	// Create status layout with improved visual hierarchy.
	status := container.NewVBox(
		container.NewHBox(
			widget.NewLabelWithStyle(
				"Progress:",
				fyne.TextAlignLeading,
				fyne.TextStyle{Bold: true},
			),
			hs.progress,
		),
		hs.counter,
		hs.tpsLabel,
	)

	// Create buttons layout with padding.
	buttons := container.NewPadded(
		container.NewHBox(
			hs.sendBtn,
			hs.stopBtn,
		),
	)

	// Add a checkbox to toggle logging of command history.
	hs.logHistoryCheckbox = widget.NewCheck("Log Command History", func(checked bool) {
		hs.logHistory = checked
	})
	hs.logHistoryCheckbox.SetChecked(
		hs.logHistory,
	) // Set initial state based on the logHistory flag.

	// Layout everything in the container
	topContent := container.NewVBox(
		form,
		status,
		buttons,
		hs.logHistoryCheckbox, // Add the checkbox here.
		widget.NewSeparator(),
		hs.commandResponseField,
	)

	// Use Border layout to make the history window expand to the bottom.
	hs.container = container.NewBorder(
		topContent,             // top
		nil,                    // bottom
		nil,                    // left
		nil,                    // right
		hs.commandHistoryField, // center expands to fill space
	)

	return hs
}

func (hs *HSMCommandSender) initializeCommandResponseUI() {
	// Create a read-only text area for the latest command response.
	hs.commandResponseField = widget.NewMultiLineEntry()
	hs.commandResponseField.Disable() // Set to read-only.
	hs.commandResponseField.SetPlaceHolder("Latest command response will appear here.")

	// Create a read-only text area for the command history.
	hs.commandHistoryField = widget.NewMultiLineEntry()
	hs.commandHistoryField.Disable() // Set to read-only.
	hs.commandHistoryField.SetPlaceHolder("Command history will appear here.")
}

func (hs *HSMCommandSender) addResponse(req, resp string, latency time.Duration) {
	fyne.Do(func() {
		// Update the latest command response field.
		hs.commandResponseField.SetText(resp)

		if hs.logHistory {
			// Format the new history entry.
			newEntry := fmt.Sprintf(
				"[%s] Command: %s\n[%s] Response: %s\nLatency: %d ms\n\n",
				time.Now().Format("2006-01-02 15:04:05"), req,
				time.Now().Format("2006-01-02 15:04:05"), resp,
				latency.Milliseconds(),
			)

			// Append the new entry to the command history.
			currentHistory := hs.commandHistoryField.Text
			hs.commandHistoryField.SetText(currentHistory + newEntry)

			// Scroll to the bottom of the command history field.
			hs.commandHistoryField.CursorRow = len(hs.commandHistoryField.Text)
		}
	})
}

func (hs *HSMCommandSender) onSend() {
	hs.sendMutex.Lock()
	if hs.isSending {
		hs.sendMutex.Unlock()
		return
	}

	// Check connection status before attempting to send
	connState := hs.connection.GetState()
	if connState != hsm.Connected {
		hs.sendMutex.Unlock()
		dialog.ShowError(
			errors.New("hsm not connected - please wait for reconnection to complete"),
			fyne.CurrentApp().Driver().AllWindows()[0],
		)

		return
	}

	lastError := hs.connection.GetLastError()
	if lastError != nil {
		// Show the error but allow the user to try sending anyway
		dialog.ShowInformation(
			"Connection Warning",
			fmt.Sprintf(
				"HSM connection reported an error but is still marked as connected: %v\nAttempting to send anyway.",
				lastError,
			),
			fyne.CurrentApp().Driver().AllWindows()[0],
		)
	}

	if hs.command.Text == "" {
		hs.sendMutex.Unlock()
		dialog.ShowError(
			errors.New("command cannot be empty"),
			fyne.CurrentApp().Driver().AllWindows()[0],
		)

		return
	}

	// Parse request count
	reqCount, err := strconv.Atoi(hs.reqCount.Text)
	if err != nil || reqCount < 0 {
		reqCount = 0
	}
	if reqCount == 0 {
		reqCount = 1
		hs.reqCount.SetText("1")
	}

	// Reset state for new command
	hs.stopChan = make(chan struct{}) // Create new channel for this send operation
	hs.progress.SetValue(0)
	hs.progress.Max = float64(reqCount)
	hs.isSending = true
	hs.sendBtn.Disable()
	hs.stopBtn.Enable()

	if hs.tpsLabel != nil {
		if reqCount <= 10 {
			hs.tpsLabel.SetText("")
		} else {
			hs.tpsLabel.SetText("TPS: calculating...")
		}
	}

	poolCapacity := hs.connection.GetPoolCapacity()
	hs.sendMutex.Unlock() // Unlock before starting goroutine

	if !hs.logHistory {
		// Performance mode: send commands concurrently.
		go hs.sendConcurrent(reqCount, int(poolCapacity))
	} else {
		// Default mode: send commands sequentially.
		go hs.sendSequential(reqCount)
	}
}

func (hs *HSMCommandSender) sendSequential(reqCount int) {
	var batchStartTime time.Time
	if reqCount > 10 {
		batchStartTime = time.Now()
	}
	var completed int32

	defer func() {
		fyne.Do(func() {
			hs.sendMutex.Lock()
			hs.isSending = false
			hs.sendBtn.Enable()
			hs.stopBtn.Disable()
			hs.progress.SetValue(float64(completed))
			if hs.tpsLabel != nil {
				if reqCount <= 10 || int(completed) != reqCount {
					hs.tpsLabel.SetText("")
				}
			}
			hs.sendMutex.Unlock()
		})
	}()

	for i := 0; i < reqCount; i++ {
		select {
		case <-hs.stopChan:
			return // Exit the loop immediately if stop is signaled.
		default:
			// Check connection state before each send
			if hs.connection.GetState() != hsm.Connected {
				fyne.Do(func() {
					if hs.tpsLabel != nil {
						hs.tpsLabel.SetText("HSM disconnected - reconnecting...")
					}
					dialog.ShowError(
						errors.New("hsm connection lost during command sequence"),
						fyne.CurrentApp().Driver().AllWindows()[0],
					)
				})

				return
			}

			startTime := time.Now()
			respText, err := hs.connection.ExecuteCommand([]byte(hs.command.Text), 5*time.Second)
			latency := time.Since(startTime)

			var response string
			switch {
			case err != nil:
				response = "Error: " + err.Error()
				// If this is a connection/broker error, stop the sequence
				if err.Error() == "hsm client not connected" ||
					err.Error() == "broker is closed" ||
					err.Error() == "command timed out" {
					fyne.Do(func() {
						if hs.tpsLabel != nil {
							hs.tpsLabel.SetText("HSM disconnected - reconnecting...")
						}
					})
					hs.addResponse(hs.command.Text, response, latency)

					return
				}
			case respText != nil:
				response = string(respText)
			default:
				response = "No response"
			}

			hs.addResponse(hs.command.Text, response, latency)
			completed++

			fyne.Do(func() {
				hs.progress.SetValue(float64(completed))
				hs.counter.SetText(fmt.Sprintf("Completed: %d", completed))
				if hs.tpsLabel != nil && reqCount > 10 {
					elapsedTime := time.Since(batchStartTime)
					if elapsedTime.Seconds() > 0 {
						tps := float64(completed) / elapsedTime.Seconds()
						hs.tpsLabel.SetText(fmt.Sprintf("TPS: %.2f", tps))
					}
				}
			})

			// Add a small delay between commands to prevent overwhelming the connection
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// sendConcurrent sends commands using multiple goroutines.
func (hs *HSMCommandSender) sendConcurrent(reqCount, numWorkers int) {
	var batchStartTime time.Time
	if reqCount > 10 {
		batchStartTime = time.Now()
	}
	var completedCount atomic.Int32
	var wg sync.WaitGroup
	var stopSending atomic.Bool

	jobs := make(chan int, reqCount)
	for i := 0; i < reqCount; i++ {
		jobs <- i
	}
	close(jobs)

	defer func() {
		wg.Wait() // Wait for all workers to finish
		finalCompleted := completedCount.Load()

		fyne.Do(func() {
			hs.sendMutex.Lock()
			hs.isSending = false
			hs.sendBtn.Enable()
			hs.stopBtn.Disable()
			hs.progress.SetValue(float64(finalCompleted))

			if hs.tpsLabel != nil {
				if reqCount <= 10 || int(finalCompleted) != reqCount {
					hs.tpsLabel.SetText("")
				}
			}
			hs.sendMutex.Unlock()
		})
	}()

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				// Stop processing if signaled by another worker
				if stopSending.Load() {
					return
				}

				// Check for external stop signal
				select {
				case <-hs.stopChan:
					stopSending.Store(true)
					return // Exit the worker loop immediately if stop is signaled.
				default:
					// Check connection state before each send
					if hs.connection.GetState() != hsm.Connected {
						stopSending.Store(true)
						fyne.Do(func() {
							if hs.tpsLabel != nil {
								hs.tpsLabel.SetText("HSM disconnected - reconnecting...")
							}
							dialog.ShowError(
								errors.New("hsm connection lost during command sequence"),
								fyne.CurrentApp().Driver().AllWindows()[0],
							)
						})

						return
					}

					startTime := time.Now()
					cmdText := hs.command.Text
					respText, err := hs.connection.ExecuteCommand([]byte(cmdText), 5*time.Second)
					latency := time.Since(startTime)
					response := ""
					switch {
					case err != nil:
						response = "Error: " + err.Error()
						// If this is a connection/broker error, stop the sequence
						if err.Error() == "hsm client not connected" ||
							err.Error() == "broker is closed" ||
							err.Error() == "command timed out" {
							stopSending.Store(true)
							fyne.Do(func() {
								if hs.tpsLabel != nil {
									hs.tpsLabel.SetText("HSM disconnected - reconnecting...")
								}
							})
							hs.addResponse(cmdText, response, latency)

							return
						}
					case respText != nil:
						response = string(respText)
					default:
						response = "No response"
					}

					// Record response and update UI
					hs.addResponse(cmdText, response, latency)
					newCount := completedCount.Add(1)

					// Update progress and TPS if needed
					fyne.Do(func() {
						hs.progress.SetValue(float64(newCount))
						hs.counter.SetText(fmt.Sprintf("Completed: %d", newCount))
						if hs.tpsLabel != nil && reqCount > 10 {
							elapsedTime := time.Since(batchStartTime)
							if elapsedTime.Seconds() > 0 {
								tps := float64(newCount) / elapsedTime.Seconds()
								hs.tpsLabel.SetText(fmt.Sprintf("TPS: %.2f", tps))
							}
						}
					})
				}
			}
		}()
	}
}

func (hs *HSMCommandSender) onStop() {
	hs.sendMutex.Lock()
	defer hs.sendMutex.Unlock()

	if !hs.isSending {
		return
	}

	if hs.stopChan != nil {
		close(hs.stopChan)
		// do not nil the channel so sequential send can detect closure
	}

	// Reset the state immediately for UI responsiveness.
	hs.isSending = false
	hs.sendBtn.Enable()
	hs.stopBtn.Disable()
	if hs.tpsLabel != nil {
		hs.tpsLabel.SetText("")
	}
}

func (hs *HSMCommandSender) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(hs.container)
}

func (hs *HSMCommandSender) Cleanup() {
	hs.sendMutex.Lock()
	defer hs.sendMutex.Unlock()

	if hs.isSending && hs.stopChan != nil {
		close(hs.stopChan)
		// do not nil the channel to allow proper channel semantics
		hs.isSending = false // Ensure state is reset.
	}

	// Reset all UI elements
	hs.command.SetText("")
	hs.reqCount.SetText("0")
	if hs.duration != nil {
		hs.duration.SetText("")
	}
	if hs.tpsLabel != nil {
		hs.tpsLabel.SetText("")
	}
	if hs.counter != nil {
		hs.counter.SetText("Completed: 0")
	}
	if hs.progress != nil {
		hs.progress.SetValue(0)
	}
	if hs.commandResponseField != nil {
		hs.commandResponseField.SetText("")
	}
	if hs.commandHistoryField != nil {
		hs.commandHistoryField.SetText("")
	}

	// Reset control elements
	if hs.sendBtn != nil {
		hs.sendBtn.Enable()
	}
	if hs.stopBtn != nil {
		hs.stopBtn.Disable()
	}
}
