package controller

import (
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
	osEnabled          bool
}

func NewApp(
	view app.AppView,
	navigations map[string]app.Controller,
	eventBus app.EventBus,
	repo Repository,
	secrets app.Secrets,
	cacheDir string,
) *App {
	return &App{
		view:     view,
		targets:  navigations,
		eventBus: eventBus,
		repo:     repo,
		secrets:  secrets,
		cacheDir: cacheDir,
	}
}

func (a *App) OnEnter() {
	osSecret, err := a.secrets.GetOpenSubtitles()
	if err != nil {
		logAndPub(a.eventBus, err, "Failed to retrieve open subtitles password")
	}

	settings, err := a.repo.LoadSettings()
	if err != nil {
		logAndPub(a.eventBus, err, "Failed to load settings")
	}

	a.view.Show(app.AppData{
		CacheDir: a.cacheDir,
		OpenSubtitles: app.OpenSubtitles{
			Username: settings.OpenSubtitles.Username,
			Password: osSecret.Password,
		},
	})

	a.osEnabled = settings.OpenSubtitles.Username != "" && osSecret.Password != ""

	if !a.osEnabled {
		a.view.DisableAllTabsButSettings()
	}
}

func (a *App) reenableTabs() {
	if a.canReenableTabs() {
		a.view.EnableTabs(true)
	}
}

func (a *App) canReenableTabs() bool {
	return a.osEnabled
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
