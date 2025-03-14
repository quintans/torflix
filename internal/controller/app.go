package controller

import (
	"fmt"
	"time"

	"github.com/pkg/browser"
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/lib/navigator"
)

type OnExit interface {
	OnExit()
}

type OnBack interface {
	OnBack()
}

type App struct {
	view               app.AppView
	targets            map[string]app.Controller
	oldChildController app.Controller
	eventBus           app.EventBus
	repo               Repository
	secrets            app.Secrets
	cacheDir           string
	libAuth            app.LibraryAuth
	osEnabled          bool
	traktEnabled       bool
}

func NewApp(
	view app.AppView,
	navigations map[string]app.Controller,
	eventBus app.EventBus,
	repo Repository,
	secrets app.Secrets,
	cacheDir string,
	libAuth app.LibraryAuth,
) *App {
	return &App{
		view:     view,
		targets:  navigations,
		eventBus: eventBus,
		repo:     repo,
		secrets:  secrets,
		cacheDir: cacheDir,
		libAuth:  libAuth,
	}
}

func (a *App) OnEnter() {
	traktData, err := a.secrets.GetTrackt()
	if err != nil {
		logAndPub(a.eventBus, err, "Failed to retrieve Trakt data")
	}

	osSecret, err := a.secrets.GetOpenSubtitles()
	if err != nil {
		logAndPub(a.eventBus, err, "Failed to retrieve open subtitles password")
	}

	settings, err := a.repo.LoadSettings()
	if err != nil {
		logAndPub(a.eventBus, err, "Failed to load settings")
	}

	a.traktEnabled = traktData != app.TraktSecret{}
	a.view.Show(app.AppData{
		CacheDir: a.cacheDir,
		OpenSubtitles: app.OpenSubtitles{
			Username: settings.OpenSubtitles.Username,
			Password: osSecret.Password,
		},
		Trakt: app.Trakt{
			Connected: a.traktEnabled,
		},
	})

	a.osEnabled = settings.OpenSubtitles.Username != "" && osSecret.Password != ""

	if !a.traktEnabled || !a.osEnabled {
		a.view.DisableAllTabsButSettings()
	}
}

func (a *App) reenableTabs() {
	if a.canReenableTabs() {
		a.view.EnableTabs(true)
	}
}
func (a *App) canReenableTabs() bool {
	return a.traktEnabled && a.osEnabled
}

func (a *App) OnNavigation(vc navigator.To) {
	ctrl, ok := a.targets[vc.Target]
	if !ok {
		logAndPub(a.eventBus, nil, "No controller found", "controler", vc.Target)
		return
	}

	if a.oldChildController != nil {
		c, ok := a.oldChildController.(OnBack)
		if vc.Back && ok {
			c.OnBack()
		} else if c, ok := a.oldChildController.(OnExit); ok {
			c.OnExit()
		}
	}

	a.oldChildController = ctrl

	a.view.EnableTabs(vc.Target == SearchNavigation && a.canReenableTabs())

	ctrl.OnEnter()
}

func (a *App) OnExit() {
	if a.oldChildController == nil {
		return
	}

	if v, ok := a.oldChildController.(OnExit); ok {
		v.OnExit()
	}
}

func (a *App) ClearCache() {
	a.eventBus.Publish(app.ClearCache{})
}

func (a *App) ShowNotification(evt app.Notify) {
	a.view.ShowNotification(evt)
}

func (a *App) SetOpenSubtitles(username, password string) {
	err := a.secrets.SetOpenSubtitles(app.OpenSubtitlesSecret{
		Password: password,
	})
	if err != nil {
		logAndPub(a.eventBus, err, "Failed to set open subtitles password")
	}

	settings, err := a.repo.LoadSettings()
	if err != nil {
		logAndPub(a.eventBus, err, "Failed to load settings")
	}

	settings.OpenSubtitles.Username = username
	err = a.repo.SaveSettings(settings)
	if err != nil {
		logAndPub(a.eventBus, err, "Failed to save settings")
	}

	a.osEnabled = true
	a.reenableTabs()

	a.eventBus.Success("OpenSubtitles credentials saved")
}

func (a *App) TraktLogin(done func()) {
	deviceCodeResponse, err := a.libAuth.GetDeviceCode()
	if err != nil {
		logAndPub(a.eventBus, err, "Failed to get device code")
		return
	}

	url := fmt.Sprintf("%s/%s", deviceCodeResponse.VerificationURL, deviceCodeResponse.UserCode)
	err = browser.OpenURL(url)
	if err != nil {
		logAndPub(a.eventBus, err, "Failed to open browser")
		return
	}

	go func() {
		token, err := a.libAuth.PollForToken(deviceCodeResponse)
		if err != nil {
			logAndPub(a.eventBus, err, "Failed to get token")
			return
		}

		err = a.secrets.SetTrakt(app.TraktSecret{
			AccessToken:  token.AccessToken,
			RefreshToken: token.RefreshToken,
			ExpiresAt:    time.Unix(int64(token.CreatedAt), 0).Add(time.Second * time.Duration(token.ExpiresIn)),
		})
		if err != nil {
			logAndPub(a.eventBus, err, "Failed to save Trakt token")
			return
		}

		a.traktEnabled = true
		a.reenableTabs()

		a.eventBus.Success("Trakt credentials saved")
		done()
	}()
}
