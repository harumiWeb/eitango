package stats

import (
	"fmt"
	"strings"

	"github.com/harumiWeb/eitango/internal/i18n"
	"github.com/harumiWeb/eitango/internal/tui"
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
	_, _ = fmt.Fprintf(&b, "%s: %d\n", tui.AlignLabel(i18n.T(i18n.StatsDue), 14), snapshot.DueCount)
	_, _ = fmt.Fprintf(&b, "%s: %d\n", tui.AlignLabel(i18n.T(i18n.StatsNew), 14), snapshot.NewCount)
	_, _ = fmt.Fprintf(&b, "%s: %d\n", tui.AlignLabel(i18n.T(i18n.StatsStreak), 14), snapshot.StreakDays)
	return b.String()
}

func renderWindow(window Window) string {
	return fmt.Sprintf("%s %s=%d %s=%d %s=%.1f%% %s=%.1fm\n",
		tui.AlignLabel(window.Label+":", 12),
		i18n.T(i18n.StatsReviews), window.Reviews,
		i18n.T(i18n.StatsCorrect), window.Correct,
		i18n.T(i18n.StatsAccuracy), window.Accuracy(),
		i18n.T(i18n.StatsWait), window.WaitMinutes,
	)
}
