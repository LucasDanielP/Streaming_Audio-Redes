package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

const (
	StationName  = "Back2Past"
	StationFreq  = "88.5"
	StationCity  = "UFES"
	StationGenre = "Nostalgia"
	FooterText   = "Back2Past · Transmissão contínua"
)

// BrandHeader cabeçalho com ícone dimensionado e texto alinhado.
func BrandHeader(title, subtitle string, accent bool) fyne.CanvasObject {
	return brandHeader(title, subtitle, accent, nil)
}

// BrandHeaderRow igual ao BrandHeader, com elemento à direita (ex.: badge).
func BrandHeaderRow(title, subtitle string, accent bool, trailing fyne.CanvasObject) fyne.CanvasObject {
	return brandHeader(title, subtitle, accent, trailing)
}

func brandHeader(title, subtitle string, accent bool, trailing fyne.CanvasObject) fyne.CanvasObject {
	titleLabel := canvas.NewText(title, ColorForeground)
	titleLabel.TextSize = 22
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	subtitleLabel := canvas.NewText(subtitle, ColorMuted)
	subtitleLabel.TextSize = 10
	subtitleLabel.TextStyle = fyne.TextStyle{Monospace: true}

	textCol := container.NewVBox(titleLabel, VGap(2), subtitleLabel)
	iconWrap := container.NewBorder(VGap(1), VGap(1), nil, nil, newBrandIcon(accent))
	left := container.NewHBox(iconWrap, HGap(InlineGap), textCol)

	if trailing == nil {
		return left
	}
	return container.NewBorder(nil, nil, left, trailing, nil)
}

// MonoCaption rótulo em caixa alta com espaçamento (estilo painel FM).
func MonoCaption(text string) *canvas.Text {
	l := canvas.NewText(text, ColorMuted)
	l.TextSize = 10
	l.TextStyle = fyne.TextStyle{Monospace: true}
	return l
}

// DisplayPanel painel escuro para “tocando agora” / status.
func DisplayPanel(content fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(ColorDisplay)
	bg.StrokeColor = ColorSecondary
	bg.StrokeWidth = 1
	bg.CornerRadius = 8
	return container.NewStack(bg, Pad(PanelPad, content))
}

// NewOnAirBadge cria o indicador de transmissão (atualize com SetOnAir).
func NewOnAirBadge() *canvas.Text {
	l := canvas.NewText("Fora do ar", ColorMuted)
	l.TextSize = 10
	l.TextStyle = fyne.TextStyle{Monospace: true}
	return l
}

// SetOnAir atualiza o badge de transmissão.
func SetOnAir(badge *canvas.Text, onAir bool) {
	if onAir {
		badge.Text = "● No ar"
		badge.Color = ColorPrimary
		badge.TextStyle = fyne.TextStyle{Monospace: true, Bold: true}
	} else {
		badge.Text = "Fora do ar"
		badge.Color = ColorMuted
		badge.TextStyle = fyne.TextStyle{Monospace: true}
	}
	canvas.Refresh(badge)
}

// Footer rodapé do design system.
func Footer() fyne.CanvasObject {
	l := canvas.NewText(FooterText, ColorMuted)
	l.TextSize = 9
	l.TextStyle = fyne.TextStyle{Monospace: true}
	return container.NewCenter(Pad(2, l))
}

// CardWrap envolve conteúdo em cartão com borda grossa (estilo radio-player).
func CardWrap(content fyne.CanvasObject) fyne.CanvasObject {
	bg := canvas.NewRectangle(ColorCard)
	bg.StrokeColor = ColorBorder
	bg.StrokeWidth = 3
	bg.CornerRadius = 12

	accent := canvas.NewRectangle(ColorPrimary)
	accent.SetMinSize(fyne.NewSize(0, 4))

	inner := container.NewVBox(accent, Pad(CardPad, content))
	return container.NewStack(bg, inner)
}
