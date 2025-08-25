package app

import (
	"context"
	"net/http"

	"github.com/anacrolix/torrent"
	"github.com/quintans/torflix/internal/lib/extractor"
	"github.com/quintans/torflix/internal/model"
)

const (
	Version = "0.2"
	Name    = "torflix"
)

type EventBus interface {
	Publish(m Message)
}

type Message interface {
	Kind() string
}

type Stats struct {
	Stream           string
	ReadyForPlayback bool
	Complete         int64
	Size             int64
	DownloadSpeed    int64
	UploadSpeed      int64
	Seeders          int
	Done             bool
	Pieces           []bool
}

type TorrentClient interface {
	Stats() Stats
	Close()
	GetFile(filename string) func(w http.ResponseWriter, r *http.Request)
	ReadyForPlayback() bool
	GetFilteredFiles() []*torrent.File
	Play(file *torrent.File)
	PauseTorrent()
}

// VideoPlayer opens a stream URL in a video player.
type VideoPlayer interface {
	Open(ctx context.Context, player model.Player, url string, subtitlesDir string) error
}

type SubtitlesClient interface {
	Login() (string, error)
	Logout(token string) error
	Search(query string, season, episode int, languages []string) ([]SubtitleAttributes, error)
	Download(token string, fileID int) (DownloadResponse, error)
}

type SubtitleAttributes struct {
	ID       string
	Language string
	FileID   int
	Filename string
}

type DownloadResponse struct {
	Link     string `json:"link"`
	Filename string `json:"file_name"`
}

type Extractor interface {
	Slugs() []string
	Accept(slug string) bool
	Extract(slug string, query string) ([]extractor.Result, error)
}

type Secrets interface {
	GetOpenSubtitles() (OpenSubtitlesSecret, error)
	SetOpenSubtitles(value OpenSubtitlesSecret) error
}

type OpenSubtitlesSecret struct {
	Password string `json:"password"`
}

type AsyncError func(err error, message string, args ...any)

type SearchSettings struct {
	Model     *model.Search
	Providers []string
}
