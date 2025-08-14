package player

import (
	"context"
	"os/exec"

	"github.com/quintans/faults"
	"github.com/quintans/torflix/internal/model"
)

type Player struct{}

func (p Player) Open(ctx context.Context, player model.Player, url string, subtitlesDir string) error {
	if len(player.Args) == 0 {
		return faults.New("player is undefined")
	}

	return open(ctx, player, url, subtitlesDir)
}

// GenericPlayer represents most players. The stream URL will be appended to the arguments.
type GenericPlayer struct {
	Commands []string
	Subs     string
}

// Open the given stream in a GenericPlayer.
func open(ctx context.Context, p model.Player, url string, subtitlesDir string) error {
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
