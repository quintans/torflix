package view

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/quintans/torflix/internal/components"
	"github.com/quintans/torflix/internal/lib/navigation"
	"github.com/quintans/torflix/internal/viewmodel"
)

func Search(vm *viewmodel.ViewModel, navigator *navigation.Navigator[*viewmodel.ViewModel]) (fyne.CanvasObject, func(bool)) {
	searchBtn := widget.NewButton("Search", nil)
	searchBtn.Importance = widget.HighImportance

	query := widget.NewEntry()
	query.SetPlaceHolder("Enter search...")

	vm.Search.Query.Bind(query.SetText)
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
	unbindSearchProviders := vm.Search.SelectedProviders.Bind(func(selectedProviders map[string]bool) {
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

	data := make([]components.MagnetItem, 0)
	result := widget.NewList(
		func() int {
			return len(data)
		},
		func() fyne.CanvasObject {
			return components.NewMagnetListItem()
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*components.MagnetListItem).SetData(data[i])
		},
	)
	result.OnSelected = func(id widget.ListItemID) {
		result.Hide()
		data[id].Cached = true
		nav := vm.Search.Download(data[id].Magnet)
		if navigate(vm, navigator, nav) {
			return
		}
		result.Unselect(id)
		result.Show()
	}

	searchBtn.OnTapped = func() {
		searchBtn.Disable()

		result.UnselectAll()
		result.Refresh()

		nav := vm.Search.Search()
		if navigate(vm, navigator, nav) {
			return
		}
		searchBtn.Enable()
	}

	unbindSearchResult := vm.Search.SearchResults.Bind(func(results []*viewmodel.SearchData) {
		items := make([]components.MagnetItem, len(results))
		for i, r := range results {
			items[i] = components.MagnetItem{
				Provider: r.Provider,
				Name:     r.Name,
				Size:     r.Size,
				Seeds:    strconv.Itoa(r.Seeds),
				Magnet:   r.Magnet,
				Cached:   r.Cached,
				Quality:  r.QualityName,
			}
		}

		data = items
		result.Refresh()
	})

	unbindClearCache := vm.App.CacheCleared.Bind(func(bool) {
		for i := range data {
			data[i].Cached = false
		}
		result.Refresh()
	})

	vm.Search.Init()

	return container.NewBorder(
			container.NewBorder(nil, pillContainer, nil, searchBtn, query),
			nil,
			nil,
			nil,
			result,
		), func(bool) {
			// Handle exit
			unbindSearchProviders()
			unbindSearchResult()
			unbindClearCache()
		}
}

func navigate(vm *viewmodel.ViewModel, navigator *navigation.Navigator[*viewmodel.ViewModel], destination viewmodel.DownloadType) bool {
	switch destination {
	case viewmodel.DownloadSingle:
		navigator.To(vm, Download)
	case viewmodel.DownloadMultiple:
		navigator.To(vm, DownloadList)
	default:
		return false
	}
	return true
}
