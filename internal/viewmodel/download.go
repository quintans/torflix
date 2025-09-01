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
	DownloadTorrent(magnetLink string) ([]*torrent.File, error)
	DownloadSubtitles(file *torrent.File, originalQuery string) (string, int, error)
	ServeFile(ctx context.Context, asyncError app.AsyncError, file *torrent.File, originalQuery string, setStats func(app.Stats)) (string, error)
	Play(ctx context.Context, asyncError app.AsyncError, servingFile, subtitlesDir string, onClose func()) error
	Pause()
	Close()
}

type Download struct {
	shared         *Shared
	params         app.DownloadParams
	service        DownloadService
	queryAndSeason string
	subtitlesDir   string
	ctx            context.Context
	cancel         func()
	Status         bind.Notifier[app.Stats]
	Playable       bind.Notifier[bool]
}

func NewDownload(shared *Shared, service DownloadService, params app.DownloadParams) *Download {
	d := &Download{
		shared:   shared,
		service:  service,
		Status:   bind.NewNotifier[app.Stats](),
		Playable: bind.NewNotifier[bool](),
		params:   params,
	}

	d.ctx, d.cancel = context.WithCancel(context.Background())

	return d
}

func (d *Download) Back() {
	if d.cancel != nil {
		d.cancel()
	}

	if d.params.PauseTorrentOnClose {
		d.service.Pause()
	} else {
		d.service.Close()
	}

	d.shared.Navigate.Back()
}

func (d *Download) TorrentFilename() string {
	return d.params.FileToPlay.Torrent().Name()
}

func (d *Download) TorrentSubFilename() string {
	if d.params.PauseTorrentOnClose {
		return d.params.FileToPlay.DisplayPath()
	}
	return ""
}

func (d *Download) ServeAsync() bool {
	if d.params.Subtitles {
		t := timer.New(time.Second, func() {
			d.shared.Publish(app.Loading{
				Text: "Downloading subtitles",
				Show: true,
			})
		})

		defer func() {
			t.Stop()
			d.shared.Publish(app.Loading{}) // hide spinner
		}()

		subtitlesDir, downloaded, err := d.service.DownloadSubtitles(d.params.FileToPlay, d.params.ResourceName)
		if err != nil {
			d.shared.Error(err, "Failed to download subtitles")
			return false
		}

		if downloaded == 0 {
			d.shared.Info("No subtitles found")
		}

		d.subtitlesDir = subtitlesDir
	}

	queryAndSeason, err := d.service.ServeFile(d.ctx, d.shared.Error, d.params.FileToPlay, d.params.ResourceName, func(stats app.Stats) {
		d.Status.NotifyAsync(stats)
	})
	if err != nil {
		d.shared.Error(err, "Failed to serve file")
		return false
	}

	d.queryAndSeason = queryAndSeason

	return true
}

func (d *Download) Play() {
	d.Playable.Notify(false)

	err := d.service.Play(d.ctx, d.shared.Error, d.queryAndSeason, d.subtitlesDir, func() {
		d.Playable.NotifyAsync(true)
	})
	if err != nil {
		d.shared.Error(err, "Failed to play file")
	}
}
