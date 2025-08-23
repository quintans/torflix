package components

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type PillChoice struct {
	widget.BaseWidget

	text      *canvas.Text
	rectangle *canvas.Rectangle
	selected  bool

	OnSelected func(bool) `json:"-"`
}

func NewPillChoice(text string, selected bool) *PillChoice {
	p := &PillChoice{
		text:      canvas.NewText(text, nil),
		rectangle: canvas.NewRectangle(color.Transparent),
		selected:  selected,
	}
	p.ExtendBaseWidget(p)

	p.updateSelection()
	return p
}

func (p *PillChoice) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewCenter(p.rectangle, p.text))
}

func (p *PillChoice) SetText(text string) {
	p.text.Text = text
	p.updateSelection()

	p.Refresh()
}

func (p *PillChoice) Text() string {
	return p.text.Text
}

func (p *PillChoice) SetSelect(selected bool) {
	p.selected = selected
	p.updateSelection()

	p.Refresh()
}

func (p *PillChoice) Selected() bool {
	return p.selected
}

func (p *PillChoice) Tapped(_ *fyne.PointEvent) {
	p.selected = !p.selected
	p.updateSelection()

	if p.OnSelected != nil {
		p.OnSelected(p.selected)
	}

	p.rectangle.Refresh()
}

func (p *PillChoice) updateSelection() {
	var c color.Color
	if p.selected {
		c = theme.Color(theme.ColorNamePrimary)
	} else {
		c = theme.Color(theme.ColorNameDisabled)
	}
	p.rectangle.CornerRadius = 10
	p.rectangle.StrokeColor = c
	p.rectangle.StrokeWidth = 1
	p.rectangle.FillColor = c

	ms := p.text.MinSize()
	p.rectangle.SetMinSize(fyne.NewSize(ms.Width+10, ms.Height+5))
}

func (p *PillChoice) Refresh() {
	p.BaseWidget.Refresh()
	p.text.Refresh()
	p.rectangle.Refresh()
}
