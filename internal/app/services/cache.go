package services

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/quintans/faults"
	"github.com/quintans/torflix/internal/lib/files"
	"github.com/quintans/torflix/internal/model"
)

type Cache struct {
	cacheDir         string
	mediaDir         string
	torrentsDir      string
	subtitlesRootDir string
}

func NewCache(cacheDir, mediaDir, torrentsDir, subtitlesRootDir string) *Cache {
	return &Cache{
		cacheDir:         cacheDir,
		mediaDir:         mediaDir,
		torrentsDir:      torrentsDir,
		subtitlesRootDir: subtitlesRootDir,
	}
}

func (a *Cache) LoadAllCached() ([]*model.CacheData, error) {
	if !files.Exists(a.cacheDir) {
		return nil, nil
	}

	files, err := os.ReadDir(a.cacheDir)
	if err != nil {
		return nil, faults.Errorf("Failed to read cache directory: %w", err)
	}

	var cachedData []*model.CacheData
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fullpath := filepath.Join(a.cacheDir, file.Name())
		content, err := os.ReadFile(fullpath)
		if err != nil {
			return nil, faults.Errorf("Failed to load cache file %s: %w", fullpath, err)
		}

		var data model.CacheData
		err = json.Unmarshal(content, &data)
		if err != nil {
			return nil, faults.Errorf("Failed to unmarshal cache file %s: %w", fullpath, err)
		}

		cachedData = append(cachedData, &data)
	}

	return cachedData, nil
}

func (a *Cache) SaveCache(data *model.CacheData) error {
	err := os.MkdirAll(a.cacheDir, os.ModePerm)
	if err != nil {
		return faults.Errorf("creating cache directory: %w", err)
	}

	content, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return faults.Errorf("Failed to marshal cache data: %w", err)
	}

	fullpath := filepath.Join(a.cacheDir, data.Hash+".json")
	err = os.WriteFile(fullpath, content, os.ModePerm)
	if err != nil {
		return faults.Errorf("Failed to write cache file %s: %w", fullpath, err)
	}

	return nil
}

func (a *Cache) ClearCache() error {
	err := os.RemoveAll(a.cacheDir)
	if err != nil {
		return faults.Errorf("Failed to clear cached cache: %w", err)
	}
	err = os.RemoveAll(a.mediaDir)
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
	defSubTitlesDir := filepath.Join(a.subtitlesRootDir, "default")
	err = os.MkdirAll(defSubTitlesDir, os.ModePerm)
	if err != nil {
		return faults.Errorf("Failed to recreate default subtitles directory: %w", err)
	}

	return nil
}
