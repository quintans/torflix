package viewmodel

import (
	"context"

	"github.com/anacrolix/torrent"
	"github.com/quintans/torflix/internal/app"
)

type DownloadService interface {
	DownloadTorrent(query, magnetLink string) ([]*torrent.File, error)
	DownloadSubtitles(file *torrent.File, originalQuery string) (string, int, error)
	ServeFile(ctx context.Context, asyncError app.AsyncError, file *torrent.File, originalQuery string) (string, error)
	Play(ctx context.Context, asyncError app.AsyncError, servingFile, subtitlesDir string) error
}

type FileItem struct {
	File     *torrent.File
	Selected bool
}

type Download struct {
	root       *ViewModel
	service    DownloadService
	FileToPlay *FileItem
}

func NewDownload(service DownloadService) *Download {
	return &Download{
		service: service,
	}
}

func (c *Download) PlayFile() {
	c.FileToPlay.Selected = true

	err := c.playFile(file, true)
	if err != nil {
		c.logAndPub(err, "Failed to play file")
		return
	}
}
