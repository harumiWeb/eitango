package tui

import (
	"testing"

	"github.com/mattn/go-runewidth"
)

func TestAlignLabel(t *testing.T) {
	tests := []struct {
		name  string
		label string
		width int
		wantW int
	}{
		{"ascii short", "Word", 14, 14},
		{"ascii exact", "Response time!", 14, 14},
		{"ascii overflow", "Very long label text!", 14, 21},
		{"cjk short", "単語", 14, 14},
		{"cjk exact", "今日の学習時間", 14, 14},
		{"cjk mixed paren", "例文（英語）", 14, 14},
		{"cjk short 2", "正答率", 14, 14},
		{"mixed ascii cjk", "30日間:", 12, 12},
		{"zero width", "", 14, 14},
		{"width zero", "abc", 0, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AlignLabel(tt.label, tt.width)
			gotW := runewidth.StringWidth(got)
			if gotW != tt.wantW {
				t.Errorf("AlignLabel(%q, %d) display width = %d, want %d (result=%q)",
					tt.label, tt.width, gotW, tt.wantW, got)
			}
		})
	}
}
