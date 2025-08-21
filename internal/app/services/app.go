package services

import (
	"os"

	"github.com/quintans/faults"
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/viewmodel"
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

func (a *App) LoadData() (viewmodel.AppData, error) {
	osSecret, err := a.secrets.GetOpenSubtitles()
	if err != nil {
		return viewmodel.AppData{}, faults.Errorf("Failed to retrieve open subtitles password: %w", err)
	}

	settings, err := a.repo.LoadSettings()
	if err != nil {
		return viewmodel.AppData{}, faults.Errorf("Failed to load settings: %w", err)
	}

	return viewmodel.AppData{
		CacheDir: a.cacheDir,
		OpenSubtitles: viewmodel.OpenSubtitles{
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

func (a *App) ClearCache() error {
	err := os.RemoveAll(a.mediaDir)
	if err != nil {
		return faults.Errorf("Failed to clear media cache: %w", err)
	}
	err = os.RemoveAll(a.torrentsDir)
	if err != nil {
		return faults.Errorf("Failed to clear torrent cache: %w", err)
	}
	err = os.RemoveAll(a.subtitlesRootDir)
	if err != nil {
		return faults.Errorf("Failed to clear subtitles cache: %w", err)
	}

	return nil
}
