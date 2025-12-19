package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Widget represents a UI component that can be rendered and updated
type Widget interface {
	Update(event UIEvent) bool // returns true if widget changed and needs re-render
	Render(width, height int) string
	Focused() bool
	SetFocus(focused bool)
}

// BaseWidget provides common widget functionality
type BaseWidget struct {
	focused bool
	style   lipgloss.Style
}

func NewBaseWidget() BaseWidget {
	return BaseWidget{
		focused: false,
		style:   lipgloss.NewStyle(),
	}
}

func (w *BaseWidget) Focused() bool {
	return w.focused
}

func (w *BaseWidget) SetFocus(focused bool) {
	w.focused = focused
}

// Layout manager for arranging widgets
type Layout struct {
	widgets    []Widget
	focusIndex int
}

func NewLayout() *Layout {
	return &Layout{
		widgets:    make([]Widget, 0),
		focusIndex: 0,
	}
}

func (l *Layout) AddWidget(w Widget) {
	l.widgets = append(l.widgets, w)
	// Focus first widget by default
	if len(l.widgets) == 1 {
		w.SetFocus(true)
	}
}

func (l *Layout) NextFocus() {
	if len(l.widgets) == 0 {
		return
	}
	l.widgets[l.focusIndex].SetFocus(false)
	l.focusIndex = (l.focusIndex + 1) % len(l.widgets)
	l.widgets[l.focusIndex].SetFocus(true)
}

func (l *Layout) PrevFocus() {
	if len(l.widgets) == 0 {
		return
	}
	l.widgets[l.focusIndex].SetFocus(false)
	l.focusIndex = (l.focusIndex - 1 + len(l.widgets)) % len(l.widgets)
	l.widgets[l.focusIndex].SetFocus(true)
}

func (l *Layout) UpdateAll(event UIEvent) bool {
	changed := false
	for _, widget := range l.widgets {
		if widget.Update(event) {
			changed = true
		}
	}
	return changed
}

func (l *Layout) GetWidgets() []Widget {
	return l.widgets
}
