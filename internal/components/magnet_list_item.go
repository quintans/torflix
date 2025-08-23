package components

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

type MagnetItem struct {
	Provider string
	Name     string
	Size     string
	Seeds    string
	Quality  string
	Magnet   string
	Cached   bool
}

type MagnetListItem struct {
	widget.BaseWidget

	Provider *widget.Label
	Name     *widget.Label
	Bytes    *widget.Label
	Seeds    *widget.Label
	Quality  *Pill
	Cached   *Pill
}

func NewMagnetListItem() *MagnetListItem {
	li := &MagnetListItem{
		Provider: widget.NewLabel(""),
		Name:     widget.NewLabel(""),
		Bytes:    widget.NewLabel(""),
		Seeds:    widget.NewLabel(""),
		Cached:   NewPill("Cached"),
		Quality:  NewPill(""),
	}
	li.ExtendBaseWidget(li)
	return li
}

func (item *MagnetListItem) SetData(data *MagnetItem) {
	item.Provider.SetText("Source: " + data.Provider)
	item.Name.SetText(data.Name)
	item.Seeds.SetText("Seeds: " + data.Seeds)
	item.Bytes.SetText(data.Size)
	item.Quality.SetText(data.Quality)
	if data.Cached {
		item.Cached.Show()
	} else {
		item.Cached.Hide()
	}
}

func (item *MagnetListItem) CreateRenderer() fyne.WidgetRenderer {
	c := container.NewVBox(
		item.Name,
		container.NewHBox(
			item.Quality,
			item.Seeds,
			item.Bytes,
			layout.NewSpacer(),
			item.Cached,
			layout.NewSpacer(),
			item.Provider,
		),
	)

	r := canvas.NewRectangle(color.Transparent)
	r.CornerRadius = 5
	r.StrokeColor = color.White
	r.StrokeWidth = 1
	r.SetMinSize(c.MinSize())

	return widget.NewSimpleRenderer(container.NewStack(r, c))
}
