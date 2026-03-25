package main

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"github.com/yourname/eitango/internal/app"
	"github.com/yourname/eitango/internal/config"
	"github.com/yourname/eitango/internal/dict"
	"github.com/yourname/eitango/internal/session"
	"github.com/yourname/eitango/internal/stats"
	"github.com/yourname/eitango/internal/store"
)

func main() {
	if err := newRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "eitango",
		Short:        "Offline TUI English vocabulary trainer",
		SilenceUsage: true,
	}
	cmd.AddCommand(newLearnCommand(), newReviewCommand(), newStatsCommand())
	return cmd
}

func newLearnCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "learn",
		Short: "Start a learning session",
		RunE:  runLearn,
	}
	addFocusModeFlag(cmd)
	return cmd
}

func newReviewCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "review",
		Short: "Start a due-only review session",
		RunE:  runReview,
	}
	addFocusModeFlag(cmd)
	cmd.Flags().Bool("restart", false, "Abandon the active session and start a fresh review session")
	return cmd
}

func runLearn(cmd *cobra.Command, args []string) error {
	ctx := commandContext(cmd)
	st, settings, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = st.Close()
	}()

	focusMode, err := resolveFocusModeOverride(cmd)
	if err != nil {
		return err
	}

	program := tea.NewProgram(app.NewModel(st, app.Options{
		Plan: sessionOptionsFromSettings(settings, focusMode),
	}))
	_, err = program.Run()
	return err
}

func runReview(cmd *cobra.Command, args []string) error {
	ctx := commandContext(cmd)
	st, settings, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = st.Close()
	}()

	focusMode, err := resolveFocusModeOverride(cmd)
	if err != nil {
		return err
	}
	restart, err := cmd.Flags().GetBool("restart")
	if err != nil {
		return fmt.Errorf("get restart flag: %w", err)
	}

	program := tea.NewProgram(app.NewModel(st, app.Options{
		Plan: sessionOptionsFromSettings(settings, focusMode),
		Startup: &app.StartupRequest{
			Mode:          store.ModeReview,
			ReplaceActive: restart,
		},
	}))
	_, err = program.Run()
	return err
}

func newStatsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show review statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, _, err := openStore(commandContext(cmd))
			if err != nil {
				return err
			}
			defer func() {
				_ = st.Close()
			}()

			snapshot, err := st.LoadStatsSnapshot(commandContext(cmd))
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), stats.RenderText(snapshot))
			return err
		},
	}
}

func addFocusModeFlag(cmd *cobra.Command) {
	cmd.Flags().Bool("focus-mode", false, "Use a 5-question session")
}

func resolveFocusModeOverride(cmd *cobra.Command) (*bool, error) {
	flag := cmd.Flags().Lookup("focus-mode")
	if flag == nil {
		return nil, nil
	}
	focusMode, err := cmd.Flags().GetBool("focus-mode")
	if err != nil {
		return nil, fmt.Errorf("get focus-mode flag: %w", err)
	}
	if !flag.Changed {
		return nil, nil
	}
	return &focusMode, nil
}

func sessionOptionsFromSettings(settings config.Settings, focusModeOverride *bool) session.PlanOptions {
	options := session.PlanOptions{
		QuestionCount: settings.SessionSize,
		ReviewRatio:   settings.ReviewRatio,
	}
	focusMode := settings.FocusModeDefault
	if focusModeOverride != nil {
		focusMode = *focusModeOverride
	}
	if focusMode {
		options.QuestionCount = session.FocusQuestionCount
	}
	return options.Normalize()
}

func openStore(ctx context.Context) (*store.Store, config.Settings, error) {
	paths, err := config.Ensure()
	if err != nil {
		return nil, config.Settings{}, fmt.Errorf("prepare data dir: %w", err)
	}

	settings, err := config.Load(paths.ConfigPath)
	if err != nil {
		return nil, config.Settings{}, fmt.Errorf("load config: %w", err)
	}

	st, err := store.Open(ctx, paths.DBPath)
	if err != nil {
		return nil, config.Settings{}, fmt.Errorf("open db: %w", err)
	}

	if err := st.Migrate(ctx); err != nil {
		_ = st.Close()
		return nil, config.Settings{}, fmt.Errorf("migrate db: %w", err)
	}

	entries, err := dict.LoadCoreWords()
	if err != nil {
		_ = st.Close()
		return nil, config.Settings{}, fmt.Errorf("load core words: %w", err)
	}

	if err := st.SeedWords(ctx, entries, dict.CoreWordsVersion); err != nil {
		_ = st.Close()
		return nil, config.Settings{}, fmt.Errorf("seed words: %w", err)
	}

	return st, settings, nil
}

func commandContext(cmd *cobra.Command) context.Context {
	ctx := cmd.Context()
	if ctx == nil {
		return context.Background()
	}
	return ctx
}
