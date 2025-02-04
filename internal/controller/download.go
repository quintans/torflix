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
	gslices "slices"
	"strconv"
	"strings"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/fails"
	"github.com/quintans/torflix/internal/lib/https"
	"github.com/quintans/torflix/internal/lib/retry"
	"github.com/quintans/torflix/internal/lib/slices"
	"github.com/quintans/torflix/internal/lib/timers"
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
	files    []fileItem
	eventBus app.EventBus
	secrets  app.Secrets
}

type fileItem struct {
	file     *torrent.File
	selected bool
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
		c.showList()
		c.fromList = false
		return
	}

	c.client.Close()
	c.client = nil

	c.nav.Back()
}

func (c *Download) showList() {
	c.downloadListView.Show(slices.Map(c.files, func(it fileItem) app.FileItem {
		return app.FileItem{
			Name:     it.file.DisplayPath(),
			Size:     it.file.Length(),
			Selected: it.selected,
		}
	}))
}

func (c *Download) OnEnter() {
	err := c.onEnter()
	if err != nil {
		logAndPub(c.eventBus, err, "Something went wrong")
	}
}

func (c *Download) onEnter() error {
	d := timers.NewDebounce(time.Second, func() {
		c.eventBus.Publish(app.Loading{
			Text: "Downloading torrent metadata",
			Show: true,
		})
	})

	defer func() {
		d.Stop()
		c.eventBus.Publish(app.Loading{}) // hide spinner
	}()

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

	gslices.SortFunc(files, func(i, j *torrent.File) int {
		return cmp.Compare(i.DisplayPath(), j.DisplayPath())
	})

	c.files = slices.Map(files, func(file *torrent.File) fileItem {
		file.BytesCompleted()
		return fileItem{
			file:     file,
			selected: file.BytesCompleted() >= file.Length(),
		}
	})
	c.showList()

	return nil
}

func (c *Download) PlayFile(findex int) {
	c.files[findex].selected = true

	file := c.files[findex].file
	d := timers.NewDebounce(time.Second, func() {
		c.eventBus.Publish(app.Loading{
			Show: true,
		})
	})

	defer func() {
		d.Stop()
		c.eventBus.Publish(app.Loading{}) // hide spinner
	}()

	err := c.playFile(file, true)
	if err != nil {
		logAndPub(c.eventBus, err, "Failed to play file")
		return
	}
}

func (c *Download) playFile(file *torrent.File, fromList bool) error {
	cleanedQuery, queryAndSeason, season, episode := c.getQueryComponents(file)
	err := c.downloadSubtitles(file, cleanedQuery, queryAndSeason, season, episode)
	if err != nil {
		return fmt.Errorf("downloading subtitles: %w", err)
	}

	c.fromList = fromList
	return c.downloadTorrentFile(file, queryAndSeason)
}

func (c *Download) getQueryComponents(file *torrent.File) (cleanedQuery, queryAndSeason string, season, episode int) {
	model := c.repo.LoadDownload()
	query := model.OriginalQuery()

	_, season, episode = extractSeasonEpisode(file.DisplayPath(), false)
	cleanedQuery, _, _ = extractSeasonEpisode(query, true)

	queryAndSeason = strings.ReplaceAll(cleanedQuery, " ", "_")
	if season > 0 {
		if episode == 0 {
			queryAndSeason = queryAndSeason + fmt.Sprintf("_%02d", season)
		} else {
			queryAndSeason = queryAndSeason + fmt.Sprintf("_s%02de%02d", season, episode)
		}
	}

	return
}

func (c *Download) downloadSubtitles(file *torrent.File, cleanedQuery, queryAndSeason string, season, episode int) error {
	c.eventBus.Publish(app.Loading{
		Text: "Downloading subtitles",
	})

	settings, err := c.repo.LoadSettings()
	if err != nil {
		return fmt.Errorf("loading settings: %w", err)
	}
	if settings.OpenSubtitles.Username == "" {
		return nil
	}

	secret, err := c.secrets.GetOpenSubtitles()
	if err != nil {
		return fmt.Errorf("getting OpenSubtitles password: %w", err)
	}
	subtitlesClient := c.subtitlesClientFactory(settings.OpenSubtitles.Username, secret.Password)

	subsDir := filepath.Join(c.subtitlesRootDir, queryAndSeason)
	c.subtitlesDir = subsDir
	// if subtitles already exist we will use it
	if _, err := os.Stat(subsDir); err == nil {
		return nil
	}

	err = os.MkdirAll(subsDir, os.ModePerm)
	if err != nil {
		return fmt.Errorf("creating subtitles directory: %w", err)
	}

	languages := settings.Languages()
	subtitles, err := subtitlesClient.Search(cleanedQuery, season, episode, languages)
	if err != nil {
		return fmt.Errorf("searching subtitles: %w", err)
	}

	if len(subtitles) == 0 {
		c.subtitlesDir = ""
		c.eventBus.Info("No subtitles found")
		return nil
	}

	token, err := subtitlesClient.Login()
	if err != nil {
		return fmt.Errorf("login: %w", err)
	}

	defer func() {
		err := subtitlesClient.Logout(token)
		if err != nil {
			logAndPub(c.eventBus, err, "Failed to logout")
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

	downloaded := 1
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
		if downloaded > 10 {
			break
		}
	}

	return nil
}

func wordMatch(tokens []string, name string) bool {
	name = strings.ToLower(name)
	for _, token := range tokens {
		if !strings.Contains(name, token) {
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
		fn := func() {
			stats := c.client.Stats()
			if stats == (app.Stats{}) || stats.ReadyForPlayback {
				stats.Stream = fmt.Sprintf(localhost, settings.Port(), c.servingFile)
			} else {
				stats.Stream = "Not ready for playback"
			}
			c.downloadView.SetStats(stats)
		}
		fn()

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fn()
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
			logAndPub(c.eventBus, err, "Failed to serve file")
		}
	}()

	c.shutdownServer = func() {
		cancel()

		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := server.Shutdown(ctx); err != nil {
				logAndPub(c.eventBus, err, "Failed to shutdown server")
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
		logAndPub(c.eventBus, err, "Failed to load settings on Play")
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
			logAndPub(c.eventBus, err, "Failed to open player")
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
		`(?i)(S(\d{1,2})E(\d{1,2}))`,    // Pattern: S01E01
		`(?i)((\d{1,2})x(\d{1,2}))`,     // Pattern: 1x01
		`(?i)(S(\d{1,2}))`,              // Pattern: S01
		`(?i)(Season (\d{1,2}))`,        // Pattern: Season 01
		`(?i)(Season-(\d{1,2}))`,        // Pattern: Season-01
		`(?i)([\s\-]?(\d{1,2})[\s\-]?)`, // Pattern: 01 may or may not have a space or a dash at the beginning
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
					if loc[0] == 0 {
						name = name[loc[1]:] // remove the pattern match
					} else {
						name = name[:loc[0]] // inclusive truncate on pattern math
					}
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
		logAndPub(c.eventBus, err, "Failed to clear media cache")
	}
	err = os.RemoveAll(c.torrentsDir)
	if err != nil {
		logAndPub(c.eventBus, err, "Failed to clear torrent cache")
	}
	err = os.RemoveAll(c.subtitlesRootDir)
	if err != nil {
		logAndPub(c.eventBus, err, "Failed to clear subtitles cache")
	}

	c.eventBus.Success("Cache cleared")
}
