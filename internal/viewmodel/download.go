package viewmodel

import (
	"context"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/bind"
	"github.com/quintans/torflix/internal/lib/timer"
)

type DownloadService interface {
	DownloadTorrent(query, magnetLink string) ([]*torrent.File, error)
	DownloadSubtitles(file *torrent.File, originalQuery string) (string, int, error)
	ServeFile(ctx context.Context, asyncError app.AsyncError, file *torrent.File, originalQuery string, setStats func(app.Stats)) (string, error)
	Play(ctx context.Context, asyncError app.AsyncError, servingFile, subtitlesDir string) error
}

type Download struct {
	root          *ViewModel
	service       DownloadService
	fileToPlay    *torrent.File
	originalQuery string
	cancel        func()
	Status        *bind.Bind[app.Stats]
}

func NewDownload(service DownloadService) *Download {
	return &Download{
		service: service,
		Status:  bind.NewNotifier[app.Stats](),
	}
}

func (d *Download) Init(fileToPlay *torrent.File, originalQuery string) {
	d.fileToPlay = fileToPlay
	d.originalQuery = originalQuery
}

func (d *Download) Play() bool {
	t := timer.New(time.Second, func() {
		d.root.eventBus.Publish(app.Loading{
			Text: "Downloading subtitles",
			Show: true,
		})
	})

	defer func() {
		t.Stop()
		d.root.eventBus.Publish(app.Loading{}) // hide spinner
	}()

	subtitlesDir, downloaded, err := d.service.DownloadSubtitles(d.fileToPlay, d.originalQuery)
	if err != nil {
		d.root.App.logAndPub(err, "Failed to download subtitles")
		return false
	}

	if downloaded == 0 {
		d.root.App.ShowNotification.Notify(app.NewNotifyInfo("No subtitles found"))
	}

	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel
	queryAndSeason, err := d.service.ServeFile(ctx, d.root.App.logAndPub, d.fileToPlay, d.originalQuery, func(stats app.Stats) {
		d.Status.Notify(stats)
	})
	if err != nil {
		d.root.App.logAndPub(err, "Failed to download subtitles")
		return false
	}

	err = d.service.Play(ctx, d.root.App.logAndPub, queryAndSeason, subtitlesDir)
	if err != nil {
		d.root.App.logAndPub(err, "Failed to play file")
		return false
	}

	return true
}

func (d *Download) Back() {
	if d.cancel != nil {
		d.cancel()
		d.cancel = nil
	}
}
