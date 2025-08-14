package repository

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/quintans/faults"
	"github.com/quintans/torflix/internal/lib/files"
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
	LastQuery         string          `json:"lastQuery"`
	SelectedProviders map[string]bool `json:"selectedProviders"`
}

func (d *DB) SaveSearch(search *model.Search) error {
	err := d.write("search.json", Search{
		LastQuery:         search.Query(),
		SelectedProviders: search.SelectedProviders(),
	})
	if err != nil {
		return faults.Errorf("saving search: %w", err)
	}
	d.search = search

	return nil
}

func (d *DB) LoadSearch() (*model.Search, error) {
	if d.search == nil {
		s := Search{}
		err := d.read("search.json", &s)
		if err != nil {
			return nil, faults.Errorf("loading search: %w", err)
		}

		search := model.NewSearch()
		search.Hydrate(s.LastQuery, s.SelectedProviders)

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
	TorrentPort             int                 `json:"torrentPort"`
	Port                    int                 `json:"port"`
	Player                  model.Player        `json:"player"`
	Tcp                     bool                `json:"tcp"`
	MaxConnections          int                 `json:"maxConnections"`
	Seed                    bool                `json:"seed"`
	SeedAfterComplete       bool                `json:"seedAfterComplete"`
	Languages               []string            `json:"languages"`
	HtmlSearchConfig        json.RawMessage     `json:"htmlSearchConfig"`
	HtmlDetailsSearchConfig json.RawMessage     `json:"htmlDetailsSearchConfig"`
	ApiSearchConfig         json.RawMessage     `json:"apiSearchConfig"`
	Qualities               []string            `json:"qualities"`
	OpenSubtitles           model.OpenSubtitles `json:"openSubtitles"`
	UploadRate              int                 `json:"uploadRate"`
}

func (d *DB) SaveSettings(settings *model.Settings) error {
	err := d.write("settings.json", Settings{
		TorrentPort:    settings.TorrentPort(),
		Port:           settings.Port(),
		Player:         settings.Player(),
		Tcp:            settings.TCP(),
		MaxConnections: settings.MaxConnections(),
		Seed:           settings.Seed(),
		Languages:      settings.Languages(),
		Qualities:      settings.Qualities(),
		UploadRate:     settings.UploadRate(),
		OpenSubtitles:  settings.OpenSubtitles,
	})
	if err != nil {
		return faults.Errorf("saving settings: %w", err)
	}

	d.settings = settings

	return nil
}

func (d *DB) LoadSettings() (*model.Settings, error) {
	if d.settings == nil {
		settings := Settings{}
		err := d.read("settings.json", &settings)
		if err != nil {
			return nil, faults.Errorf("loading settings: %w", err)
		}

		s := model.NewSettings()
		s.Hydrate(
			settings.TorrentPort,
			settings.Port,
			settings.Player,
			settings.Tcp,
			settings.MaxConnections,
			settings.Seed,
			settings.SeedAfterComplete,
			settings.Languages,
			settings.HtmlSearchConfig,
			settings.HtmlDetailsSearchConfig,
			settings.ApiSearchConfig,
			settings.Qualities,
			settings.UploadRate,
			settings.OpenSubtitles,
		)

		d.settings = s
	}

	return d.settings, nil
}

func (d *DB) write(file string, data any) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return faults.Errorf("marshalling data for '%s': %w", file, err)
	}

	err = os.WriteFile(filepath.Join(d.dir, file), b, os.ModePerm)
	if err != nil {
		return faults.Errorf("writing data for '%s': %w", file, err)
	}

	return nil
}

func (d *DB) Exists(file string) bool {
	return files.Exists(filepath.Join(d.dir, file))
}

func (d *DB) read(file string, data any) error {
	b, err := os.ReadFile(filepath.Join(d.dir, file))
	if err != nil {
		return faults.Errorf("reading data for '%s': %w", file, err)
	}

	err = json.Unmarshal(b, data)
	if err != nil {
		return faults.Errorf("unmarshalling data for '%s': %w", file, err)
	}

	return nil
}
