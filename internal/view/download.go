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
	"github.com/quintans/torflix/internal/lib/navigation"
	"github.com/quintans/torflix/internal/viewmodel"
)

func Download(vm *viewmodel.ViewModel, navigator *navigation.Navigator[*viewmodel.ViewModel]) (fyne.CanvasObject, func(bool)) {
	stream := widget.NewLabel("")
	progress := widget.NewLabel("")
	downloadSpeed := widget.NewLabel("")
	uploadSpeed := widget.NewLabel("")
	seeders := widget.NewLabel("")

	play := widget.NewButton("Play", nil)
	play.OnTapped = func() {
		if vm.Download.Play() {
			play.Disable()
		}
	}
	play.Importance = widget.HighImportance
	play.Disable()

	back := widget.NewButton("Back", func() {
		navigator.Back(vm)
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

	progressTxt := canvas.NewText("Progress", color.White)
	progressTxt.Alignment = fyne.TextAlignTrailing
	widgets = append(widgets, progressTxt, progress)

	down := canvas.NewText("Download speed", color.White)
	down.Alignment = fyne.TextAlignTrailing
	widgets = append(widgets, down, downloadSpeed)

	up := canvas.NewText("Upload speed", color.White)
	up.Alignment = fyne.TextAlignTrailing
	widgets = append(widgets, up, uploadSpeed)

	seedersTxt := canvas.NewText("Seeders", color.White)
	seedersTxt.Alignment = fyne.TextAlignTrailing
	widgets = append(widgets, seedersTxt, seeders)

	streamTxt := canvas.NewText("Stream", color.White)
	streamTxt.Alignment = fyne.TextAlignTrailing
	widgets = append(widgets, streamTxt, stream)

	tracker := components.NewPieceTracker(nil)

	unbindStats := vm.Download.Status.Bind(func(stats app.Stats) {
		if stats.Pieces == nil {
			return
		}

		stream.SetText(stats.Stream)

		if stats.Size > 0 {
			percentage := float64(stats.Complete) / float64(stats.Size) * 100
			complete := humanize.Bytes(uint64(stats.Complete))
			size := humanize.Bytes(uint64(stats.Size))
			progress.SetText(fmt.Sprintf("%s / %s  %.2f%%", complete, size, percentage))
		}

		if stats.Done {
			downloadSpeed.SetText("Download complete")
		} else {
			downloadSpeed.SetText(humanize.Bytes(uint64(stats.DownloadSpeed)) + "/s")
		}
		uploadSpeed.SetText(humanize.Bytes(uint64(stats.UploadSpeed)) + "/s")
		seeders.SetText(fmt.Sprintf("%d", stats.Seeders))

		tracker.SetPieces(stats.Pieces)
	})

	vm.Download.Play()

	content := container.NewVBox(
		container.New(layout.NewFormLayout(), widgets...),
		tracker,
		container.NewHBox(
			layout.NewSpacer(),
			play,
			layout.NewSpacer(),
			back,
			layout.NewSpacer(),
		),
	)
	return content, func(bool) {
		unbindStats()

		vm.Download.Back()
	}
}
