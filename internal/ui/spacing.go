package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

const (
	CardPad       float32 = 16
	PanelPad      float32 = 10
	SectionGap    float32 = 10
	InlineGap     float32 = 10
	IconSize      float32 = 40
	IconGlyphSize float32 = 18
)

// GUIWindowSize tamanho inicial da janela do ouvinte.
func GUIWindowSize() fyne.Size { return fyne.NewSize(480, 400) }

// GUISectionGap espaçamento vertical entre blocos do ouvinte.
const GUISectionGap float32 = 8

// StudioWindowSize tamanho inicial da janela do estúdio.
func StudioWindowSize() fyne.Size { return fyne.NewSize(520, 600) }

// HGap espaço horizontal entre ícone e texto.
func HGap(w float32) fyne.CanvasObject {
	s := canvas.NewRectangle(color.Transparent)
	s.SetMinSize(fyne.NewSize(w, 0))
	return s
}

// VGap espaço vertical entre blocos.
func VGap(h float32) fyne.CanvasObject {
	s := canvas.NewRectangle(color.Transparent)
	s.SetMinSize(fyne.NewSize(0, h))
	return s
}

// SpacedVBox empilha elementos com intervalo uniforme.
func SpacedVBox(spacing float32, items ...fyne.CanvasObject) fyne.CanvasObject {
	if len(items) == 0 {
		return container.NewVBox()
	}
	objs := make([]fyne.CanvasObject, 0, len(items)*2-1)
	for i, item := range items {
		if i > 0 {
			objs = append(objs, VGap(spacing))
		}
		objs = append(objs, item)
	}
	return container.NewVBox(objs...)
}

// Pad aplica margem interna uniforme ao conteúdo.
func Pad(pad float32, content fyne.CanvasObject) fyne.CanvasObject {
	top := VGap(pad)
	left := HGap(pad)
	right := HGap(pad)
	bottom := VGap(pad)
	return container.NewBorder(top, bottom, left, right, content)
}
