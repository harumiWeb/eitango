package store

import (
	"time"

	"github.com/yourname/eitango/internal/srs"
)

const (
	ModeLearn  = "learn"
	ModeReview = "review"
)

const (
	SessionStatusActive    = "active"
	SessionStatusCompleted = "completed"
	SessionStatusAbandoned = "abandoned"
)

const (
	ItemStatusPending  = "pending"
	ItemStatusAnswered = "answered"
)

const (
	ItemKindReview = "review"
	ItemKindNew    = "new"
	ItemKindRetry  = "retry"
)

type Progress = srs.Progress

type Word struct {
	ID              int64
	Lemma           string
	Pos             string
	MeaningJA       string
	Level           string
	FrequencyRank   int
	DistractorGroup string
	ExampleEN       string
	ExampleJA       string
	CreatedAt       time.Time
}

type SessionRecord struct {
	ID                string
	StartedAt         time.Time
	FinishedAt        *time.Time
	Mode              string
	TotalQuestions    int
	AnsweredQuestions int
	Status            string
}

type SessionItem struct {
	SessionID        string
	Ordinal          int
	WordID           int64
	Kind             string
	Status           string
	SourceOrdinal    *int
	AnsweredReviewID *int64
	CreatedAt        time.Time
}

type SessionItemPlan struct {
	WordID        int64
	Kind          string
	SourceOrdinal int
}

type HomeSnapshot struct {
	DueCount      int
	NewCount      int
	StreakDays    int
	ActiveSession *SessionRecord
}

type ReviewEvent struct {
	SessionID      string
	ItemOrdinal    int
	WordID         int64
	Kind           string
	SelectedChoice int
	CorrectChoice  int
	IsCorrect      bool
	Rating         srs.Rating
	AnsweredAt     time.Time
	ResponseMS     int64
}

type SessionSummary struct {
	SessionID      string
	TotalQuestions int
	CorrectAnswers int
	Accuracy       float64
	NewCount       int
	ReviewCount    int
	RetryCount     int
	HardWords      []Word
}

type ExportReviewStats struct {
	TotalReviews   int
	CorrectReviews int
	WrongReviews   int
	LastAnsweredAt *time.Time
	LastWrongAt    *time.Time
	LastCorrectAt  *time.Time
}

type ExportWordSnapshot struct {
	Word        Word
	Progress    Progress
	ReviewStats ExportReviewStats
}
