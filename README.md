# TorFlix
Start watching a movie while the torrent it is still downloading.

Features:
- search torrents
- automatic download of subtitles from if user and password for opensubtitles.com
- immediate watch movie while it is still downloading
- download using a magnet link in the search box

> This project is under development

## Build

get the API key from [Open Subtitles API consumers](https://www.opensubtitles.com/en/consumers) and use it in the build command. 

example using as an environment variable (`OS_API_KEY`):

```bash
go build -ldflags "-X github.com/quintans/torflix/internal/gateways/opensubtitles.apiKey=$OS_API_KEY" -o ./builds/ .
```

## Usage
If you want to use subtitles, you have to provide credentials for https://www.opensubtitles.com/ in the settings tab

If you want to scale use:
```dh
FYNE_SCALE=3 ./torflix
```

torflix will create a cache directory in:
- On Unix systems, in $XDG_CACHE_HOME/torflix as specified by
https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html if
non-empty, else $HOME/.cache/torflix.
- On Darwin, in $HOME/Library/Caches/torflix.
- On Windows, in returns %LocalAppData%\torflix.

You can override the cache directory by specifying $TORFLIX_CACHE_DIR

## Troubleshooting

On arch linux if you experience 4K stuttering install flatpak mpv and change the settings `player.args` from `"mpv"` to `"flatpak", "run", "io.mpv.Mpv"`:

```
{
  ...
  player": {
    "args": ["flatpak", "run", "io.mpv.Mpv", ...],
    "subs": ...
  }
  ...
} 
```

## Credits
- [go-peerflix](https://github.com/Sioro-Neoku/go-peerflix) for the awesome example provided
- [torrent](https://github.com/anacrolix/torrent) for the torrent client
- [fyne](https://fyne.io/) for the GUI framework

## License
[MIT](https://raw.githubusercontent.com/quintans/torflix/master/LICENSE)
