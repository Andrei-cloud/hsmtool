package tabs

import (
	"encoding/hex"
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	descrypto "github.com/andrei-cloud/hsmtool/internal/backend/crypto"
	"github.com/andrei-cloud/hsmtool/pkg/utils"
)

var (
	// PaddingModes available for DES operations.
	PaddingModes = []string{"None", "ISO 9797-1 Method 1", "ISO 9797-1 Method 2"}

	// CipherModes available for DES operations.
	CipherModes = []string{"ECB", "CBC"}

	// Operations available for DES calculator.
	Operations = []string{"Encrypt", "Decrypt"}
)

// DESCalculator represents the DES Calculator tab.
type DESCalculator struct {
	widget.BaseWidget
	container *fyne.Container
	form      *widget.Form // Added form field for grouped dropdowns

	// Input fields.
	dataInput   *widget.Entry
	keyInput    *widget.Entry
	padding     *widget.Select
	mode        *widget.Select
	operation   *widget.Select
	ivInput     *widget.Entry   // iv input for CBC mode
	ivContainer *fyne.Container // container for iv row

	// Output fields.
	kcv    *widget.Label
	result *widget.Entry
}

// NewDESCalculator creates a new DES Calculator tab.
func NewDESCalculator() *DESCalculator {
	c := &DESCalculator{}
	c.ExtendBaseWidget(c)

	// Create IV input for CBC mode first
	c.ivInput = widget.NewEntry()
	c.ivInput.SetPlaceHolder("Enter IV in hex format (16 hex digits)")
	c.ivInput.Resize(fyne.NewSize(320, 36))

	c.ivContainer = container.NewHBox(
		container.NewGridWrap(fyne.NewSize(60, 36), widget.NewLabel("IV:")),
		container.NewGridWrap(fyne.NewSize(480, 36), c.ivInput),
		layout.NewSpacer(),
	)

	// Create Mode/Operation/Padding group items
	c.mode = widget.NewSelect([]string{"ECB", "CBC"}, func(value string) {
		if value == "CBC" {
			c.ivContainer.Show()
		} else {
			c.ivContainer.Hide()
		}
	})
	c.mode.SetSelected("ECB")

	c.operation = widget.NewSelect([]string{"Encrypt", "Decrypt"}, nil)
	c.operation.SetSelected("Encrypt")

	c.padding = widget.NewSelect([]string{"None", "PKCS7"}, nil)
	c.padding.SetSelected("None")

	// Create form with Mode/Operation/Padding group.
	c.form = &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Mode", Widget: c.mode},
			{Text: "Operation", Widget: c.operation},
			{Text: "Padding", Widget: c.padding},
		},
	}

	// Create data input field (640px, multi-line).
	c.dataInput = widget.NewMultiLineEntry()
	c.dataInput.SetPlaceHolder("Enter data in hex format")
	c.dataInput.Wrapping = fyne.TextWrapBreak
	c.dataInput.Resize(fyne.NewSize(640, 100)) // Set initial size

	// Create key input field with proper sizing for 48 hex digits
	c.keyInput = widget.NewEntry()
	c.keyInput.SetPlaceHolder("Enter DES key in hex format (16/32/48 hex digits)")
	c.keyInput.Resize(fyne.NewSize(480, 36))
	c.keyInput.OnChanged = func(key string) {
		c.calculateKCV(key)
	}

	// Create KCV label
	c.kcv = widget.NewLabelWithStyle("", fyne.TextAlignCenter, fyne.TextStyle{})

	// Create result field with proper sizing.
	c.result = widget.NewMultiLineEntry()
	c.result.Wrapping = fyne.TextWrapBreak
	c.result.Resize(fyne.NewSize(640, 100))
	c.result.Disable() // Make result read-only

	// Create calculate button.
	calculate := widget.NewButton("Calculate", func() {
		c.calculate()
	})

	// Layout with visual separators and proper spacing.
	c.container = container.NewVBox(
		// Mode/Operation/Padding group.
		widget.NewCard("Settings", "", c.form),

		// Data input section.
		widget.NewCard("Input Data", "",
			container.NewVBox(
				c.dataInput,
			),
		),

		// Key and KCV section with fixed widths and consistent alignment.
		widget.NewCard("Key", "",
			container.NewVBox(
				container.NewHBox(
					container.NewGridWrap(fyne.NewSize(480, 36), c.keyInput),
					layout.NewSpacer(),
					widget.NewLabelWithStyle(
						"KCV:",
						fyne.TextAlignLeading,
						fyne.TextStyle{Bold: true},
					),
					container.NewGridWrap(fyne.NewSize(120, 36), c.kcv),
				),
				widget.NewLabel(""), // Add subtle spacing
				c.ivContainer,
			),
		),

		// Result section.
		widget.NewCard("Result", "",
			container.NewVBox(
				c.result,
			),
		),

		// Calculate button.
		calculate,
	)

	return c
}

// calculateKCV calculates and displays the Key Check Value for the given key.
func (c *DESCalculator) calculateKCV(key string) {
	// Remove any spaces from key.
	key = strings.ToUpper(strings.ReplaceAll(key, " ", ""))

	// Validate key length.
	if len(key)%16 != 0 || len(key) > 48 || len(key) == 0 {
		c.kcv.SetText("Invalid key length")
		return
	}

	// Convert key from hex to bytes.
	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		c.kcv.SetText("Invalid hex format")
		return
	}

	// Calculate KCV: encrypt 8 zero bytes with the key.
	zeros := make([]byte, 8)
	params := &descrypto.DESParams{
		Data:    zeros,
		Key:     keyBytes,
		Mode:    descrypto.ECB,
		Padding: descrypto.NoPadding,
		Encrypt: true,
	}

	result, err := descrypto.ProcessDES(params)
	if err != nil {
		c.kcv.SetText("KCV error")
		return
	}

	// Display first 3 bytes of result as KCV in uppercase.
	kcv := strings.ToUpper(hex.EncodeToString(result[:3]))
	c.kcv.SetText(kcv)
}

// calculate processes the input data according to the selected options.
func (c *DESCalculator) calculate() {
	// Get and validate the key.
	key := strings.ReplaceAll(c.keyInput.Text, " ", "")
	if len(key)%16 != 0 || len(key) > 48 || len(key) == 0 {
		c.result.SetText("Invalid key length")
		return
	}
	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		c.result.SetText("Invalid key format")
		return
	}

	// Get and validate the data.
	data := strings.ToUpper(strings.ReplaceAll(c.dataInput.Text, " ", ""))
	if len(data) == 0 {
		c.result.SetText("No data provided")
		return
	}
	dataBytes, err := hex.DecodeString(data)
	if err != nil {
		c.result.SetText("Invalid data format")
		return
	}

	// Get and validate IV if in CBC mode.
	var iv []byte
	if c.mode.Selected == "CBC" {
		ivStr := strings.ToUpper(strings.ReplaceAll(c.ivInput.Text, " ", ""))
		if len(ivStr) != 16 {
			c.result.SetText("Invalid IV length (must be 16 hex digits)")
			return
		}
		iv, err = hex.DecodeString(ivStr)
		if err != nil {
			c.result.SetText("Invalid IV format")
			return
		}
	}

	// Prepare parameters.
	var mode descrypto.CipherMode
	switch c.mode.Selected {
	case "CBC":
		mode = descrypto.CBC
	default:
		mode = descrypto.ECB
	}

	var padding descrypto.PaddingMode
	switch c.padding.Selected {
	case "None":
		padding = descrypto.NoPadding
	case "ISO 9797-1 Method 1":
		padding = descrypto.ISO97971
	case "ISO 9797-1 Method 2":
		padding = descrypto.ISO97972
	default:
		padding = descrypto.NoPadding
	}

	params := &descrypto.DESParams{
		Data:    dataBytes,
		Key:     keyBytes,
		Mode:    mode,
		Padding: padding,
		Encrypt: c.operation.Selected == "Encrypt",
		IV:      iv,
	}

	// Process the data.
	result, err := descrypto.ProcessDES(params)
	if err != nil {
		c.result.SetText(fmt.Sprintf("Error: %v", err))
		return
	}

	// Display the result in uppercase.
	c.result.SetText(strings.ToUpper(hex.EncodeToString(result)))
}

// onModeChanged shows or hides iv input based on mode.
func (dc *DESCalculator) onModeChanged(mode string) {
	if mode == "CBC" {
		dc.ivContainer.Show()
	} else {
		dc.ivContainer.Hide()
	}
	dc.container.Refresh()
}

// onKeyChanged updates KCV when key input changes.
func (dc *DESCalculator) onKeyChanged(text string) {
	clean := strings.ReplaceAll(text, " ", "")
	if err := utils.ValidateHex(clean); err != nil {
		dc.kcv.SetText("invalid hex string")
		return
	}
	byteLen := len(clean) / 2
	if byteLen != 8 && byteLen != 16 && byteLen != 24 {
		dc.kcv.SetText("invalid key length")
		return
	}
	key, err := utils.DecodeHex(clean)
	if err != nil {
		dc.kcv.SetText("invalid hex string")
		return
	}
	kcvVal, err := descrypto.CalculateKCV(key)
	if err != nil {
		dc.kcv.SetText("error calculating kcv")
		return
	}
	dc.kcv.SetText("KCV: " + strings.ToUpper(kcvVal))
}

// onDataChanged validates data input.
func (dc *DESCalculator) onDataChanged(text string) {
	// no-op; validation on calculate
}

// onIVChanged validates iv input.
func (dc *DESCalculator) onIVChanged(text string) {
	// no-op; validation on calculate
}

// onCalculate performs DES encrypt/decrypt.
func (dc *DESCalculator) onCalculate() {
	w := fyne.CurrentApp().Driver().AllWindows()[0]

	// validate data
	dataClean := strings.ToUpper(strings.ReplaceAll(dc.dataInput.Text, " ", ""))
	if err := utils.ValidateHex(dataClean); err != nil {
		dialog.ShowError(err, w)

		return
	}
	data, _ := hex.DecodeString(dataClean)

	// validate key
	keyClean := strings.ToUpper(strings.ReplaceAll(dc.keyInput.Text, " ", ""))
	if err := utils.ValidateHex(keyClean); err != nil {
		dialog.ShowError(err, w)

		return
	}
	keyBytes, _ := hex.DecodeString(keyClean)
	if len(keyBytes) != 8 && len(keyBytes) != 16 && len(keyBytes) != 24 {
		dialog.ShowError(fmt.Errorf("invalid key length"), w)

		return
	}

	// prepare params
	params := &descrypto.DESParams{
		Data:    data,
		Key:     keyBytes,
		Encrypt: dc.operation.Selected == "Encrypt",
	}
	// mode
	switch dc.mode.Selected {
	case "ECB":
		params.Mode = descrypto.ECB

	case "CBC":
		params.Mode = descrypto.CBC
		ivClean := strings.ToUpper(strings.ReplaceAll(dc.ivInput.Text, " ", ""))
		if err := utils.ValidateHex(ivClean); err != nil {
			dialog.ShowError(err, w)

			return
		}
		ivBytes, _ := hex.DecodeString(ivClean)
		if len(ivBytes) != 8 {
			dialog.ShowError(fmt.Errorf("invalid iv length"), w)

			return
		}
		params.IV = ivBytes

	default:
		dialog.ShowError(fmt.Errorf("unsupported mode"), w)

		return
	}
	// padding
	switch dc.padding.Selected {
	case "None":
		params.Padding = descrypto.NoPadding
	case "ISO 9797-1 Method 1":
		params.Padding = descrypto.ISO97971
	case "ISO 9797-1 Method 2":
		params.Padding = descrypto.ISO97972
	default:
		params.Padding = descrypto.NoPadding
	}
	// perform operation
	resultBytes, err := descrypto.ProcessDES(params)
	if err != nil {
		dialog.ShowError(err, w)

		return
	}
	dc.result.SetText(strings.ToUpper(hex.EncodeToString(resultBytes)))
}

// CreateRenderer returns a new renderer for the DESCalculator widget.
func (c *DESCalculator) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.container)
}

// Cleanup implements TabContent interface.
func (dc *DESCalculator) Cleanup() {
	// Clear sensitive data.
	dc.keyInput.SetText("")
	dc.dataInput.SetText("")
	dc.result.SetText("")
	dc.kcv.SetText("KCV: ")
}
