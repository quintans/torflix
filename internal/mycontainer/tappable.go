package mycontainer

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type Tappable struct {
	widget.BaseWidget

	OnTapped func() `json:"-"`
	object   fyne.CanvasObject
}

func NewTappable(obj fyne.CanvasObject) *Tappable {
	t := &Tappable{
		object: obj,
	}
	t.ExtendBaseWidget(t)

	return t
}

func (t *Tappable) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(t.object)
}

func (t *Tappable) Tapped(ev *fyne.PointEvent) {
	if t.OnTapped != nil {
		t.OnTapped()
	}
}
