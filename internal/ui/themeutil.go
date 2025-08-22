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
		// Windows: 尝试检测注册表或环境变量
		// 检查环境变量
		if v := os.Getenv("APPDATA"); v != "" {
			// 检查是否在AppData\Roaming\Microsoft\Windows\Themes\appsUseLightTheme
			// 但由于权限问题，这里使用环境变量作为备选方案
			if strings.Contains(strings.ToLower(v), "dark") {
				return true, true
			}
		}
		// 检查Windows 10+的深色模式环境变量
		if v := os.Getenv("APPS_DEFAULT_THEME"); v != "" {
			if strings.Contains(strings.ToLower(v), "dark") {
				return true, true
			}
		}
		// 默认返回浅色主题（Windows传统默认）
		return false, true
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
		// 应用浅色主题的自适应主题
		lightTheme := newAdaptiveTheme(theme.LightTheme())
		set(lightTheme)
	case "dark":
		// 应用深色主题的自适应主题
		darkTheme := newAdaptiveTheme(theme.DarkTheme())
		set(darkTheme)
	default: // "system" or unknown
		if dark, known := detectSystemDark(); known {
			if dark {
				// 系统深色模式：应用深色主题的自适应主题
				darkTheme := newAdaptiveTheme(theme.DarkTheme())
				set(darkTheme)
			} else {
				// 系统浅色模式：应用浅色主题的自适应主题
				lightTheme := newAdaptiveTheme(theme.LightTheme())
				set(lightTheme)
			}
		} else {
			// 无法检测：默认使用浅色主题的自适应主题
			lightTheme := newAdaptiveTheme(theme.LightTheme())
			set(lightTheme)
		}
	}
}
