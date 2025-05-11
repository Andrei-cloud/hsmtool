package tabs

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

var (
	// LogLevels available in the application.
	LogLevels = []string{"Info", "Debug", "Error"}

	// LMKPairIndices available for encryption.
	LMKPairIndices = []string{"00", "01", "02", "03", "04"}
)

// Settings represents the Settings tab.
type Settings struct {
	widget.BaseWidget
	container *fyne.Container

	// Connection settings.
	hsmIP    *widget.Entry
	hsmPort  *widget.Entry
	lmkIndex *widget.Select

	// Other settings.
	pluginPath  *widget.Entry
	logLevel    *widget.Select
	themeToggle *widget.Check
}

// NewSettings creates a new Settings tab.
func NewSettings() *Settings {
	s := &Settings{}
	s.ExtendBaseWidget(s)

	// Initialize connection fields.
	s.hsmIP = widget.NewEntry()
	s.hsmIP.SetPlaceHolder("Enter HSM IP address...")

	s.hsmPort = widget.NewEntry()
	s.hsmPort.SetPlaceHolder("Enter port number...")
	s.hsmPort.OnChanged = func(text string) {
		// Validate port number.
		if text != "" {
			if _, err := strconv.Atoi(text); err != nil {
				s.hsmPort.SetText(text[:len(text)-1])
			}
		}
	}

	s.lmkIndex = widget.NewSelect(LMKPairIndices, nil)

	// Initialize other settings.
	s.pluginPath = widget.NewEntry()
	s.pluginPath.SetPlaceHolder("Path to WASM plugins...")

	s.logLevel = widget.NewSelect(LogLevels, nil)
	s.themeToggle = widget.NewCheck("Dark Theme", nil)

	// Create form layout.
	form := widget.NewForm(
		&widget.FormItem{Text: "HSM IP", Widget: s.hsmIP},
		&widget.FormItem{Text: "HSM Port", Widget: s.hsmPort},
		&widget.FormItem{Text: "LMK Pair Index", Widget: s.lmkIndex},
		&widget.FormItem{Text: "Plugin Path", Widget: s.pluginPath},
		&widget.FormItem{Text: "Log Level", Widget: s.logLevel},
		&widget.FormItem{Text: "Theme", Widget: s.themeToggle},
	)

	// Create test connection button.
	testBtn := widget.NewButton("Test Connection", s.onTestConnection)

	s.container = container.NewVBox(
		form,
		container.NewHBox(
			testBtn,
		),
	)

	return s
}

func (s *Settings) onTestConnection() {
	// TODO: Implement HSM connection test.
	// For now, just show a placeholder dialog.
	dialog.ShowInformation("Connection Test",
		"Testing connection to HSM...",
		fyne.CurrentApp().Driver().AllWindows()[0])
}

// CreateRenderer implements fyne.Widget interface.
func (s *Settings) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(s.container)
}

// Cleanup implements TabContent interface.
func (s *Settings) Cleanup() {
	// No sensitive data to clean in settings tab.
}
