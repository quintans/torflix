package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/quintans/torflix/internal/model"
)

type DB struct {
	dir      string
	search   *model.Search
	download *model.Download
	settings *model.Settings
}

func NewDB(cacheDir string) *DB {
	dir := filepath.Join(cacheDir, "data")
	os.MkdirAll(dir, os.ModePerm)

	return &DB{
		dir:      dir,
		download: model.NewDownload(),
	}
}

type Search struct {
	SelectedProviders map[string]bool `json:"selectedProviders"`
}

func (d *DB) SaveSearch(search *model.Search) error {
	err := d.write("search.json", Search{
		SelectedProviders: search.SelectedProviders(),
	})
	if err != nil {
		return fmt.Errorf("saving search: %w", err)
	}
	d.search = search

	return nil
}

func (d *DB) LoadSearch() (*model.Search, error) {
	if d.search == nil {
		s := Search{}
		err := d.read("search.json", &s)
		if err != nil {
			return nil, fmt.Errorf("loading search: %w", err)
		}

		search := model.NewSearch()
		search.Hydrate(search.Query(), s.SelectedProviders)

		d.search = search
	}

	return d.search, nil
}

func (d *DB) SaveDownload(download *model.Download) {
	d.download = download
}

func (d *DB) LoadDownload() *model.Download {
	return d.download
}

type Settings struct {
	TorrentPort         int                 `json:"torrentPort"`
	Port                int                 `json:"port"`
	Player              string              `json:"player"`
	Tcp                 bool                `json:"tcp"`
	MaxConnections      int                 `json:"maxConnections"`
	Seed                bool                `json:"seed"`
	Languages           []string            `json:"languages"`
	SearchConfig        json.RawMessage     `json:"searchConfig"`
	DetailsSearchConfig json.RawMessage     `json:"detailsSearchConfig"`
	Providers           []string            `json:"providers"`
	Qualities           []string            `json:"qualities"`
	OpenSubtitles       model.OpenSubtitles `json:"openSubtitles"`
}

func (d *DB) SaveSettings(settings *model.Settings) error {
	err := d.write("settings.json", Settings{
		TorrentPort:         settings.TorrentPort(),
		Port:                settings.Port(),
		Player:              settings.Player().String(),
		Tcp:                 settings.TCP(),
		MaxConnections:      settings.MaxConnections(),
		Seed:                settings.Seed(),
		Languages:           settings.Languages(),
		SearchConfig:        settings.SearchConfig(),
		DetailsSearchConfig: settings.DetailsSearchConfig(),
		Providers:           settings.Providers(),
		Qualities:           settings.Qualities(),
		OpenSubtitles:       settings.OpenSubtitles,
	})
	if err != nil {
		return fmt.Errorf("saving settings: %w", err)
	}

	d.settings = settings

	return nil
}

func (d *DB) LoadSettings() (*model.Settings, error) {
	if d.settings == nil {
		settings := Settings{}
		err := d.read("settings.json", &settings)
		if err != nil {
			return nil, fmt.Errorf("loading settings: %w", err)
		}

		player, err := model.ParsePlayer(settings.Player)
		if err != nil {
			return nil, fmt.Errorf("parsing player: %w", err)
		}

		s := model.NewSettings()
		s.Hydrate(
			settings.TorrentPort,
			settings.Port,
			player,
			settings.Tcp,
			settings.MaxConnections,
			settings.Seed,
			settings.Languages,
			settings.SearchConfig,
			settings.DetailsSearchConfig,
			settings.Providers,
			settings.Qualities,
			settings.OpenSubtitles,
		)

		d.settings = s
	}

	return d.settings, nil
}

func (d *DB) write(file string, data any) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling data for '%s': %w", file, err)
	}

	err = os.WriteFile(filepath.Join(d.dir, file), b, os.ModePerm)
	if err != nil {
		return fmt.Errorf("writing data for '%s': %w", file, err)
	}

	return nil
}

func (d *DB) Exists(file string) bool {
	_, err := os.Stat(filepath.Join(d.dir, file))
	return !errors.Is(err, os.ErrNotExist)
}

func (d *DB) read(file string, data any) error {
	b, err := os.ReadFile(filepath.Join(d.dir, file))
	if err != nil {
		return fmt.Errorf("reading data for '%s': %w", file, err)
	}

	err = json.Unmarshal(b, data)
	if err != nil {
		return fmt.Errorf("unmarshalling data for '%s': %w", file, err)
	}

	return nil
}
