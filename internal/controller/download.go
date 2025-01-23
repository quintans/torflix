package controller

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
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/anacrolix/torrent"
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
	downloadListView       app.DownloadListView
	downloadView           app.DownloadView
	repo                   Repository
	nav                    app.Navigator
	client                 app.TorrentClient
	shutdownServer         func()
	closePlayer            func()
	torCliFact             TorrentClientFactory
	videoPlayer            app.VideoPlayer
	subtitlesClientFactory OpenSubtitlesClientFactory
	mediaDir               string
	torrentsDir            string
	subtitlesRootDir       string
	subtitlesDir           string
	servingFile            string

	fromList bool
	files    []*torrent.File
	eventBus app.EventBus
	secrets  app.Secrets
}

func NewDownload(
	downloadView app.DownloadView,
	downloadListView app.DownloadListView,
	repo Repository,
	nav app.Navigator,
	videoPlayer app.VideoPlayer,
	torCliFact TorrentClientFactory,
	subtitlesClientFactory OpenSubtitlesClientFactory,
	mediaDir string,
	torrentsDir string,
	subtitlesDir string,
	eventBus app.EventBus,
	secrets app.Secrets,
) *Download {
	return &Download{
		downloadView:           downloadView,
		downloadListView:       downloadListView,
		repo:                   repo,
		nav:                    nav,
		closePlayer:            func() {},
		shutdownServer:         func() {},
		torCliFact:             torCliFact,
		videoPlayer:            videoPlayer,
		subtitlesClientFactory: subtitlesClientFactory,
		mediaDir:               mediaDir,
		torrentsDir:            torrentsDir,
		subtitlesRootDir:       subtitlesDir,
		eventBus:               eventBus,
		secrets:                secrets,
	}
}

func (c *Download) Back() {
	c.subtitlesDir = ""

	c.closePlayer()
	c.closePlayer = func() {}

	c.shutdownServer()
	c.shutdownServer = func() {}

	if c.fromList {
		c.client.PauseTorrent()
		c.downloadListView.Show(c.files)
		c.fromList = false
		return
	}

	c.client.Close()
	c.client = nil

	c.nav.Back()
}

func (c *Download) OnEnter() {
	err := c.onEnter()
	if err != nil {
		c.eventBus.Publish(app.NewNotifyError("Something went wrong: %s", err))
		slog.Error("Something went wrong.", "error", err.Error())
		os.Exit(1)
	}
}

func (c *Download) onEnter() error {
	model := c.repo.LoadDownload()

	var err error
	c.client, err = c.torCliFact(model.Link())
	if err != nil {
		return fmt.Errorf("creating torrent client: %w", err)
	}

	files := c.client.GetFilteredFiles()
	if files == nil {
		return fmt.Errorf("no media file found")
	}

	if len(files) == 1 {
		return c.playFile(files[0], false)
	}

	slices.SortFunc(files, func(i, j *torrent.File) int {
		return cmp.Compare(i.DisplayPath(), j.DisplayPath())
	})

	c.files = files
	c.downloadListView.Show(files)

	return nil
}

func (c *Download) PlayFile(file *torrent.File) error {
	return c.playFile(file, true)
}

func (c *Download) playFile(file *torrent.File, fromList bool) error {
	c.fromList = fromList

	cleanedQuery, err := c.downloadSubtitles(file)
	if err != nil {
		return fmt.Errorf("downloading subtitles: %w", err)
	}

	return c.downloadTorrentFile(file, cleanedQuery)
}

func (c *Download) downloadSubtitles(file *torrent.File) (string, error) {
	model := c.repo.LoadDownload()
	query := model.OriginalQuery()

	_, season, episode := extractSeasonEpisode(file.DisplayPath(), false)
	cleanedQuery, _, _ := extractSeasonEpisode(query, true)

	queryAndSeason := strings.ReplaceAll(cleanedQuery, " ", "_")
	if season > 0 {
		queryAndSeason = queryAndSeason + fmt.Sprintf("_s%de%d", season, episode)
	}

	settings, err := c.repo.LoadSettings()
	if err != nil {
		return "", fmt.Errorf("loading settings: %w", err)
	}
	if settings.OpenSubtitles.Username == "" {
		return queryAndSeason, nil
	}

	password, err := c.secrets.GetOpenSubtitles()
	if err != nil {
		return "", fmt.Errorf("getting OpenSubtitles password: %w", err)
	}
	subtitlesClient := c.subtitlesClientFactory(settings.OpenSubtitles.Username, password)

	subsDir := filepath.Join(c.subtitlesRootDir, queryAndSeason)
	c.subtitlesDir = subsDir
	// if subtitles already exist we will use it
	if _, err := os.Stat(subsDir); err == nil {
		return queryAndSeason, nil
	}

	err = os.MkdirAll(subsDir, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("creating subtitles directory: %w", err)
	}

	languages := settings.Languages()
	subtitles, err := subtitlesClient.Search(cleanedQuery, season, episode, languages)
	if err != nil {
		return "", fmt.Errorf("searching subtitles: %w", err)
	}

	if len(subtitles) == 0 {
		c.subtitlesDir = ""
		c.eventBus.Publish(app.NewNotifyInfo("No subtitles found"))
		return queryAndSeason, nil
	}

	token, err := subtitlesClient.Login()
	if err != nil {
		return "", fmt.Errorf("login: %w", err)
	}

	defer func() {
		err := subtitlesClient.Logout(token)
		if err != nil {
			c.eventBus.Publish(app.NewNotifyError("Failed to logout: %s", err))
			slog.Error("Failed to logout", "error", err)
		}
	}()

	slices.SortFunc(subtitles, func(i, j app.SubtitleAttributes) int {
		slices.Index(languages, i.Language)
		return cmp.Compare(
			slices.Index(languages, i.Language),
			slices.Index(languages, j.Language),
		)
	})

	for k, sub := range subtitles {
		download, err := subtitlesClient.Download(token, sub.FileID)
		if err != nil {
			return "", fmt.Errorf("downloading subtitle: %w", err)
		}

		subPath := filepath.Join(subsDir, insertLang(k, download.Filename, sub.Language))
		err = saveSubtitleFileRetry(download.Link, subPath)
		if err != nil {
			return "", fmt.Errorf("saving subtitle file: %w", err)
		}
	}

	return queryAndSeason, nil
}

func insertLang(index int, filename, language string) string {
	ext := filepath.Ext(filename)
	return fmt.Sprintf("%02d-subtitle.%s%s", index, language, ext)
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
		return retry.NewPermanentError(fmt.Errorf("failed to fetch subtitle file, status code: %d", resp.StatusCode))
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

func (c *Download) downloadTorrentFile(file *torrent.File, filename string) error {
	c.downloadView.Show(file.Torrent().Name(), file.DisplayPath())

	c.client.Play(file)

	settings, err := c.repo.LoadSettings()
	if err != nil {
		return fmt.Errorf("loading settings: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				stats := c.client.Stats()
				if stats.ReadyForPlayback {
					stats.Stream = fmt.Sprintf(localhost, settings.Port(), c.servingFile)
				} else {
					stats.Stream = "Not ready for playback"
				}
				c.downloadView.SetStats(stats)
				time.Sleep(time.Second)
			}
		}
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/"+filename, c.client.GetFile(filename))

	server := &http.Server{
		Addr:    ":" + strconv.Itoa(settings.Port()),
		Handler: mux,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("Failed to serve file", "error", err)
			os.Exit(1)
		}
	}()

	c.shutdownServer = func() {
		cancel()

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := server.Shutdown(ctx); err != nil {
				c.eventBus.Publish(app.NewNotifyError("Failed to shutdown server: %s", err))
				slog.Error("Server Shutdown Failed.", "error", err)
			}
		}()
	}

	c.servingFile = filename
	c.Play()

	return nil
}

func (c *Download) Play() {
	c.downloadView.DisablePlay()

	settings, err := c.repo.LoadSettings()
	if err != nil {
		c.eventBus.Publish(app.NewNotifyError("Failed to load settings on Play: %s", err))
		slog.Error("Failed to load settings on Play.", "error", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			if c.client == nil {
				return
			}

			if c.client.ReadyForPlayback() {
				break
			}

			time.Sleep(time.Second)
		}

		err := c.videoPlayer.Open(ctx, settings.Player(), fmt.Sprintf(localhost, settings.Port(), c.servingFile), c.subtitlesDir)
		if err != nil {
			c.eventBus.Publish(app.NewNotifyError("Failed to open player: %s", err))
			slog.Error("Failed to open player.", "error", err)
		}

		c.downloadView.EnablePlay()
	}()

	c.closePlayer = cancel
}

var MediaExtensions = []string{".mp4", ".mkv", ".avi", ".mov", ".flv", ".wmv", ".webm"}

// cleanTorrentFilename parses a torrent filename and determines if it's a movie or TV show.
// It returns the cleaned name, type (movie/TV show), and season/episode as integers (if applicable).
func extractSeasonEpisode(name string, clean bool) (string, int, int) {
	// Remove file extension
	for _, ext := range MediaExtensions {
		name = strings.TrimSuffix(name, ext)
	}

	// Replace common separators with spaces
	name = strings.ReplaceAll(name, ".", " ")
	name = strings.ReplaceAll(name, "_", " ")

	// Define patterns for TV show (season and episode)
	tvShowPatterns := []string{
		`(?i)(S(\d{1,2})E(\d{1,2}))`, // Pattern: S01E01
		`(?i)((\d{1,2})x(\d{1,2}))`,  // Pattern: 1x01
		`(?i)(S(\d{1,2}))`,           // Pattern: S01
		`(?i)(Season (\d{1,2}))`,     // Pattern: Season 01
		`(?i)(Season-(\d{1,2}))`,     // Pattern: Season-01
	}

	var season, episode int

	for _, pattern := range tvShowPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(name)
		if matches != nil {
			season = parseInt(matches[2]) // Extract season as an integer
			if len(matches) > 3 {
				episode = parseInt(matches[3]) // Extract episode as an integer
			}
			if clean {
				loc := re.FindStringIndex(name)
				if loc != nil {
					name = name[:loc[0]] // inclusive truncate on pattern math
				}
			}
			break
		}
	}

	if clean {
		patternsToRemove := []string{
			`(?i)(720p|1080p|2160p|4k)`, // Resolutions
			`(?i)(x264|x265|h264|h265)`, // Codecs
		}

		for _, pattern := range patternsToRemove {
			re := regexp.MustCompile(pattern)
			loc := re.FindStringIndex(name)
			if loc != nil {
				name = name[:loc[0]] // inclusive truncate on pattern math
			}
		}
	}

	name = strings.TrimSpace(name)

	return name, season, episode
}

// parseInt is a utility function to safely parse an integer from a string
func parseInt(input string) int {
	var result int
	_, err := fmt.Sscanf(input, "%d", &result)
	if err != nil {
		slog.Error("Failed to parse integer", "error", err)
	}
	return result
}

func (c *Download) ClearCache(_ app.ClearCache) {
	err := os.RemoveAll(c.mediaDir)
	if err != nil {
		c.eventBus.Publish(app.NewNotifyError("Failed to clear media cache: %s", err))
		slog.Error("Failed to clear media cache.", "error", err)
	}
	err = os.RemoveAll(c.torrentsDir)
	if err != nil {
		c.eventBus.Publish(app.NewNotifyError("Failed to clear torrent cache: %s", err))
		slog.Error("Failed to clear torrent cache.", "error", err)
	}
	err = os.RemoveAll(c.subtitlesRootDir)
	if err != nil {
		c.eventBus.Publish(app.NewNotifyError("Failed to clear subtitles cache: %s", err))
		slog.Error("Failed to clear subtitles cache.", "error", err)
	}

	c.eventBus.Publish(app.NewNotifySuccess("Cache cleared"))
}
