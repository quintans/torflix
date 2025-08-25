package viewmodel

import (
	"github.com/quintans/torflix/internal/lib/bind"
	"github.com/quintans/torflix/internal/model"
)

type CacheService interface {
	LoadAllCached() ([]*model.CacheData, error)
	SaveCache(data *model.CacheData) error
	ClearCache() error
}

type Cache struct {
	shared          *Shared
	cacheService    CacheService
	downloadService DownloadService
	CacheDir        string
	CacheCleared    bind.Notifier[bool]
	Results         bind.Notifier[[]*model.CacheData]
}

func NewCache(shared *Shared, cacheDir string, cacheService CacheService, downloadService DownloadService) *Cache {
	return &Cache{
		shared:          shared,
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
		c.shared.Error(err, "Failed to load cached data")
		return
	}
	c.Results.Notify(data)
}

func (c *Cache) Unmount() {
	c.Results.UnbindAll()
	c.CacheCleared.UnbindAll()
}

func (c *Cache) Download(data *model.CacheData, subtitles bool) bool {
	return download(c.shared, c.downloadService, data.OriginalQuery, data.Magnet, subtitles)
}

func (c *Cache) Add(originalSearchQuery string, data *model.CacheData) {
	for _, d := range c.Results.Get() {
		if d.Magnet == data.Magnet {
			return
		}
	}

	data.OriginalQuery = originalSearchQuery
	if err := c.cacheService.SaveCache(data); err != nil {
		c.shared.Error(err, "Failed to save cache")
		return
	}
	list := c.Results.Get()
	c.Results.Notify(append(list, data))
}

func (c *Cache) ClearCache() {
	err := c.cacheService.ClearCache()
	if err != nil {
		c.shared.Error(err, "Failed to clear cache")
		return
	}

	c.CacheCleared.Notify(true)

	c.Results.Notify(nil)

	c.shared.Success("Cache cleared")
}
