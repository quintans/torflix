package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
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
	searchScraper, err := extractor.NewScraper(settings.SearchConfig())
	if err != nil {
		panic(fmt.Sprintf("creating search scraper: %s", err))
	}
	detailsScraper, err := extractor.NewScraper(settings.DetailsSearchConfig())
	if err != nil {
		panic(fmt.Sprintf("creating details search scraper: %s", err))
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

	searchCtrl, err := controller.NewSearch(searchView, nav, db, searchScraper, detailsScraper, settings.Providers(), eventBus)
	if err != nil {
		panic(fmt.Sprintf("creating search controller: %s", err))
	}

	downloadCtrl := controller.NewDownload(
		downloadView,
		downloadListView,
		db,
		nav,
		player.Player{},
		func(torrentPath string) (gapp.TorrentClient, error) {
			settings, err := db.LoadSettings()
			if err != nil {
				return nil, fmt.Errorf("torrent client factory loading settings: %w", err)
			}
			return tor.NewTorrentClient(
				tor.ClientConfig{
					TorrentPort:          settings.TorrentPort(),
					MaxConnections:       settings.MaxConnections(),
					Seed:                 settings.Seed(),
					TCP:                  settings.TCP(),
					DownloadAheadPercent: 3,
					ValidMediaExtensions: controller.MediaExtensions,
				},
				torrentsDir,
				torrentPath,
			)
		},
		openSubtitlesClientFactory,
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
