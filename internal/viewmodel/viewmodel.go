package viewmodel

import (
	"cmp"
	gslices "slices"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/timer"
)

type ViewModel struct {
	eventBus     app.EventBus
	App          *App
	Search       *Search
	Download     *Download
	DownloadList *DownloadList
	Cache        *Cache
}

func New(
	eventBus app.EventBus,
	app *App,
	cache *Cache,
	search *Search,
	download *Download,
	downloadList *DownloadList,
) *ViewModel {
	vm := &ViewModel{
		eventBus:     eventBus,
		App:          app,
		Cache:        cache,
		Search:       search,
		Download:     download,
		DownloadList: downloadList,
	}

	vm.App.root = vm
	vm.Search.root = vm
	vm.Download.root = vm
	vm.DownloadList.root = vm
	vm.Cache.root = vm

	return vm
}

func download(vm *ViewModel, downloadService DownloadService, originalQuery, magnetLink string) DownloadType {
	t := timer.New(time.Second, func() {
		vm.eventBus.Publish(app.Loading{
			Text: "Downloading torrent metadata",
			Show: true,
		})
	})

	defer func() {
		t.Stop()
		vm.eventBus.Publish(app.Loading{}) // hide spinner
	}()

	files, err := downloadService.DownloadTorrent(magnetLink)
	if err != nil {
		vm.App.logAndPub(err, "Failed to download torrent metadata")
		return 0
	}

	if len(files) == 0 {
		vm.App.ShowNotification.Notify(app.NewNotifyWarn("No media files found for magnet link"))
		return 0
	}

	if len(files) == 1 {
		vm.Download.Init(files[0], false, originalQuery)
		return DownloadSingle
	}

	gslices.SortFunc(files, func(i, j *torrent.File) int {
		return cmp.Compare(i.DisplayPath(), j.DisplayPath())
	})

	vm.DownloadList.Init(files, originalQuery)

	return DownloadMultiple
}
