package services

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/quintans/faults"
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/extractor"
	"github.com/quintans/torflix/internal/lib/files"
	"github.com/quintans/torflix/internal/lib/values"
	"github.com/quintans/torflix/internal/model"
	"github.com/quintans/torflix/internal/viewmodel"
)

type Search struct {
	repo       Repository
	extractors []app.Extractor
	providers  []string
	torrentDir string
}

func NewSearch(
	repo Repository,
	extractors []app.Extractor,
	torrentDir string,
) (*Search, error) {
	slugSet := map[string]struct{}{}
	for _, xtr := range extractors {
		for _, slug := range xtr.Slugs() {
			slugSet[slug] = struct{}{}
		}
	}

	providers := make([]string, 0, len(slugSet))
	for k := range slugSet {
		providers = append(providers, k)
	}

	slices.Sort(providers)

	model, err := repo.LoadSearch()
	if err != nil {
		return nil, faults.Errorf("loading search: %w", err)
	}
	selectedProviders := model.SelectedProviders()
	changed := false
	for k := range selectedProviders {
		if !slices.Contains(providers, k) {
			delete(selectedProviders, k)
			changed = true
		}
	}
	if changed {
		model.SetSelectedProviders(selectedProviders)
		repo.SaveSearch(model)
	}

	return &Search{
		repo:       repo,
		extractors: extractors,
		providers:  providers,
		torrentDir: torrentDir,
	}, nil
}

func (c *Search) LoadSearch() (*app.SearchSettings, error) {
	model, err := c.repo.LoadSearch()
	if err != nil {
		return nil, faults.Errorf("loading search: %w", err)
	}

	return &app.SearchSettings{
		Model:     model,
		Providers: c.providers,
	}, nil
}

func (c Search) SearchModel() (*model.Search, error) {
	return c.repo.LoadSearch()
}

func (c Search) SaveSearch(model *model.Search) error {
	return c.repo.SaveSearch(model)
}

func (c Search) Search(query string, selectedProviders []string) ([]*viewmodel.SearchResult, error) {
	settings, err := c.repo.LoadSettings()
	if err != nil {
		return nil, faults.Errorf("loading settings: %w", err)
	}
	qualities := settings.Qualities()
	count := 0
	ch := make(chan *viewmodel.SearchResult, len(selectedProviders))
	for _, slug := range selectedProviders {
		for _, xtr := range c.extractors {
			if !xtr.Accept(slug) {
				continue
			}

			count++
			go func(slug string) {
				res, err := xtr.Extract(slug, query)
				if err != nil {
					ch <- &viewmodel.SearchResult{
						Error: faults.Errorf("extracting from %s: %w", slug, err),
					}
					return
				}

				r, err := c.transformToMyResult(slug, res, qualities)
				if err != nil {
					ch <- &viewmodel.SearchResult{
						Error: faults.Errorf("transforming result from %s: %w", slug, err),
					}
					return
				}
				ch <- &viewmodel.SearchResult{
					Data: slices.DeleteFunc(r, func(r *viewmodel.SearchData) bool {
						return r.Seeds == 0
					}),
				}
			}(slug)
		}
	}

	results := make([]*viewmodel.SearchResult, 0, count)
	for range count {
		res := <-ch
		if len(res.Data) == 0 && res.Error == nil {
			continue // skip results with no seeds
		}
		results = append(results, res)
	}

	return results, nil
}

var reHash = regexp.MustCompile(`urn:btih:([a-fA-F0-9]+)`)

func (c Search) transformToMyResult(slug string, r []extractor.Result, qualities []string) ([]*viewmodel.SearchData, error) {
	var results []*viewmodel.SearchData
	for _, r := range r {
		rep := strings.NewReplacer(",", "", ".", "")
		seeds, err := strconv.Atoi(rep.Replace(r.Seeds))
		if err != nil {
			return nil, faults.Errorf("converting seeds '%s' for '%s': %s", r.Seeds, r.Name, err)
		}

		var hash string
		match := reHash.FindStringSubmatch(r.Magnet)
		if len(match) > 1 {
			hash = match[1]
		}

		result := &viewmodel.SearchData{
			Provider: values.Coalesce(r.Source, slug),
			Name:     r.Name,
			Magnet:   r.Magnet,
			Size:     r.Size,
			Seeds:    seeds,
			Cached:   files.Exists(c.torrentDir, strings.ToUpper(hash)+".torrent"),
		}

		for i, q := range qualities {
			name := strings.ToLower(r.Name)
			if strings.Contains(name, q) {
				result.Quality = i + 1
				break
			}
		}

		if result.Quality != 0 {
			result.QualityName = qualities[result.Quality-1]
		} else {
			result.QualityName = "SD"
		}

		results = append(results, result)
	}
	return results, nil
}
