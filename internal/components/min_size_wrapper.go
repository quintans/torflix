package components

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type MinSizeWrapper struct {
	widget.BaseWidget                   // Embeds Fyne's base widget functionality
	wrapped           fyne.CanvasObject // The widget to wrap
	minSize           fyne.Size         // The minimum size to enforce
}

func NewMinSizeWrapper(object fyne.CanvasObject, minSize fyne.Size) *MinSizeWrapper {
	wrapper := &MinSizeWrapper{
		wrapped: object,
		minSize: minSize,
	}
	wrapper.ExtendBaseWidget(wrapper) // Initialize as a widget
	return wrapper
}

// Implement the CanvasObject methods

// MinSize returns the enforced minimum size
func (m *MinSizeWrapper) MinSize() fyne.Size {
	return m.minSize
}

// Resize resizes the wrapped object
func (m *MinSizeWrapper) Resize(size fyne.Size) {
	m.wrapped.Resize(size)
	m.BaseWidget.Resize(size)
}

// Layout ensures the wrapped object fills the wrapper
func (m *MinSizeWrapper) Layout(size fyne.Size) {
	m.wrapped.Resize(size)
}

// Refresh redraws the wrapped object
func (m *MinSizeWrapper) Refresh() {
	m.wrapped.Refresh()
	m.BaseWidget.Refresh()
}

// CreateRenderer ensures the wrapped object is rendered
func (m *MinSizeWrapper) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(m.wrapped)
}
