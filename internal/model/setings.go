package model

import (
	"errors"
	"fmt"
)

var ErrInvalidPlayer = errors.New("invalid player")

type Player struct {
	val string
}

func (t Player) String() string {
	return t.val
}

func ParsePlayer(s string) (Player, error) {
	for _, p := range Players {
		if p.val == s {
			return p, nil
		}
	}

	return Player{}, fmt.Errorf("%w: %s", ErrInvalidPlayer, s)
}

var (
	MPV     = Player{"MPV"}
	VLC     = Player{"VLC"}
	MPlayer = Player{"MPlayer"}
)

var Players = []Player{
	MPV,
	VLC,
	MPlayer,
}

type Settings struct {
	torrentPort             int
	port                    int
	player                  Player
	tcp                     bool
	maxConnections          int
	seed                    bool
	seedAfterComplete       bool
	languages               []string
	htmlSearchConfig        []byte
	htmlDetailsSearchConfig []byte
	apiSearchConfig         []byte
	qualities               []string
	OpenSubtitles           OpenSubtitles
}

type OpenSubtitles struct {
	Username string `json:"username"`
}

func NewSettings() *Settings {
	return &Settings{
		port:                    8080,
		player:                  MPV,
		torrentPort:             50007,
		seed:                    true,
		seedAfterComplete:       false,
		tcp:                     true,
		maxConnections:          200,
		languages:               []string{"po-PT", "pt-BR", "en"},
		qualities:               qualities,
		htmlSearchConfig:        htmlSearchConfig,
		htmlDetailsSearchConfig: detailsScrapeConfig,
		apiSearchConfig:         apiSearchConfig,
		OpenSubtitles: OpenSubtitles{
			Username: "",
		},
	}
}

func (m *Settings) Port() int {
	return m.port
}

func (m *Settings) SetPort(port int) {
	// TODO validate port. Must be between 1024 and 65535
	m.port = port
}

func (m *Settings) Player() Player {
	return m.player
}

func (m *Settings) SetPlayer(player Player) {
	m.player = player
}

func (m *Settings) TorrentPort() int {
	return m.torrentPort
}

func (m *Settings) SetTorrentPort(port int) {
	// TODO validate port. Must be between 1024 and 65535
	m.torrentPort = port
}

func (m *Settings) TCP() bool {
	return m.tcp
}

func (m *Settings) SetTCP(tcp bool) {
	m.tcp = tcp
}

func (m *Settings) MaxConnections() int {
	return m.maxConnections
}

func (m *Settings) SetMaxConnections(maxConnections int) {
	m.maxConnections = maxConnections
}

func (m *Settings) Seed() bool {
	return m.seed
}

func (m *Settings) SetSeed(seed bool) {
	m.seed = seed
}

func (m *Settings) SeedAfterComplete() bool {
	return m.seedAfterComplete
}

func (m *Settings) SetSeedAfterComplete(seedAfterComplete bool) {
	m.seedAfterComplete = seedAfterComplete
}

func (m *Settings) Languages() []string {
	return m.languages
}

func (m *Settings) SetLanguages(languages []string) {
	m.languages = languages
}

func (m *Settings) HtmlSearchConfig() []byte {
	return m.htmlSearchConfig
}

func (m *Settings) SetHtmlSearchConfig(searchConfig []byte) {
	m.htmlSearchConfig = searchConfig
}

func (m *Settings) HtmlDetailsSearchConfig() []byte {
	return m.htmlDetailsSearchConfig
}

func (m *Settings) SetHtmlDetailsSearchConfig(detailsSearchConfig []byte) {
	m.htmlDetailsSearchConfig = detailsSearchConfig
}

func (m *Settings) ApiSearchConfig() []byte {
	return m.apiSearchConfig
}

func (m *Settings) SetApiSearchConfig(apiSearchConfig []byte) {
	m.apiSearchConfig = apiSearchConfig
}

func (m *Settings) Qualities() []string {
	return m.qualities
}

func (m *Settings) SetQualities(qualities []string) {
	m.qualities = qualities
}

func (m *Settings) Hydrate(
	torrentPort int,
	port int,
	player Player,
	tcp bool,
	maxConnections int,
	seed bool,
	seedAfterComplete bool,
	languages []string,
	searchConfig []byte,
	detailsSearchConfig []byte,
	apiSearchConfig []byte,
	qualities []string,
	OpenSubtitles OpenSubtitles,
) {
	m.torrentPort = torrentPort
	m.port = port
	m.player = player
	m.tcp = tcp
	m.maxConnections = maxConnections
	m.seed = seed
	m.seedAfterComplete = seedAfterComplete
	m.languages = languages
	m.htmlSearchConfig = searchConfig
	m.htmlDetailsSearchConfig = detailsSearchConfig
	m.apiSearchConfig = apiSearchConfig
	m.qualities = qualities
	m.OpenSubtitles = OpenSubtitles
}

var (
	qualities        = []string{"720p", "1080p", "2160p"}
	htmlSearchConfig = []byte(`{
	"tgx": {
		"name": "TORRENT GALAXY",
		"url": "https://torrentgalaxy.to/torrents.php?search={{query}}&lang=0&nox=2&sort=seeders&order=desc",
		"list": "div.tgxtablerow",
		"result": {
			"name": ["div.tgxtablecell > div > a[title]", "@title"],
			"magnet": ["div.tgxtablecell > a[role='button']", "@href"],
			"size": "div.tgxtablecell > span.badge.badge-secondary",
			"seeds": "div.tgxtablecell > span[title='Seeders/Leechers'] > font[color='green'] > b"
		}
	},
	"thb": {
		"queryInPath": true,
		"name": "THE PIRATE BAY",
		"url": "https://thehiddenbay.com/search/{{query}}/1/99/0",
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
		"url": "https://nyaa.si/?f=0&c=0_0&q={{query}}&s=seeders&o=desc",
		"list": "table.torrent-list > tbody > tr",
		"result": {
			"name": ["td:nth-child(2) > a", "@title"],
			"magnet": ["td:nth-child(3) > a:nth-child(2)", "@href"],
			"size": "td:nth-child(4)",
			"seeds": "td:nth-child(6)"
		}
	},
	"1337x": {
		"name": "1337x",
		"url": "https://1337x.to/sort-search/{{query}}/seeders/desc/1/",
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
		"url": "https://bt4gprx.com/search?q={{query}}&category=movie&orderby=seeders&p=1",
		"list": "div.list-group > div.list-group-item",
		"result": {
			"name": ["h5 > a", "@title"],
			"follow": ["h5 > a", "@href"],
			"size": "p > span:nth-child(4) > b",
			"seeds": "p > span:nth-child(5) > b"
		}
	}
}`)
	detailsScrapeConfig = []byte(`{
	"1337x": {
		"name": "1337x",
		"url": "https://1337x.to{{link}}",
		"list": "div.torrent-detail-page",
		"result": {
			"magnet": ["a#openPopup", "@href"]
		}
	},
	"bt4g": {
		"name": "bt4g",
		"url": "https://bt4gprx.com{{link}}",
		"list": "div.card-body",
		"result": {
			"magnet":["a:nth-child(3)", "@href", "/magnet:\\?.*/"]
		}
	}
}`)

	apiSearchConfig = []byte(`{
	"tpb": {
		"url": "https://apibay.org/q.php?q={{.query}}&cat=",
		"result": {
			"name": "name",
			"hash": "info_hash",
			"ssize": "size",
			"seeds": "seeders"
		}
	}
}`)
)
