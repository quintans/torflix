package controller

import "github.com/quintans/torflix/internal/model"

type Repository interface {
	LoadSearch() (*model.Search, error)
	SaveSearch(search *model.Search) error
	LoadDownload() *model.Download
	SaveDownload(download *model.Download)
	LoadSettings() (*model.Settings, error)
	SaveSettings(model *model.Settings) error
}

const (
	SearchNavigation   = "SEARCH_NAV"
	DownloadNavigation = "DOWNLOAD_NAV"
)
