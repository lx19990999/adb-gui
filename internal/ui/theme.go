package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// 智能自适应主题
type adaptiveTheme struct {
	baseTheme fyne.Theme
}

var _ fyne.Theme = (*adaptiveTheme)(nil)

func (t *adaptiveTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	// 获取基础主题颜色
	baseColor := t.baseTheme.Color(name, variant)

	// 特殊处理某些颜色名称
	switch name {
	case theme.ColorNameForeground:
		// 前景色：深色主题用白色，浅色主题用深色
		if variant == theme.VariantLight {
			return color.NRGBA{R: 33, G: 33, B: 33, A: 255} // 深灰色，接近黑色
		}
		return color.NRGBA{R: 245, G: 245, B: 245, A: 255} // 浅灰色，接近白色

	case theme.ColorNameBackground:
		// 背景色：深色主题用深色，浅色主题用浅色
		if variant == theme.VariantLight {
			return color.NRGBA{R: 250, G: 250, B: 250, A: 255} // 浅灰白色
		}
		return color.NRGBA{R: 40, G: 40, B: 40, A: 255} // 深灰色

	case theme.ColorNameButton:
		// 按钮颜色：深色主题用深色，浅色主题用浅色
		if variant == theme.VariantLight {
			return color.NRGBA{R: 240, G: 240, B: 240, A: 255} // 浅灰色
		}
		return color.NRGBA{R: 60, G: 60, B: 60, A: 255} // 深灰色

	case theme.ColorNameHover:
		// 悬停颜色：深色主题用亮色，浅色主题用暗色
		if variant == theme.VariantLight {
			return color.NRGBA{R: 220, G: 220, B: 220, A: 255} // 稍暗的灰色
		}
		return color.NRGBA{R: 80, G: 80, B: 80, A: 255} // 稍亮的灰色

	case theme.ColorNamePressed:
		// 按下颜色：深色主题用亮色，浅色主题用暗色
		if variant == theme.VariantLight {
			return color.NRGBA{R: 200, G: 200, B: 200, A: 255} // 更暗的灰色
		}
		return color.NRGBA{R: 100, G: 100, B: 100, A: 255} // 更亮的灰色

	case theme.ColorNameInputBackground:
		// 输入框背景：深色主题用深色，浅色主题用白色
		if variant == theme.VariantLight {
			return color.White
		}
		return color.NRGBA{R: 50, G: 50, B: 50, A: 255} // 深灰色

	case theme.ColorNamePlaceHolder:
		// 占位符颜色：深色主题用浅色，浅色主题用深色
		if variant == theme.VariantLight {
			return color.NRGBA{R: 150, G: 150, B: 150, A: 255} // 中等灰色
		}
		return color.NRGBA{R: 180, G: 180, B: 180, A: 255} // 浅灰色

	case theme.ColorNameScrollBar:
		// 滚动条颜色：深色主题用浅色，浅色主题用深色
		if variant == theme.VariantLight {
			return color.NRGBA{R: 180, G: 180, B: 180, A: 255} // 中等灰色
		}
		return color.NRGBA{R: 120, G: 120, B: 120, A: 255} // 浅灰色

	case theme.ColorNameShadow:
		// 阴影颜色：深色主题用亮色，浅色主题用暗色
		if variant == theme.VariantLight {
			return color.NRGBA{R: 0, G: 0, B: 0, A: 30} // 半透明黑色
		}
		return color.NRGBA{R: 255, G: 255, B: 255, A: 30} // 半透明白色
	}

	return baseColor
}

func (t *adaptiveTheme) Font(style fyne.TextStyle) fyne.Resource {
	return t.baseTheme.Font(style)
}

func (t *adaptiveTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return t.baseTheme.Icon(name)
}

func (t *adaptiveTheme) Size(name fyne.ThemeSizeName) float32 {
	return t.baseTheme.Size(name)
}

// 创建新的自适应主题
func newAdaptiveTheme(base fyne.Theme) fyne.Theme {
	return &adaptiveTheme{baseTheme: base}
}

// 获取文件列表项的颜色（根据主题自适应）
func getFileItemColor(isDir bool, variant fyne.ThemeVariant) color.Color {
	if isDir {
		// 目录颜色：深色主题用蓝色，浅色主题用深蓝色
		if variant == theme.VariantLight {
			return color.NRGBA{R: 0, G: 100, B: 200, A: 255} // 深蓝色
		}
		return color.NRGBA{R: 100, G: 150, B: 255, A: 255} // 亮蓝色
	} else {
		// 文件颜色：深色主题用白色，浅色主题用深色
		if variant == theme.VariantLight {
			return color.NRGBA{R: 33, G: 33, B: 33, A: 255} // 深灰色，接近黑色
		}
		return color.NRGBA{R: 245, G: 245, B: 245, A: 255} // 浅灰色，接近白色
	}
}

// 获取文本颜色（根据主题自适应）
func getTextColor(variant fyne.ThemeVariant) color.Color {
	if variant == theme.VariantLight {
		return color.NRGBA{R: 33, G: 33, B: 33, A: 255} // 深灰色，接近黑色
	}
	return color.NRGBA{R: 245, G: 245, B: 245, A: 255} // 浅灰色，接近白色
}

// isDarkColor 判断颜色是否为深色
func isDarkColor(c color.Color) bool {
	// 转换为RGBA
	r, g, b, _ := c.RGBA()
	// 计算亮度 (使用标准亮度公式)
	luminance := 0.299*float64(r)/65535 + 0.587*float64(g)/65535 + 0.114*float64(b)/65535
	return luminance < 0.5
}
