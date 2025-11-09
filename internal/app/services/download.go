package services

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	gslices "slices"
	"strconv"
	"strings"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/quintans/faults"
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/fails"
	"github.com/quintans/torflix/internal/lib/https"
	"github.com/quintans/torflix/internal/lib/retry"
)

const localhost = "http://localhost:%d/%s"

type (
	TorrentClientFactory       func(string) (app.TorrentClient, error)
	OpenSubtitlesClientFactory func(usr, pwd string) app.SubtitlesClient
)

type Download struct {
	repo                   Repository
	client                 app.TorrentClient
	torCliFact             TorrentClientFactory
	videoPlayer            app.VideoPlayer
	subtitlesClientFactory OpenSubtitlesClientFactory
	torrentsDir            string
	subtitlesRootDir       string

	secrets app.Secrets
}

func NewDownload(
	repo Repository,
	videoPlayer app.VideoPlayer,
	torCliFact TorrentClientFactory,
	subtitlesClientFactory OpenSubtitlesClientFactory,
	torrentsDir string,
	subtitlesDir string,
	secrets app.Secrets,
) *Download {
	return &Download{
		repo:                   repo,
		torCliFact:             torCliFact,
		videoPlayer:            videoPlayer,
		subtitlesClientFactory: subtitlesClientFactory,
		torrentsDir:            torrentsDir,
		subtitlesRootDir:       subtitlesDir,
		secrets:                secrets,
	}
}

func (c *Download) Pause() {
	c.client.PauseTorrent()
}

func (c *Download) Close() {
	c.client.Close()
	c.client = nil
}

func (c *Download) DownloadTorrent(link string) ([]*torrent.File, error) {
	if c.client != nil {
		c.Close()
	}

	var err error
	c.client, err = c.torCliFact(link)
	if err != nil {
		return nil, faults.Errorf("creating torrent client: %w", err)
	}

	return c.client.GetFilteredFiles(), nil
}

func (c *Download) ServeFile(
	ctx context.Context,
	asyncError app.AsyncError,
	file *torrent.File,
	mediaName string,
	setStats func(app.Stats),
) error {
	settings, err := c.repo.LoadSettings()
	if err != nil {
		return faults.Errorf("loading settings: %w", err)
	}

	go func() {
		const interval = 2
		fn := func() {
			stats := c.client.Stats()
			if stats.Pieces == nil || stats.ReadyForPlayback {
				stats.Stream = fmt.Sprintf(localhost, settings.Port(), mediaName)
			} else {
				stats.Stream = "Not ready for playback"
			}
			// normalize speed
			stats.DownloadSpeed = stats.DownloadSpeed / interval
			stats.UploadSpeed = stats.UploadSpeed / interval

			setStats(stats)
		}
		fn()

		for {
			select {
			case <-ctx.Done():
				return
			case <-time.After(interval * time.Second):
				fn()
			}
		}
	}()

	c.client.Play(file)

	mux := http.NewServeMux()
	mux.HandleFunc("/"+mediaName, c.client.GetFile(mediaName))

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(settings.Port()),
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			asyncError(err, "Failed to serve file")
		}
	}()

	go func() {
		<-ctx.Done()
		ctx2, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx2); err != nil {
			asyncError(err, "Failed to shutdown server")
		}
	}()

	return nil
}

func (c *Download) Play(ctx context.Context, asyncError app.AsyncError, servingFile, subtitlesDir string, onClose func()) error {
	settings, err := c.repo.LoadSettings()
	if err != nil {
		return faults.Errorf("loading settings on play: %w", err)
	}

	go func() {
		for {
			if c.client == nil {
				return
			}

			if c.client.ReadyForPlayback() {
				break
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Second):
			}
		}

		err := c.videoPlayer.Open(ctx, settings.Player(), fmt.Sprintf(localhost, settings.Port(), servingFile), subtitlesDir)
		if err != nil {
			asyncError(err, "Failed to open player")
		}

		onClose()
	}()

	return nil
}

func (c *Download) DownloadSubtitles(
	file *torrent.File,
	mediaName string,
	cleanedQuery string,
	season int,
	episode int,
) (string, int, error) {
	settings, err := c.repo.LoadSettings()
	if err != nil {
		return "", 0, faults.Errorf("loading settings: %w", err)
	}
	if settings.OpenSubtitles.Username == "" {
		return "", 0, nil
	}

	secret, err := c.secrets.GetOpenSubtitles()
	if err != nil {
		return "", 0, faults.Errorf("getting OpenSubtitles password: %w", err)
	}
	subtitlesClient := c.subtitlesClientFactory(settings.OpenSubtitles.Username, secret.Password)

	subsDir := filepath.Join(c.subtitlesRootDir, mediaName)
	// if subtitles already exist we will use it
	if _, err := os.Stat(subsDir); err == nil {
		return subsDir, -1, nil
	}

	err = os.MkdirAll(subsDir, os.ModePerm)
	if err != nil {
		return "", 0, faults.Errorf("creating subtitles directory: %w", err)
	}

	languages := settings.Languages()
	subtitles, err := subtitlesClient.Search(cleanedQuery, season, episode, languages)
	if err != nil {
		return "", 0, faults.Errorf("searching subtitles: %w", err)
	}

	if len(subtitles) == 0 {
		return "", 0, nil
	}

	token, err := subtitlesClient.Login()
	if err != nil {
		return "", 0, faults.Errorf("login: %w", err)
	}

	defer func() {
		err := subtitlesClient.Logout(token)
		if err != nil {
			slog.Error("Failed to logout from OpenSubtitles", "error", fmt.Sprintf("%+v", err))
		}
	}()

	gslices.SortFunc(subtitles, func(i, j app.SubtitleAttributes) int {
		gslices.Index(languages, i.Language)
		return cmp.Compare(
			gslices.Index(languages, i.Language),
			gslices.Index(languages, j.Language),
		)
	})

	qry := strings.ToLower(cleanedQuery)
	tokens := strings.Split(qry, " ")

	downloaded := 0
	for _, sub := range subtitles {
		if !wordMatch(tokens, sub.Filename) {
			continue
		}

		download, err := subtitlesClient.Download(token, sub.FileID)
		if err != nil {
			slog.Error("Failed to download subtitle", "error", err)
			continue
		}

		subPath := filepath.Join(subsDir, insertLang(downloaded, download.Filename, sub.Language))
		err = saveSubtitleFileRetry(download.Link, subPath)
		if err != nil {
			slog.Error("Failed to save subtitle file", "link", download.Link, "error", err)
			continue
		}

		downloaded++
		// Download only the first 10 subtitles
		if downloaded >= 10 {
			break
		}
	}

	return subsDir, downloaded, nil
}

func wordMatch(tokens []string, name string) bool {
	name = strings.ToLower(name)
	for _, token := range tokens {
		if !regexp.MustCompile(`\b` + regexp.QuoteMeta(token) + `\b`).MatchString(name) {
			return false
		}
	}
	return true
}

func insertLang(index int, filename, language string) string {
	ext := filepath.Ext(filename)
	name := strings.TrimSuffix(filename, ext)
	return fmt.Sprintf("%02d-%s.%s%s", index, name, language, ext)
}

// saveSubtitleFileRetry downloads the subtitle file from the given URL and saves it locally
func saveSubtitleFileRetry(downloadLink, fileName string) error {
	return retry.Do(func() error {
		return saveSubtitleFile(downloadLink, fileName)
	}, retry.WithDelayFunc(https.DelayFunc))
}

func saveSubtitleFile(downloadLink, fileName string) error {
	resp, err := http.Get(downloadLink)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			return fails.New("too many requests for downloading subtitle", "retry-after", resp.Header.Get("Retry-After"))
		}
		return retry.NewPermanentError(faults.Errorf("failed to fetch subtitle file, status code: %d", resp.StatusCode))
	}

	// Create a file locally to save the subtitle
	outFile, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
