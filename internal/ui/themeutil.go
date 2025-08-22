package ui

import (
	"os"
	"os/exec"
	"runtime"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// detectSystemDark tries to detect if the OS prefers dark mode.
// Returns (isDark, known). If known is false, detection was not conclusive.
func detectSystemDark() (bool, bool) {
	// Respect FYNE_THEME if set (fyne convention)
	if v := strings.ToLower(strings.TrimSpace(os.Getenv("FYNE_THEME"))); v == "dark" {
		return true, true
	} else if v == "light" {
		return false, true
	}

	switch runtime.GOOS {
	case "darwin":
		// macOS: defaults read -g AppleInterfaceStyle -> "Dark" when dark mode enabled
		out, err := exec.Command("defaults", "read", "-g", "AppleInterfaceStyle").CombinedOutput()
		if err == nil && strings.Contains(strings.ToLower(string(out)), "dark") {
			return true, true
		}
		return false, true // if the key is missing, it's light
	case "linux":
		// GNOME 42+: color-scheme -> 'prefer-dark' or 'default'
		out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "color-scheme").CombinedOutput()
		if err == nil {
			s := strings.ToLower(string(out))
			if strings.Contains(s, "prefer-dark") {
				return true, true
			}
			if strings.Contains(s, "default") || strings.Contains(s, "prefer-light") {
				return false, true
			}
		}
		// Fallback: gtk-theme name contains "-dark"
		out2, err2 := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "gtk-theme").CombinedOutput()
		if err2 == nil && strings.Contains(strings.ToLower(string(out2)), "dark") {
			return true, true
		}
		// KDE and others: no simple standard detection
		return false, false
	case "windows":
		// No registry access here; try env var only
		return false, false
	default:
		return false, false
	}
}

// ApplyThemeMode applies the theme based on mode: "light", "dark", "system".
func ApplyThemeMode(mode string) {
	app := fyne.CurrentApp()
	if app == nil {
		return
	}
	set := func(th fyne.Theme) {
		app.Settings().SetTheme(th)
	}
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "light":
		set(theme.LightTheme())
	case "dark":
		set(theme.DarkTheme())
	default: // "system" or unknown
		if dark, known := detectSystemDark(); known {
			if dark {
				set(theme.DarkTheme())
			} else {
				set(theme.LightTheme())
			}
		} else {
			// Fallback to default (usually light)
			set(theme.LightTheme())
		}
	}
}