# TorFlix
Start watching a movie while the torrent it is still downloading.

Features:
- search torrents
- automatic download of subtitles from if user and password for opensubtitles.com
- immediate watch movie while it is still downloading
- download using a magnet link in the search box

> This project is under development

## Installation

You to have [golang](https://go.dev/dl/) configured to install torflix:

```sh
go get github.com/quintans/torflix
```

Install mpv for your system from https://mpv.io/installation/

## Usage
If you want to use subtitles, you have to provide credentials for https://www.opensubtitles.com/ in the settings tab


If you want to scale use:
```dh
FYNE_SCALE=3 ./torflix
```

## Build

Building only for the current platform:

```bash
go build -ldflags "-X github.com/quintans/torflix/internal/gateways/opensubtitles.APIKey=$OS_API_KEY" -o ./builds/ .
```

where `OS_API_KEY` is an environment variable with the opensubtitles apikey.

## Credits
- [go-peerflix](https://github.com/Sioro-Neoku/go-peerflix) for the awesome example provided
- [torrent](https://github.com/anacrolix/torrent) for the torrent client
- [fyne](https://fyne.io/) for the GUI framework

## License
[MIT](https://raw.githubusercontent.com/quintans/torflix/master/LICENSE)
