package viewmodel

import (
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/bind"
	"github.com/quintans/torflix/internal/model"
)

type CacheService interface {
	LoadAllCached() ([]*model.CacheData, error)
	SaveCache(data *model.CacheData) error
	ClearCache() error
}

type Cache struct {
	root            *ViewModel
	cacheService    CacheService
	downloadService DownloadService
	CacheDir        string
	CacheCleared    bind.Notifier[bool]
	Results         bind.Notifier[[]*model.CacheData]
}

func NewCache(cacheDir string, cacheService CacheService, downloadService DownloadService) *Cache {
	return &Cache{
		cacheService:    cacheService,
		downloadService: downloadService,
		CacheDir:        cacheDir,
		CacheCleared:    bind.NewNotifier[bool](),
		Results:         bind.NewNotifier[[]*model.CacheData](),
	}
}

func (c *Cache) Mount() {
	data, err := c.cacheService.LoadAllCached()
	if err != nil {
		c.root.App.logAndPub(err, "Failed to load cached data")
		return
	}
	c.Results.Notify(data)
}

func (c *Cache) Unmount() {
	c.Results.UnbindAll()
	c.CacheCleared.UnbindAll()
}

func (c *Cache) Download(data *model.CacheData) DownloadType {
	return download(c.root, c.downloadService, data.OriginalQuery, data.Magnet)
}

func (c *Cache) Add(data *model.CacheData) {
	for _, d := range c.Results.Get() {
		if d.Magnet == data.Magnet {
			return
		}
	}

	data.OriginalQuery = c.root.Search.originalQuery
	if err := c.cacheService.SaveCache(data); err != nil {
		c.root.App.logAndPub(err, "Failed to save cache")
		return
	}
	list := c.Results.Get()
	c.Results.Notify(append(list, data))
}

func (c *Cache) ClearCache() {
	err := c.cacheService.ClearCache()
	if err != nil {
		c.root.App.logAndPub(err, "Failed to clear cache")
		return
	}

	c.CacheCleared.Notify(true)

	c.Results.Notify(nil)

	c.root.App.ShowNotification.Notify(app.NewNotifyInfo("Cache cleared"))
}
