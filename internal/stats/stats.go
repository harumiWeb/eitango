package stats

import (
	"fmt"
	"strings"

	"github.com/yourname/eitango/internal/i18n"
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
	title := i18n.T(i18n.StatsTitle)
	b.WriteString(title + "\n")
	b.WriteString(strings.Repeat("=", len([]rune(title))))
	b.WriteString("\n\n")
	b.WriteString(renderWindow(snapshot.Today))
	b.WriteString(renderWindow(snapshot.SevenDays))
	b.WriteString(renderWindow(snapshot.ThirtyDays))
	b.WriteString(renderWindow(snapshot.Total))
	_, _ = fmt.Fprintf(&b, "%-14s: %d\n", i18n.T(i18n.StatsDue), snapshot.DueCount)
	_, _ = fmt.Fprintf(&b, "%-14s: %d\n", i18n.T(i18n.StatsNew), snapshot.NewCount)
	_, _ = fmt.Fprintf(&b, "%-14s: %d\n", i18n.T(i18n.StatsStreak), snapshot.StreakDays)
	return b.String()
}

func renderWindow(window Window) string {
	return fmt.Sprintf("%-12s %s=%d %s=%d %s=%.1f%% %s=%.1fm\n",
		window.Label+":",
		i18n.T(i18n.StatsReviews), window.Reviews,
		i18n.T(i18n.StatsCorrect), window.Correct,
		i18n.T(i18n.StatsAccuracy), window.Accuracy(),
		i18n.T(i18n.StatsWait), window.WaitMinutes,
	)
}
