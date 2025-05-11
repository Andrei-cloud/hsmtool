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

	// Response table.
	responseScroll *container.Scroll
	responseTable  *widget.Table
	responseLabel  *widget.Label

	// Control.
	sendBtn   *widget.Button
	stopBtn   *widget.Button
	isSending bool
	stopChan  chan struct{}
}

// NewHSMCommandSender creates a new HSM Command Sender tab.
func NewHSMCommandSender(conn *hsm.Connection) *HSMCommandSender {
	hs := &HSMCommandSender{
		connection: conn,
		responses:  make([]Response, 0),
		stopChan:   make(chan struct{}),
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

	// Initialize response table.
	hs.initializeTable()

	// Create control buttons.
	hs.sendBtn = widget.NewButton("Send", hs.onSend)
	hs.stopBtn = widget.NewButton("Stop", hs.onStop)
	hs.stopBtn.Disable()

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

	// Create responses section with emphasized header.
	hs.responseLabel = widget.NewLabelWithStyle(
		"Command History",
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)
	hs.responseScroll = container.NewScroll(hs.responseTable)

	// Layout everything in the container
	topContent := container.NewVBox(
		form,
		status,
		buttons,
		widget.NewSeparator(),
		hs.responseLabel,
	)

	// Use Border layout to make response section expand
	hs.container = container.NewBorder(
		topContent,                             // top
		nil,                                    // bottom
		nil,                                    // left
		nil,                                    // right
		container.NewPadded(hs.responseScroll), // center expands to fill space
	)

	return hs
}

func (hs *HSMCommandSender) initializeTable() {
	hs.responseTable = widget.NewTable(
		func() (int, int) {
			hs.respMutex.Lock()
			defer hs.respMutex.Unlock()
			return len(hs.responses), 4 // Columns: Time, Request, Response, Latency.
		},
		func() fyne.CanvasObject {
			return container.NewMax(widget.NewLabel(""))
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			// Ensure o is a *fyne.Container and its first object is a *widget.Label.
			ctr, ok := o.(*fyne.Container)
			if !ok || len(ctr.Objects) == 0 {
				return
			}
			label, ok := ctr.Objects[0].(*widget.Label)
			if !ok {
				return
			}

			hs.respMutex.Lock()
			defer hs.respMutex.Unlock()

			if i.Row >= len(hs.responses) {
				label.SetText("")
				return
			}

			resp := hs.responses[i.Row]
			label.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
			label.Wrapping = fyne.TextWrapWord
			label.Alignment = fyne.TextAlignLeading

			switch i.Col {
			case 0:
				label.SetText(resp.Timestamp.Format("15:04:05.000"))
			case 1:
				label.SetText(resp.Request)
			case 2:
				label.SetText(resp.Response)
			case 3:
				label.SetText(resp.Latency.String())
			}

			label.Refresh()
		},
	)

	hs.responseTable.SetColumnWidth(0, 130) // Timestamp.
	hs.responseTable.SetColumnWidth(1, 258) // Request.
	hs.responseTable.SetColumnWidth(2, 258) // Response.
	hs.responseTable.SetColumnWidth(3, 120) // Latency.
}

func (hs *HSMCommandSender) addResponse(req, resp string, latency time.Duration) {
	fyne.Do(func() {
		hs.respMutex.Lock()
		hs.responses = append(hs.responses, Response{
			Timestamp: time.Now(),
			Request:   req,
			Response:  resp,
			Latency:   latency,
		})
		hs.respMutex.Unlock()
		if hs.responseTable != nil {
			hs.responseTable.Refresh()
		}
	})
}

func (hs *HSMCommandSender) onSend() {
	if hs.isSending {
		return
	}

	if hs.connection.GetState() != hsm.Connected {
		dialog.ShowError(
			errors.New("hsm not connected"), // Changed to errors.New.
			fyne.CurrentApp().Driver().AllWindows()[0],
		)

		return
	}

	if hs.command.Text == "" {
		dialog.ShowError(
			errors.New("command cannot be empty"), // Changed to errors.New.
			fyne.CurrentApp().Driver().AllWindows()[0],
		)

		return
	}

	// Parse request count.
	reqCount, err := strconv.Atoi(hs.reqCount.Text)
	if err != nil || reqCount < 0 {
		reqCount = 0
	}
	if reqCount == 0 {
		reqCount = 1
		hs.reqCount.SetText("1")
	}

	hs.isSending = true
	hs.sendBtn.Disable()
	hs.stopBtn.Enable()
	hs.stopChan = make(chan struct{})
	hs.progress.SetValue(0)
	hs.progress.Max = float64(reqCount)

	if hs.tpsLabel != nil {
		if reqCount <= 10 {
			hs.tpsLabel.SetText("")
		} else {
			hs.tpsLabel.SetText("TPS: calculating...")
		}
	}

	poolCapacity := hs.connection.GetPoolCapacity()

	if reqCount > 1 && poolCapacity > 1 {
		numWorkers := int(poolCapacity)
		if reqCount < numWorkers {
			numWorkers = reqCount
		}
		go hs.sendConcurrent(reqCount, numWorkers)
	} else {
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
		isLoopCompleted := int(completed) == reqCount
		fyne.Do(func() {
			hs.isSending = false
			hs.sendBtn.Enable()
			hs.stopBtn.Disable()
			hs.progress.SetValue(float64(completed))

			if hs.tpsLabel != nil {
				if !isLoopCompleted || reqCount <= 10 {
					hs.tpsLabel.SetText("")
				}
			}
		})
	}()

	for i := 0; i < reqCount; i++ {
		select {
		case <-hs.stopChan:
			fyne.Do(func() {
				if hs.tpsLabel != nil {
					hs.tpsLabel.SetText("")
				}
			})

			return
		default:
			startTime := time.Now()
			respText, err := hs.connection.ExecuteCommand([]byte(hs.command.Text))
			latency := time.Since(startTime)
			response := ""
			if err != nil {
				response = "Error: " + err.Error()
			} else {
				response = string(respText)
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
					} else if completed == 0 {
						hs.tpsLabel.SetText("TPS: calculating...")
					}
				}
			})
		}
	}
}

// sendConcurrent sends commands using multiple goroutines.
func (hs *HSMCommandSender) sendConcurrent(reqCount, numWorkers int) { // Grouped parameters.
	var batchStartTime time.Time
	if reqCount > 10 {
		batchStartTime = time.Now()
	}
	var completedCount atomic.Int32
	var wg sync.WaitGroup

	jobs := make(chan int, reqCount)
	for i := 0; i < reqCount; i++ {
		jobs <- i
	}
	close(jobs)

	defer func() {
		finalCompleted := completedCount.Load()
		isLoopCompleted := int(finalCompleted) == reqCount
		fyne.Do(func() {
			hs.isSending = false
			hs.sendBtn.Enable()
			hs.stopBtn.Disable()
			hs.progress.SetValue(float64(finalCompleted))

			if hs.tpsLabel != nil {
				if !isLoopCompleted || reqCount <= 10 {
					hs.tpsLabel.SetText("")
				}
			}
		})
	}()

	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range jobs {
				select {
				case <-hs.stopChan:

					return
				default:
					// Check stopChan again before doing work, in case jobs were queued before stop.
					select {
					case <-hs.stopChan:

						return
					default:
					}

					startTime := time.Now()
					cmdText := hs.command.Text
					respText, err := hs.connection.ExecuteCommand([]byte(cmdText))
					latency := time.Since(startTime)
					response := ""
					if err != nil {
						response = "Error: " + err.Error()
					} else {
						response = string(respText)
					}

					hs.addResponse(cmdText, response, latency)
					currentCompleted := completedCount.Add(1)

					fyne.Do(func() {
						hs.progress.SetValue(float64(currentCompleted))
						hs.counter.SetText(fmt.Sprintf("Completed: %d", currentCompleted))
						if hs.tpsLabel != nil && reqCount > 10 {
							elapsedTime := time.Since(batchStartTime)
							if elapsedTime.Seconds() > 0 {
								tps := float64(currentCompleted) / elapsedTime.Seconds()
								hs.tpsLabel.SetText(fmt.Sprintf("TPS: %.2f", tps))
							} else if currentCompleted == 0 {
								hs.tpsLabel.SetText("TPS: calculating...")
							}
						}
					})
				}
			}
		}()
	}

	go func() {
		wg.Wait()
		select {
		case <-hs.stopChan:
			fyne.Do(func() {
				if hs.tpsLabel != nil {
					if int(completedCount.Load()) != reqCount {
						hs.tpsLabel.SetText("")
					}
				}
			})
		default:
		}
	}()
}

func (hs *HSMCommandSender) onStop() {
	if !hs.isSending {
		return
	}

	close(hs.stopChan)
}

func (hs *HSMCommandSender) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(hs.container)
}

func (hs *HSMCommandSender) Cleanup() {
	if hs.isSending {
		hs.onStop()
	}

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
}
