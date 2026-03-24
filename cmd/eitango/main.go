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
	cmd.AddCommand(newLearnCommand(), newStatsCommand())
	return cmd
}

func newLearnCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "learn",
		Short: "Start a learning session",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			store, err := openStore(ctx)
			if err != nil {
				return err
			}
			defer func() {
				_ = store.Close()
			}()

			program := tea.NewProgram(app.NewModel(store))
			_, err = program.Run()
			return err
		},
	}
}

func newStatsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show review statistics",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
			store, err := openStore(ctx)
			if err != nil {
				return err
			}
			defer func() {
				_ = store.Close()
			}()

			snapshot, err := store.LoadStatsSnapshot(ctx)
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), stats.RenderText(snapshot))
			return err
		},
	}
}

func openStore(ctx context.Context) (*store.Store, error) {
	paths, err := config.Ensure()
	if err != nil {
		return nil, fmt.Errorf("prepare data dir: %w", err)
	}

	st, err := store.Open(ctx, paths.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := st.Migrate(ctx); err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("migrate db: %w", err)
	}

	entries, err := dict.LoadCoreWords()
	if err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("load core words: %w", err)
	}

	if err := st.SeedWords(ctx, entries, dict.CoreWordsVersion); err != nil {
		_ = st.Close()
		return nil, fmt.Errorf("seed words: %w", err)
	}

	return st, nil
}
