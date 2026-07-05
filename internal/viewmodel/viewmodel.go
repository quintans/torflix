package viewmodel

import (
	"cmp"
	gslices "slices"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/timer"
)

func download(shared *Shared, downloadService DownloadService, originalQuery, link string, subtitles bool) (DownloadTorrentResponse, bool) {
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

	response, err := downloadService.DownloadTorrent(link)
	if err != nil {
		shared.Error(err, "Failed to download torrent metadata")
		return DownloadTorrentResponse{}, false
	}

	if len(response.Files) == 0 {
		shared.Warn("No media files found for magnet link")
		return DownloadTorrentResponse{}, false
	}

	if len(response.Files) == 1 {
		shared.Navigate.To(app.DownloadParams{
			FileToPlay:          response.Files[0],
			PauseTorrentOnClose: false,
			OriginalQuery:       originalQuery,
			Subtitles:           subtitles,
		})
		return response, true
	}

	gslices.SortFunc(response.Files, func(i, j *torrent.File) int {
		return cmp.Compare(i.DisplayPath(), j.DisplayPath())
	})

	shared.Navigate.To(app.DownloadListParams{
		Files:         response.Files,
		OriginalQuery: originalQuery,
		Subtitles:     subtitles,
	})
	return response, true
}
