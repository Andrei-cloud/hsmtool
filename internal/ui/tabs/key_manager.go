package tabs

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

var KeyTypes = []string{"ZMK", "ZPK", "TMK", "PVK", "KEK"}

var KeyLengths = []string{
	"Single DES (8 bytes)",
	"Double DES (16 bytes)",
	"Triple DES (24 bytes)",
	"AES-128",
	"AES-256",
}

// KeyManager represents the Key Manager tab.
type KeyManager struct {
	widget.BaseWidget
	container *fyne.Container

	// Input fields.
	keyName   *widget.Entry
	keyType   *widget.Select
	keyLength *widget.Select
	keyInput  *widget.Entry
	kcv       *widget.Label

	// Stored keys table.
	keysTable *widget.Table
}

// NewKeyManager creates a new Key Manager tab.
func NewKeyManager() *KeyManager {
	km := &KeyManager{}
	km.ExtendBaseWidget(km)

	// Initialize input fields.
	km.keyName = widget.NewEntry()
	km.keyName.SetPlaceHolder("Enter key name...")

	km.keyType = widget.NewSelect(KeyTypes, nil)
	km.keyLength = widget.NewSelect(KeyLengths, nil)

	km.keyInput = widget.NewEntry()
	km.keyInput.SetPlaceHolder("Hex format key value...")

	km.kcv = widget.NewLabel("KCV: ")
	// Form actions will be set after form creation.// Create form layout.
	form := widget.NewForm(
		&widget.FormItem{Text: "Key Name", Widget: km.keyName},
		&widget.FormItem{Text: "Key Type", Widget: km.keyType},
		&widget.FormItem{Text: "Key Length", Widget: km.keyLength},
		&widget.FormItem{Text: "Key Value", Widget: km.keyInput},
		&widget.FormItem{Text: "Check Value", Widget: km.kcv},
	)

	// Add buttons to form.
	form.SubmitText = "Generate in HSM"
	form.OnSubmit = km.onGenerateKey
	form.CancelText = "Store Key"
	form.OnCancel = km.onStoreKey

	// Initialize stored keys table.
	km.initializeTable()

	// Layout everything in a container.
	km.container = container.NewVBox(
		form,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Stored Keys", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		km.keysTable,
	)

	return km
}

func (km *KeyManager) initializeTable() {
	km.keysTable = widget.NewTable(
		func() (int, int) { return 0, 4 }, // Initial size.
		func() fyne.CanvasObject { // Template object.
			return widget.NewLabel("Template")
		},
		func(_ widget.TableCellID, o fyne.CanvasObject) {
			// Will populate data here.
		},
	)
}

func (km *KeyManager) onGenerateKey() {
	// TODO: Implement key generation via HSM.
}

func (km *KeyManager) onStoreKey() {
	// TODO: Implement key storage.
}

// CreateRenderer implements fyne.Widget interface.
func (km *KeyManager) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(km.container)
}

// Cleanup implements TabContent interface.
func (km *KeyManager) Cleanup() {
	// Clear sensitive data.
	km.keyInput.SetText("")
	km.kcv.SetText("KCV: ")
}
