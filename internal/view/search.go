package view

import (
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/components"
	"github.com/quintans/torflix/internal/model"
)

type SearchController interface {
	Download(originalQuery string, magnetLink string) error
	Search(query string, providers []string) ([]components.MagnetItem, error)
	SearchModel() (*model.Search, error)
}

type Search struct {
	query  *widget.Entry
	result *widget.List

	controller SearchController
	data       []components.MagnetItem
	showViewer ShowViewer
}

func NewSearch(showViewer ShowViewer) *Search {
	return &Search{
		showViewer: showViewer,
	}
}

func (v *Search) SetController(controller SearchController) {
	v.controller = controller
}

func (v *Search) Show(searchModel *model.Search, providers []string) {
	search := widget.NewButton("Search", nil)
	search.Importance = widget.HighImportance

	v.query = widget.NewEntry()
	v.query.Text = searchModel.Query()
	v.query.SetPlaceHolder("Enter search...")

	originalQuery := searchModel.Query()

	if v.query.Text == "" {
		search.Disable()
	}

	v.query.OnChanged = func(text string) {
		if text == "" {
			search.Disable()
		} else {
			search.Enable()
		}
	}
	v.query.OnSubmitted = func(text string) {
		if v.query.Text == "" {
			return
		}
		search.OnTapped()
	}

	selectedProviders := searchModel.SelectedProviders()
	pills := make([]*components.PillChoice, 0, len(providers))

	pillCont := container.NewHBox()
	for _, v := range providers {
		selected, ok := selectedProviders[v]
		if !ok {
			selected = true
		}
		pill := components.NewPillChoice(v, selected)
		pills = append(pills, pill)
		pillCont.Add(pill)
	}

	search.OnTapped = func() {
		search.Disable()

		v.data = nil
		v.result.Refresh()
		providers := []string{}
		for _, p := range pills {
			if p.Selected() {
				providers = append(providers, p.Text())
			}
		}

		originalQuery = v.query.Text
		items, err := v.controller.Search(v.query.Text, providers)
		if err != nil {
			slog.Error("Failed to search.", "error", err.Error())
		} else {
			v.data = items
			v.result.Refresh()
		}

		search.Enable()
	}

	v.result = widget.NewList(
		func() int {
			return len(v.data)
		},
		func() fyne.CanvasObject {
			return components.NewMagnetListItem()
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*components.MagnetListItem).SetData(v.data[i])
		},
	)
	v.result.OnSelected = func(id widget.ListItemID) {
		v.result.Hide()
		v.data[id].Cached = true
		err := v.controller.Download(originalQuery, v.data[id].Magnet)
		if err != nil {
			slog.Error("Failed to download.", "error", err.Error())
		}
		v.result.Show()
	}

	v.showViewer.ShowView(container.NewBorder(
		container.NewBorder(nil, pillCont, nil, search, v.query),
		nil,
		nil,
		nil,
		v.result,
	))
}

func (v *Search) OnExit() {
	v.query = nil
	v.result = nil
}

func (v *Search) ClearCache(_ app.ClearCache) {
	for i := range v.data {
		v.data[i].Cached = false
	}
	v.result.Refresh()
}
