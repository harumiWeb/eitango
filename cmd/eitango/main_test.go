package main

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/yourname/eitango/internal/config"
	"github.com/yourname/eitango/internal/dict"
	"github.com/yourname/eitango/internal/session"
	"github.com/yourname/eitango/internal/store"
)

func TestNewRootCommandIncludesReviewCommandAndFlags(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	learn := findSubcommand(cmd, "learn")
	if learn == nil {
		t.Fatal("learn command not found")
	}
	if learn.Flags().Lookup("focus-mode") == nil {
		t.Fatal("learn focus-mode flag not found")
	}

	review := findSubcommand(cmd, "review")
	if review == nil {
		t.Fatal("review command not found")
	}
	if review.Flags().Lookup("focus-mode") == nil {
		t.Fatal("review focus-mode flag not found")
	}
	if review.Flags().Lookup("restart") == nil {
		t.Fatal("review restart flag not found")
	}

	doctor := findSubcommand(cmd, "doctor")
	if doctor == nil {
		t.Fatal("doctor command not found")
	}
}

func TestSessionOptionsFromSettings(t *testing.T) {
	t.Parallel()

	settings := config.Settings{
		SessionSize:      12,
		ReviewRatio:      0.25,
		FocusModeDefault: true,
	}

	options := sessionOptionsFromSettings(settings, nil)
	if options.QuestionCount != session.FocusQuestionCount {
		t.Fatalf("QuestionCount with config focus mode = %d, want %d", options.QuestionCount, session.FocusQuestionCount)
	}
	if options.ReviewRatio != 0.25 {
		t.Fatalf("ReviewRatio = %v, want 0.25", options.ReviewRatio)
	}

	override := false
	options = sessionOptionsFromSettings(settings, &override)
	if options.QuestionCount != 12 {
		t.Fatalf("QuestionCount with explicit false = %d, want 12", options.QuestionCount)
	}

	settings.FocusModeDefault = false
	override = true
	options = sessionOptionsFromSettings(settings, &override)
	if options.QuestionCount != session.FocusQuestionCount {
		t.Fatalf("QuestionCount with explicit true = %d, want %d", options.QuestionCount, session.FocusQuestionCount)
	}
}

func findSubcommand(root *cobra.Command, name string) *cobra.Command {
	for _, command := range root.Commands() {
		if command.Name() == name {
			return command
		}
	}
	return nil
}

func TestDoctorCommandRunsDiagnostics(t *testing.T) {
	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, "user.db")

	ctx := context.Background()
	st, err := store.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if err := st.SeedWords(ctx, []dict.Entry{
		{Lemma: "adopt", Pos: "verb", MeaningJA: "採用する", Level: "toeic600", FrequencyRank: 100, DistractorGroup: "basic-verb-action"},
		{Lemma: "apply", Pos: "verb", MeaningJA: "応募する", Level: "toeic600", FrequencyRank: 120, DistractorGroup: "basic-verb-action"},
		{Lemma: "cancel", Pos: "verb", MeaningJA: "取り消す", Level: "toeic600", FrequencyRank: 140, DistractorGroup: "basic-verb-action"},
		{Lemma: "deliver", Pos: "verb", MeaningJA: "届ける", Level: "toeic600", FrequencyRank: 160, DistractorGroup: "basic-verb-action"},
	}, dict.CoreWordsVersion); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	t.Setenv("EITANGO_DATA_DIR", dataDir)

	var out bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"doctor"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "eitango doctor") {
		t.Fatalf("doctor output = %q, want header", output)
	}
	if !strings.Contains(output, "Summary: OK") {
		t.Fatalf("doctor output = %q, want OK summary", output)
	}
}

func TestDoctorCommandReturnsExitCodeForIssues(t *testing.T) {
	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, "user.db")

	ctx := context.Background()
	st, err := store.Open(ctx, dbPath)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if err := st.SeedWords(ctx, []dict.Entry{
		{Lemma: "abandon", Pos: "verb", MeaningJA: "捨てる", Level: "toeic600", FrequencyRank: 100, DistractorGroup: "basic-verb-action"},
		{Lemma: "apply", Pos: "verb", MeaningJA: "応募する", Level: "toeic600", FrequencyRank: 200, DistractorGroup: "basic-verb-action"},
		{Lemma: "benefit", Pos: "noun", MeaningJA: "利益", Level: "toeic600", FrequencyRank: 300, DistractorGroup: "basic-noun-business"},
	}, dict.CoreWordsVersion); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}
	if err := st.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	t.Setenv("EITANGO_DATA_DIR", dataDir)

	var out bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"doctor"})

	err = cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want exit code error")
	}

	var withExitCode interface{ ExitCode() int }
	if !errors.As(err, &withExitCode) {
		t.Fatalf("Execute() error = %T, want exit code error", err)
	}
	if withExitCode.ExitCode() != 1 {
		t.Fatalf("ExitCode() = %d, want 1", withExitCode.ExitCode())
	}
	if !strings.Contains(out.String(), "quizability") {
		t.Fatalf("doctor output = %q, want quizability failure", out.String())
	}
}
