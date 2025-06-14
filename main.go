package main

import (
	"fmt"
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

var (
	htmlSearchConfig = []byte(`{
	"knaben": {
		"name": "KNABEN",
		"queryInPath": true,
		"url": "https://knaben.org/search/{{query}}/0/1/seeders",
		"list": "table > tbody > tr",
		"result": {
			"name": ["td:nth-child(2) > a:first-of-type", "@title"],
			"magnet": ["td:nth-child(2) > a:first-of-type", "@href"],
			"size": "td:nth-child(3)",
			"seeds": "td:nth-child(5)",
			"source": "td:nth-child(7)"
		}
	},
	"nyaa": {
		"name": "NYAA",
		"url": "https://nyaa.si/?f=0&c=0_0&q={{query}}&s=seeders&o=desc",
		"list": "table.torrent-list > tbody > tr",
		"result": {
			"name": ["td:nth-child(2) > a:last-child", "@title"],
			"magnet": ["td:nth-child(3) > a:nth-child(2)", "@href"],
			"size": "td:nth-child(4)",
			"seeds": "td:nth-child(6)"
		}
	},
	"1337x": {
		"name": "1337x",
		"queryInPath": true,
		"url": "https://www.1377x.to/sort-search/{{query}}/seeders/desc/1/",
		"list": "table.table-list > tbody > tr",
		"result": {
			"name": ["td.name > a:nth-child(2)"],
			"follow": ["td.name > a:nth-child(2)", "@href"],
			"size": ["td.size"],
			"seeds": "td.seeds"
		}
	},
	"bt4g": {
		"name": "bt4g",
		"url": "https://bt4gprx.com/search?q={{query}}&category=movie&orderby=seeders&p=1",
		"list": "div.list-group > div.list-group-item",
		"result": {
			"name": ["h5 > a", "@title"],
			"follow": ["h5 > a", "@href"],
			"size": "p > span:nth-child(4) > b",
			"seeds": "p > span:nth-child(5) > b"
		}
	}
}`)
	detailsScrapeConfig = []byte(`{
	"1337x": {
		"name": "1337x",
		"url": "https://1337x.to{{link}}",
		"list": "div.torrent-detail-page",
		"result": {
			"magnet": ["a#openPopup", "@href"]
		}
	},
	"bt4g": {
		"name": "bt4g",
		"url": "https://bt4gprx.com{{link}}",
		"list": "div.card-body",
		"result": {
			"magnet":["a:nth-child(3)", "@href", "/magnet:\\?.*/"]
		}
	}
}`)

	apiSearchConfig = []byte(`{
	"tpb": {
		"url": "https://apibay.org/q.php?q={{.query}}&cat=",
		"result": {
			"name": "name",
			"hash": "info_hash",
			"ssize": "size",
			"seeds": "seeders"
		}
	}
}`)
)

func main() {
	cacheDir := os.Getenv("TORFLIX_CACHE_DIR")
	if cacheDir == "" {
		path, err := os.UserCacheDir()
		if err != nil {
			panic(err)
		}

		cacheDir = filepath.Join(path, "torflix")
	}

	err := os.MkdirAll(cacheDir, os.ModePerm)
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
	w.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
		if k.Name == fyne.KeyEscape && escapeHandler != nil {
			escapeHandler()
		}
	})

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

	extractors := []gapp.Extractor{}
	searchScraper, err := extractor.NewScraper(htmlSearchConfig, detailsScrapeConfig)
	if err != nil {
		panic(fmt.Sprintf("creating search scraper: %s", err))
	}
	extractors = append(extractors, searchScraper)

	if len(apiSearchConfig) > 0 {
		apiSearch, err := extractor.NewApi(apiSearchConfig)
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

	searchCtrl, err := controller.NewSearch(searchView, nav, db, extractors, eventBus, torrentsDir)
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
		cacheDir,
	)

	searchView.SetController(searchCtrl)

	downloadView.SetController(downloadCtrl)
	downloadListView.SetController(downloadCtrl)
	appView.SetController(appCtrl)

	bus.Register(b, appCtrl.OnNavigation)
	bus.Register(b, downloadCtrl.ClearCache)
	bus.Register(b, searchView.ClearCache)
	bus.Register(b, appCtrl.ShowNotification)
	bus.Register(b, appView.Loading)
	bus.Register(b, onEscape)

	appCtrl.OnEnter()
	nav.Go(controller.SearchNavigation)

	w.ShowAndRun()
}

func torrentClientFactory(db *repository.DB, mediaDir, torrentFileDir string) func(torrentPath string) (gapp.TorrentClient, error) {
	return func(link string) (gapp.TorrentClient, error) {
		settings, err := db.LoadSettings()
		if err != nil {
			return nil, fmt.Errorf("torrent client factory loading settings: %w", err)
		}
		tCli, err := tor.NewTorrentClient(
			tor.ClientConfig{
				TorrentPort:          settings.TorrentPort(),
				MaxConnections:       settings.MaxConnections(),
				Seed:                 settings.Seed(),
				SeedAfterComplete:    settings.SeedAfterComplete(),
				TCP:                  settings.TCP(),
				DownloadAheadPercent: 1,
				ValidMediaExtensions: controller.MediaExtensions,
				UploadRate:           settings.UploadRate(),
			},
			torrentFileDir,
			mediaDir,
			link,
		)
		if err != nil {
			return nil, fmt.Errorf("creating torrent client: %w", err)
		}

		return tCli, nil
	}
}

var escapeHandler func()

func onEscape(e gapp.EscapeHandler) {
	escapeHandler = e.Handler
}
