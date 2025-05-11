package tabs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// HSMCommandSender represents the HSM Command Sender tab.
type HSMCommandSender struct {
	widget.BaseWidget
	container *fyne.Container

	// Input fields.
	command  *widget.Entry
	reqCount *widget.Entry
	duration *widget.Entry

	// Status indicators.
	progress *widget.ProgressBar
	counter  *widget.Label

	// Response table.
	responseTable *widget.Table

	// Control.
	sendBtn   *widget.Button
	stopBtn   *widget.Button
	isSending bool
}

// NewHSMCommandSender creates a new HSM Command Sender tab.
func NewHSMCommandSender() *HSMCommandSender {
	hs := &HSMCommandSender{}
	hs.ExtendBaseWidget(hs)

	// Initialize input fields.
	hs.command = widget.NewMultiLineEntry()
	hs.command.SetPlaceHolder("Enter hex command...")

	hs.reqCount = widget.NewEntry()
	hs.reqCount.SetPlaceHolder("Number of requests...")

	hs.duration = widget.NewEntry()
	hs.duration.SetPlaceHolder("Duration in ms (optional)...")

	// Initialize status indicators.
	hs.progress = widget.NewProgressBar()
	hs.counter = widget.NewLabel("Completed: 0")

	// Initialize response table.
	hs.initializeTable()

	// Create control buttons.
	hs.sendBtn = widget.NewButton("Send", hs.onSend)
	hs.stopBtn = widget.NewButton("Stop", hs.onStop)
	hs.stopBtn.Disable()

	// Create form layout.
	form := widget.NewForm(
		&widget.FormItem{Text: "Host Command (Hex)", Widget: hs.command},
		&widget.FormItem{Text: "Request Count", Widget: hs.reqCount},
		&widget.FormItem{Text: "Duration (ms)", Widget: hs.duration},
	)

	// Create status layout.
	status := container.NewVBox(
		container.NewHBox(
			widget.NewLabel("Progress:"),
			hs.progress,
		),
		hs.counter,
	)

	// Create buttons layout.
	buttons := container.NewHBox(
		hs.sendBtn,
		hs.stopBtn,
	)

	// Layout everything in the container.
	hs.container = container.NewVBox(
		form,
		status,
		buttons,
		widget.NewSeparator(),
		widget.NewLabel("Responses"),
		hs.responseTable,
	)

	return hs
}

func (hs *HSMCommandSender) initializeTable() {
	hs.responseTable = widget.NewTable(
		func() (int, int) { return 0, 3 }, // Columns: Timestamp, Request Header, Response
		func() fyne.CanvasObject {
			return widget.NewLabel("Template")
		},
		func(i widget.TableCellID, o fyne.CanvasObject) {
			// Will populate response data here.
		},
	)
}

func (hs *HSMCommandSender) onSend() {
	if hs.isSending {
		return
	}

	// TODO: Implement command sending logic.
	hs.isSending = true
	hs.sendBtn.Disable()
	hs.stopBtn.Enable()
}

func (hs *HSMCommandSender) onStop() {
	if !hs.isSending {
		return
	}

	// TODO: Implement stop logic.
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
