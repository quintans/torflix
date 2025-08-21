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
	Play(ctx context.Context, asyncError app.AsyncError, servingFile, subtitlesDir string, onClose func()) error
}

type Download struct {
	root           *ViewModel
	service        DownloadService
	fileToPlay     *torrent.File
	isFromList     bool
	originalQuery  string
	queryAndSeason string
	subtitlesDir   string
	ctx            context.Context
	cancel         func()
	Status         bind.Notifier[app.Stats]
	Playable       bind.Notifier[bool]
}

func NewDownload(service DownloadService) *Download {
	return &Download{
		service:  service,
		Status:   bind.NewNotifier[app.Stats](),
		Playable: bind.NewNotifier[bool](),
	}
}

func (d *Download) Back() {
	if d.cancel != nil {
		d.cancel()
	}
	d.cancel = nil
	d.ctx = nil
	d.fileToPlay = nil
	d.isFromList = false
	d.originalQuery = ""
	d.queryAndSeason = ""
	d.subtitlesDir = ""
	d.Status.Clear()
	d.Playable.Clear()
}

func (d *Download) Init(fileToPlay *torrent.File, isFromList bool, originalQuery string) {
	d.fileToPlay = fileToPlay
	d.isFromList = isFromList
	d.originalQuery = originalQuery

	d.root.App.EscapeKey.Notify(func() {
		d.Back()
	})

	d.ctx, d.cancel = context.WithCancel(context.Background())
}

func (d *Download) TorrentFilename() string {
	return d.fileToPlay.Torrent().Name()
}

func (d *Download) TorrentSubFilename() string {
	if d.isFromList {
		return d.fileToPlay.DisplayPath()
	}
	return ""
}

func (d *Download) Serve() bool {
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

	queryAndSeason, err := d.service.ServeFile(d.ctx, d.root.App.logAndPub, d.fileToPlay, d.originalQuery, func(stats app.Stats) {
		d.Status.Notify(stats)
	})
	if err != nil {
		d.root.App.logAndPub(err, "Failed to serve file")
		return false
	}

	d.queryAndSeason = queryAndSeason
	d.subtitlesDir = subtitlesDir

	return true
}

func (d *Download) Play() {
	d.Playable.Notify(false)

	err := d.service.Play(d.ctx, d.root.App.logAndPub, d.queryAndSeason, d.subtitlesDir, func() {
		d.Playable.Notify(true)
	})
	if err != nil {
		d.root.App.logAndPub(err, "Failed to play file")
	}
}
