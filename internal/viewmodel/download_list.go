package viewmodel

import (
	"github.com/anacrolix/torrent"
	"github.com/quintans/torflix/internal/lib/bind"
	"github.com/quintans/torflix/internal/lib/slices"
)

type DownloadList struct {
	root          *ViewModel
	service       DownloadService
	originalQuery string
	FileItems     *bind.Bind[[]*FileItem]
}

type FileItem struct {
	Selected bool
	File     *torrent.File
}

func NewDownloadList(service DownloadService) *DownloadList {
	return &DownloadList{
		service:   service,
		FileItems: bind.NewNotifier[[]*FileItem](),
	}
}

func (d *DownloadList) Back() {
	d.FileItems.Clear()
	d.originalQuery = ""
}

func (d *DownloadList) Init(files []*torrent.File, originalQuery string) {
	fileItems := slices.Map(files, func(it *torrent.File) *FileItem {
		return &FileItem{
			Selected: it.BytesCompleted() >= it.Length(),
			File:     it,
		}
	})
	d.FileItems.Notify(fileItems)
	d.originalQuery = originalQuery

	d.root.App.EscapeKey.Notify(func() {
		d.Back()
	})
}

func (d *DownloadList) Select(item *FileItem) {
	item.Selected = true
	d.root.Download.Init(item.File, true, d.originalQuery)
}
