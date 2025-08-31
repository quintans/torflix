package main

import (
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/quintans/faults"
	gapp "github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/app/services"
	"github.com/quintans/torflix/internal/gateways/opensubtitles"
	"github.com/quintans/torflix/internal/gateways/player"
	"github.com/quintans/torflix/internal/gateways/repository"
	"github.com/quintans/torflix/internal/gateways/secrets"
	"github.com/quintans/torflix/internal/gateways/tor"
	"github.com/quintans/torflix/internal/lib/bind"
	"github.com/quintans/torflix/internal/lib/bus"
	"github.com/quintans/torflix/internal/lib/extractor"
	"github.com/quintans/torflix/internal/lib/navigation"
	"github.com/quintans/torflix/internal/model"
	"github.com/quintans/torflix/internal/mycontainer"
	"github.com/quintans/torflix/internal/view"
	"github.com/quintans/torflix/internal/viewmodel"
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
	w := a.NewWindow(fmt.Sprintf("TorFlix v%s", gapp.Version))
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
	bus.Register(b, createDialogListener(w))

	openSubtitlesClientFactory := func(usr, pwd string) gapp.SubtitlesClient {
		return opensubtitles.New(usr, pwd)
	}

	sec := secrets.NewSecrets()
	appSvc := services.NewApp(db, sec, cacheDir, mediaDir, torrentsDir, subtitlesDir)
	searchSvc, err := services.NewSearch(db, extractors, torrentsDir)
	if err != nil {
		panic(fmt.Sprintf("creating search service: %s", err))
	}
	downloadSvc := services.NewDownload(
		db,
		player.Player{},
		torrentClientFactory(db, mediaDir, torrentsDir),
		openSubtitlesClientFactory,
		torrentsDir,
		subtitlesDir,
		sec,
	)

	cachedDir := filepath.Join(cacheDir, "cached")
	cacheSvc := services.NewCache(cachedDir, mediaDir, torrentsDir, subtitlesDir)

	// Root container where screens are swapped
	content := container.NewStack()

	// Notification container
	notification := mycontainer.NewNotification()
	shared := &viewmodel.Shared{
		Publish:          b.Publish,
		ShowNotification: bind.NewNotifier[gapp.Notify](),
	}
	shared.ShowNotification.Listen(showNotification(notification))

	anchor := mycontainer.NewAnchor()
	anchor.Add(content, mycontainer.FillConstraint)
	margin := float32(10)
	anchor.Add(notification.Container(), mycontainer.AnchorConstraints{Top: &margin, Right: &margin})

	navigator := navigation.New(content)
	shared.Navigate = navigator

	navigator.Factory = func(to any) navigation.ViewFactory {
		switch t := to.(type) {
		case gapp.AppParams:
			vm := viewmodel.NewApp(
				shared,
				appSvc,
				searchSvc,
				cacheSvc,
				downloadSvc,
				cachedDir,
			)
			return func() (fyne.CanvasObject, func(bool)) {
				return view.App(vm)
			}
		case gapp.DownloadListParams:
			vm := viewmodel.NewDownloadList(shared, downloadSvc, t)
			return func() (fyne.CanvasObject, func(bool)) {
				return view.DownloadList(vm)
			}
		case gapp.DownloadParams:
			vm := viewmodel.NewDownload(shared, downloadSvc, t)
			return func() (fyne.CanvasObject, func(bool)) {
				return view.Download(vm)
			}
		}

		return nil
	}
	navigator.To(gapp.AppParams{})

	w.SetContent(anchor.Container)
	w.ShowAndRun()
}

func torrentClientFactory(db *repository.DB, mediaDir, torrentFileDir string) func(torrentPath string) (gapp.TorrentClient, error) {
	return func(link string) (gapp.TorrentClient, error) {
		settings, err := db.LoadSettings()
		if err != nil {
			return nil, faults.Errorf("torrent client factory loading settings: %w", err)
		}
		tCli, err := tor.NewTorrentClient(
			tor.ClientConfig{
				TorrentPort:          settings.TorrentPort(),
				MaxConnections:       settings.MaxConnections(),
				Seed:                 settings.Seed(),
				SeedAfterComplete:    settings.SeedAfterComplete(),
				TCP:                  settings.TCP(),
				DownloadAheadPercent: 1,
				FirstDownloadPercent: 0.25,
				ValidMediaExtensions: services.MediaExtensions,
				UploadRate:           settings.UploadRate(),
			},
			torrentFileDir,
			mediaDir,
			link,
		)
		if err != nil {
			return nil, faults.Errorf("creating torrent client: %w", err)
		}

		return tCli, nil
	}
}

func createDialogListener(w fyne.Window) func(msg gapp.Loading) {
	inifiniteProgress := widget.NewProgressBarInfinite()
	inifiniteProgress.Start()
	// Custom content for the dialog
	loadingText := widget.NewLabel("")
	customContent := container.NewVBox(
		loadingText,
		inifiniteProgress,
	)

	// Create the dialog
	loading := dialog.NewCustomWithoutButtons("Working...", customContent, w)
	return func(evt gapp.Loading) {
		if evt.Text != "" {
			loadingText.SetText(evt.Text)
		}

		if evt.Show {
			loading.Show()
			return
		}

		if evt.Text == "" && !evt.Show {
			loading.Hide()
			loadingText.SetText("")
		}
	}
}

func showNotification(notification *mycontainer.NotificationContainer) func(evt gapp.Notify) {
	return func(evt gapp.Notify) {
		go fyne.Do(func() {
			switch evt.Type {
			case gapp.NotifyError:
				notification.ShowError(evt.Message)
			case gapp.NotifyWarn:
				notification.ShowWarning(evt.Message)
			case gapp.NotifyInfo:
				notification.ShowInfo(evt.Message)
			case gapp.NotifySuccess:
				notification.ShowSuccess(evt.Message)
			}
		})
	}
}
