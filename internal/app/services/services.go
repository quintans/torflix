package services

import "github.com/quintans/torflix/internal/model"

type Repository interface {
	LoadSearch() (*model.Search, error)
	SaveSearch(search *model.Search) error
	LoadSettings() (*model.Settings, error)
	SaveSettings(model *model.Settings) error
}
