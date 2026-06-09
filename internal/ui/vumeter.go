package ui

import (
	"image/color"
	"math"
	"math/rand"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

const vuBarCount = 15

// VuMeter barras animadas no estilo do design system (pílulas em curva).
type VuMeter struct {
	widget.BaseWidget
	bars    []*canvas.Rectangle
	levels  [vuBarCount]float32
	active  bool
	mu      sync.Mutex
	objects []fyne.CanvasObject
}

// NewVuMeter cria o medidor de nível.
func NewVuMeter() *VuMeter {
	v := &VuMeter{}
	v.ExtendBaseWidget(v)
	return v
}

// SetActive liga ou desliga a animação conforme áudio em reprodução.
func (v *VuMeter) SetActive(active bool) {
	v.mu.Lock()
	v.active = active
	v.mu.Unlock()
}

func (v *VuMeter) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(ColorDisplay)
	bg.CornerRadius = 6

	barObjs := make([]fyne.CanvasObject, 0, vuBarCount)
	v.bars = make([]*canvas.Rectangle, vuBarCount)
	for i := range v.bars {
		bar := canvas.NewRectangle(vuColorByIndex(i))
		bar.CornerRadius = 4
		v.bars[i] = bar
		barObjs = append(barObjs, bar)
	}

	barRow := container.NewWithoutLayout(barObjs...)
	v.objects = []fyne.CanvasObject{bg, Pad(8, barRow)}

	r := &vuRenderer{meter: v, bg: bg, barRow: barRow}
	go r.animate()
	return r
}

type vuRenderer struct {
	meter  *VuMeter
	bg     *canvas.Rectangle
	barRow *fyne.Container
}

func (r *vuRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.bg.Move(fyne.NewPos(0, 0))
	r.meter.objects[1].Resize(size)
	r.meter.objects[1].Move(fyne.NewPos(0, 0))
}

func (r *vuRenderer) MinSize() fyne.Size {
	return fyne.NewSize(280, 48)
}

func (r *vuRenderer) Refresh() {
	r.meter.mu.Lock()
	active := r.meter.active
	levels := r.meter.levels
	r.meter.mu.Unlock()

	innerW := r.barRow.Size().Width
	innerH := r.barRow.Size().Height
	if innerH < 32 {
		innerH = 32
	}

	barW := float32(8)
	gap := float32(4)
	totalW := float32(vuBarCount)*barW + float32(vuBarCount-1)*gap
	startX := (innerW - totalW) / 2
	if startX < 0 {
		startX = 0
	}
	baseY := innerH

	for i, bar := range r.meter.bars {
		lvl := levels[i]
		if lvl < 0.08 {
			lvl = 0.08
		}
		h := (innerH - 2) * lvl
		bar.Resize(fyne.NewSize(barW, h))
		bar.Move(fyne.NewPos(startX+float32(i)*(barW+gap), baseY-h))
		bar.CornerRadius = barW / 2
		bar.FillColor = vuColorByIndex(i)
		if !active {
			bar.FillColor = withAlpha(bar.FillColor, 0x55)
		}
	}

	r.barRow.Refresh()
	canvas.Refresh(r.meter)
}

func (r *vuRenderer) Objects() []fyne.CanvasObject {
	return r.meter.objects
}

func (r *vuRenderer) Destroy() {}

func (r *vuRenderer) animate() {
	ticker := time.NewTicker(90 * time.Millisecond)
	defer ticker.Stop()
	for range ticker.C {
		r.meter.mu.Lock()
		active := r.meter.active
		center := float32(vuBarCount-1) / 2
		for i := range r.meter.levels {
			v := r.meter.levels[i]
			var target float32
			if active {
				dist := 1 - abs32(float32(i)-center)/center
				if dist < 0 {
					dist = 0
				}
				// curva em sino + variação aleatória suave
				peak := 0.2 + 0.8*dist*dist
				target = peak * (0.75 + rand.Float32()*0.25)
			} else {
				target = 0.08 + 0.04*abs32(float32(i)-center)/center
			}
			r.meter.levels[i] = v + (target-v)*0.45
		}
		r.meter.mu.Unlock()
		fyne.Do(func() { r.Refresh() })
	}
}

// vuColorByIndex cor por posição: bege nas pontas, dourado no meio, vermelho no centro.
func vuColorByIndex(i int) color.Color {
	center := float64(vuBarCount-1) / 2
	dist := math.Abs(float64(i) - center)
	switch {
	case dist < 0.6:
		return ColorVuHigh
	case dist < 2.5:
		return ColorVuMid
	default:
		return ColorVuLow
	}
}

func withAlpha(c color.Color, a uint8) color.Color {
	r, g, b, _ := c.RGBA()
	return color.NRGBA{
		R: uint8(r >> 8),
		G: uint8(g >> 8),
		B: uint8(b >> 8),
		A: a,
	}
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
