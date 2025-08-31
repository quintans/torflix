package view

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/quintans/torflix/internal/components"
	"github.com/quintans/torflix/internal/model"
	"github.com/quintans/torflix/internal/viewmodel"
)

func buildSearch(vm *viewmodel.App) fyne.CanvasObject {
	searchBtn := widget.NewButton("SEARCH", nil)
	searchBtn.Importance = widget.HighImportance

	query := widget.NewEntry()
	query.SetPlaceHolder("Enter search...")

	vm.Search.Query.BindPtr(&query.Text)
	query.OnChanged = func(text string) {
		vm.Search.Query.Set(text)
		if text == "" {
			searchBtn.Disable()
		} else {
			searchBtn.Enable()
		}
	}
	query.OnSubmitted = func(text string) {
		if query.Text == "" {
			return
		}
		searchBtn.OnTapped()
	}

	var pills []*components.PillChoice
	pillContainer := container.NewHBox()

	vm.Search.SelectedProviders.Bind(func(selectedProviders map[string]bool) {
		providers := vm.Search.Providers
		pills = make([]*components.PillChoice, 0, len(providers))

		for _, v := range providers {
			selected := selectedProviders[v]

			pill := components.NewPillChoice(v, selected)
			pill.OnSelected = func(selected bool) {
				if selected {
					selectedProviders[v] = selected
				} else {
					delete(selectedProviders, v)
				}
				vm.Search.SelectedProviders.Set(selectedProviders)
			}
			pills = append(pills, pill)
			pillContainer.Add(pill)
		}
	})

	subtitles := widget.NewCheck("Download Subtitles", nil)
	vm.Search.DownloadSubtitles.BindPtr(&subtitles.Checked)
	subtitles.OnChanged = vm.Search.DownloadSubtitles.Set

	var data []*viewmodel.SearchData
	result := widget.NewList(
		func() int {
			return len(data)
		},
		func() fyne.CanvasObject {
			return components.NewMagnetListItem()
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			r := data[i]
			o.(*components.MagnetListItem).SetData(&components.MagnetItem{
				Provider: r.Provider,
				Name:     r.Name,
				Size:     r.Size,
				Seeds:    strconv.Itoa(r.Seeds),
				Magnet:   r.Magnet,
				Cached:   r.Cached,
				Quality:  r.QualityName,
			})
		},
	)
	result.OnSelected = func(id widget.ListItemID) {
		result.Hide()
		d := data[id]
		d.Cached = true
		vm.Cache.Add(vm.Search.OriginalQuery, &model.CacheData{
			Provider: d.Provider,
			Name:     d.Name,
			Magnet:   d.Magnet,
			Size:     d.Size,
			Seeds:    strconv.Itoa(d.Seeds),
			Quality:  d.QualityName,
			Hash:     d.Hash,
		})
		go vm.Search.Download(d.Magnet)
	}

	searchBtn.OnTapped = func() {
		searchBtn.Disable()

		result.UnselectAll()
		result.Refresh()

		go func() {
			if !vm.Search.SearchAsync() {
				fyne.DoAndWait(func() {
					searchBtn.Enable()
				})
			}
		}()
	}

	vm.Search.SearchResults.Bind(func(results []*viewmodel.SearchData) {
		data = results
		searchBtn.Enable()
		result.Show()
		result.UnselectAll()
		result.Refresh()
	})

	vm.Cache.CacheCleared.Listen(func(bool) {
		for i := range data {
			data[i].Cached = false
		}
		result.Refresh()
	})

	options := container.NewVBox(pillContainer, subtitles)

	return container.NewBorder(
		container.NewBorder(nil, options, nil, searchBtn, query),
		nil,
		nil,
		nil,
		result,
	)
}
