package model

import (
	"errors"

	"github.com/quintans/faults"
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

	return Player{}, faults.Errorf("%w: %s", ErrInvalidPlayer, s)
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
	torrentPort       int
	port              int
	player            Player
	tcp               bool
	maxConnections    int
	seed              bool
	seedAfterComplete bool
	languages         []string
	qualities         []string
	uploadRate        int
	OpenSubtitles     OpenSubtitles
}

type OpenSubtitles struct {
	Username string `json:"username"`
}

func NewSettings() *Settings {
	return &Settings{
		port:              8080,
		player:            MPV,
		torrentPort:       50007,
		seed:              true,
		seedAfterComplete: false,
		tcp:               true,
		maxConnections:    200,
		languages:         []string{"po-PT", "pt-BR", "en"},
		qualities:         qualities,
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

func (m *Settings) Qualities() []string {
	return m.qualities
}

func (m *Settings) SetQualities(qualities []string) {
	m.qualities = qualities
}

func (m *Settings) UploadRate() int {
	return m.uploadRate
}

func (m *Settings) SetUploadRate(uploadRate int) {
	m.uploadRate = uploadRate
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
	uploadRate int,
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
	m.qualities = qualities
	m.uploadRate = uploadRate
	m.OpenSubtitles = OpenSubtitles
}

var (
	qualities = []string{"720p", "1080p", "2160p"}
)
