package view

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/anacrolix/torrent"
	"github.com/dustin/go-humanize"
	"github.com/quintans/torflix/internal/app"
)

type DownloadListController interface {
	Back()
	PlayFile(*torrent.File)
}

type DownloadList struct {
	showViewer ShowViewer
	controller DownloadListController
	eventBus   app.EventBus
}

func NewDownloadList(showViewer ShowViewer, eventBus app.EventBus) *DownloadList {
	return &DownloadList{
		showViewer: showViewer,
		eventBus:   eventBus,
	}
}

func (s *DownloadList) SetController(controller DownloadListController) {
	s.controller = controller
}

func (s *DownloadList) Show(files []*torrent.File) {
	result := widget.NewList(
		func() int {
			return len(files)
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
			it.Objects[0].(*widget.Label).SetText(files[i].DisplayPath())
			it.Objects[1].(*widget.Label).SetText(humanize.Bytes(uint64(files[i].Length())))
		},
	)
	result.OnSelected = func(id widget.ListItemID) {
		s.controller.PlayFile(files[id])
	}

	s.showViewer.ShowView(container.NewBorder(
		nil,
		container.NewHBox(layout.NewSpacer(), widget.NewButton("Back", s.controller.Back)),
		nil,
		nil,
		result,
	))
}
