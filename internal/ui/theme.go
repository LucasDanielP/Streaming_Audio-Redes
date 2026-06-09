package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// RadioTheme aplica a paleta retrô do design system à interface Fyne.
type RadioTheme struct{}

var _ fyne.Theme = (*RadioTheme)(nil)

func (RadioTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground:
		return ColorBackground
	case theme.ColorNameButton:
		return ColorSecondary
	case theme.ColorNameDisabledButton:
		return ColorBorder
	case theme.ColorNameInputBackground:
		return ColorDisplay
	case theme.ColorNamePlaceHolder:
		return ColorMuted
	case theme.ColorNamePrimary:
		return ColorPrimary
	case theme.ColorNameHover:
		return ColorAccent
	case theme.ColorNameFocus:
		return ColorPrimary
	case theme.ColorNameScrollBar:
		return ColorBorder
	case theme.ColorNameShadow:
		return color.NRGBA{A: 0x80}
	case theme.ColorNameSelection:
		return ColorAccent
	case theme.ColorNameSeparator:
		return ColorBorder
	case theme.ColorNameInputBorder:
		return ColorSecondary
	case theme.ColorNameForeground:
		return ColorForeground
	case theme.ColorNameDisabled:
		return ColorMuted
	case theme.ColorNameHeaderBackground:
		return ColorCard
	default:
		return theme.DefaultTheme().Color(name, variant)
	}
}

func (t RadioTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(style)
}

func (t RadioTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t RadioTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 10
	case theme.SizeNameInnerPadding:
		return 8
	case theme.SizeNameInputBorder:
		return 1
	case theme.SizeNameScrollBar:
		return 8
	case theme.SizeNameScrollBarSmall:
		return 4
	default:
		return theme.DefaultTheme().Size(name)
	}
}

// ApplyTheme define o tema retrô no app Fyne.
func ApplyTheme(a fyne.App) {
	a.Settings().SetTheme(&RadioTheme{})
}
