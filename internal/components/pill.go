package components

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type Pill struct {
	widget.BaseWidget

	text      *canvas.Text
	rectangle *canvas.Rectangle
}

func NewPill(text string) *Pill {
	r := canvas.NewRectangle(color.RGBA{128, 128, 128, 255})
	r.CornerRadius = 10
	p := &Pill{
		text:      canvas.NewText(text, nil),
		rectangle: r,
	}
	p.ExtendBaseWidget(p)

	p.updateSelection()
	return p
}

func (p *Pill) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewPadded(container.NewCenter(p.rectangle, p.text)))
}

func (p *Pill) SetText(text string) {
	p.text.Text = text
	p.updateSelection()

	p.Refresh()
}

func (p *Pill) updateSelection() {
	ms := p.text.MinSize()
	p.rectangle.SetMinSize(fyne.NewSize(ms.Width+10, ms.Height+5))
}

func (p *Pill) Refresh() {
	p.BaseWidget.Refresh()
	p.text.Refresh()
	p.rectangle.Refresh()
}
