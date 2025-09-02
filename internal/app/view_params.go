package app

import "github.com/anacrolix/torrent"

type AppParams struct{}

type DownloadListParams struct {
	Files         []*torrent.File
	OriginalQuery string
	Subtitles     bool
}

type DownloadParams struct {
	FileToPlay          *torrent.File
	PauseTorrentOnClose bool
	OriginalQuery       string
	Subtitles           bool
}
