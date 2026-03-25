package main

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/yourname/eitango/internal/config"
	"github.com/yourname/eitango/internal/store"
)

func newExportCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export learning data",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	cmd.AddCommand(newExportWrongWordsCommand(), newExportProgressCommand())
	return cmd
}

func newExportWrongWordsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wrong-words",
		Short: "Export wrong-answer words as CSV",
		Args:  cobra.NoArgs,
		RunE:  runExportWrongWords,
	}
	addExportOutputFlag(cmd)
	cmd.Flags().String("format", "csv", "Output format (csv)")
	return cmd
}

func newExportProgressCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "progress",
		Short: "Export word progress as JSON",
		Args:  cobra.NoArgs,
		RunE:  runExportProgress,
	}
	addExportOutputFlag(cmd)
	cmd.Flags().String("format", "json", "Output format (json)")
	return cmd
}

func runExportWrongWords(cmd *cobra.Command, args []string) error {
	if err := validateExportFormat(cmd, "csv"); err != nil {
		return err
	}

	ctx := commandContext(cmd)
	st, err := openExportStore(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = st.Close()
	}()

	snapshots, err := st.ListWrongWordSnapshots(ctx)
	if err != nil {
		return err
	}

	data, err := formatWrongWordsCSV(snapshots)
	if err != nil {
		return err
	}
	return writeExportOutput(cmd, data)
}

func runExportProgress(cmd *cobra.Command, args []string) error {
	if err := validateExportFormat(cmd, "json"); err != nil {
		return err
	}

	ctx := commandContext(cmd)
	st, err := openExportStore(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = st.Close()
	}()

	dictVersion, err := st.DictionaryVersion(ctx)
	if err != nil {
		return err
	}
	snapshots, err := st.ListExportWordSnapshots(ctx)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	data, err := formatProgressJSON(dictVersion, snapshots, now)
	if err != nil {
		return err
	}
	return writeExportOutput(cmd, data)
}

func addExportOutputFlag(cmd *cobra.Command) {
	cmd.Flags().String("output", "", "Write output to a file instead of stdout")
}

func validateExportFormat(cmd *cobra.Command, expected string) error {
	format, err := cmd.Flags().GetString("format")
	if err != nil {
		return fmt.Errorf("get format flag: %w", err)
	}
	if format != expected {
		return fmt.Errorf("%s only supports --format %s", cmd.CommandPath(), expected)
	}
	return nil
}

func openExportStore(ctx context.Context) (*store.Store, error) {
	paths, err := config.Resolve()
	if err != nil {
		return nil, fmt.Errorf("resolve data dir: %w", err)
	}

	st, err := store.OpenReadOnly(ctx, paths.DBPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("open db read-only: no database found at %s", paths.DBPath)
		}
		return nil, fmt.Errorf("open db read-only: %w", err)
	}
	return st, nil
}

func writeExportOutput(cmd *cobra.Command, data []byte) error {
	outputPath, err := cmd.Flags().GetString("output")
	if err != nil {
		return fmt.Errorf("get output flag: %w", err)
	}
	if outputPath == "" || outputPath == "-" {
		_, err := cmd.OutOrStdout().Write(data)
		return err
	}
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("write export %s: %w", outputPath, err)
	}
	return nil
}

func formatWrongWordsCSV(snapshots []store.ExportWordSnapshot) ([]byte, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	if err := writer.Write([]string{
		"lemma",
		"pos",
		"meaning_ja",
		"level",
		"frequency_rank",
		"distractor_group",
		"source",
		"state",
		"wrong_reviews",
		"correct_reviews",
		"total_reviews",
		"last_wrong_at",
		"last_correct_at",
		"due_at",
		"example_en",
		"example_ja",
	}); err != nil {
		return nil, fmt.Errorf("write wrong-words csv header: %w", err)
	}

	for _, snapshot := range snapshots {
		if err := writer.Write([]string{
			snapshot.Word.Lemma,
			snapshot.Word.Pos,
			snapshot.Word.MeaningJA,
			snapshot.Word.Level,
			strconv.Itoa(snapshot.Word.FrequencyRank),
			snapshot.Word.DistractorGroup,
			snapshot.Word.Source,
			snapshot.Progress.State,
			strconv.Itoa(snapshot.ReviewStats.WrongReviews),
			strconv.Itoa(snapshot.ReviewStats.CorrectReviews),
			strconv.Itoa(snapshot.ReviewStats.TotalReviews),
			formatNullableExportTimestamp(snapshot.ReviewStats.LastWrongAt),
			formatNullableExportTimestamp(snapshot.ReviewStats.LastCorrectAt),
			formatNullableExportTimestamp(snapshot.Progress.DueAt),
			snapshot.Word.ExampleEN,
			snapshot.Word.ExampleJA,
		}); err != nil {
			return nil, fmt.Errorf("write wrong-words csv row for %s: %w", snapshot.Word.Lemma, err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("flush wrong-words csv: %w", err)
	}
	return buffer.Bytes(), nil
}

func formatProgressJSON(dictVersion string, snapshots []store.ExportWordSnapshot, now time.Time) ([]byte, error) {
	document := buildProgressExportDocument(dictVersion, snapshots, now)
	data, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal progress json: %w", err)
	}
	return append(data, '\n'), nil
}

func buildProgressExportDocument(dictVersion string, snapshots []store.ExportWordSnapshot, now time.Time) progressExportDocument {
	document := progressExportDocument{
		ExportedAt:  formatRequiredExportTimestamp(now),
		DictVersion: dictVersion,
		Summary: progressExportSummary{
			TotalWords: len(snapshots),
		},
		Words: make([]progressExportWord, 0, len(snapshots)),
	}

	for _, snapshot := range snapshots {
		switch snapshot.Progress.State {
		case "learning":
			document.Summary.LearningWords++
		case "review":
			document.Summary.ReviewWords++
		default:
			document.Summary.NewWords++
		}
		if snapshot.Progress.DueAt != nil && !snapshot.Progress.DueAt.After(now) {
			document.Summary.DueWords++
		}
		if snapshot.ReviewStats.TotalReviews > 0 {
			document.Summary.ReviewedWords++
		}
		if snapshot.ReviewStats.WrongReviews > 0 {
			document.Summary.WrongWords++
		}

		document.Words = append(document.Words, progressExportWord{
			Word: progressExportWordInfo{
				ID:              snapshot.Word.ID,
				Lemma:           snapshot.Word.Lemma,
				Pos:             snapshot.Word.Pos,
				MeaningJA:       snapshot.Word.MeaningJA,
				Level:           snapshot.Word.Level,
				FrequencyRank:   snapshot.Word.FrequencyRank,
				DistractorGroup: snapshot.Word.DistractorGroup,
				ExampleEN:       snapshot.Word.ExampleEN,
				ExampleJA:       snapshot.Word.ExampleJA,
				Source:          snapshot.Word.Source,
			},
			Progress: progressExportProgress{
				State:         snapshot.Progress.State,
				DueAt:         nullableExportTimestamp(snapshot.Progress.DueAt),
				IntervalDays:  snapshot.Progress.IntervalDays,
				EaseFactor:    snapshot.Progress.EaseFactor,
				LastSeenAt:    nullableExportTimestamp(snapshot.Progress.LastSeenAt),
				StreakCorrect: snapshot.Progress.StreakCorrect,
				TotalCorrect:  snapshot.Progress.TotalCorrect,
				TotalWrong:    snapshot.Progress.TotalWrong,
				Lapses:        snapshot.Progress.Lapses,
			},
			ReviewStats: progressExportReviewStats{
				TotalReviews:   snapshot.ReviewStats.TotalReviews,
				CorrectReviews: snapshot.ReviewStats.CorrectReviews,
				WrongReviews:   snapshot.ReviewStats.WrongReviews,
				LastAnsweredAt: nullableExportTimestamp(snapshot.ReviewStats.LastAnsweredAt),
				LastWrongAt:    nullableExportTimestamp(snapshot.ReviewStats.LastWrongAt),
				LastCorrectAt:  nullableExportTimestamp(snapshot.ReviewStats.LastCorrectAt),
			},
		})
	}

	return document
}

func nullableExportTimestamp(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := formatRequiredExportTimestamp(*value)
	return &formatted
}

func formatNullableExportTimestamp(value *time.Time) string {
	if value == nil {
		return ""
	}
	return formatRequiredExportTimestamp(*value)
}

func formatRequiredExportTimestamp(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

type progressExportDocument struct {
	ExportedAt  string                `json:"exported_at"`
	DictVersion string                `json:"dict_version"`
	Summary     progressExportSummary `json:"summary"`
	Words       []progressExportWord  `json:"words"`
}

type progressExportSummary struct {
	TotalWords    int `json:"total_words"`
	NewWords      int `json:"new_words"`
	LearningWords int `json:"learning_words"`
	ReviewWords   int `json:"review_words"`
	DueWords      int `json:"due_words"`
	ReviewedWords int `json:"reviewed_words"`
	WrongWords    int `json:"wrong_words"`
}

type progressExportWord struct {
	Word        progressExportWordInfo    `json:"word"`
	Progress    progressExportProgress    `json:"progress"`
	ReviewStats progressExportReviewStats `json:"review_stats"`
}

type progressExportWordInfo struct {
	ID              int64  `json:"id"`
	Lemma           string `json:"lemma"`
	Pos             string `json:"pos"`
	MeaningJA       string `json:"meaning_ja"`
	Level           string `json:"level"`
	FrequencyRank   int    `json:"frequency_rank"`
	DistractorGroup string `json:"distractor_group"`
	ExampleEN       string `json:"example_en"`
	ExampleJA       string `json:"example_ja"`
	Source          string `json:"source"`
}

type progressExportProgress struct {
	State         string  `json:"state"`
	DueAt         *string `json:"due_at"`
	IntervalDays  float64 `json:"interval_days"`
	EaseFactor    float64 `json:"ease_factor"`
	LastSeenAt    *string `json:"last_seen_at"`
	StreakCorrect int     `json:"streak_correct"`
	TotalCorrect  int     `json:"total_correct"`
	TotalWrong    int     `json:"total_wrong"`
	Lapses        int     `json:"lapses"`
}

type progressExportReviewStats struct {
	TotalReviews   int     `json:"total_reviews"`
	CorrectReviews int     `json:"correct_reviews"`
	WrongReviews   int     `json:"wrong_reviews"`
	LastAnsweredAt *string `json:"last_answered_at"`
	LastWrongAt    *string `json:"last_wrong_at"`
	LastCorrectAt  *string `json:"last_correct_at"`
}
