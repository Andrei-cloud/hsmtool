package tabs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var (
	// PaddingModes available for DES operations.
	PaddingModes = []string{"None", "ISO 9797-1 Method 1", "ISO 9797-1 Method 2"}

	// CipherModes available for DES operations.
	CipherModes = []string{"ECB", "CBC", "CFB"}

	// Operations available for DES calculator.
	Operations = []string{"Encrypt", "Decrypt"}
)

// DESCalculator represents the DES Calculator tab.
type DESCalculator struct {
	widget.BaseWidget
	container *fyne.Container

	// Input fields.
	dataInput *widget.Entry
	keyInput  *widget.Entry
	padding   *widget.Select
	mode      *widget.Select
	operation *widget.Select

	// Output fields.
	kcv    *widget.Label
	result *widget.Entry
}

// NewDESCalculator creates a new DES Calculator tab.
func NewDESCalculator() *DESCalculator {
	dc := &DESCalculator{}
	dc.ExtendBaseWidget(dc)

	// Initialize input fields.
	dc.dataInput = widget.NewMultiLineEntry()
	dc.dataInput.SetPlaceHolder("Enter hex data...")

	dc.keyInput = widget.NewEntry()
	dc.keyInput.SetPlaceHolder("Enter 8/16/24-byte hex key...")

	dc.padding = widget.NewSelect(PaddingModes, nil)
	dc.mode = widget.NewSelect(CipherModes, nil)
	dc.operation = widget.NewSelect(Operations, nil)

	// Initialize output fields.
	dc.kcv = widget.NewLabel("KCV: ")
	dc.result = widget.NewMultiLineEntry()
	dc.result.Disable() // Read-only result field.

	// Create form layout.
	form := widget.NewForm(
		&widget.FormItem{Text: "Data (Hex)", Widget: dc.dataInput},
		&widget.FormItem{Text: "Key", Widget: dc.keyInput},
		&widget.FormItem{Text: "Padding", Widget: dc.padding},
		&widget.FormItem{Text: "Mode", Widget: dc.mode},
		&widget.FormItem{Text: "Operation", Widget: dc.operation},
		&widget.FormItem{Text: "Key Check Value", Widget: dc.kcv},
		&widget.FormItem{Text: "Result", Widget: dc.result},
	)

	// Set form actions.
	form.SubmitText = "Calculate"
	form.OnSubmit = dc.onCalculate

	dc.container = container.NewVBox(form)

	return dc
}

func (dc *DESCalculator) onCalculate() {
	// TODO: Implement DES calculation logic.
}

// CreateRenderer implements fyne.Widget interface.
func (dc *DESCalculator) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(dc.container)
}

// Cleanup implements TabContent interface.
func (dc *DESCalculator) Cleanup() {
	// Clear sensitive data.
	dc.keyInput.SetText("")
	dc.dataInput.SetText("")
	dc.result.SetText("")
	dc.kcv.SetText("KCV: ")
}
