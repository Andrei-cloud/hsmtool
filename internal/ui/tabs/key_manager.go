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
	"000 ZMK - Zone Master Key (also known as ZCMK)",
	"100 ZMK(C) - Zone Master Key Component (legacy commands only)",
	"200 KML - Master Load Key (Visa Cash)",
	"300 KEKr - (AS 2805) Key Encryption Key of Recipient)",
	"400 KEKs - (AS 2805) Key Encryption Key of Sender)",
	"001 ZPK - Zone PIN Key)",
	"002 PVK - PIN Verification Key)",
	"002 PVVK - (OBKM) PVV Key)",
	"002 TPK - Terminal PIN Key)",
	"002 PEK - (AS 2805) PIN Encipherment Key)",
	"002 PEK - (LIC031) PIN Encryption Key)",
	"002 TMK - Terminal Master Key)",
	"002 KT - (AS 2805) Transaction Key)",
	"002 TK - (AS 2805) Terminal Key)",
	"002 KI - (AS 2805) Initial Transport Key)",
	"002 KCA - (AS 2805) Sponsor Cross Acquirer Key)",
	"002 KMA - (AS 2805) Acquirer Master Key Encrypting Key)",
	"002 TKR - Terminal Key Register)",
	"102 TMK1 - (AS 2805) Terminal Master Key)",
	"202 TMK2 - (AS 2805) Terminal Master Key)",
	"302 IKEY - Initial Key (DUKPT)",
	"30D CK-ENK - (Issuing) Card Key for Cryptograms)",
	"402 CVK - Card Verification Key)",
	"402 CSCK - Card Security Code Key)",
	"40D CK-MAC - (Issuing) Card Key for Authentication)",
	"50D CK-DEK - (Issuing) Card Key for Authentication)",
	"602 KIA - (AS 2805) Acquirer Initialization Key)",
	"003 TAK - Terminal Authentication Key)",
	"103 TAKr - (AS 2805) Terminal Authentication Key of Recipient)",
	"103 TAKs - (AS 2805) Terminal Authentication Key of Sender)",
	"105 KML - (OBKM) Master Load Key)",
	"105 KMLISS - (OBKM) Master Load Key for Issuer)",
	"205 KMX - (OBKM) Master Currency Exchange Key)",
	"205 KMXISS - (OBKM) Master Currency Exchange Key for Issuer)",
	"305 KMP - (OBKM) Master Purchase Key)",
	"305 KMPISS - (OBKM) Master Purchase Key for Issuer)",
	"405 KIS.5 - (OBKM) S5 Issuer Key)",
	"505 KM3L - (OBKM) Master Key for Load & Unload Verification)",
	"505 KM3LISS - (OBKM) Master Key for Load & Unload Verification for Issuer)",
	"605 KM3X - (OBKM) Master Key for Currency Exchange Verification)",
	"605 KM3XISS - (OBKM) Master Key for Currency Exchange Verification for Issuer)",
	"705 KMACS4 - (OBKM))",
	"805 KMACS5 - (OBKM))",
	"905 KMACACQ - (OBKM))",
	"006 WWK - Watchword Key)",
	"106 KMACUPD - (OBKM))",
	"206 KMACMA - (OBKM))",
	"306 KMACCI - (OBKM))",
	"306 KMACISS - (OBKM))",
	"406 KMSCISS - (OBKM) Secure Messaging Master Key)",
	"506 BKEM - (OBKM) Transport key for key encryption)",
	"606 BKAM - (OBKM) Transport key for message authentication)",
	"107 KEK - (Issuing) Key Encryption Key)",
	"207 KMC - (Issuing) Master Personalization Key)",
	"307 SK-ENC - (Issuing) Session Key for cryptograms and encrypting card messages)",
	"407 SK-MAC - (Issuing) Session Key for authenticating card messages)",
	"507 SK-DEK - (Issuing) Session Key for encrypting secret card data)",
	"507 KD-PERSO - (Issuing) KD Personalization Key)",
	"607 ZKA MK - Master key for GBIC/ZKA key derivation)",
	"807 MK-KE - (Issuing) Master KTU Encipherment key)",
	"907 MK-AS - (Issuing) Master Application Signature (MAC) key)",
	"008 ZAK - Zone Authentication Key)",
	"108 ZAKs - (AS 2805) Zone Authentication Key of Sender)",
	"208 ZAKr - (AS 2805) Zone Authentication Key of Recipient)",
	"009 BDK-1 - Base Derivation Key (type 1)",
	"109 MK-AC - Master Key for Application Cryptograms)",
	"209 MK-SMI - Master Key for Secure Messaging (for Integrity)",
	"309 MK-SMC - Master Key for Secure Messaging (for Confidentiality)",
	"409 MK-DAC - Master Key for Data Authentication Codes)",
	"509 MK-DN - Master Key for Dynamic Numbers)",
	"609 BDK-2 - Base Derivation Key (type 2)",
	"709 MK-CVC3 - Master Key for CVC3 (Contactless)",
	"809 BDK-3 - Base Derivation Key (type 3)",
	"909 BDK-4 - Base Derivation Key (type 4)",
	"00A ZEK - Zone Encryption Key)",
	"10A ZEKs - (AS 2805) Zone Encryption Key of Sender)",
	"20A ZEKr - (AS 2805) Zone Encryption Key of Recipient)",
	"00B DEK - Data Encryption Key)",
	"00B TEK - (AS 2805) Terminal Encryption Key)",
	"10B TEKs - (AS 2805) Terminal Encryption Key of Sender)",
	"10B TEKr - (AS 2805) Terminal Encryption Key of recipient)",
	"30B TEK - Terminal Encryption Key)",
	"00C RSA-SK - RSA Private Key)",
	"10C HMAC - HMAC key)",
	"00D RSA-PK - RSA Public Key)",
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
	fields := strings.Fields(km.keyType.Selected)
	keyCode := fields[0]
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
