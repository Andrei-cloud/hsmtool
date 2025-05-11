package tabs

import (
	"encoding/hex"
	"fmt"
	"math/bits"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	hsm "github.com/andrei-cloud/go_hsm/pkg/crypto"
	cryptobackend "github.com/andrei-cloud/hsmtool/internal/backend/crypto"
)

var BitwiseOperations = []string{
	"XOR",
	"AND",
	"OR",
	"NOT",
}

var ModeOptions = []string{"Regular", "Key Sharing"}

// BitwiseCalculator represents the Bitwise Calculator tab.
type BitwiseCalculator struct {
	widget.BaseWidget
	container *fyne.Container
	content   *fyne.Container

	// Mode toggle implemented as a horizontal radio group.
	modeToggle *widget.RadioGroup

	// Regular mode inputs.
	operation *widget.RadioGroup
	blockA    *widget.Entry
	blockB    *widget.Entry
	result    *widget.Entry

	// Key sharing mode inputs.
	combinedKey   *widget.Entry
	comp1         *widget.Entry
	comp2         *widget.Entry
	comp3         *widget.Entry
	comp3Label    *widget.Label
	numComponents *widget.RadioGroup
	parityBits    *widget.RadioGroup
	combinedKCV   *widget.Label
	comp1KCV      *widget.Label
	comp2KCV      *widget.Label
	comp3KCV      *widget.Label
	generate64    *widget.Button
	generate128   *widget.Button
	generate192   *widget.Button
	generate256   *widget.Button
	splitBtn      *widget.Button
	combineBtn    *widget.Button
	helpText      *widget.Label
}

// Initialize all UI components for the calculator.
func (bc *BitwiseCalculator) initializeComponents() {
	// Regular mode fields.
	bc.operation = widget.NewRadioGroup(BitwiseOperations, nil)
	bc.operation.Horizontal = true
	bc.operation.SetSelected(BitwiseOperations[0])
	bc.blockA = widget.NewEntry()
	bc.blockA.SetPlaceHolder("Enter hex value (up to 16 digits)...")
	bc.blockA.OnChanged = func(s string) { bc.validateHex(s, bc.blockA, 16) }
	bc.blockB = widget.NewEntry()
	bc.blockB.SetPlaceHolder("Enter hex value (up to 16 digits)...")
	bc.blockB.OnChanged = func(s string) { bc.validateHex(s, bc.blockB, 16) }
	bc.result = widget.NewEntry()
	bc.result.Disable()

	// Key sharing mode fields.
	bc.combinedKey = widget.NewEntry()
	bc.combinedKey.SetPlaceHolder("Combined key (hex, up to 64 chars)...")
	bc.combinedKey.OnChanged = func(s string) { bc.validateHex(s, bc.combinedKey, 64) }
	bc.combinedKey.SetMinSize(fyne.NewSize(320, 0)) // minimum width for ~32 hex chars.

	bc.comp1 = widget.NewEntry()
	bc.comp1.SetPlaceHolder("Component 1 (hex, up to 64 chars)...")
	bc.comp1.OnChanged = func(s string) { bc.validateHex(s, bc.comp1, 64) }
	bc.comp1.SetMinSize(fyne.NewSize(320, 0)) // minimum width for ~32 hex chars.

	bc.comp2 = widget.NewEntry()
	bc.comp2.SetPlaceHolder("Component 2 (hex, up to 64 chars)...")
	bc.comp2.OnChanged = func(s string) { bc.validateHex(s, bc.comp2, 64) }
	bc.comp2.SetMinSize(fyne.NewSize(320, 0)) // minimum width for ~32 hex chars.

	bc.comp3 = widget.NewEntry()
	bc.comp3.SetPlaceHolder("Component 3 (hex, up to 64 chars, optional)...")
	bc.comp3.OnChanged = func(s string) { bc.validateHex(s, bc.comp3, 64) }
	bc.comp3.SetMinSize(fyne.NewSize(320, 0)) // minimum width for ~32 hex chars.
	bc.comp3.Hide()                           // Initially hidden.

	// Component labels.
	bc.comp3Label = widget.NewLabel("Component 3")
	bc.comp3Label.Hide()

	// KCV labels.
	bc.combinedKCV = widget.NewLabel("KCV:")
	bc.comp1KCV = widget.NewLabel("KCV:")
	bc.comp2KCV = widget.NewLabel("KCV:")
	bc.comp3KCV = widget.NewLabel("KCV:")
	bc.comp3KCV.Hide()

	// Radio groups for options
	bc.numComponents = widget.NewRadioGroup([]string{"2", "3"}, bc.onNumComponentsChanged)
	bc.numComponents.SetSelected("2")
	bc.parityBits = widget.NewRadioGroup([]string{"Ignore", "Force Odd"}, nil)
	bc.parityBits.SetSelected("Ignore")

	// Action buttons
	bc.generate64 = widget.NewButton("64-bit", bc.onGenerateKey(64))
	bc.generate128 = widget.NewButton("128-bit", bc.onGenerateKey(128))
	bc.generate192 = widget.NewButton("192-bit", bc.onGenerateKey(192))
	bc.generate256 = widget.NewButton("256-bit", bc.onGenerateKey(256))
	bc.splitBtn = widget.NewButton("Split", bc.onSplit)
	bc.combineBtn = widget.NewButton("Combine", bc.onCombine)

	// Help text
	bc.helpText = widget.NewLabel(
		"KCV (Key Check Value) is calculated as the first 3 bytes of DES encryption " +
			"of zeros (0x0000000000000000) with the key. Key components are combined using XOR operation.",
	)
	bc.helpText.Wrapping = fyne.TextWrapWord
}

// NewBitwiseCalculator creates a new Bitwise Calculator tab.
func NewBitwiseCalculator() *BitwiseCalculator {
	bc := &BitwiseCalculator{}
	bc.ExtendBaseWidget(bc)

	// Initialize all components first.
	bc.initializeComponents()

	// Create containers.
	bc.content = container.NewVBox()
	bc.container = container.NewVBox()

	// Mode toggle should be created last since its callback depends on other components.
	bc.modeToggle = widget.NewRadioGroup(ModeOptions, bc.onModeChange)
	bc.modeToggle.Horizontal = true
	bc.container.Add(bc.modeToggle)
	bc.container.Add(bc.content)

	// Only set the initial mode after everything is initialized.
	bc.modeToggle.SetSelected(ModeOptions[0])

	return bc
}

func (bc *BitwiseCalculator) onModeChange(mode string) {
	bc.content.Objects = nil
	if mode == "Key Sharing" {
		combinedKeyRow := container.NewGridWithColumns(3,
			widget.NewLabel("Combined Key"),
			bc.combinedKey,
			bc.combinedKCV,
		)
		component1Row := container.NewGridWithColumns(3,
			widget.NewLabel("Component 1"),
			bc.comp1,
			bc.comp1KCV,
		)
		component2Row := container.NewGridWithColumns(3,
			widget.NewLabel("Component 2"),
			bc.comp2,
			bc.comp2KCV,
		)
		component3Row := container.NewGridWithColumns(3,
			bc.comp3Label,
			bc.comp3,
			bc.comp3KCV,
		)

		keyInputs := container.NewVBox(
			combinedKeyRow,
			widget.NewSeparator(),
			component1Row,
			component2Row,
			component3Row,
		)

		options := container.NewHBox(
			container.NewVBox(
				widget.NewLabel("Number of Components"),
				bc.numComponents,
			),
			layout.NewSpacer(),
			container.NewVBox(
				widget.NewLabel("Parity Bits"),
				bc.parityBits,
			),
			layout.NewSpacer(),
		)
		centeredOptions := container.NewCenter(options)

		genButtons := container.NewHBox(
			layout.NewSpacer(),
			bc.generate64,
			bc.generate128,
			bc.generate192,
			bc.generate256,
			layout.NewSpacer(),
		)

		actionButtons := container.NewHBox(
			layout.NewSpacer(),
			bc.splitBtn,
			bc.combineBtn,
			layout.NewSpacer(),
		)

		bc.content.Add(keyInputs)
		bc.content.Add(widget.NewSeparator())
		bc.content.Add(centeredOptions)
		bc.content.Add(widget.NewSeparator())
		bc.content.Add(genButtons)
		bc.content.Add(widget.NewSeparator())
		bc.content.Add(actionButtons)
		bc.content.Add(widget.NewSeparator())
		bc.content.Add(bc.helpText)
	} else {
		calc := container.NewVBox(
			bc.operation,
			bc.blockA,
			bc.blockB,
			bc.result,
			widget.NewButton("Calculate", bc.onCalculate),
		)
		bc.content.Add(calc)
	}
	bc.content.Refresh()
}

func (bc *BitwiseCalculator) onCalculate() {
	op := bc.operation.Selected
	a := bc.blockA.Text
	b := bc.blockB.Text
	params := &cryptobackend.BitwiseParams{
		Operation: cryptobackend.BitwiseOperation(op),
		BlockA:    a,
		BlockB:    b,
	}
	result, err := cryptobackend.PerformBitwise(params)
	if err != nil {
		bc.result.SetText(err.Error())

		return
	}

	bc.result.SetText(result)
}

// onSplit handles splitting the combined key into components.
func (bc *BitwiseCalculator) onSplit() {
	num := 2
	if bc.numComponents.Selected == "3" {
		num = 3
	}
	parity := bc.parityBits.Selected

	combined := bc.combinedKey.Text
	_, err := hex.DecodeString(combined)
	if err != nil {
		bc.combinedKCV.SetText("KCV: Invalid Key")
		return
	}

	components, origKCVHexStr, err := hsm.SplitKey(combined, num)
	if err != nil {
		bc.combinedKCV.SetText("KCV: Split Error")
		return
	}

	if parity == "Force Odd" {
		for i := range components {
			compHex, pErr := enforceOddParity(components[i])
			if pErr != nil {
				switch i {
				case 0:
					bc.comp1.SetText("Parity Error")
					bc.comp1KCV.SetText("KCV: Error")
				case 1:
					bc.comp2.SetText("Parity Error")
					bc.comp2KCV.SetText("KCV: Error")
				case 2:
					bc.comp3.SetText("Parity Error")
					bc.comp3KCV.SetText("KCV: Error")
				}
				components[i] = ""
			} else {
				components[i] = compHex
			}
		}
	}

	bc.combinedKey.SetText(strings.ToUpper(combined))
	bc.combinedKCV.SetText("KCV: " + strings.ToUpper(origKCVHexStr))

	if len(components) > 0 {
		bc.comp1.SetText(strings.ToUpper(components[0]))
		data1, err1 := hex.DecodeString(components[0])
		if err1 == nil && len(data1) > 0 {
			kcv1 := hsm.CalculateKCV(data1)
			bc.comp1KCV.SetText("KCV: " + strings.ToUpper(hex.EncodeToString(kcv1)))
		} else {
			bc.comp1KCV.SetText("KCV:")
			if err1 != nil {
				bc.comp1KCV.SetText("KCV: Invalid")
			}
		}
	}

	if len(components) > 1 {
		bc.comp2.SetText(strings.ToUpper(components[1]))
		data2, err2 := hex.DecodeString(components[1])
		if err2 == nil && len(data2) > 0 {
			kcv2 := hsm.CalculateKCV(data2)
			bc.comp2KCV.SetText("KCV: " + strings.ToUpper(hex.EncodeToString(kcv2)))
		} else {
			bc.comp2KCV.SetText("KCV:")
			if err2 != nil {
				bc.comp2KCV.SetText("KCV: Invalid")
			}
		}
	}

	if num == 3 {
		if len(components) > 2 {
			bc.comp3.SetText(strings.ToUpper(components[2]))
			data3, err3 := hex.DecodeString(components[2])
			if err3 == nil && len(data3) > 0 {
				kcv3 := hsm.CalculateKCV(data3)
				bc.comp3KCV.SetText("KCV: " + strings.ToUpper(hex.EncodeToString(kcv3)))
			} else {
				bc.comp3KCV.SetText("KCV:")
				if err3 != nil {
					bc.comp3KCV.SetText("KCV: Invalid")
				}
			}
		} else {
			bc.comp3.SetText("")
			bc.comp3KCV.SetText("KCV:")
		}
	} else {
		bc.comp3.SetText("")
		bc.comp3KCV.SetText("KCV:")
	}

	bc.container.Refresh()
}

// onCombine handles combining components into a single key.
func (bc *BitwiseCalculator) onCombine() {
	num := 2
	if bc.numComponents.Selected == "3" {
		num = 3
	}
	dcomps := []string{bc.comp1.Text, bc.comp2.Text}
	if num == 3 {
		dcomps = append(dcomps, bc.comp3.Text)
	}

	for i, c := range dcomps {
		_, err := hex.DecodeString(c)
		if err != nil {
			bc.combinedKey.SetText("")
			bc.combinedKCV.SetText(fmt.Sprintf("KCV: Comp %d Invalid", i+1))
			return
		}
	}

	keyHex, err := hsm.CombineComponents(dcomps)
	if err != nil {
		bc.combinedKey.SetText("")
		bc.combinedKCV.SetText("KCV: Combine Error")
		return
	}

	bc.combinedKey.SetText(strings.ToUpper(keyHex))

	data, _ := hex.DecodeString(keyHex)
	if len(data) > 0 {
		kcv := hsm.CalculateKCV(data)
		bc.combinedKCV.SetText("KCV: " + strings.ToUpper(hex.EncodeToString(kcv)))
	} else {
		bc.combinedKCV.SetText("KCV:")
	}

	bc.container.Refresh()
}

// validateHex checks if the input is valid hexadecimal, enforces maxLength, and calculates KCV.
func (bc *BitwiseCalculator) validateHex(originalS string, entry *widget.Entry, maxLength int) {
	processedS := strings.Builder{}
	processedS.Grow(len(originalS))
	for _, r := range originalS {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			processedS.WriteRune(r)
		}
	}

	hexInput := processedS.String()
	if len(hexInput) > maxLength {
		hexInput = hexInput[:maxLength]
	}

	hexInput = strings.ToUpper(hexInput)

	if entry.Text != hexInput {
		entry.SetText(hexInput)
	}

	var kcvLabel *widget.Label
	switch entry {
	case bc.combinedKey:
		kcvLabel = bc.combinedKCV
	case bc.comp1:
		kcvLabel = bc.comp1KCV
	case bc.comp2:
		kcvLabel = bc.comp2KCV
	case bc.comp3:
		kcvLabel = bc.comp3KCV
	case bc.blockA, bc.blockB:
		return
	default:
		return
	}

	hexLen := len(hexInput)
	if hexLen == 16 || hexLen == 32 || hexLen == 48 || hexLen == 64 {
		data, err := hex.DecodeString(hexInput)
		if err == nil && len(data) > 0 {
			kcv := hsm.CalculateKCV(data)
			kcvLabel.SetText("KCV: " + strings.ToUpper(hex.EncodeToString(kcv)))
		} else {
			kcvLabel.SetText("KCV: Invalid")
		}
	} else {
		kcvLabel.SetText("KCV:")
	}
}

// onGenerateKey returns a handler for generating and displaying DES key components.
func (bc *BitwiseCalculator) onGenerateKey(bits int) func() {
	return func() {
		bc.clearKeySharingFields()
		num := 2
		if bc.numComponents.Selected == "3" {
			num = 3
		}
		parity := bc.parityBits.Selected

		keyHex, combinedKCVHexStr, err := hsm.GenerateKey(bits)
		if err != nil {
			bc.combinedKey.SetText("Error generating key")
			bc.combinedKCV.SetText("KCV: Error")
			return
		}
		bc.combinedKey.SetText(strings.ToUpper(keyHex))
		bc.combinedKCV.SetText("KCV: " + strings.ToUpper(combinedKCVHexStr))

		components, _, err := hsm.SplitKey(keyHex, num)
		if err != nil {
			bc.comp1.SetText("Split Error")
			bc.comp1KCV.SetText("KCV: Error")
			bc.comp2.SetText("")
			bc.comp2KCV.SetText("KCV:")
			if num == 3 {
				bc.comp3.SetText("")
				bc.comp3KCV.SetText("KCV:")
			}
			return
		}

		if parity == "Force Odd" {
			for i := range components {
				compHex, pErr := enforceOddParity(components[i])
				if pErr != nil {
					switch i {
					case 0:
						bc.comp1.SetText("Parity Error")
						bc.comp1KCV.SetText("KCV: Error")
					case 1:
						bc.comp2.SetText("Parity Error")
						bc.comp2KCV.SetText("KCV: Error")
					case 2:
						bc.comp3.SetText("Parity Error")
						bc.comp3KCV.SetText("KCV: Error")
					}
					components[i] = ""
				} else {
					components[i] = compHex
				}
			}
		}

		if len(components) > 0 {
			bc.comp1.SetText(strings.ToUpper(components[0]))
			data1, err1 := hex.DecodeString(components[0])
			if err1 == nil && len(data1) > 0 {
				kcv1 := hsm.CalculateKCV(data1)
				bc.comp1KCV.SetText("KCV: " + strings.ToUpper(hex.EncodeToString(kcv1)))
			} else {
				bc.comp1KCV.SetText("KCV:")
				if err1 != nil {
					bc.comp1KCV.SetText("KCV: Invalid")
				}
			}
		}
		if len(components) > 1 {
			bc.comp2.SetText(strings.ToUpper(components[1]))
			data2, err2 := hex.DecodeString(components[1])
			if err2 == nil && len(data2) > 0 {
				kcv2 := hsm.CalculateKCV(data2)
				bc.comp2KCV.SetText("KCV: " + strings.ToUpper(hex.EncodeToString(kcv2)))
			} else {
				bc.comp2KCV.SetText("KCV:")
				if err2 != nil {
					bc.comp2KCV.SetText("KCV: Invalid")
				}
			}
		}

		if num == 3 {
			if len(components) > 2 {
				bc.comp3.SetText(strings.ToUpper(components[2]))
				data3, err3 := hex.DecodeString(components[2])
				if err3 == nil && len(data3) > 0 {
					kcv3 := hsm.CalculateKCV(data3)
					bc.comp3KCV.SetText("KCV: " + strings.ToUpper(hex.EncodeToString(kcv3)))
				} else {
					bc.comp3KCV.SetText("KCV:")
					if err3 != nil {
						bc.comp3KCV.SetText("KCV: Invalid")
					}
				}
			} else {
				bc.comp3.SetText("")
				bc.comp3KCV.SetText("KCV:")
			}
		}
		bc.onNumComponentsChanged(bc.numComponents.Selected)

		bc.container.Refresh()
	}
}

// clearKeySharingFields clears all input and KCV fields in key sharing mode.
func (bc *BitwiseCalculator) clearKeySharingFields() {
	bc.combinedKey.SetText("")
	bc.comp1.SetText("")
	bc.comp2.SetText("")
	bc.comp3.SetText("")
	bc.clearKCVs()
}

// clearKCVs resets all KCV labels.
func (bc *BitwiseCalculator) clearKCVs() {
	bc.combinedKCV.SetText("KCV:")
	bc.comp1KCV.SetText("KCV:")
	bc.comp2KCV.SetText("KCV:")
	bc.comp3KCV.SetText("KCV:")
}

// onNumComponentsChanged handles visibility of component 3 inputs.
func (bc *BitwiseCalculator) onNumComponentsChanged(value string) {
	if value == "3" {
		bc.comp3Label.Show()
		bc.comp3.Show()
		bc.comp3KCV.Show()
	} else {
		bc.comp3Label.Hide()
		bc.comp3.Hide()
		bc.comp3KCV.Hide()
	}

	if bc.container != nil {
		bc.container.Refresh()
	}
}

// enforceOddParity sets odd parity bit for each byte in the hex string.
func enforceOddParity(hexStr string) (string, error) {
	data, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", err
	}
	for i := range data {
		b := data[i]
		high := b >> 1
		ones := bits.OnesCount8(high)
		if ones%2 == 0 {
			data[i] = (b & 0xFE) | 1
		} else {
			data[i] = b & 0xFE
		}
	}

	return hex.EncodeToString(data), nil
}

// CreateRenderer implements fyne.Widget interface.
func (bc *BitwiseCalculator) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(bc.container)
}

// MinSize implements fyne.CanvasObject interface.
func (bc *BitwiseCalculator) MinSize() fyne.Size {
	return bc.container.MinSize()
}

// Move implements fyne.CanvasObject interface.
func (bc *BitwiseCalculator) Move(pos fyne.Position) {
	bc.container.Move(pos)
}

// Position implements fyne.CanvasObject interface.
func (bc *BitwiseCalculator) Position() fyne.Position {
	return bc.container.Position()
}

// Size implements fyne.CanvasObject interface.
func (bc *BitwiseCalculator) Size() fyne.Size {
	return bc.container.Size()
}

// Show implements fyne.CanvasObject interface.
func (bc *BitwiseCalculator) Show() {
	bc.container.Show()
}

// Hide implements fyne.CanvasObject interface.
func (bc *BitwiseCalculator) Hide() {
	bc.container.Hide()
}

// Visible implements fyne.CanvasObject interface.
func (bc *BitwiseCalculator) Visible() bool {
	return bc.container.Visible()
}

// Refresh implements fyne.CanvasObject interface.
func (bc *BitwiseCalculator) Refresh() {
	bc.container.Refresh()
}

// Cleanup implements TabContent interface.
func (bc *BitwiseCalculator) Cleanup() {
	bc.blockA.SetText("")
	bc.blockB.SetText("")
	bc.result.SetText("")

	bc.clearKeySharingFields()

	bc.numComponents.SetSelected("2")
	bc.parityBits.SetSelected("Ignore")
}
