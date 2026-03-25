package main

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/yourname/eitango/internal/config"
	"github.com/yourname/eitango/internal/session"
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
