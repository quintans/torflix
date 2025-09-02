package viewmodel

import (
	"cmp"
	gslices "slices"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/timer"
)

func download(shared *Shared, downloadService DownloadService, originalQuery, link string, subtitles bool) bool {
	t := timer.New(time.Second, func() {
		shared.Publish(app.Loading{
			Text: "Downloading torrent metadata",
			Show: true,
		})
	})

	defer func() {
		t.Stop()
		shared.Publish(app.Loading{}) // hide spinner
	}()

	files, err := downloadService.DownloadTorrent(link)
	if err != nil {
		shared.Error(err, "Failed to download torrent metadata")
		return false
	}

	if len(files) == 0 {
		shared.Warn("No media files found for magnet link")
		return false
	}

	if len(files) == 1 {
		shared.Navigate.To(app.DownloadParams{
			FileToPlay:          files[0],
			PauseTorrentOnClose: false,
			OriginalQuery:       originalQuery,
			Subtitles:           subtitles,
		})
		return true
	}

	gslices.SortFunc(files, func(i, j *torrent.File) int {
		return cmp.Compare(i.DisplayPath(), j.DisplayPath())
	})

	shared.Navigate.To(app.DownloadListParams{
		Files:         files,
		OriginalQuery: originalQuery,
		Subtitles:     subtitles,
	})
	return true
}
