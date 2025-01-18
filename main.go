package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"github.com/anacrolix/torrent"
	gapp "github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/controller"
	"github.com/quintans/torflix/internal/gateways/eventbus"
	"github.com/quintans/torflix/internal/gateways/opensubtitles"
	"github.com/quintans/torflix/internal/gateways/player"
	"github.com/quintans/torflix/internal/gateways/repository"
	"github.com/quintans/torflix/internal/gateways/secrets"
	"github.com/quintans/torflix/internal/gateways/tor"
	"github.com/quintans/torflix/internal/lib/bus"
	"github.com/quintans/torflix/internal/lib/extractor"
	"github.com/quintans/torflix/internal/lib/files"
	"github.com/quintans/torflix/internal/lib/magnet"
	"github.com/quintans/torflix/internal/lib/navigator"
	"github.com/quintans/torflix/internal/model"
	"github.com/quintans/torflix/internal/view"
)

func main() {
	path, err := os.UserCacheDir()
	if err != nil {
		panic(err)
	}
	slog.Info("cache", "dir", path)

	cacheDir := filepath.Join(path, "torflix")
	err = os.MkdirAll(cacheDir, os.ModePerm)
	if err != nil {
		panic(err)
	}

	mediaDir := filepath.Join(cacheDir, "media")
	err = os.MkdirAll(mediaDir, os.ModePerm)
	if err != nil {
		panic(err)
	}

	torrentsDir := filepath.Join(cacheDir, "torrents")
	err = os.MkdirAll(torrentsDir, os.ModePerm)
	if err != nil {
		panic(err)
	}

	subtitlesDir := filepath.Join(cacheDir, "subtitles")
	err = os.MkdirAll(subtitlesDir, os.ModePerm)
	if err != nil {
		panic(err)
	}

	a := app.New()
	w := a.NewWindow("TorFlix")
	w.Resize(fyne.NewSize(800, 600))

	db := repository.NewDB(cacheDir)
	if !db.Exists("search.json") {
		err := db.SaveSearch(model.NewSearch())
		if err != nil {
			panic(fmt.Sprintf("creating search: %s", err))
		}
	}

	if !db.Exists("settings.json") {
		err := db.SaveSettings(model.NewSettings())
		if err != nil {
			panic(fmt.Sprintf("creating settings: %s", err))
		}
	}

	settings, err := db.LoadSettings()
	if err != nil {
		panic(fmt.Sprintf("loading settings: %s", err))
	}

	extractors := []gapp.Extractor{}
	searchScraper, err := extractor.NewScraper(settings.HtmlSearchConfig(), settings.HtmlDetailsSearchConfig())
	if err != nil {
		panic(fmt.Sprintf("creating search scraper: %s", err))
	}
	extractors = append(extractors, searchScraper)

	if len(settings.ApiSearchConfig()) > 0 {
		apiSearch, err := extractor.NewApi(settings.ApiSearchConfig())
		if err != nil {
			panic(fmt.Sprintf("creating api search: %s", err))
		}
		extractors = append(extractors, apiSearch)
	}

	b := bus.New()
	eventBus := eventbus.New(b)
	nav := navigator.New(b)

	openSubtitlesClientFactory := func(usr, pwd string) gapp.SubtitlesClient {
		return opensubtitles.New(usr, pwd)
	}

	sec := secrets.NewSecrets()

	appView := view.NewApp(w)
	searchView := view.NewSearch(appView)
	downloadView := view.NewDownload(appView)
	downloadListView := view.NewDownloadList(appView, eventBus)

	searchCtrl, err := controller.NewSearch(searchView, nav, db, extractors, eventBus)
	if err != nil {
		panic(fmt.Sprintf("creating search controller: %s", err))
	}

	downloadCtrl := controller.NewDownload(
		downloadView,
		downloadListView,
		db,
		nav,
		player.Player{},
		torrentClientFactory(db, mediaDir, torrentsDir),
		openSubtitlesClientFactory,
		mediaDir,
		torrentsDir,
		subtitlesDir,
		eventBus,
		sec,
	)

	appCtrl := controller.NewApp(
		appView,
		map[string]gapp.Controller{
			controller.SearchNavigation:   searchCtrl,
			controller.DownloadNavigation: downloadCtrl,
		},
		eventBus,
		db,
		sec,
	)

	searchView.SetController(searchCtrl)

	downloadView.SetController(downloadCtrl)
	downloadListView.SetController(downloadCtrl)
	appView.SetController(appCtrl)

	bus.Listen(b, appCtrl.OnNavigation)
	bus.Listen(b, downloadCtrl.ClearCache)
	bus.Listen(b, appCtrl.ShowNotification)

	appCtrl.OnEnter()
	nav.Go(controller.SearchNavigation)

	// w.FullScreen()
	w.ShowAndRun()
}

func torrentClientFactory(db *repository.DB, mediaDir, torrentFileDir string) func(torrentPath string) (gapp.TorrentClient, error) {
	return func(magnetLink string) (gapp.TorrentClient, error) {
		torrentPath, err := checkIfTorrentExists(torrentFileDir, magnetLink)
		if err != nil {
			return nil, fmt.Errorf("parsing magnet or torrent: %w", err)
		}
		var link string
		if torrentPath != "" {
			link = torrentPath
		} else {
			link = magnetLink
		}

		settings, err := db.LoadSettings()
		if err != nil {
			return nil, fmt.Errorf("torrent client factory loading settings: %w", err)
		}
		tCli, err := tor.NewTorrentClient(
			tor.ClientConfig{
				TorrentPort:          settings.TorrentPort(),
				MaxConnections:       settings.MaxConnections(),
				Seed:                 settings.Seed(),
				TCP:                  settings.TCP(),
				DownloadAheadPercent: 1,
				ValidMediaExtensions: controller.MediaExtensions,
			},
			mediaDir,
			link,
		)
		if err != nil {
			return nil, fmt.Errorf("creating torrent client: %w", err)
		}

		if torrentPath == "" {
			err = saveTorrent(torrentFileDir, tCli.Torrent)
			if err != nil {
				return nil, fmt.Errorf("saving torrent: %w", err)
			}
		}

		return tCli, nil
	}
}

func checkIfTorrentExists(torrentFileDir, torrentPath string) (string, error) {
	m, err := magnet.Parse(torrentPath)
	if err != nil {
		return "", fmt.Errorf("parsing magnet: %w", err)
	}

	if m.InfoHash != "" {
		filename := fmt.Sprintf("%s.torrent", m.InfoHash)
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
		return fmt.Errorf("creating torrent directory: %w", err)
	}

	hash := t.InfoHash().HexString()
	filename := fmt.Sprintf("%s.torrent", hash)
	file := filepath.Join(torrentFileDir, filename)

	f, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("creating torrent file: %w", err)
	}
	defer f.Close()

	mi := t.Metainfo()
	err = mi.Write(f)
	if err != nil {
		return fmt.Errorf("saving torrent: %w", err)
	}

	return nil
}
