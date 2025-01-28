package app

import (
	"context"
	"net/http"

	"github.com/anacrolix/torrent"
	"github.com/quintans/torflix/internal/lib/extractor"
	"github.com/quintans/torflix/internal/model"
)

type Controller interface {
	OnEnter()
}

type Navigator interface {
	Go(string)
	Back()
}

type EventBus interface {
	Publish(m Message)
}

type Message interface {
	Kind() string
}

type AppView interface {
	Show(AppData)
	ShowNotification(evt Notify)
	EnableTabs(bool)
	Loading(Loading)
}

type AppData struct {
	CacheDir      string
	OpenSubtitles OpenSubtitles
}

type OpenSubtitles struct {
	Username string
	Password string
}

type DownloadView interface {
	Show(torName string, subFile string)
	SetStats(stats Stats)
	EnablePlay()
	DisablePlay()
}

type DownloadListView interface {
	Show(files []FileItem)
}

type FileItem struct {
	Name     string
	Size     int64
	Selected bool
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
	GetOpenSubtitles() (string, error)
	SetOpenSubtitles(value string) error
}
