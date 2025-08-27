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
	"github.com/quintans/torflix/internal/viewmodel"
)

func Download(vm *viewmodel.Download) (fyne.CanvasObject, func(bool)) {
	stream := widget.NewLabel("")
	progress := widget.NewLabel("")
	downloadSpeed := widget.NewLabel("")
	uploadSpeed := widget.NewLabel("")
	seeders := widget.NewLabel("")

	back := widget.NewButton("BACK", func() {
		vm.Back()
	})

	play := widget.NewButton("PLAY", nil)
	play.Disable()
	play.OnTapped = func() {
		vm.Play()
	}
	play.Importance = widget.HighImportance
	vm.Playable.BindInMain(func(playable bool) {
		if playable {
			play.Enable()
		} else {
			play.Disable()
		}
	})

	widgets := []fyne.CanvasObject{}
	name := canvas.NewText("Name", color.White)
	name.Alignment = fyne.TextAlignTrailing
	widgets = append(widgets, name, widget.NewLabel(vm.TorrentFilename()))

	subFile := vm.TorrentSubFilename()
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

	vm.Status.BindInMain(func(stats app.Stats) {
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

	go func() {
		if vm.Serve() {
			vm.Play()
		}
	}()

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

	// the close func is nil, because it can only go back
	// going back the viewmodel will be GC
	return content, nil
}
