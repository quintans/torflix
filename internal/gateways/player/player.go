package player

import (
	"context"
	"os/exec"

	"github.com/quintans/faults"
	"github.com/quintans/torflix/internal/model"
)

type Player struct{}

func (p Player) Open(ctx context.Context, player model.Player, url string, subtitlesDir string) error {
	for _, pl := range genericPlayers {
		if pl.Name == player.String() {
			return pl.Open(ctx, url, subtitlesDir)
		}
	}

	return faults.Errorf("player %s not found", player)
}

var genericPlayers = []GenericPlayer{
	{Name: "MPV", Args: []string{"mpv", "--save-position-on-quit", "--sub-auto=all", "--hwdec=auto"}, Subs: "--sub-file-paths="},
	{Name: "VLC", Args: []string{"vlc", "--sub-autodetect-fuzzy=1"}, Subs: "--sub-autodetect-path="},
	{Name: "MPlayer", Args: []string{"mplayer"}},
}

// GenericPlayer represents most players. The stream URL will be appended to the arguments.
type GenericPlayer struct {
	Name string
	Args []string
	Subs string
}

// Open the given stream in a GenericPlayer.
func (p GenericPlayer) Open(ctx context.Context, url string, subtitlesDir string) error {
	if p.Subs != "" && subtitlesDir != "" {
		p.Args = append(p.Args, p.Subs+subtitlesDir)
	}
	command := append(p.Args, url)

	// #nosec
	// It is the user's responsibility to pass the correct arguments to open the url.
	c := exec.CommandContext(ctx, command[0], command[1:]...)
	err := c.Start()
	if err != nil {
		return faults.Errorf("error opening player: %w", err)
	}

	err = c.Wait()
	if err != nil {
		return faults.Errorf("waiting player to close: %w", err)
	}

	return nil
}
