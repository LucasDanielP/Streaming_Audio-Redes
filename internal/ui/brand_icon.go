package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// brandIcon círculo com glifo centralizado (tamanho fixo).
type brandIcon struct {
	widget.BaseWidget
	bg     *canvas.Rectangle
	glyph  *canvas.Text
	accent bool
}

func newBrandIcon(accent bool) *brandIcon {
	b := &brandIcon{accent: accent}
	b.ExtendBaseWidget(b)
	return b
}

func (b *brandIcon) CreateRenderer() fyne.WidgetRenderer {
	bgColor := ColorPrimary
	fgColor := ColorPrimaryText
	if b.accent {
		bgColor = ColorAccent
		fgColor = ColorAccentText
	}

	b.bg = canvas.NewRectangle(bgColor)
	b.bg.CornerRadius = IconSize / 2

	b.glyph = canvas.NewText("◉", fgColor)
	b.glyph.TextSize = IconGlyphSize
	b.glyph.TextStyle = fyne.TextStyle{Bold: true}
	b.glyph.Alignment = fyne.TextAlignCenter

	return &brandIconRenderer{icon: b}
}

type brandIconRenderer struct {
	icon *brandIcon
}

func (r *brandIconRenderer) Layout(size fyne.Size) {
	r.icon.bg.Resize(size)
	r.icon.bg.Move(fyne.NewPos(0, 0))

	glyphSize := r.icon.glyph.MinSize()
	r.icon.glyph.Resize(glyphSize)
	r.icon.glyph.Move(fyne.NewPos(
		(size.Width-glyphSize.Width)/2,
		(size.Height-glyphSize.Height)/2,
	))
}

func (r *brandIconRenderer) MinSize() fyne.Size {
	return fyne.NewSize(IconSize, IconSize)
}

func (r *brandIconRenderer) Refresh() {
	canvas.Refresh(r.icon.bg)
	canvas.Refresh(r.icon.glyph)
}

func (r *brandIconRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.icon.bg, r.icon.glyph}
}

func (r *brandIconRenderer) Destroy() {}
