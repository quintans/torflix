package view

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/quintans/torflix/internal/components"
	"github.com/quintans/torflix/internal/lib/navigation"
	"github.com/quintans/torflix/internal/model"
	"github.com/quintans/torflix/internal/viewmodel"
)

func Cache(vm *viewmodel.ViewModel, navigator *navigation.Navigator[*viewmodel.ViewModel]) (fyne.CanvasObject, func(bool)) {
	vbox := container.NewVBox()

	clear := widget.NewButton("CLEAR CACHE", func() {
		vm.Cache.ClearCache()
	})
	clear.Importance = widget.WarningImportance

	vbox.Add(widget.NewLabel("Cache"))
	vbox.Add(canvas.NewLine(color.Gray{128}))
	vbox.Add(container.NewHBox(
		widget.NewForm(
			widget.NewFormItem("Directory", widget.NewLabel(vm.Cache.CacheDir)),
		),
		layout.NewSpacer(),
	))
	vbox.Add(container.NewHBox(clear, layout.NewSpacer()))
	vbox.Add(widget.NewSeparator())

	var data []*model.CacheData
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
				Seeds:    r.Seeds,
				Magnet:   r.Magnet,
				Quality:  r.Quality,
			})
		},
	)
	result.OnSelected = func(id widget.ListItemID) {
		result.Hide()
		nav := vm.Cache.Download(data[id])
		if navigate(vm, navigator, nav) {
			return
		}
		result.Unselect(id)
		result.Show()
	}

	unbindSearchResult := vm.Cache.Results.Bind(func(results []*model.CacheData) {
		data = results
		result.Refresh()
	})
	vbox.Add(result)

	vm.Cache.Mount()

	return vbox, func(bool) {
		unbindSearchResult()

		vm.Cache.Unmount()
	}
}
