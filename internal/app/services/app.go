package services

import (
	"github.com/quintans/faults"
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/model"
)

type App struct {
	repo             Repository
	secrets          app.Secrets
	cacheDir         string
	mediaDir         string
	torrentsDir      string
	subtitlesRootDir string
}

func NewApp(
	repo Repository,
	secrets app.Secrets,
	cacheDir string,
	mediaDir string,
	torrentsDir string,
	subtitlesRootDir string,
) *App {
	return &App{
		repo:             repo,
		secrets:          secrets,
		cacheDir:         cacheDir,
		mediaDir:         mediaDir,
		torrentsDir:      torrentsDir,
		subtitlesRootDir: subtitlesRootDir,
	}
}

func (a *App) LoadData() (model.AppData, error) {
	osSecret, err := a.secrets.GetOpenSubtitles()
	if err != nil {
		return model.AppData{}, faults.Errorf("Failed to retrieve open subtitles password: %w", err)
	}

	settings, err := a.repo.LoadSettings()
	if err != nil {
		return model.AppData{}, faults.Errorf("Failed to load settings: %w", err)
	}

	return model.AppData{
		OpenSubtitles: model.OpenSubtitles{
			Username: settings.OpenSubtitles.Username,
			Password: osSecret.Password,
		},
	}, nil
}

func (a *App) SetOpenSubtitles(username, password string) error {
	err := a.secrets.SetOpenSubtitles(app.OpenSubtitlesSecret{
		Password: password,
	})
	if err != nil {
		return faults.Errorf("Failed to set open subtitles password: %w", err)
	}

	settings, err := a.repo.LoadSettings()
	if err != nil {
		return faults.Errorf("Failed to load settings: %w", err)
	}

	settings.OpenSubtitles.Username = username
	err = a.repo.SaveSettings(settings)
	if err != nil {
		return faults.Errorf("Failed to save settings: %w", err)
	}

	return nil
}
