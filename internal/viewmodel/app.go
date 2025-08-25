package viewmodel

import (
	"github.com/quintans/torflix/internal/lib/bind"
	"github.com/quintans/torflix/internal/model"
)

type AppService interface {
	LoadData() (model.AppData, error)
	SetOpenSubtitles(username, password string) error
}

type App struct {
	shared          *Shared
	searchService   SearchService
	cacheService    CacheService
	downloadService DownloadService
	appService      AppService
	SelectedTab     int
	OSUsername      *bind.Bind[string]
	OSPassword      *bind.Bind[string]
	Cache           *Cache
	Search          *Search
}

func NewApp(shared *Shared,
	appService AppService,
	searchService SearchService,
	cacheService CacheService,
	downloadService DownloadService,
	cacheDir string,
) *App {
	a := &App{
		shared:          shared,
		appService:      appService,
		searchService:   searchService,
		cacheService:    cacheService,
		downloadService: downloadService,
		Cache:           NewCache(shared, cacheDir, cacheService, downloadService),
		Search:          NewSearch(shared, searchService, downloadService),
	}

	data, err := a.appService.LoadData()
	if err != nil {
		a.shared.Error(err, "Failed to load app data")
		return a
	}

	a.OSUsername = bind.New[string](data.OpenSubtitles.Username)
	a.OSPassword = bind.New[string](data.OpenSubtitles.Password)

	a.shared.EscapeKey.Notify(nil)

	return a
}

func (a *App) Unmount() {
	a.OSUsername.UnbindAll()
	a.OSPassword.UnbindAll()

	a.Cache.Unmount()
	a.Search.Unmount()
}

func (a *App) SetOpenSubtitles(username, password string) {
	err := a.appService.SetOpenSubtitles(username, password)
	if err != nil {
		a.shared.Error(err, "Failed to set open subtitles")
		return
	}

	a.shared.Success("OpenSubtitles credentials saved")

	a.OSUsername.Set(username)
	a.OSPassword.Set(password)
}
