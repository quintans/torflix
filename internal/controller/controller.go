package controller

import (
	"fmt"
	"log/slog"

	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/values"
	"github.com/quintans/torflix/internal/model"
)

type Repository interface {
	LoadSearch() (*model.Search, error)
	SaveSearch(search *model.Search) error
	LoadDownload() *model.Download
	SaveDownload(download *model.Download)
	LoadSettings() (*model.Settings, error)
	SaveSettings(model *model.Settings) error
}

const (
	SearchNavigation   = "SEARCH_NAV"
	DownloadNavigation = "DOWNLOAD_NAV"
)

func logAndPub(eb app.EventBus, err error, msg string, args ...any) {
	s := msg
	if len(args) > 0 {
		m := values.ToMap(args)
		s = fmt.Sprintf("%s %s", msg, values.ToStr(m))
	}

	if err != nil {
		eb.Error("%s: %s", s, err)
		slog.Error(msg, append(args, "error", err)...)
		return
	}

	eb.Error(s)
	slog.Error(msg, args...)
}
