package main

import (
	"log"

	"adb-gui/internal/adb"
	"adb-gui/internal/config"
	"adb-gui/internal/ui"

	"fyne.io/fyne/v2/app"
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

	// Start GUI. The App ID must be set here for the fyne tool to work correctly.
	a := app.NewWithID("io.github.lx19990999.adb-gui")

	// Apply theme from config (system by default)
	ui.ApplyThemeMode(cfg.ThemeMode)

	w := a.NewWindow("ADB GUI")
	w.Resize(ui.DefaultWindowSize())

	// Build UI and run
	ui.BuildUI(w, a, mgr, cfg)
	w.ShowAndRun()
}