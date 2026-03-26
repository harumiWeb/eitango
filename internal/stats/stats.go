package stats

import (
	"fmt"
	"strings"
)

type Window struct {
	Label       string
	Reviews     int
	Correct     int
	WaitMinutes float64
}

func (w Window) Accuracy() float64 {
	if w.Reviews == 0 {
		return 0
	}
	return float64(w.Correct) / float64(w.Reviews) * 100
}

type Snapshot struct {
	Today      Window
	SevenDays  Window
	ThirtyDays Window
	Total      Window
	DueCount   int
	NewCount   int
	StreakDays int
}

func RenderText(snapshot Snapshot) string {
	var b strings.Builder
	b.WriteString("Eitango stats\n")
	b.WriteString(strings.Repeat("=", 13))
	b.WriteString("\n\n")
	b.WriteString(renderWindow(snapshot.Today))
	b.WriteString(renderWindow(snapshot.SevenDays))
	b.WriteString(renderWindow(snapshot.ThirtyDays))
	b.WriteString(renderWindow(snapshot.Total))
	_, _ = fmt.Fprintf(&b, "Due now      : %d\n", snapshot.DueCount)
	_, _ = fmt.Fprintf(&b, "New available: %d\n", snapshot.NewCount)
	_, _ = fmt.Fprintf(&b, "Streak days  : %d\n", snapshot.StreakDays)
	return b.String()
}

func renderWindow(window Window) string {
	return fmt.Sprintf("%-12s reviews=%d correct=%d accuracy=%.1f%% wait=%.1fm\n", window.Label+":", window.Reviews, window.Correct, window.Accuracy(), window.WaitMinutes)
}
