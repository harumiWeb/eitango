package tui

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// AlignLabel pads label with trailing spaces so its display width equals width.
// CJK characters that occupy 2 terminal columns are handled correctly.
func AlignLabel(label string, width int) string {
	w := runewidth.StringWidth(label)
	if w >= width {
		return label
	}
	return label + strings.Repeat(" ", width-w)
}
