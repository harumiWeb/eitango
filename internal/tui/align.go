package tui

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// AlignText pads text with trailing spaces so its display width equals width.
// CJK characters that occupy 2 terminal columns are handled correctly.
func AlignText(text string, width int) string {
	w := runewidth.StringWidth(text)
	if w >= width {
		return text
	}
	return text + strings.Repeat(" ", width-w)
}

// AlignLabel pads label with trailing spaces so its display width equals width.
// CJK characters that occupy 2 terminal columns are handled correctly.
func AlignLabel(label string, width int) string {
	return AlignText(label, width)
}
