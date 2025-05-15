package tabs

import (
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/andrei-cloud/hsmtool/internal/backend/hsm"
)

// KeySchemes holds supported Variant-LMK key scheme tags.
var KeySchemes = []string{"Z", "U", "T", "X", "Y"}

// KeyTypes holds A0-supported DES and related keys with display names.
var KeyTypes = []string{
	"000 - ZMK",
	"001 - ZPK",
	"002 - PVK/Generic",
	"003 - TAK",
	"008 - ZAK",
	"009 - BDK type-1",
	"00A - ZEK",
	"00B - DEK/TEK",
}

// KeyManager represents the Key Manager tab.
type KeyManager struct {
	widget.BaseWidget
	container *fyne.Container

	connection *hsm.Connection

	// Input fields.
	keyType   *widget.Select
	keyScheme *widget.Select
	keyInput  *widget.Entry
	kcv       *widget.Label
}

// NewKeyManager creates a new Key Manager tab.
func NewKeyManager(conn *hsm.Connection) *KeyManager {
	km := &KeyManager{connection: conn}
	km.ExtendBaseWidget(km)

	// Initialize input fields.
	km.keyType = widget.NewSelect(KeyTypes, nil)
	km.keyScheme = widget.NewSelect(KeySchemes, nil)

	km.keyInput = widget.NewEntry()
	km.keyInput.SetPlaceHolder("Hex format key value...")

	km.kcv = widget.NewLabel("KCV: ")

	// Create form layout.
	form := widget.NewForm(
		&widget.FormItem{Text: "Key Type", Widget: km.keyType},
		&widget.FormItem{Text: "Key Scheme", Widget: km.keyScheme},
		&widget.FormItem{Text: "Key Value", Widget: km.keyInput},
		&widget.FormItem{Text: "Check Value", Widget: km.kcv},
	)

	// Add generate button to form.
	form.SubmitText = "Generate in HSM"
	form.OnSubmit = km.onGenerateKey

	// Layout everything in a container - storage removed.
	km.container = container.NewVBox(
		form,
	)

	return km
}

func (km *KeyManager) onGenerateKey() {
	// check HSM connection.
	if km.connection.GetState() != hsm.Connected {
		dialog.ShowError(
			fmt.Errorf("hsm not connected - please connect first"),
			fyne.CurrentApp().Driver().AllWindows()[0],
		)

		return
	}

	// validate selected key scheme.
	if km.keyScheme.Selected == "" {
		dialog.ShowError(
			fmt.Errorf("select key scheme"),
			fyne.CurrentApp().Driver().AllWindows()[0],
		)

		return
	}

	// build A0 command: generate key under Variant LMK with scheme.
	parts := strings.SplitN(km.keyType.Selected, " - ", 2)
	keyCode := parts[0]
	scheme := km.keyScheme.Selected
	// mode '0' = generate under LMK only.
	mode := '0'
	cmdText := fmt.Sprintf("A0%c%s%s", mode, keyCode, scheme)
	respBytes, err := km.connection.ExecuteCommand([]byte(cmdText), 5*time.Second)
	if err != nil {
		dialog.ShowError(err, fyne.CurrentApp().Driver().AllWindows()[0])

		return
	}
	respStr := string(respBytes)

	// check response code.
	if !strings.HasPrefix(respStr, "A1") {
		dialog.ShowError(
			fmt.Errorf("unexpected response code: %s", respStr[:2]),
			fyne.CurrentApp().Driver().AllWindows()[0],
		)

		return
	}

	// parse error code.
	errCode := respStr[2:4]
	if errCode != "00" {
		var msg string
		switch errCode {
		case "07":
			msg = "invalid zka master key type"
		case "10":
			msg = "zmk or tmk parity error"
		case "68":
			msg = "command disabled"
		default:
			msg = "error code " + errCode
		}

		dialog.ShowError(
			fmt.Errorf(msg),
			fyne.CurrentApp().Driver().AllWindows()[0],
		)

		return
	}

	// extract encrypted key and kcv.
	encrypted := respStr[4 : len(respStr)-6]
	kcvVal := respStr[len(respStr)-6:]

	// display results.
	km.keyInput.SetText(encrypted)
	km.kcv.SetText("KCV: " + kcvVal)
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
