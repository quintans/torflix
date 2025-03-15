package view

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/dustin/go-humanize"
	"github.com/quintans/torflix/internal/app"
	"github.com/quintans/torflix/internal/components"
)

type DownloadController interface {
	Back()
	OnEnter()
	Play()
}

type Download struct {
	showViewer ShowViewer
	controller DownloadController

	stream        *widget.Label
	progress      *widget.Label
	downloadSpeed *widget.Label
	uploadSpeed   *widget.Label
	seeders       *widget.Label
	tracker       *components.PieceTracker

	play *widget.Button
}

func NewDownload(showViewer ShowViewer) *Download {
	return &Download{
		showViewer: showViewer,
	}
}

func (d *Download) SetController(controller DownloadController) {
	d.controller = controller
}

func (v *Download) Show(torName string, subFile string) {
	v.stream = widget.NewLabel("")
	v.progress = widget.NewLabel("")
	v.downloadSpeed = widget.NewLabel("")
	v.uploadSpeed = widget.NewLabel("")
	v.seeders = widget.NewLabel("")

	v.play = widget.NewButton("Play", func() {
		v.controller.Play()
	})
	v.play.Importance = widget.HighImportance
	v.play.Disable()

	back := widget.NewButton("Back", func() {
		v.controller.Back()
	})

	widgets := []fyne.CanvasObject{}
	name := canvas.NewText("Name", color.White)
	name.Alignment = fyne.TextAlignTrailing
	widgets = append(widgets, name, widget.NewLabel(torName))

	if subFile != "" {
		sf := canvas.NewText("Sub File", color.White)
		sf.Alignment = fyne.TextAlignTrailing
		widgets = append(widgets, sf, widget.NewLabel(subFile))
	}

	progress := canvas.NewText("Progress", color.White)
	progress.Alignment = fyne.TextAlignTrailing
	widgets = append(widgets, progress, v.progress)

	down := canvas.NewText("Download speed", color.White)
	down.Alignment = fyne.TextAlignTrailing
	widgets = append(widgets, down, v.downloadSpeed)

	up := canvas.NewText("Upload speed", color.White)
	up.Alignment = fyne.TextAlignTrailing
	widgets = append(widgets, up, v.uploadSpeed)

	seeders := canvas.NewText("Seeders", color.White)
	seeders.Alignment = fyne.TextAlignTrailing
	widgets = append(widgets, seeders, v.seeders)

	stream := canvas.NewText("Stream", color.White)
	stream.Alignment = fyne.TextAlignTrailing
	widgets = append(widgets, stream, v.stream)

	v.tracker = components.NewPieceTracker(1)

	content := container.NewVBox(
		container.New(layout.NewFormLayout(), widgets...),
		v.tracker,
		container.NewHBox(
			layout.NewSpacer(),
			v.play,
			layout.NewSpacer(),
			back,
			layout.NewSpacer(),
		),
	)
	v.showViewer.ShowView(content)
}

func (v *Download) OnEnter() {
	v.controller.OnEnter()
}

func (v *Download) OnExit() {
	v.stream = nil
	v.progress = nil
	v.downloadSpeed = nil
	v.uploadSpeed = nil
	v.seeders = nil
	v.tracker = nil

	v.play = nil
}

func (v *Download) SetStats(stats app.Stats) {
	if stats.Pieces == nil {
		return
	}

	v.stream.SetText(stats.Stream)

	if stats.Size > 0 {
		percentage := float64(stats.Complete) / float64(stats.Size) * 100
		complete := humanize.Bytes(uint64(stats.Complete))
		size := humanize.Bytes(uint64(stats.Size))
		v.progress.SetText(fmt.Sprintf("%s / %s  %.2f%%", complete, size, percentage))
	}

	if stats.Done {
		v.downloadSpeed.SetText("Download complete")
	} else {
		v.downloadSpeed.SetText(humanize.Bytes(uint64(stats.DownloadSpeed)) + "/s")
	}
	v.uploadSpeed.SetText(humanize.Bytes(uint64(stats.UploadSpeed)) + "/s")
	v.seeders.SetText(fmt.Sprintf("%d", stats.Seeders))

	v.tracker.SetPieces(stats.Pieces)
}

func (v *Download) EnablePlay() {
	v.play.Enable()
}

func (v *Download) DisablePlay() {
	v.play.Disable()
}
