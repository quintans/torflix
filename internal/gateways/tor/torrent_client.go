package tor

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/quintans/faults"
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/files"
	"github.com/quintans/torflix/internal/lib/magnet"
	"golang.org/x/time/rate"
)

var isHTTP = regexp.MustCompile(`^https?:\/\/`)

// TorrentClient manages the torrent downloading.
type TorrentClient struct {
	Client     *torrent.Client
	Torrent    *torrent.Torrent
	Progress   int64
	Uploaded   int64
	Size       int64
	Config     ClientConfig
	File       *torrent.File
	TorrentDir string
	paused     bool

	torrentConfig *torrent.ClientConfig
}

// ClientConfig specifies the behaviour of a client.
type ClientConfig struct {
	TorrentPort          int
	Seed                 bool
	SeedAfterComplete    bool
	TCP                  bool
	MaxConnections       int
	DownloadAheadPercent int64 // Prioritize first % of the file.
	ValidMediaExtensions []string
	UploadRate           int // bytes per second
}

// NewTorrentClient creates a new torrent client based on a magnet or a torrent file.
// If the torrent file is on http, we try downloading it.
func NewTorrentClient(cfg ClientConfig, torrentDir, mediaDir string, resource string) (*TorrentClient, error) {
	torrentFile, err := checkIfTorrentExists(torrentDir, resource)
	if err != nil {
		return nil, faults.Errorf("parsing magnet or torrent: %w", err)
	}
	var torrentPath string
	if torrentFile != "" {
		torrentPath = torrentFile
	} else {
		torrentPath = resource
	}

	var t *torrent.Torrent
	var c *torrent.Client

	err = os.MkdirAll(mediaDir, os.ModePerm)
	if err != nil {
		return nil, faults.Errorf("creating data directory: %w", err)
	}

	if cfg.DownloadAheadPercent == 0 {
		cfg.DownloadAheadPercent = 5
	}

	client := &TorrentClient{
		TorrentDir: mediaDir,
		Config:     cfg,
	}

	torrentConfig := torrent.NewDefaultClientConfig()
	torrentConfig.DataDir = mediaDir
	torrentConfig.Seed = cfg.Seed
	torrentConfig.NoUpload = !cfg.Seed
	torrentConfig.DisableTCP = !cfg.TCP
	torrentConfig.ListenPort = cfg.TorrentPort
	if cfg.UploadRate > 0 {
		torrentConfig.UploadRateLimiter = rate.NewLimiter(rate.Limit(cfg.UploadRate), cfg.UploadRate)
	}

	// Create client.
	c, err = torrent.NewClient(torrentConfig)
	if err != nil {
		return client, faults.Errorf("creating lib torrent client: %w", err)
	}

	client.Client = c
	client.torrentConfig = torrentConfig

	// Add torrent.

	// Add as magnet url.
	if strings.HasPrefix(torrentPath, "magnet:") {
		if t, err = c.AddMagnet(torrentPath); err != nil {
			return client, faults.Errorf("adding torrent: %w", err)
		}
	} else {
		// Otherwise add as a torrent file.

		// If it's online, we try downloading the file.
		if isHTTP.MatchString(torrentPath) {
			if torrentPath, err = downloadFile(mediaDir, torrentPath); err != nil {
				return client, faults.Errorf("downloading torrent file: %w", err)
			}
		}

		if t, err = c.AddTorrentFromFile(torrentPath); err != nil {
			return client, faults.Errorf("adding torrent '%s' to the client: %w", torrentPath, err)
		}
	}

	client.Torrent = t
	client.Torrent.SetMaxEstablishedConns(cfg.MaxConnections)

	<-t.GotInfo()

	if torrentFile == "" {
		err = saveTorrent(torrentDir, t)
		if err != nil {
			return nil, faults.Errorf("saving torrent: %w", err)
		}
	}

	return client, nil
}

func checkIfTorrentExists(torrentFileDir, torrentPath string) (string, error) {
	m, err := magnet.Parse(torrentPath)
	if err != nil {
		return "", faults.Errorf("parsing magnet: %w", err)
	}

	if m.InfoHash != "" {
		filename := fmt.Sprintf("%s.torrent", strings.ToUpper(m.InfoHash))
		file := filepath.Join(torrentFileDir, filename)
		if files.Exists(file) {
			return file, nil
		}
	}

	return "", nil
}

func saveTorrent(torrentFileDir string, t *torrent.Torrent) error {
	err := os.MkdirAll(torrentFileDir, os.ModePerm)
	if err != nil {
		return faults.Errorf("creating torrent directory: %w", err)
	}

	hash := t.InfoHash().HexString()
	filename := fmt.Sprintf("%s.torrent", strings.ToUpper(hash))
	file := filepath.Join(torrentFileDir, filename)

	f, err := os.Create(file)
	if err != nil {
		return faults.Errorf("creating torrent file: %w", err)
	}
	defer f.Close()

	mi := t.Metainfo()
	err = mi.Write(f)
	if err != nil {
		return faults.Errorf("saving torrent: %w", err)
	}

	return nil
}

func (c *TorrentClient) Play(file *torrent.File) {
	c.paused = false
	c.torrentConfig.Seed = c.Config.Seed
	c.torrentConfig.NoUpload = !c.Config.Seed

	c.File = file

	t := c.Torrent
	// downloading only the pieces we need
	t.DownloadPieces(file.BeginPieceIndex(), file.EndPieceIndex())

	firstPieceIndex := file.Offset() * int64(t.NumPieces()) / t.Length()
	endPieceIndex := (file.Offset() + file.Length()) * int64(t.NumPieces()) / t.Length()
	// Prioritize the first % of the file.
	firstPercentage := endPieceIndex * c.Config.DownloadAheadPercent / 400 // 0.25%
	for idx := firstPieceIndex; idx <= firstPercentage; idx++ {
		t.Piece(int(idx)).SetPriority(torrent.PiecePriorityNow)
	}
}

func (c *TorrentClient) PauseTorrent() {
	c.paused = true
	c.torrentConfig.NoUpload = true
	c.torrentConfig.Seed = false
	c.Torrent.CancelPieces(0, c.Torrent.NumPieces())
}

// Close cleans up the connections.
func (c *TorrentClient) Close() {
	c.Torrent.Closed()
	c.Torrent.Drop()

	errs := c.Client.Close()
	for _, err := range errs {
		slog.Error("Failed closing torrent client.", "error", err)
	}
}

// Render outputs the command line interface for the client.
func (c *TorrentClient) Stats() app.Stats {
	if c.paused {
		return app.Stats{}
	}

	t := c.Torrent

	c.File.BytesCompleted()

	if t.Info() == nil {
		return app.Stats{}
	}

	tStats := t.Stats()
	currentProgress := c.File.BytesCompleted()
	size := c.File.Length()

	// upload
	bytesWrittenData := tStats.BytesWrittenData
	currentUpload := (&bytesWrittenData).Int64()

	fps := c.File.State()
	pieces := make([]bool, len(fps))
	for i, fp := range fps {
		pieces[i] = fp.Complete
	}

	stats := app.Stats{
		Complete:      currentProgress,
		Size:          size,
		DownloadSpeed: currentProgress - c.Progress,
		UploadSpeed:   currentUpload - c.Uploaded,
		Seeders:       tStats.ConnectedSeeders,
		Done:          currentProgress >= size,
		Pieces:        pieces,
	}

	c.Progress = currentProgress
	c.Uploaded = currentUpload
	c.Size = size

	if !c.paused && stats.Done && !c.Config.SeedAfterComplete {
		c.PauseTorrent()
		stats.UploadSpeed = 0
		stats.DownloadSpeed = 0
		stats.Seeders = 0
	}

	stats.ReadyForPlayback = c.ReadyForPlayback()

	return stats
}

func (c TorrentClient) GetFilteredFiles() []*torrent.File {
	var maxSize int64

	files := c.Torrent.Files()
	validFiles := make([]*torrent.File, 0, len(files))
	// gets the largest file
	for _, file := range files {
		if len(c.Config.ValidMediaExtensions) > 0 &&
			!slices.Contains(c.Config.ValidMediaExtensions, strings.ToLower(filepath.Ext(file.Path()))) {
			continue
		}

		validFiles = append(validFiles, file)
		if maxSize < file.Length() {
			maxSize = file.Length()
		}
	}

	result := make([]*torrent.File, 0, len(validFiles))
	// return all files that are at least a percentage (30%) of the largest file
	// this way we avoid returning small files like samples
	for _, file := range validFiles {
		if file.Length() > maxSize*30/100 {
			result = append(result, file)
		}
	}

	return result
}

// ReadyForPlayback checks if the torrent is ready for playback or not.
// We wait until 0.5% of the torrent to start playing.
func (c TorrentClient) ReadyForPlayback() bool {
	percentage := float64(c.Progress) / float64(c.Size) * 200

	return percentage > float64(c.Config.DownloadAheadPercent)
}

// GetFile is an http handler to serve the biggest file managed by the client.
func (c TorrentClient) GetFile(filename string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		target := c.File
		entry, err := NewFileReader(target)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		defer func() {
			if err := entry.Close(); err != nil {
				slog.Error("Failed closing file reader.", "error", err)
			}
		}()

		w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
		http.ServeContent(w, r, target.DisplayPath(), time.Now(), entry)
	}
}

func downloadFile(torrentDir, URL string) (fileName string, err error) {
	var file *os.File
	if file, err = os.CreateTemp(torrentDir, "download"); err != nil {
		return
	}

	defer func() {
		if ferr := file.Close(); ferr != nil {
			slog.Error("Failed closing torrent file.", "error", ferr)
		}
	}()

	// #nosec
	// We are downloading the url the user passed to us, we trust it is a torrent file.
	response, err := http.Get(URL)
	if err != nil {
		return
	}

	defer func() {
		if ferr := response.Body.Close(); ferr != nil {
			slog.Error("Failed closing torrent file.", "error", ferr)
		}
	}()

	_, err = io.Copy(file, response.Body)

	return file.Name(), err
}
