package ui

import (
	"github.com/andrei-cloud/hsmtool/internal/ui/tabs"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

const (
	appTitle  = "HSM Key Management Tool"
	appWidth  = 1024
	appHeight = 768
)

// StartApp initializes and runs the main application window.
func StartApp() {
	application := app.New()
	mainWindow := application.NewWindow(appTitle)

	// Create settings tab with HSM connection first
	settingsTab := tabs.NewSettings()

	// Create tab container with all app tabs
	tabContainer := container.NewAppTabs(
		container.NewTabItemWithIcon("Key Manager", theme.HomeIcon(), tabs.NewKeyManager()),
		container.NewTabItemWithIcon(
			"DES Calculator",
			theme.ConfirmIcon(),
			tabs.NewDESCalculator(),
		),
		container.NewTabItem("Bitwise Calculator", tabs.NewBitwiseCalculator()),
		container.NewTabItemWithIcon(
			"HSM Command",
			theme.FileIcon(),
			tabs.NewHSMCommandSender(settingsTab.GetConnection()),
		),
		container.NewTabItemWithIcon("Settings", theme.SettingsIcon(), settingsTab),
	)
	tabContainer.SetTabLocation(container.TabLocationTop)

	// Set window content and size.
	mainWindow.SetContent(tabContainer)
	mainWindow.Resize(fyne.NewSize(appWidth, appHeight))
	mainWindow.CenterOnScreen()

	mainWindow.SetOnClosed(func() {
		// Clean up HSM connection on exit
		if conn := settingsTab.GetConnection(); conn != nil {
			conn.Disconnect()
		}
	})

	mainWindow.SetMaster()
	mainWindow.Show()
	application.Run()
}
