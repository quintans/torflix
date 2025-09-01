package viewmodel

import (
	"github.com/anacrolix/torrent"
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/bind"
	"github.com/quintans/torflix/internal/lib/slices"
)

type DownloadList struct {
	shared        *Shared
	params        app.DownloadListParams
	service       DownloadService
	originalQuery string
	FileItems     bind.Notifier[[]*FileItem]
}

type FileItem struct {
	Selected bool
	File     *torrent.File
}

func NewDownloadList(shared *Shared, service DownloadService, params app.DownloadListParams) *DownloadList {
	d := &DownloadList{
		shared:    shared,
		service:   service,
		params:    params,
		FileItems: bind.NewNotifier[[]*FileItem](),
	}

	fileItems := slices.Map(params.Files, func(it *torrent.File) *FileItem {
		return &FileItem{
			Selected: it.BytesCompleted() >= it.Length(),
			File:     it,
		}
	})
	d.FileItems.Notify(fileItems)

	return d
}

func (d *DownloadList) Unmount() {
	d.FileItems.UnbindAll()
}

func (d *DownloadList) Back() {
	d.service.Close()
	d.shared.Navigate.Back()
}

func (d *DownloadList) Select(item *FileItem) {
	item.Selected = true
	d.shared.Navigate.To(app.DownloadParams{
		FileToPlay:          item.File,
		PauseTorrentOnClose: true,
		ResourceName:        d.originalQuery,
		Subtitles:           d.params.Subtitles,
	})
}
