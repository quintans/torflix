package view

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"
	"github.com/quintans/torflix/internal/lib/navigation"
	"github.com/quintans/torflix/internal/viewmodel"
)

type DownloadListController interface {
	Back()
	PlayFile(idx int)
}

func DownloadList(vm *viewmodel.ViewModel, navigator *navigation.Navigator[*viewmodel.ViewModel]) (fyne.CanvasObject, func(bool)) {
	var fileItems []*viewmodel.FileItem
	result := widget.NewList(
		func() int {
			return len(fileItems)
		},
		func() fyne.CanvasObject {
			nameLbl := widget.NewLabel("")
			nameLbl.Alignment = fyne.TextAlignLeading
			sizeLbl := widget.NewLabel("")
			sizeLbl.Alignment = fyne.TextAlignTrailing
			return container.NewBorder(nil, nil, nil, sizeLbl, nameLbl)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			it := o.(*fyne.Container)
			name := it.Objects[0].(*widget.Label)
			name.SetText(fileItems[i].File.DisplayPath())
			size := it.Objects[1].(*widget.Label)
			size.SetText(humanize.Bytes(uint64(fileItems[i].File.Length())))
			if fileItems[i].Selected {
				name.Importance = widget.HighImportance
				size.Importance = widget.HighImportance
			} else {
				name.Importance = widget.MediumImportance
				size.Importance = widget.MediumImportance
			}
		},
	)
	result.OnSelected = func(id widget.ListItemID) {
		vm.DownloadList.Select(fileItems[id])
		navigator.To(vm, Download)
		// result.Unselect(id)
		// result.Refresh()
	}

	unbindFileItems := vm.DownloadList.FileItems.Bind(func(items []*viewmodel.FileItem) {
		fileItems = items
		result.Refresh()
	})

	return container.NewBorder(
			nil,
			container.NewHBox(layout.NewSpacer(), widget.NewButton("Back", func() {
				vm.DownloadList.Back()
				navigator.Back(vm)
			})),
			nil,
			nil,
			result,
		), func(bool) {
			unbindFileItems()
		}
}
