package components

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type PieceTracker struct {
	widget.BaseWidget
	pieces []bool
}

func NewPieceTracker(pieces []bool) *PieceTracker {
	w := &PieceTracker{
		pieces: pieces,
	}
	w.ExtendBaseWidget(w)
	return w
}

func (w *PieceTracker) CreateRenderer() fyne.WidgetRenderer {
	background := canvas.NewRectangle(color.NRGBA{0, 0, 255, 255}) // Blue background
	objects := []fyne.CanvasObject{background}

	return &pieceWidgetRenderer{
		widget:     w,
		background: background,
		objects:    objects,
	}
}

func (w *PieceTracker) SetPieces(pieces []bool) {
	w.pieces = pieces
	w.Refresh()
}

type pieceWidgetRenderer struct {
	widget     *PieceTracker
	background *canvas.Rectangle
	objects    []fyne.CanvasObject
}

func (r *pieceWidgetRenderer) Layout(size fyne.Size) {
	r.background.Resize(size)

	totalPieces := len(r.widget.pieces)
	var barWidth float32
	if totalPieces > 0 {
		barWidth = size.Width / float32(totalPieces)
	} else {
		barWidth = size.Width
	}

	// Remove old bars before creating new ones
	if len(r.objects) > 1 {
		r.objects = r.objects[:1]
	}

	start := -1
	for i := range totalPieces {
		if r.widget.pieces[i] {
			if start == -1 {
				start = i
			}
		} else {
			if start != -1 {
				bar := canvas.NewRectangle(color.NRGBA{0, 255, 0, 255}) // Green bar
				bar.Resize(fyne.NewSize(barWidth*float32(i-start), size.Height))
				bar.Move(fyne.NewPos(barWidth*float32(start), 0))
				r.objects = append(r.objects, bar)
				start = -1
			}
		}
	}
	if start != -1 {
		bar := canvas.NewRectangle(color.NRGBA{0, 255, 0, 255}) // Green bar
		bar.Resize(fyne.NewSize(barWidth*float32(totalPieces-start), size.Height))
		bar.Move(fyne.NewPos(barWidth*float32(start), 0))
		r.objects = append(r.objects, bar)
	}

	// Layout objects:
	for _, obj := range r.objects {
		obj.Refresh()
	}
}

func (r *pieceWidgetRenderer) MinSize() fyne.Size {
	return fyne.NewSize(100, 20)
}

func (r *pieceWidgetRenderer) Refresh() {
	r.Layout(r.widget.Size())
	canvas.Refresh(r.widget)
}

func (r *pieceWidgetRenderer) BackgroundColor() color.Color {
	return theme.BackgroundColor()
}

func (r *pieceWidgetRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *pieceWidgetRenderer) Destroy() {}
