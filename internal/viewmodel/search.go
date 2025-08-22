package viewmodel

import (
	"cmp"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	gslices "slices"
	"sort"
	"strings"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/quintans/faults"
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/bind"
	"github.com/quintans/torflix/internal/lib/magnet"
	"github.com/quintans/torflix/internal/lib/timer"
	"github.com/quintans/torflix/internal/model"
)

type DownloadType int

const (
	_ DownloadType = iota
	DownloadSingle
	DownloadMultiple
)

type SearchService interface {
	LoadSearch() (*SearchSettings, error)
	Search(query string, providers []string) ([]*SearchResult, error)
}

type SearchSettings struct {
	Model     *model.Search
	Providers []string
}

type Search struct {
	root              *ViewModel
	searchService     SearchService
	downloadService   DownloadService
	originalQuery     string
	Query             *bind.Bind[string]
	SelectedProviders *bind.Bind[map[string]bool]
	Providers         []string
	DownloadSubtitles bool
	SearchResults     *bind.Bind[[]*SearchData]
}

type SearchResult struct {
	Data  []*SearchData
	Error error
}

type SearchData struct {
	Provider    string
	Name        string
	Magnet      string
	Size        string
	Seeds       int
	Quality     int
	QualityName string
	Hash        string
	Cached      bool
}

func NewSearch(searchService SearchService, downloadService DownloadService) *Search {
	return &Search{
		searchService:     searchService,
		downloadService:   downloadService,
		Query:             bind.New[string](),
		SelectedProviders: bind.NewMap[string, bool](),
		SearchResults:     bind.NewSlicePtr[SearchData](),
	}
}

func (s *Search) Init() {
	data, err := s.searchService.LoadSearch()
	if err != nil {
		s.root.App.logAndPub(err, "Failed to load search data")
		return
	}
	s.Query.Set(data.Model.Query())

	if len(data.Providers) == 0 {
		s.root.App.logAndPub(nil, "No providers available for search")
		return
	}

	s.Providers = data.Providers
	// the following code must come after setting the providers
	selected := make(map[string]bool, len(data.Providers))
	oldSelection := data.Model.SelectedProviders()
	for _, v := range data.Providers {
		if oldSelection[v] {
			selected[v] = true
		}
	}
	s.SelectedProviders.Set(selected)

	s.root.App.EscapeKey.Notify(nil)
}

func (s *Search) Search(subtitles bool) DownloadType {
	s.root.Download.DownloadSubtitles = subtitles
	query := s.Query.Get()
	if strings.HasPrefix(query, "magnet:") {
		mag, err := magnet.Parse(query)
		if err != nil {
			s.root.App.logAndPub(err, "Failed to parse magnet link")
			return 0
		}
		dn := mag.DisplayName
		if dn == "" {
			dn = fmt.Sprintf("Torrent-%s", mag.Hash)
		} else {
			dn = cleanTorrentName(dn)
		}
		s.originalQuery = dn
		return s.Download(query)
	}

	d := timer.New(time.Second, func() {
		s.root.eventBus.Publish(app.Loading{
			Text: "Searching torrents",
			Show: true,
		})
	})

	defer func() {
		d.Stop()
		s.root.eventBus.Publish(app.Loading{}) // hide spinner
	}()

	s.originalQuery = query
	selectedProviders := s.SelectedProviders.Get()
	providers := []string{}
	for k, v := range selectedProviders {
		if v {
			providers = append(providers, k)
		}
	}

	results, err := s.searchService.Search(query, providers)
	if err != nil {
		s.root.App.logAndPub(err, "Failed to search")
		return 0
	}

	data := []*SearchData{}
	for _, r := range results {
		if r.Error != nil {
			s.root.App.logAndPub(r.Error, "Failed to search")
			continue
		}
		data = append(data, r.Data...)
	}
	// results may be >= 0 but the data may be empty (where seeds = 0)
	if len(data) == 0 {
		s.root.App.ShowNotification.Notify(app.NewNotifyWarn("No results found for query"))
	}

	data, err = s.collapseByHash(data)
	if err != nil {
		s.root.App.logAndPub(err, "Failed to collapse by hash")
		return 0
	}

	gslices.SortFunc(data, func(b, a *SearchData) int {
		if a.Quality != b.Quality {
			return cmp.Compare(a.Quality, b.Quality)
		}
		return cmp.Compare(a.Seeds, b.Seeds)
	})

	s.SearchResults.Set(data)

	return 0
}

func (s *Search) Download(magnetLink string) DownloadType {
	t := timer.New(time.Second, func() {
		s.root.eventBus.Publish(app.Loading{
			Text: "Downloading torrent metadata",
			Show: true,
		})
	})

	defer func() {
		t.Stop()
		s.root.eventBus.Publish(app.Loading{}) // hide spinner
	}()

	files, err := s.downloadService.DownloadTorrent(s.originalQuery, magnetLink)
	if err != nil {
		s.root.App.logAndPub(err, "Failed to download torrent metadata")
		return 0
	}

	if len(files) == 0 {
		s.root.App.ShowNotification.Notify(app.NewNotifyWarn("No media files found for magnet link"))
		return 0
	}

	if len(files) == 1 {
		s.root.Download.Init(files[0], false, s.originalQuery)
		return DownloadSingle
	}

	gslices.SortFunc(files, func(i, j *torrent.File) int {
		return cmp.Compare(i.DisplayPath(), j.DisplayPath())
	})

	s.root.DownloadList.Init(files, s.originalQuery)

	return DownloadMultiple
}

func (s *Search) collapseByHash(results []*SearchData) ([]*SearchData, error) {
	groups := map[string][]*SearchData{}
	for k, r := range results {
		hash := r.Hash
		if hash == "" {
			hash = fmt.Sprintf("no_hash_%d", k)
		}
		h, ok := groups[hash]
		if !ok {
			groups[hash] = []*SearchData{r}
		} else {
			groups[hash] = append(h, r)
		}
	}

	var merged []*SearchData
	for _, group := range groups {
		if len(group) == 1 {
			merged = append(merged, group[0])
		} else {
			magnets := make([]string, 0, len(group))
			for _, r := range group {
				if r.Magnet == "" {
					s.root.App.ShowNotification.Notify(app.NewNotifyWarn("Empty magnet link. name=%s, provider=%s", r.Name, r.Provider))
					continue
				}
				magnets = append(magnets, r.Magnet)
			}

			magnet, dn, err := mergeMagnetLinks(magnets)
			if err != nil {
				return nil, faults.Errorf("merging magnet links: %w", err)
			}

			providers := make([]string, 0, len(group))
			for _, r := range group {
				providers = append(providers, r.Provider)
			}
			sort.Strings(providers)

			maxSeeded := gslices.MaxFunc(group, func(a, b *SearchData) int {
				return cmp.Compare(a.Seeds, b.Seeds)
			})

			merged = append(merged, &SearchData{
				Provider: strings.Join(providers, ","),
				Name:     dn,
				Magnet:   magnet,
				Size:     maxSeeded.Size,
				Seeds:    maxSeeded.Seeds,
				Quality:  maxSeeded.Quality,
				Hash:     maxSeeded.Hash,
			})
		}
	}

	return merged, nil
}

func mergeMagnetLinks(links []string) (string, string, error) {
	if len(links) == 0 {
		return "", "", faults.Errorf("no magnet links provided")
	}

	// Maps to store unique values for each component
	var hash string
	var dns []string
	trackers := make(map[string]struct{})
	webSeeds := make(map[string]struct{})

	for _, link := range links {
		u, err := url.Parse(link)
		if err != nil {
			return "", "", faults.Errorf("failed to parse link (%s): %w", link, err)
		}

		if strings.HasPrefix(u.Scheme, "http") {
			// stop redirect
			checkRedirect := http.DefaultClient.CheckRedirect
			http.DefaultClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			}
			defer func() {
				http.DefaultClient.CheckRedirect = checkRedirect
			}()

			res, err := http.Get(link)
			if err != nil {
				return "", "", faults.Errorf("failed to fetch link (%s): %w", link, err)
			}
			defer res.Body.Close()
			if res.StatusCode != http.StatusTemporaryRedirect &&
				res.StatusCode != http.StatusPermanentRedirect {
				return "", "", faults.Errorf("don't know how to handle non-magnet link: %s; Status Code: %d", link, res.StatusCode)
			}

			link = res.Header.Get("Location")
		}

		mag, err := magnet.Parse(link)
		if err != nil {
			return "", "", faults.Errorf("failed to parse magnet link (%s): %w", link, err)
		}

		dns = append(dns, mag.DisplayName)

		if hash == "" {
			hash = mag.Hash
		} else if hash != mag.Hash {
			return "", "", faults.Errorf("different hashes found when merging: %s and %s", hash, mag.Hash)
		}

		for _, value := range mag.Trackers {
			trackers[value] = struct{}{}
		}

		for _, value := range mag.WebSeeds {
			webSeeds[value] = struct{}{}
		}
	}

	if hash == "" {
		return "", "", faults.Errorf("no hash (xt) found in magnet links")
	}

	// Determine the smallest `dn`
	var smallestDN string
	if len(dns) > 0 {
		sort.Strings(dns)
		smallestDN = dns[0]
	}

	// Build the merged magnet link
	mergedParams := url.Values{}
	mergedParams.Add("xt", hash)
	if smallestDN != "" {
		mergedParams.Add("dn", smallestDN)
	}
	for tracker := range trackers {
		mergedParams.Add("tr", tracker)
	}
	for webSeed := range webSeeds {
		mergedParams.Add("ws", webSeed)
	}

	return fmt.Sprintf("magnet:?%s", mergedParams.Encode()), smallestDN, nil
}

func cleanTorrentName(torrentName string) string {
	// Pattern to identify and retain special markers (Season/Episode info)
	seasonEpisodePattern := regexp.MustCompile(`(?i)\b(S\d{2}(E\d{2})?|Season \d+)\b`)

	// Find the first occurrence of the pattern
	loc := seasonEpisodePattern.FindStringIndex(torrentName)
	if loc != nil {
		// Keep everything up to and including the matched pattern
		torrentName = torrentName[:loc[1]]
	}

	// Remove common metadata from the trimmed name
	patterns := []string{
		`(?i)\b(720p|1080p|2160p|4k|8k)\b`,           // Resolutions
		`(?i)\b(x264|x265|h264|h265)\b`,              // Codecs
		`(?i)\b(WEBRip|BRRip|BluRay|HDTV|WEB-DL)\b`,  // Sources
		`(?i)\b(DTS|DD5\.1|AAC|Atmos|TrueHD|MP3)\b`,  // Audio formats
		`$begin:math:display$\\w+$end:math:display$`, // Text in square brackets
		`$begin:math:text$[^)]+$end:math:text$`,      // Text in parentheses
		`-.*$`,                                       // Trailing release group name
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		loc := re.FindStringIndex(torrentName)
		if loc != nil {
			torrentName = torrentName[:loc[0]]
		}
	}

	// Replace dots and underscores with spaces
	torrentName = strings.ReplaceAll(torrentName, ".", " ")
	torrentName = strings.ReplaceAll(torrentName, "_", " ")

	// Trim extra spaces
	torrentName = strings.TrimSpace(torrentName)

	return torrentName
}
