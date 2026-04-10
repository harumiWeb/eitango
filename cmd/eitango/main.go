package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/harumiWeb/eitango"
	"github.com/harumiWeb/eitango/internal/app"
	"github.com/harumiWeb/eitango/internal/config"
	"github.com/harumiWeb/eitango/internal/dict"
	"github.com/harumiWeb/eitango/internal/i18n"
	"github.com/harumiWeb/eitango/internal/session"
	"github.com/harumiWeb/eitango/internal/stats"
	"github.com/harumiWeb/eitango/internal/store"
	"github.com/harumiWeb/eitango/internal/updatecheck"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	// These are overwritten by GoReleaser ldflags for release builds.
	version = "dev"
	commit  = "unknown"
	date    = "unknown"

	readBuildInfo = debug.ReadBuildInfo

	newUpdateService = func(dataDir string) updatecheck.Service {
		return updatecheck.New(updatecheck.DefaultStatePath(dataDir))
	}
)

func main() {
	if err := newRootCommand().Execute(); err != nil {
		exitCode := 1
		var withExitCode interface{ ExitCode() int }
		if errors.As(err, &withExitCode) {
			exitCode = withExitCode.ExitCode()
		}
		if message := err.Error(); message != "" {
			fmt.Fprintln(os.Stderr, message)
		}
		os.Exit(exitCode)
	}
}

func newRootCommand() *cobra.Command {
	currentVersion := resolvedVersion()
	cmd := &cobra.Command{
		Use:           "eitango",
		Short:         i18n.T(i18n.CLIRootShort),
		SilenceErrors: true,
		SilenceUsage:  true,
		Version:       formatBuildVersion("eitango", currentVersion, commit, date),
		RunE:          runDashboard,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return maybePrintLicense(cmd)
		},
	}
	cmd.SetVersionTemplate("{{ .Version }}\n")
	cmd.PersistentFlags().Bool("license", false, "Print bundled license information and exit")
	cmd.AddCommand(newVersionCommand(), newPlayCommand(), newReviewCommand(), newStatsCommand(), newDoctorCommand(), newImportCommand(), newExportCommand(), newResetCommand(), newValidateCommand())
	return cmd
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show build and release version information",
		RunE:  runVersion,
	}
}

func newPlayCommand() *cobra.Command {
	cmd := newSessionCommand("play", []string{"learn"}, store.ModeLearn, "Start a learning session")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runSession(cmd, store.ModeLearn, store.AnswerModeChoice)
	}
	return cmd
}

func newReviewCommand() *cobra.Command {
	cmd := newSessionCommand("review", nil, store.ModeReview, "Start a review session")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return runSession(cmd, store.ModeReview, store.AnswerModeChoice)
	}
	return cmd
}

func newSessionCommand(name string, aliases []string, mode, short string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     name,
		Aliases: aliases,
		Short:   short,
		Args:    cobra.NoArgs,
	}
	addSessionFlags(cmd.PersistentFlags())
	if mode == store.ModeReview {
		cmd.PersistentFlags().Bool("restart", false, "Abandon the active session and start a fresh review session")
	}
	cmd.AddCommand(
		newAnswerModeCommand("choice", mode, "Start with 4-choice answers", store.AnswerModeChoice),
		newAnswerModeCommand("write", mode, "Start with typed answers", store.AnswerModeWrite),
	)
	return cmd
}

func newAnswerModeCommand(name, sessionMode, short, answerMode string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSession(cmd, sessionMode, answerMode)
		},
	}
}

func runDashboard(cmd *cobra.Command, args []string) error {
	ctx := commandContext(cmd)
	currentVersion := resolvedVersion()
	st, settings, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = st.Close()
	}()
	paths, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}

	program := tea.NewProgram(app.NewModel(st, app.Options{
		Plan:           mustSessionOptions(settings, nil, nil),
		Settings:       settings,
		ConfigPath:     paths.ConfigPath,
		CurrentVersion: currentVersion,
		UpdateService:  newUpdateService(paths.DataDir),
	}))
	_, err = program.Run()
	return err
}

func runSession(cmd *cobra.Command, sessionMode, answerMode string) error {
	ctx := commandContext(cmd)
	currentVersion := resolvedVersion()
	st, settings, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = st.Close()
	}()
	paths, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}

	focusMode, err := resolveFocusModeOverride(cmd)
	if err != nil {
		return err
	}
	questionCount, err := resolveQuestionCountOverride(cmd)
	if err != nil {
		return err
	}
	restart := false
	if sessionMode == store.ModeReview {
		restart, err = cmd.Flags().GetBool("restart")
		if err != nil {
			return fmt.Errorf("get restart flag: %w", err)
		}
	}
	options, err := sessionOptionsFromSettings(settings, questionCount, focusMode)
	if err != nil {
		return err
	}

	program := tea.NewProgram(app.NewModel(st, app.Options{
		Plan:           options,
		Settings:       settings,
		ConfigPath:     paths.ConfigPath,
		CurrentVersion: currentVersion,
		UpdateService:  newUpdateService(paths.DataDir),
		Startup: &app.StartupRequest{
			Mode:          sessionMode,
			AnswerMode:    answerMode,
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

func runVersion(cmd *cobra.Command, args []string) error {
	currentVersion := resolvedVersion()
	result := updatecheck.Result{}
	if paths, err := config.Resolve(); err == nil {
		if service := newUpdateService(paths.DataDir); service != nil {
			result, _ = service.CheckNow(commandContext(cmd), currentVersion)
		}
	}
	_, err := fmt.Fprint(cmd.OutOrStdout(), formatVersionReport(result))
	return err
}

func newDoctorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run read-only database diagnostics",
		RunE:  runDoctor,
	}
}

func newResetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reset",
		Short: "Reset progress and/or reload embedded core words",
		RunE:  runReset,
	}
	cmd.Flags().Bool("progress", false, "Reset learning history (sessions, reviews, progress)")
	cmd.Flags().Bool("reseed", false, "Reload embedded core words and clear learning history")
	return cmd
}

func runDoctor(cmd *cobra.Command, args []string) error {
	paths, err := config.Resolve()
	if err != nil {
		return fmt.Errorf("resolve data dir: %w", err)
	}

	st, err := store.OpenReadOnly(commandContext(cmd), paths.DBPath)
	if err != nil {
		return fmt.Errorf("open db read-only: %w", err)
	}
	defer func() {
		_ = st.Close()
	}()

	report := st.RunDiagnostics(commandContext(cmd))
	if _, err := fmt.Fprint(cmd.OutOrStdout(), formatDoctorReport(report)); err != nil {
		return err
	}
	if report.HasIssues() {
		return commandExitError{code: 1}
	}
	return nil
}

func runReset(cmd *cobra.Command, args []string) error {
	options, err := resetOptionsFromFlags(cmd)
	if err != nil {
		return err
	}
	if err := options.Validate(); err != nil {
		return err
	}

	ctx := commandContext(cmd)
	st, _, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer func() {
		_ = st.Close()
	}()

	var entries []dict.Entry
	if options.Reseed {
		entries, err = dict.LoadCoreWords()
		if err != nil {
			return fmt.Errorf("load core words: %w", err)
		}
	}

	result, err := st.Reset(ctx, options, entries, dict.CoreWordsVersion)
	if err != nil {
		return err
	}

	_, err = fmt.Fprint(cmd.OutOrStdout(), formatResetReport(result))
	return err
}

func addSessionFlags(flags *pflag.FlagSet) {
	flags.Bool("focus-mode", false, "Use a 5-question session")
	flags.Int("questions", 0, "Override the lesson size with a specific question count")
}

func resetOptionsFromFlags(cmd *cobra.Command) (store.ResetOptions, error) {
	progress, err := cmd.Flags().GetBool("progress")
	if err != nil {
		return store.ResetOptions{}, fmt.Errorf("get progress flag: %w", err)
	}
	reseed, err := cmd.Flags().GetBool("reseed")
	if err != nil {
		return store.ResetOptions{}, fmt.Errorf("get reseed flag: %w", err)
	}
	return store.ResetOptions{
		Progress: progress,
		Reseed:   reseed,
	}, nil
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

func resolveQuestionCountOverride(cmd *cobra.Command) (*int, error) {
	flag := cmd.Flags().Lookup("questions")
	if flag == nil || !flag.Changed {
		return nil, nil
	}
	questionCount, err := cmd.Flags().GetInt("questions")
	if err != nil {
		return nil, fmt.Errorf("get questions flag: %w", err)
	}
	if questionCount <= 0 {
		return nil, fmt.Errorf("questions must be greater than 0")
	}
	return &questionCount, nil
}

func sessionOptionsFromSettings(settings config.Settings, questionCountOverride *int, focusModeOverride *bool) (session.PlanOptions, error) {
	options := session.PlanOptions{
		QuestionCount: settings.SessionSize,
		ReviewRatio:   settings.ReviewRatio,
	}
	if questionCountOverride != nil {
		if focusModeOverride != nil && *focusModeOverride {
			return session.PlanOptions{}, fmt.Errorf("cannot use --questions with --focus-mode")
		}
		options.QuestionCount = *questionCountOverride
		return options.Normalize(), nil
	}
	focusMode := settings.FocusModeDefault
	if focusModeOverride != nil {
		focusMode = *focusModeOverride
	}
	if focusMode {
		options.QuestionCount = session.FocusQuestionCount
	}
	return options.Normalize(), nil
}

func mustSessionOptions(settings config.Settings, questionCountOverride *int, focusModeOverride *bool) session.PlanOptions {
	options, err := sessionOptionsFromSettings(settings, questionCountOverride, focusModeOverride)
	if err != nil {
		panic(err)
	}
	return options
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

	if err := i18n.Load(settings.Language); err != nil {
		return nil, config.Settings{}, fmt.Errorf("load locale: %w", err)
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

func maybePrintLicense(cmd *cobra.Command) error {
	showLicense, err := cmd.Flags().GetBool("license")
	if err != nil {
		return fmt.Errorf("get license flag: %w", err)
	}
	if !showLicense {
		return nil
	}

	text, err := eitango.LicenseText()
	if err != nil {
		return fmt.Errorf("load embedded licenses: %w", err)
	}
	if _, err := fmt.Fprintln(cmd.OutOrStdout(), text); err != nil {
		return err
	}
	cmd.Run = func(*cobra.Command, []string) {}
	cmd.RunE = func(*cobra.Command, []string) error { return nil }
	return nil
}

func buildVersionText() string {
	return formatBuildVersion("eitango", resolvedVersion(), commit, date)
}

func formatVersionReport(result updatecheck.Result) string {
	var b strings.Builder
	b.WriteString(buildVersionText())
	if result.Disabled {
		fmt.Fprintf(&b, "\nlatest: disabled (%s=1)", updatecheck.DisableEnv)
		return b.String()
	}
	if tag := strings.TrimSpace(result.Latest.TagName); tag != "" {
		fmt.Fprintf(&b, "\nlatest: %s", tag)
		if url := strings.TrimSpace(result.Latest.HTMLURL); url != "" {
			fmt.Fprintf(&b, "\nrelease: %s", url)
		}
		switch {
		case result.UpdateAvailable:
			fmt.Fprint(&b, "\nstatus: update available")
		case result.Compared:
			fmt.Fprint(&b, "\nstatus: up to date")
		}
	}
	return b.String()
}

func resolvedVersion() string {
	currentVersion := strings.TrimSpace(version)
	if currentVersion != "" && currentVersion != "dev" {
		return currentVersion
	}
	if buildVersion := moduleBuildVersion(); buildVersion != "" {
		return buildVersion
	}
	return defaultBuildValue(currentVersion, "dev")
}

func moduleBuildVersion() string {
	info, ok := readBuildInfo()
	if !ok || info == nil {
		return ""
	}

	buildVersion := strings.TrimSpace(info.Main.Version)
	switch buildVersion {
	case "", "(devel)":
		return ""
	default:
		return buildVersion
	}
}

func formatBuildVersion(name, version, commit, date string) string {
	return fmt.Sprintf("%s %s\ncommit: %s\ndate: %s",
		defaultBuildValue(name, "eitango"),
		defaultBuildValue(version, "dev"),
		defaultBuildValue(commit, "unknown"),
		defaultBuildValue(date, "unknown"),
	)
}

func defaultBuildValue(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

type commandExitError struct {
	code int
}

func (e commandExitError) Error() string {
	return ""
}

func (e commandExitError) ExitCode() int {
	return e.code
}

func formatDoctorReport(report store.DiagnosticReport) string {
	var b strings.Builder
	header := i18n.T(i18n.CLIDoctorHeader)
	fmt.Fprintln(&b, header)
	fmt.Fprintln(&b, strings.Repeat("=", len([]rune(header))))
	fmt.Fprintln(&b)

	for _, check := range report.Checks {
		fmt.Fprintf(&b, "[%s] %-20s %s\n", doctorStatusText(check.Status), check.Name, check.Summary)
		for _, detail := range check.Details {
			fmt.Fprintf(&b, "      - %s\n", detail)
		}
	}

	fmt.Fprintln(&b)
	switch warnings, errors := report.WarningCount(), report.ErrorCount(); {
	case warnings == 0 && errors == 0:
		fmt.Fprintln(&b, i18n.T(i18n.CLIDoctorOK))
	case warnings == 0:
		fmt.Fprintln(&b, i18n.Tf(i18n.CLIDoctorErrors, errors))
	case errors == 0:
		fmt.Fprintln(&b, i18n.Tf(i18n.CLIDoctorWarnings, warnings))
	default:
		fmt.Fprintln(&b, i18n.Tf(i18n.CLIDoctorBoth, warnings, errors))
	}

	return b.String()
}

func formatResetReport(result store.ResetResult) string {
	var b strings.Builder
	header := i18n.T(i18n.CLIResetHeader)
	fmt.Fprintln(&b, header)
	fmt.Fprintln(&b, strings.Repeat("=", len([]rune(header))))
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, i18n.Tf(i18n.CLIResetCleared,
		result.ClearedSessions,
		result.ClearedSessionItems,
		result.ClearedReviews,
		result.ClearedProgress,
	))
	if result.Options.Reseed {
		fmt.Fprintln(&b, i18n.Tf(i18n.CLIResetReseeded,
			result.ClearedWords,
			result.SeededWords,
			result.DictVersion,
		))
	}
	return b.String()
}

func doctorStatusText(status store.DiagnosticStatus) string {
	switch status {
	case store.DiagnosticStatusOK:
		return "OK"
	case store.DiagnosticStatusWarning:
		return "WARN"
	default:
		return "ERR"
	}
}
