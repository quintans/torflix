package components

import (
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// CustomLabel is a widget that displays a text label with custom color and word wrapping.
type CustomLabel struct {
	widget.BaseWidget

	Text       string
	Color      color.Color
	TextStyle  fyne.TextStyle
	Alignment  fyne.TextAlign
	TextSize   float32
	FontSource fyne.Resource
	WrapWidth  float32

	box *fyne.Container
}

// NewCustomLabel creates a new CustomLabel instance.
func NewCustomLabel(text string, textColor color.Color) *CustomLabel {
	label := &CustomLabel{Text: text, Color: textColor, WrapWidth: 200}
	label.ExtendBaseWidget(label)
	label.box = container.New(layout.NewCustomPaddedVBoxLayout(float32(0))) // Create a VBox to hold text lines
	return label
}

// CreateRenderer implements the fyne.WidgetRenderer interface.
func (c *CustomLabel) CreateRenderer() fyne.WidgetRenderer {
	return &customLabelRenderer{label: c}
}

// customLabelRenderer is the renderer for CustomLabel.
type customLabelRenderer struct {
	label *CustomLabel
}

// Layout arranges the VBox container within the given size.
func (r *customLabelRenderer) Layout(size fyne.Size) {
	r.updateTexts(size.Width)
	r.label.box.Resize(size)
}

// MinSize calculates the minimum size required by the widget.
func (r *customLabelRenderer) MinSize() fyne.Size {
	r.updateTexts(r.label.WrapWidth) // Default width for MinSize calculation
	return r.label.box.MinSize()
}

// Refresh updates the text and redraws the widget.
func (r *customLabelRenderer) Refresh() {
	canvas.Refresh(r.label.box)
}

// Destroy cleans up resources (no-op here).
func (r *customLabelRenderer) Destroy() {}

// Objects returns the canvas objects used for rendering.
func (r *customLabelRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.label.box}
}

// updateTexts splits the text into lines and populates the VBox.
func (r *customLabelRenderer) updateTexts(width float32) {
	r.label.box.Objects = nil // Clear the existing objects
	words := strings.Fields(r.label.Text)
	line := ""

	for _, word := range words {
		testLine := line
		if line != "" {
			testLine += " "
		}
		testLine += word

		// Test line width
		text := r.newText(testLine)
		if text.MinSize().Width > width && line != "" {
			r.label.box.Add(r.newText(line)) // Add completed line
			line = word
		} else {
			line = testLine
		}
	}
	if line != "" {
		r.label.box.Add(r.newText(line)) // Add remaining line
	}
}

func max(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

func (r *customLabelRenderer) newText(txt string) *canvas.Text {
	text := canvas.NewText(txt, r.label.Color)
	if r.label.TextSize > 0 {
		text.TextSize = r.label.TextSize
	}

	if r.label.TextStyle != (fyne.TextStyle{}) {
		text.TextStyle = r.label.TextStyle
	}

	text.Alignment = r.label.Alignment

	if r.label.FontSource != nil {
		text.FontSource = r.label.FontSource
	}

	return text
}
