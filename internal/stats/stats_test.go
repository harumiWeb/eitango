package stats

import (
	"strings"
	"testing"

	"github.com/harumiWeb/eitango/internal/i18n"
)

func TestRenderTextIncludesWaitMinutes(t *testing.T) {
	t.Parallel()

	snapshot := Snapshot{
		Today:      Window{Label: "Today", Reviews: 4, Correct: 3, WaitMinutes: 5.5},
		SevenDays:  Window{Label: "7 days", Reviews: 10, Correct: 8, WaitMinutes: 12.0},
		ThirtyDays: Window{Label: "30 days", Reviews: 20, Correct: 15, WaitMinutes: 42.0},
		Total:      Window{Label: "Total", Reviews: 30, Correct: 23, WaitMinutes: 54.5},
		DueCount:   3,
		NewCount:   8,
		StreakDays: 2,
	}

	got := RenderText(snapshot)
	waitLabel := i18n.T(i18n.StatsWait)
	if !strings.Contains(got, waitLabel+"=5.5m") {
		t.Fatalf("RenderText() missing today wait minutes:\n%s", got)
	}
	if !strings.Contains(got, waitLabel+"=54.5m") {
		t.Fatalf("RenderText() missing total wait minutes:\n%s", got)
	}
}
