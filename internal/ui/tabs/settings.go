package tabs

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/andrei-cloud/hsmtool/internal/backend/hsm"
)

// LMKPairIndices available for encryption.
var LMKPairIndices = []string{"00", "01", "02", "03", "04"}

// Settings represents the Settings tab.
type Settings struct {
	widget.BaseWidget
	container *fyne.Container

	// Connection settings.
	hsmIP       *widget.Entry
	hsmPort     *widget.Entry
	lmkIndex    *widget.Select
	statusLED   *canvas.Circle
	statusText  *canvas.Text
	connection  *hsm.Connection
	connectBtn  *widget.Button
	currentConn bool
}

// NewSettings creates a new Settings tab.
func NewSettings() *Settings {
	s := &Settings{}
	s.ExtendBaseWidget(s)

	// Initialize HSM connection manager
	s.connection = hsm.NewConnection(s.onConnectionStateChanged)
	s.currentConn = false

	// Initialize connection fields.
	s.hsmIP = widget.NewEntry()
	s.hsmIP.SetPlaceHolder("Enter HSM IP/hostname...")

	s.hsmPort = widget.NewEntry()
	s.hsmPort.SetPlaceHolder("Enter port number...")
	s.hsmPort.Text = "1500" // Default HSM port
	s.hsmPort.OnChanged = func(text string) {
		// Validate port number.
		if text != "" {
			if _, err := strconv.Atoi(text); err != nil {
				s.hsmPort.SetText(text[:len(text)-1])
			}
		}
	}

	s.lmkIndex = widget.NewSelect(LMKPairIndices, nil)
	s.lmkIndex.SetSelected("00") // Default LMK pair

	// Status indicators
	s.statusLED = canvas.NewCircle(theme.ErrorColor())
	s.statusLED.Resize(fyne.NewSize(20, 20))
	s.statusLED.StrokeWidth = 2
	s.statusLED.StrokeColor = theme.ShadowColor()

	s.statusText = canvas.NewText("Disconnected", theme.ErrorColor())
	s.statusText.TextStyle = fyne.TextStyle{Bold: true}
	s.statusText.TextSize = theme.TextSize() * 1.2

	// Connection button
	s.connectBtn = widget.NewButton("Connect", s.onConnectClick)

	// Layout forms
	connForm := widget.NewForm(
		&widget.FormItem{Text: "HSM IP/Hostname", Widget: s.hsmIP},
		&widget.FormItem{Text: "Port", Widget: s.hsmPort},
		&widget.FormItem{Text: "LMK Pair Index", Widget: s.lmkIndex},
	)

	// Create status bar with some padding around the status text
	statusBar := container.NewHBox(
		layout.NewSpacer(),
		s.statusLED,
		container.NewPadded(s.statusText),
		s.connectBtn,
	)

	// Create container
	hsmConn := widget.NewCard("HSM Connection", "", container.NewVBox(
		connForm,
		statusBar,
	))

	s.container = container.NewVBox(
		hsmConn,
	)

	return s
}

func (s *Settings) onConnectionStateChanged(state hsm.ConnectionState) {
	// Update UI on the main thread
	fyne.Do(func() {
		if state == hsm.Connected {
			s.statusLED.FillColor = theme.SuccessColor()
			s.statusLED.StrokeColor = theme.SuccessColor()
			s.statusText.Text = "Connected"
			s.statusText.Color = theme.SuccessColor()
			s.connectBtn.SetText("Disconnect")
			s.currentConn = true
			// Disable input fields when connected
			s.hsmIP.Disable()
			s.hsmPort.Disable()
			s.lmkIndex.Disable()
		} else {
			s.statusLED.FillColor = theme.ErrorColor()
			s.statusLED.StrokeColor = theme.ErrorColor()
			s.statusText.Text = "Disconnected"
			s.statusText.Color = theme.ErrorColor()
			s.connectBtn.SetText("Connect")
			s.currentConn = false
			// Re-enable input fields when disconnected
			s.hsmIP.Enable()
			s.hsmPort.Enable()
			s.lmkIndex.Enable()
		}
		s.statusLED.Refresh()
		s.statusText.Refresh()
		s.connectBtn.Refresh()
	})
}

func (s *Settings) onConnectClick() {
	if !s.currentConn {
		// Disable button while connecting - this is on UI thread already
		s.connectBtn.Disable()
		s.connectBtn.SetText("Connecting...")

		// Connect in a goroutine to avoid blocking UI
		go func() {
			err := s.connection.Connect(s.hsmIP.Text, s.hsmPort.Text)

			// Update UI on the main thread
			fyne.Do(func() {
				s.connectBtn.Enable()
				if err != nil {
					dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
					s.onConnectionStateChanged(hsm.Disconnected)
				}
			})
		}()
	} else {
		// Disable button while disconnecting - this is on UI thread already
		s.connectBtn.Disable()
		s.connectBtn.SetText("Disconnecting...")

		// Disconnect in a goroutine
		go func() {
			err := s.connection.Disconnect()

			// Update UI on the main thread
			fyne.Do(func() {
				s.connectBtn.Enable()
				if err != nil {
					dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])
				}
			})
		}()
	}
}

func (s *Settings) onTestConnection() {
	// TODO: Implement HSM connection test.
	// For now, just show a placeholder dialog.
	dialog.ShowInformation("Connection Test",
		"Testing connection to HSM...",
		fyne.CurrentApp().Driver().AllWindows()[0])
}

// GetConnection returns the HSM connection instance.
func (s *Settings) GetConnection() *hsm.Connection {
	return s.connection
}

// CreateRenderer implements fyne.Widget interface.
func (s *Settings) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(s.container)
}

// Cleanup implements TabContent interface.
func (s *Settings) Cleanup() {
	if s.currentConn {
		s.connection.Disconnect()
	}
	s.hsmIP.SetText("")
	s.hsmPort.SetText("1500")
	s.lmkIndex.SetSelected("00")
}
