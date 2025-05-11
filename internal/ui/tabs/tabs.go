package tabs

import (
	"fyne.io/fyne/v2"
)

// TabContent defines the interface for tab content.
type TabContent interface {
	fyne.CanvasObject
	Cleanup()
}
