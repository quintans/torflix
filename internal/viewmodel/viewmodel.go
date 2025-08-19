package viewmodel

import (
	"github.com/quintans/torflix/internal/app"
)

type logAndPubFunc func(err error, msg string, args ...any)

type ViewModel struct {
	eventBus     app.EventBus
	App          *App
	Search       *Search
	Download     *Download
	DownloadList *DownloadList
}

func New(eventBus app.EventBus, app *App, search *Search, download *Download, downloadList *DownloadList) *ViewModel {
	vm := &ViewModel{
		eventBus:     eventBus,
		App:          app,
		Search:       search,
		Download:     download,
		DownloadList: downloadList,
	}

	vm.App.root = vm
	vm.Search.root = vm
	vm.Download.root = vm
	vm.DownloadList.root = vm

	return vm
}
