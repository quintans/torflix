package extractor_test

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/quintans/torflix/internal/lib/extractor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractor(t *testing.T) {
	tests := []struct {
		name    string
		results []extractor.Result
		magnet  string
	}{
		{
			name: "tgx",
			results: []extractor.Result{
				{
					Name:   "Lioness. 2023. S02E08. The Compass Points Home. 1080P. AMZN WEB-DL. DDP5.1. HEVC-X265. POOTLED.mkv",
					Magnet: "magnet:?xt=urn:btih:fadd9563df3aeca535a8c0e27799234e22002160",
					Size:   "1.42 GB",
					Seeds:  "86",
				},
				{
					Name:   "Lioness.2023.S02E08.720p.WEB.h264-DiRT",
					Magnet: "magnet:?xt=urn:btih:ca94d42855031243feb50f71ec20a0f309f95b78",
					Size:   "610.46 MB",
					Seeds:  "44",
				},
				{
					Name:   "Operazione.Speciale.Lioness.S02E08.La.Bussola.Punta.Verso.Casa.1080p.AMZN.WEB-DL.DDP2.0.H264-gattopollo.mkv",
					Magnet: "magnet:?xt=urn:btih:d4d9cf624293f0bba679b1b9ddd835fea27462e1",
					Size:   "3.49 GB",
					Seeds:  "1",
				},
				{
					Name:   "Special.Ops.Lioness.S02E08.La.bussola.punta.verso.casa.ITA.ENG.2160p.AMZN.WEB-DL.DDP2.0.H.265-MeM.GP.mkv",
					Magnet: "magnet:?xt=urn:btih:3ffdae30f91485c624e13025a708918d92187fe4",
					Size:   "5.62 GB",
					Seeds:  "1",
				},
			},
		},
		{
			name: "tpb",
			results: []extractor.Result{
				{
					Name:   "SAS Rogue Heroes S02E01 1080p HEVC x265-MeGusta",
					Magnet: "magnet:?xt=urn:btih:991B63685C6BB91E2A199D8495ECE6AA605A161C",
					Size:   "346.98\u00a0MiB",
					Seeds:  "469",
				},
				{
					Name:   "SAS.Rogue.Heroes.S02E01.1080p.x265-ELiTE",
					Magnet: "magnet:?xt=urn:btih:E8FD58643AFA38A42F298401C8AEC52C061194B5",
					Size:   "795.41\u00a0MiB",
					Seeds:  "44",
				},
			},
		},
		{
			name: "nyaa",
			results: []extractor.Result{
				{
					Name:   "[SubsPlease] Ameku Takao no Suiri Karte - 01 (1080p) [EC89E1D8].mkv",
					Magnet: "magnet:?xt=urn:btih:a3ff6c270c4da41728263e530b255ad686703b5b",
					Size:   "1.4 GiB",
					Seeds:  "839",
				},
				{
					Name:   "[ANi] Ameku MD Doctor Detective / 天久鷹央的推理病歷表 - 01 [1080P][Baha][WEB-DL][AAC AVC][CHT][MP4]",
					Magnet: "magnet:?xt=urn:btih:adfc5700a18e1d8720c9a2b584c426127467986a",
					Size:   "342.0 MiB",
					Seeds:  "220",
				},
			},
		},
		{
			name: "1337x",
			results: []extractor.Result{
				{
					Name:   "Lioness-2023-S02E08-1080p-x265-ELiTE",
					Magnet: "",
					Size:   "1.3 GB",
					Seeds:  "479",
					Follow: "/torrent/6284142/Lioness-2023-S02E08-1080p-x265-ELiTE/",
				},
				{
					Name:   "Special-Ops-Lioness-S02E08-La-bussola-punta-verso-casa-ITA-ENG-2160p-AMZN-WEB-DL-DDP2-0-H-265-MeM-GP-mkv",
					Magnet: "",
					Size:   "5.6 GB",
					Seeds:  "119",
					Follow: "/torrent/6299111/Special-Ops-Lioness-S02E08-La-bussola-punta-verso-casa-ITA-ENG-2160p-AMZN-WEB-DL-DDP2-0-H-265-MeM-GP-mkv/",
				},
				{
					Name:   "Lioness-2023-S02E08-480p-x264-RUBiK",
					Magnet: "",
					Size:   "519.2 MB",
					Seeds:  "123",
					Follow: "/torrent/6284120/Lioness-2023-S02E08-480p-x264-RUBiK/",
				},
				{
					Name:   "Operazione-Speciale-Lioness-S02E08-La-Bussola-Punta-Verso-Casa-1080p-AMZN-WEB-DL-DDP2-0-H264-gattopollo-mkv",
					Magnet: "",
					Size:   "3.5 GB",
					Seeds:  "86",
					Follow: "/torrent/6299110/Operazione-Speciale-Lioness-S02E08-La-Bussola-Punta-Verso-Casa-1080p-AMZN-WEB-DL-DDP2-0-H264-gattopollo-mkv/",
				},
				{
					Name:   "Lioness-2023-S02E08-720p-x265-TiPEX",
					Magnet: "",
					Size:   "727.1 MB",
					Seeds:  "74",
					Follow: "/torrent/6284131/Lioness-2023-S02E08-720p-x265-TiPEX/",
				},
			},
			magnet: "magnet:?xt=urn:btih:5DC47BE41CC1277A7F0A4201FBF1A949B542E21B",
		},
		{
			name: "bt4g",
			results: []extractor.Result{
				{
					Name:   "Lioness (2023) Season 2 S02 (2160p AMZN WEB-DL x265 HEVC 10bit DDP 5.1 Vyndros)",
					Magnet: "",
					Size:   "25.14GB",
					Seeds:  "129",
					Follow: "/magnet/NuJ2HGbSoUIjXWexVdcmfB2EM0fZg6AvD",
				},
				{
					Name:   "Lioness.S02.2160p.PMTP.WEB-DL.DDP5.1.H.265.DUAL-PiA",
					Magnet: "",
					Size:   "26.09GB",
					Seeds:  "0",
					Follow: "/magnet/cNUzsFlNMdrWCw3GjBejNoS3ffoYoaKeN",
				},
			},
			magnet: "magnet:?xt.1=urn:btih:80bd6bbd5703e293183350e4cdf9a8638a61f24a&dn=Lioness%20%282023%29%20Season%202%20S02%20%282160p%20AMZN%20WEB-DL%20x265%20HEVC%2010bit%20DDP%205.1%20Vyndros%29",
		},
	}

	searchScraper, err := extractor.NewScraper(searchConfig)
	require.NoError(t, err)

	detailScraper, err := extractor.NewScraper(detailsSearchConfig)
	require.NoError(t, err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &http.Server{
				Addr: ":1234",
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var filename string
					if strings.HasPrefix(r.URL.Path, "/follow/") {
						filename = fmt.Sprintf("./%s-follow.html", tt.name)
					} else {
						filename = fmt.Sprintf("./%s.html", tt.name)
					}

					body, err := os.ReadFile(filename)
					require.NoError(t, err)

					w.WriteHeader(http.StatusOK)
					_, err = w.Write(body)
					require.NoError(t, err)
				}),
			}

			go func() {
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					slog.Error("ListenAndServe()", "error", err)
				}
			}()

			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := server.Shutdown(ctx); err != nil {
					slog.Error("Server Shutdown Failed.", "error", err)
				}
			}()

			results, err := searchScraper.ScrapeQuery(tt.name, "something with spaces")
			require.NoError(t, err)

			for i := range results {
				results[i].Name = removeExtraSpaces(results[i].Name)
			}

			require.Equal(t, tt.results, results)

			if tt.magnet != "" {
				details, err := detailScraper.ScrapeLink(tt.name, results[0].Follow)
				require.NoError(t, err)
				require.NotEmpty(t, details)
				assert.Equal(t, tt.magnet, details[0].Magnet)
			}
		})
	}
}

func removeExtraSpaces(input string) string {
	input = strings.ReplaceAll(input, "\n", " ")
	words := strings.Fields(input)
	return strings.Join(words, " ")
}

var searchConfig = []byte(`{
	"tgx": {
		"name": "TORRENT GALAXY",
		"url": "http://localhost:1234/search.php?search={{query}}&lang=0&nox=2&sort=seeders&order=desc",
		"list": "div.tgxtablerow",
		"result": {
			"name": ["div.tgxtablecell > div > a[title]", "@title"],
			"magnet": ["div.tgxtablecell > a[role='button']", "@href", "/(magnet:\\?xt=urn:btih:[A-Za-z0-9]+)/"],
			"size": "div.tgxtablecell > span.badge.badge-secondary",
			"seeds": "div.tgxtablecell > span[title='Seeders/Leechers'] > font[color='green'] > b"
		}
	},
	"tpb": {
		"queryInPath": true,
		"name": "THE PIRATE BAY",
		"url": "http://localhost:1234/search/{{query}}/1/99/0",
		"list": "table#searchResult > tbody > tr",
		"result": {
			"name": "td:nth-child(2) > div.detName > a",
			"magnet": ["td:nth-child(2) > a", "@href", "/(magnet:\\?xt=urn:btih:[A-Za-z0-9]+)/"],
			"size": ["td:nth-child(2) > font", "/Size (.*?B),/", ""],
			"seeds": "td:nth-child(3)"
		}
	},
	"nyaa": {
		"name": "NYAA",
		"url": "http://localhost:1234/?f=0&c=0_0&q={{query}}&s=seeders&o=desc",
		"list": "table.torrent-list > tbody > tr",
		"result": {
			"name": ["td:nth-child(2) > a", "@title"],
			"magnet": ["td:nth-child(3) > a:nth-child(2)", "@href", "/(magnet:\\?xt=urn:btih:[A-Za-z0-9]+)/"],
			"size": "td:nth-child(4)",
			"seeds": "td:nth-child(6)"
		}
	},
	"1337x": {
		"name": "1337x",
		"url": "http://localhost:1234/sort-search/{{query}}/seeders/desc/1/",
		"list": "table.table-list > tbody > tr",
		"result": {
			"name": ["td.name > a:nth-child(2)", "@href", "/\/torrent\/[0-9]+\/(.*?)\//"],
			"follow": ["td.name > a:nth-child(2)", "@href"],
			"size": ["td.size", "/^(.*?B)/"],
			"seeds": "td.seeds"
		}
	},
	"bt4g": {
		"name": "bt4g",
		"url": "http://localhost:1234/search?q={{query}}&category=movie&orderby=seeders&p=1",
		"list": "div.list-group > div.list-group-item",
		"result": {
			"name": ["h5 > a", "@title"],
			"follow": ["h5 > a", "@href"],
			"size": "p > span:nth-child(4) > b",
			"seeds": "p > span:nth-child(5) > b"
		}
	}
}`)

var detailsSearchConfig = []byte(`{
	"1337x": {
		"name": "1337x",
		"url": "http://localhost:1234/follow/{{link}}",
		"list": "div.torrent-detail-page",
		"result": {
			"magnet": ["a#openPopup", "@href", "/(magnet:\\?xt=urn:btih:[A-Za-z0-9]+)/"]
		}
	},
	"bt4g": {
		"name": "bt4g",
		"url": "http://localhost:1234/follow/{{link}}",
		"list": "div.card-body",
		"result": {
			"magnet":["a:nth-child(3)", "@href", "/magnet:\\?.*/"]
		}
	}
}`)
