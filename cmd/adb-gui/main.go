package main

import (
	"log"

	"adb-gui/internal/adb"
	"adb-gui/internal/config"
	"adb-gui/internal/ui"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/theme"
)

func main() {
	// Load persisted config
	cfg, err := config.Load()
	if err != nil {
		log.Printf("failed to load config: %v", err)
	}

	// Initialize ADB manager with configured path (or autodetect)
	mgr := adb.NewManager(cfg.ADBPath)
	// Ensure adb server is running to avoid empty device lists on first call
	mgr.EnsureServer()

	// Start GUI
	a := app.NewWithID("adb-gui")

	// Apply theme from config (system by default)
	mode := cfg.ThemeMode
	switch mode {
	case "dark":
		a.Settings().SetTheme(theme.DarkTheme())
	case "light":
		a.Settings().SetTheme(theme.LightTheme())
	default:
		// Follow system preference
		a.Settings().SetTheme(theme.DefaultTheme())
	}

	w := a.NewWindow("ADB GUI")
	w.Resize(ui.DefaultWindowSize())

	// Build UI and run
	ui.BuildUI(w, a, mgr, cfg)
	w.ShowAndRun()
}