package srs

import "time"

type Rating string

const (
	Again Rating = "again"
	Hard  Rating = "hard"
	Good  Rating = "good"
	Easy  Rating = "easy"
)

type Progress struct {
	State         string
	DueAt         *time.Time
	IntervalDays  float64
	EaseFactor    float64
	LastSeenAt    *time.Time
	StreakCorrect int
	TotalCorrect  int
	TotalWrong    int
	Lapses        int
}

func DefaultProgress() Progress {
	return Progress{
		State:        "new",
		EaseFactor:   2.5,
		IntervalDays: 0,
	}
}

func Update(progress Progress, rating Rating, now time.Time) Progress {
	now = now.UTC()
	progress = normalize(progress)
	progress.LastSeenAt = ptr(now)

	switch rating {
	case Again:
		progress.State = "learning"
		progress.IntervalDays = 0
		progress.DueAt = ptr(now.Add(10 * time.Minute))
		progress.EaseFactor = maxFloat(1.3, progress.EaseFactor-0.2)
		progress.Lapses++
		progress.StreakCorrect = 0
		progress.TotalWrong++
	case Hard:
		progress.State = "review"
		if progress.IntervalDays < 1 {
			progress.IntervalDays = 1
		} else {
			progress.IntervalDays = maxFloat(1, progress.IntervalDays*1.2)
		}
		progress.DueAt = ptr(now.Add(durationDays(progress.IntervalDays)))
		progress.EaseFactor = maxFloat(1.3, progress.EaseFactor-0.15)
		progress.TotalCorrect++
	case Good:
		progress.State = "review"
		if progress.IntervalDays < 1 {
			progress.IntervalDays = 3
		} else {
			progress.IntervalDays = progress.IntervalDays * progress.EaseFactor
		}
		progress.DueAt = ptr(now.Add(durationDays(progress.IntervalDays)))
		progress.StreakCorrect++
		progress.TotalCorrect++
	case Easy:
		progress.State = "review"
		if progress.IntervalDays < 1 {
			progress.IntervalDays = 7
		} else {
			progress.IntervalDays = progress.IntervalDays * (progress.EaseFactor + 0.3)
		}
		progress.DueAt = ptr(now.Add(durationDays(progress.IntervalDays)))
		progress.EaseFactor += 0.05
		progress.StreakCorrect++
		progress.TotalCorrect++
	default:
		return progress
	}

	return progress
}

func normalize(progress Progress) Progress {
	if progress.State == "" {
		progress.State = "new"
	}
	if progress.EaseFactor == 0 {
		progress.EaseFactor = 2.5
	}
	if progress.EaseFactor < 1.3 {
		progress.EaseFactor = 1.3
	}
	return progress
}

func durationDays(days float64) time.Duration {
	return time.Duration(days * 24 * float64(time.Hour))
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func ptr(t time.Time) *time.Time {
	return &t
}
