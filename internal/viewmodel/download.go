package viewmodel

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/timer"
)

type DownloadService interface {
	DownloadTorrent(magnetLink string) ([]*torrent.File, error)
	DownloadSubtitles(
		file *torrent.File,
		mediaName string,
		cleanedQuery string,
		season int,
		episode int,
	) (string, int, error)
	ServeFile(
		ctx context.Context,
		asyncError app.AsyncError,
		file *torrent.File,
		mediaName string,
		setStats func(app.Stats),
	) error
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
}

func NewDownload(shared *Shared, service DownloadService, params app.DownloadParams) *Download {
	d := &Download{
		shared:  shared,
		service: service,
		params:  params,
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

func (d *Download) Serve(onStats func(stats app.Stats)) bool {
	qc := d.getQueryComponents(d.params.FileToPlay, d.params.OriginalQuery)
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

		subtitlesDir, downloaded, err := d.service.DownloadSubtitles(
			d.params.FileToPlay,
			qc.mediaName,
			qc.cleanedQuery,
			qc.season,
			qc.episode,
		)
		if err != nil {
			d.shared.Error(err, "Failed to download subtitles")
			return false
		}

		if downloaded == 0 {
			d.shared.Info("No subtitles found")
		}

		d.subtitlesDir = subtitlesDir
	}

	err := d.service.ServeFile(d.ctx, d.shared.Error, d.params.FileToPlay, qc.mediaName, onStats)
	if err != nil {
		d.shared.Error(err, "Failed to serve file")
		return false
	}

	d.queryAndSeason = qc.mediaName

	return true
}

func (d *Download) Play(onClose func()) {
	err := d.service.Play(d.ctx, d.shared.Error, d.queryAndSeason, d.subtitlesDir, onClose)
	if err != nil {
		d.shared.Error(err, "Failed to play file")
	}
}

type queryComponents struct {
	cleanedQuery string
	mediaName    string
	season       int
	episode      int
}

func (c *Download) getQueryComponents(file *torrent.File, originalQuery string) queryComponents {
	season, episode := extractSeasonEpisode(file.DisplayPath())
	cleanedQuery := extractTitle(originalQuery)

	queryAndSeason := strings.ReplaceAll(cleanedQuery, " ", "_")
	if season > 0 {
		if episode == 0 {
			queryAndSeason = queryAndSeason + fmt.Sprintf("_%02d", season)
		} else {
			queryAndSeason = queryAndSeason + fmt.Sprintf("_s%02de%02d", season, episode)
		}
	}

	return queryComponents{
		cleanedQuery: cleanedQuery,
		mediaName:    queryAndSeason,
		season:       season,
		episode:      episode,
	}
}

var MediaExtensions = []string{".mp4", ".mkv", ".avi", ".mov", ".flv", ".wmv", ".webm"}

// Define patterns for TV show (season and episode)
var tvShowPatterns = []string{
	`(?i)(S(\d{1,2})E(\d{1,2}))`,        // Pattern: S01E01
	`(?i)((\d{1,2})x(\d{1,2}))`,         // Pattern: 1x01
	`(?i)(S(\d{1,2}))`,                  // Pattern: S01
	`(?i)(Season (\d{1,2}))`,            // Pattern: Season 01
	`(?i)(Season-(\d{1,2}))`,            // Pattern: Season-01
	`(?i)([\s\-]?\b(\d{1,2})\b[\s\-]?)`, // Pattern: 01 may or may not have a space or a dash at the beginning
}

// cleanTorrentFilename parses a torrent filename and determines if it's a movie or TV show.
// It returns the cleaned name, type (movie/TV show), and season/episode as integers (if applicable).
func extractSeasonEpisode(name string) (int, int) {
	name = preClean(name)

	var season, episode int

	for _, pattern := range tvShowPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(name)
		if matches != nil {
			season = parseInt(matches[2]) // Extract season as an integer
			if len(matches) > 3 {
				episode = parseInt(matches[3]) // Extract episode as an integer
			}
			break
		}
	}

	return season, episode
}

func extractTitle(name string) string {
	name = preClean(name)

	for _, pattern := range tvShowPatterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(name)
		if matches != nil {
			loc := re.FindStringIndex(name)
			if loc != nil {
				if loc[0] == 0 {
					name = name[loc[1]:] // remove the pattern match
				} else {
					name = name[:loc[0]] // inclusive truncate on pattern math
				}
			}
			break
		}
	}

	return strings.TrimSpace(name)
}

func preClean(name string) string {
	// Remove file extension
	for _, ext := range MediaExtensions {
		name = strings.TrimSuffix(name, ext)
	}

	patternsToRemove := []string{
		`(?i)(720p|1080p|2160p|4k)`,       // Resolutions
		`(?i)(x264|x265|h264|h265|H.264)`, // Codecs
	}

	for _, pattern := range patternsToRemove {
		re := regexp.MustCompile(pattern)
		loc := re.FindStringIndex(name)
		if loc != nil {
			name = name[:loc[0]] // inclusive truncate on pattern math
		}
	}

	// Replace common separators with spaces
	name = strings.ReplaceAll(name, ".", " ")
	name = strings.ReplaceAll(name, "_", " ")

	return name
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
