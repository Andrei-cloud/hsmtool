package tabs

import (
	"fmt"
	"strconv"
	"sync"
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
			return len(hs.responses), 3 // Columns: Time, Request, Response
		},
		func() fyne.CanvasObject {
			entry := widget.NewMultiLineEntry()
			entry.TextStyle = fyne.TextStyle{
				Bold: true,
			}
			entry.Wrapping = fyne.TextWrapBreak

			return entry
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			entry := o.(*widget.Entry)
			entry.Disable() // Make read-only

			hs.respMutex.Lock()
			defer hs.respMutex.Unlock()

			if i.Row >= len(hs.responses) {
				entry.SetText("")
				return
			}

			resp := hs.responses[i.Row]
			switch i.Col {
			case 0:
				entry.SetText(resp.Timestamp.Format("15:04:05.000"))
				entry.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
			case 1:
				entry.SetText(fmt.Sprintf("%s", resp.Request))
				entry.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
			case 2:
				entry.SetText(fmt.Sprintf("%s", resp.Response))
				entry.TextStyle = fyne.TextStyle{Bold: true, Monospace: true}
			}
		},
	)

	// Get main window width and calculate column widths.
	mainWindow := fyne.CurrentApp().Driver().AllWindows()[0]
	windowWidth := mainWindow.Canvas().Size().Width

	// Calculate proportional widths.
	timestampWidth := windowWidth / 4       // 25% for timestamp
	dataWidth := (windowWidth * 3 / 8) - 10 // 37.5% each for request and response, minus padding

	// Set column widths.
	hs.responseTable.SetColumnWidth(0, timestampWidth) // Timestamp
	hs.responseTable.SetColumnWidth(1, dataWidth)      // Request
	hs.responseTable.SetColumnWidth(2, dataWidth)      // Response
}

func (hs *HSMCommandSender) addResponse(req, resp string) {
	fyne.Do(func() {
		hs.respMutex.Lock()
		hs.responses = append(hs.responses, Response{
			Timestamp: time.Now(),
			Request:   req,
			Response:  resp,
		})
		hs.respMutex.Unlock()
		if hs.responseTable != nil {
			hs.responseTable.Refresh()
			if canvas := fyne.CurrentApp().Driver().AllWindows(); len(canvas) > 0 {
				canvas[0].Canvas().Refresh(hs.responseTable)
			}
		}
	})
}

func (hs *HSMCommandSender) onSend() {
	if hs.isSending {
		return
	}

	if hs.command.Text == "" {
		dialog.ShowError(
			fmt.Errorf("command cannot be empty"),
			fyne.CurrentApp().Driver().AllWindows()[0],
		)
		return
	}

	// Parse request count.
	reqCount, err := strconv.Atoi(hs.reqCount.Text)
	if err != nil {
		reqCount = 0 // Default to single request
	}
	if reqCount == 0 {
		reqCount = 1 // Always do at least one request
	}

	// Update UI state.
	hs.isSending = true
	hs.sendBtn.Disable()
	hs.stopBtn.Enable()
	hs.stopChan = make(chan struct{})
	hs.progress.SetValue(0)
	hs.progress.Max = float64(reqCount)

	// Send commands in background.
	go func() {
		defer func() {
			fyne.Do(func() {
				hs.isSending = false
				hs.sendBtn.Enable()
				hs.stopBtn.Disable()
				hs.progress.SetValue(hs.progress.Max)
			})
		}()

		completed := 0
		for i := 0; i < reqCount; i++ {
			select {
			case <-hs.stopChan:
				return
			default:
				// Send command to HSM.
				resp, err := hs.connection.ExecuteCommand([]byte(hs.command.Text))
				response := "Error: "
				if err != nil {
					response += err.Error()
				} else {
					response = string(resp)
				}

				// Update UI with response.
				hs.addResponse(hs.command.Text, response)
				completed++

				fyne.Do(func() {
					hs.progress.SetValue(float64(completed))
					hs.counter.SetText(fmt.Sprintf("Completed: %d", completed))
				})

				// Small delay between requests.
				if i < reqCount-1 {
					time.Sleep(100 * time.Millisecond)
				}
			}
		}
	}()
}

func (hs *HSMCommandSender) onStop() {
	if !hs.isSending {
		return
	}

	close(hs.stopChan)
	hs.isSending = false
	hs.sendBtn.Enable()
	hs.stopBtn.Disable()
}

// CreateRenderer implements fyne.Widget interface.
func (hs *HSMCommandSender) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(hs.container)
}

// Cleanup implements TabContent interface.
func (hs *HSMCommandSender) Cleanup() {
	// Stop any ongoing operations.
	if hs.isSending {
		hs.onStop()
	}

	// Clear sensitive data.
	hs.command.SetText("")
	hs.reqCount.SetText("")
	hs.duration.SetText("")
}
