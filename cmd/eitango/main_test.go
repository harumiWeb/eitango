package main

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
	"time"

	projectassets "github.com/harumiWeb/eitango/assets"
	"github.com/harumiWeb/eitango/internal/config"
	"github.com/harumiWeb/eitango/internal/dict"
	"github.com/harumiWeb/eitango/internal/i18n"
	"github.com/harumiWeb/eitango/internal/session"
	"github.com/harumiWeb/eitango/internal/srs"
	"github.com/harumiWeb/eitango/internal/store"
	"github.com/harumiWeb/eitango/internal/updatecheck"
	"github.com/spf13/cobra"
	_ "modernc.org/sqlite"
)

func TestNewRootCommandIncludesPlayAndReviewCommands(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	play := findSubcommand(cmd, "play")
	if play == nil {
		t.Fatal("play command not found")
	}
	if play.PersistentFlags().Lookup("focus-mode") == nil {
		t.Fatal("play focus-mode flag not found")
	}
	if play.PersistentFlags().Lookup("questions") == nil {
		t.Fatal("play questions flag not found")
	}
	if !hasAlias(play, "learn") {
		t.Fatal("play learn alias not found")
	}
	if findSubcommand(play, "choice") == nil {
		t.Fatal("play choice command not found")
	}
	if findSubcommand(play, "write") == nil {
		t.Fatal("play write command not found")
	}

	review := findSubcommand(cmd, "review")
	if review == nil {
		t.Fatal("review command not found")
	}
	if review.PersistentFlags().Lookup("focus-mode") == nil {
		t.Fatal("review focus-mode flag not found")
	}
	if review.PersistentFlags().Lookup("questions") == nil {
		t.Fatal("review questions flag not found")
	}
	if review.PersistentFlags().Lookup("restart") == nil {
		t.Fatal("review restart flag not found")
	}
	if findSubcommand(review, "choice") == nil {
		t.Fatal("review choice command not found")
	}
	if findSubcommand(review, "write") == nil {
		t.Fatal("review write command not found")
	}

	doctor := findSubcommand(cmd, "doctor")
	if doctor == nil {
		t.Fatal("doctor command not found")
	}

	importCommand := findSubcommand(cmd, "import")
	if importCommand == nil {
		t.Fatal("import command not found")
	}
	if importCommand.Flags().Lookup("file") == nil {
		t.Fatal("import file flag not found")
	}
	if importCommand.Flags().Lookup("format") == nil {
		t.Fatal("import format flag not found")
	}
	if importCommand.Flags().Lookup("source") == nil {
		t.Fatal("import source flag not found")
	}

	validate := findSubcommand(cmd, "validate")
	if validate == nil {
		t.Fatal("validate command not found")
	}
	if validate.Flags().Lookup("file") == nil {
		t.Fatal("validate file flag not found")
	}
	if validate.Flags().Lookup("format") == nil {
		t.Fatal("validate format flag not found")
	}
	if validate.Flags().Lookup("kind") == nil {
		t.Fatal("validate kind flag not found")
	}
	if validate.Flags().Lookup("embedded-core") == nil {
		t.Fatal("validate embedded-core flag not found")
	}

	export := findSubcommand(cmd, "export")
	if export == nil {
		t.Fatal("export command not found")
	}
	wrongWords := findSubcommand(export, "wrong-words")
	if wrongWords == nil {
		t.Fatal("export wrong-words command not found")
	}
	if wrongWords.Flags().Lookup("format") == nil {
		t.Fatal("export wrong-words format flag not found")
	}
	if wrongWords.Flags().Lookup("output") == nil {
		t.Fatal("export wrong-words output flag not found")
	}
	progress := findSubcommand(export, "progress")
	if progress == nil {
		t.Fatal("export progress command not found")
	}
	if progress.Flags().Lookup("format") == nil {
		t.Fatal("export progress format flag not found")
	}
	if progress.Flags().Lookup("output") == nil {
		t.Fatal("export progress output flag not found")
	}

	reset := findSubcommand(cmd, "reset")
	if reset == nil {
		t.Fatal("reset command not found")
	}
	if reset.Flags().Lookup("progress") == nil {
		t.Fatal("reset progress flag not found")
	}
	if reset.Flags().Lookup("reseed") == nil {
		t.Fatal("reset reseed flag not found")
	}
	if cmd.PersistentFlags().Lookup("license") == nil {
		t.Fatal("root license flag not found")
	}
	versionCmd := findSubcommand(cmd, "version")
	if versionCmd == nil {
		t.Fatal("version command not found")
	}
}

func TestPlayCommandFindResolvesLearnAliasAndWriteSubcommand(t *testing.T) {
	t.Parallel()

	cmd := newRootCommand()
	found, _, err := cmd.Find([]string{"learn", "write"})
	if err != nil {
		t.Fatalf("Find(learn write) error = %v", err)
	}
	if found == nil || found.Name() != "write" {
		t.Fatalf("Find(learn write) = %+v, want write subcommand", found)
	}
}

func TestNewRootCommandVersionFlag(t *testing.T) {
	originalVersion := version
	originalReadBuildInfo := readBuildInfo
	version = "dev"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{Version: "v1.2.3"},
		}, true
	}
	t.Cleanup(func() {
		version = originalVersion
		readBuildInfo = originalReadBuildInfo
	})

	var out bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "eitango v1.2.3") {
		t.Fatalf("version output = %q, want resolved module version", output)
	}
	if !strings.Contains(output, "commit: ") {
		t.Fatalf("version output = %q, want commit line", output)
	}
	if !strings.Contains(output, "date: ") {
		t.Fatalf("version output = %q, want date line", output)
	}
}

func TestNewRootCommandLicenseFlag(t *testing.T) {
	var out bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--license"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := out.String()
	for _, want := range []string{
		"===== LICENSE =====",
		"Apache License",
		"===== THIRD_PARTY_NOTICES.md =====",
		"Third-Party Notices",
		"Japanese Wordnet (v1.1)",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("license output = %q, want substring %q", output, want)
		}
	}
}

func TestNewRootCommandHelpIncludesLicenseFlag(t *testing.T) {
	var out bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--help"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "--license") {
		t.Fatalf("help output = %q, want --license", output)
	}
}

func TestFormatBuildVersion(t *testing.T) {
	t.Parallel()

	got := formatBuildVersion("eitango", "1.2.3", "abcdef0", "2026-03-26T11:30:00Z")

	for _, want := range []string{
		"eitango 1.2.3",
		"commit: abcdef0",
		"date: 2026-03-26T11:30:00Z",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatBuildVersion() = %q, want substring %q", got, want)
		}
	}
}

func TestResolvedVersionUsesBuildInfoWhenLdflagsUnset(t *testing.T) {
	originalVersion := version
	originalReadBuildInfo := readBuildInfo
	version = "dev"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{Version: "v1.2.3"},
		}, true
	}
	t.Cleanup(func() {
		version = originalVersion
		readBuildInfo = originalReadBuildInfo
	})

	if got := resolvedVersion(); got != "v1.2.3" {
		t.Fatalf("resolvedVersion() = %q, want v1.2.3", got)
	}
}

func TestResolvedVersionFallsBackToDevWithoutBuildInfo(t *testing.T) {
	originalVersion := version
	originalReadBuildInfo := readBuildInfo
	version = "dev"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return nil, false
	}
	t.Cleanup(func() {
		version = originalVersion
		readBuildInfo = originalReadBuildInfo
	})

	if got := resolvedVersion(); got != "dev" {
		t.Fatalf("resolvedVersion() = %q, want dev", got)
	}
}

func TestFormatVersionReportIncludesLatestRelease(t *testing.T) {
	t.Parallel()

	got := formatVersionReport(updatecheck.Result{
		Latest: updatecheck.ReleaseInfo{
			TagName: "v1.2.4",
			HTMLURL: "https://example.com/eitango/v1.2.4",
		},
		UpdateAvailable: true,
		Compared:        true,
	})

	for _, want := range []string{
		"latest: v1.2.4",
		"release: https://example.com/eitango/v1.2.4",
		"status: update available",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("formatVersionReport() = %q, want substring %q", got, want)
		}
	}
}

func TestVersionCommandShowsLatestRelease(t *testing.T) {
	originalVersion := version
	originalReadBuildInfo := readBuildInfo
	version = "dev"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main: debug.Module{Version: "v1.2.3"},
		}, true
	}
	t.Cleanup(func() {
		version = originalVersion
		readBuildInfo = originalReadBuildInfo
	})

	service := &mainStubUpdateService{
		checkNowResult: updatecheck.Result{
			Latest: updatecheck.ReleaseInfo{
				TagName: "v1.2.4",
				HTMLURL: "https://example.com/eitango/v1.2.4",
			},
			UpdateAvailable: true,
			Compared:        true,
		},
	}
	original := newUpdateService
	newUpdateService = func(string) updatecheck.Service {
		return service
	}
	defer func() {
		newUpdateService = original
	}()

	t.Setenv("EITANGO_DATA_DIR", t.TempDir())

	var out bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"version"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if service.checkNowCalls != 1 {
		t.Fatalf("checkNowCalls = %d, want 1", service.checkNowCalls)
	}
	if service.checkNowVersion != "v1.2.3" {
		t.Fatalf("checkNowVersion = %q, want v1.2.3", service.checkNowVersion)
	}
	output := out.String()
	for _, want := range []string{
		"eitango v1.2.3",
		"latest: v1.2.4",
		"release: https://example.com/eitango/v1.2.4",
		"status: update available",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("version output = %q, want substring %q", output, want)
		}
	}
}

func TestSessionOptionsFromSettings(t *testing.T) {
	t.Parallel()

	settings := config.Settings{
		SessionSize:      12,
		ReviewRatio:      0.25,
		FocusModeDefault: true,
	}

	options, err := sessionOptionsFromSettings(settings, nil, nil)
	if err != nil {
		t.Fatalf("sessionOptionsFromSettings() error = %v", err)
	}
	if options.QuestionCount != session.FocusQuestionCount {
		t.Fatalf("QuestionCount with config focus mode = %d, want %d", options.QuestionCount, session.FocusQuestionCount)
	}
	if options.ReviewRatio != 0.25 {
		t.Fatalf("ReviewRatio = %v, want 0.25", options.ReviewRatio)
	}

	questionOverride := 12
	options, err = sessionOptionsFromSettings(settings, &questionOverride, nil)
	if err != nil {
		t.Fatalf("sessionOptionsFromSettings(question override) error = %v", err)
	}
	if options.QuestionCount != 12 {
		t.Fatalf("QuestionCount with explicit question override = %d, want 12", options.QuestionCount)
	}

	override := false
	options, err = sessionOptionsFromSettings(settings, nil, &override)
	if err != nil {
		t.Fatalf("sessionOptionsFromSettings(explicit false) error = %v", err)
	}
	if options.QuestionCount != 12 {
		t.Fatalf("QuestionCount with explicit false = %d, want 12", options.QuestionCount)
	}

	settings.FocusModeDefault = false
	override = true
	options, err = sessionOptionsFromSettings(settings, nil, &override)
	if err != nil {
		t.Fatalf("sessionOptionsFromSettings(explicit true) error = %v", err)
	}
	if options.QuestionCount != session.FocusQuestionCount {
		t.Fatalf("QuestionCount with explicit true = %d, want %d", options.QuestionCount, session.FocusQuestionCount)
	}

	_, err = sessionOptionsFromSettings(settings, &questionOverride, &override)
	if err == nil {
		t.Fatal("sessionOptionsFromSettings(conflicting overrides) error = nil, want error")
	}
	if !strings.Contains(err.Error(), "--questions") {
		t.Fatalf("sessionOptionsFromSettings(conflicting overrides) error = %v, want questions conflict", err)
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

func hasAlias(cmd *cobra.Command, want string) bool {
	for _, alias := range cmd.Aliases {
		if alias == want {
			return true
		}
	}
	return false
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
		{Lemma: "adopt", Pos: "verb", MeaningJA: "採用する", Level: "core-1", FrequencyRank: 100, DistractorGroup: "basic-verb-action"},
		{Lemma: "apply", Pos: "verb", MeaningJA: "応募する", Level: "core-1", FrequencyRank: 120, DistractorGroup: "basic-verb-action"},
		{Lemma: "cancel", Pos: "verb", MeaningJA: "取り消す", Level: "core-1", FrequencyRank: 140, DistractorGroup: "basic-verb-action"},
		{Lemma: "deliver", Pos: "verb", MeaningJA: "届ける", Level: "core-1", FrequencyRank: 160, DistractorGroup: "basic-verb-action"},
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
	if !strings.Contains(output, i18n.T(i18n.CLIDoctorHeader)) {
		t.Fatalf("doctor output = %q, want header", output)
	}
	if !strings.Contains(output, i18n.T(i18n.CLIDoctorOK)) {
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
		{Lemma: "abandon", Pos: "verb", MeaningJA: "捨てる", Level: "core-1", FrequencyRank: 100, DistractorGroup: "basic-verb-action"},
		{Lemma: "apply", Pos: "verb", MeaningJA: "応募する", Level: "core-1", FrequencyRank: 200, DistractorGroup: "basic-verb-action"},
		{Lemma: "benefit", Pos: "noun", MeaningJA: "利益", Level: "core-1", FrequencyRank: 300, DistractorGroup: "basic-noun-business"},
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

func TestDoctorCommandHandlesLegacySchemaReadOnly(t *testing.T) {
	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, "user.db")

	createLegacyDoctorDB(t, dbPath)
	t.Setenv("EITANGO_DATA_DIR", dataDir)

	var out bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"doctor"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want exit code error for missing migration")
	}

	var withExitCode interface{ ExitCode() int }
	if !errors.As(err, &withExitCode) {
		t.Fatalf("Execute() error = %T, want exit code error", err)
	}
	if withExitCode.ExitCode() != 1 {
		t.Fatalf("ExitCode() = %d, want 1", withExitCode.ExitCode())
	}

	output := out.String()
	if !strings.Contains(output, "005_answer_modes.sql") {
		t.Fatalf("doctor output = %q, want missing 005 migration", output)
	}
	if strings.Contains(output, "active sessions could not be read") || strings.Contains(output, "no such column: sessions.answer_mode") || strings.Contains(output, "no such column: answer_mode") {
		t.Fatalf("doctor output = %q, want legacy schema fallback without read failure", output)
	}
}

func TestResetCommandRequiresScopeFlag(t *testing.T) {
	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, "user.db")

	t.Setenv("EITANGO_DATA_DIR", dataDir)

	var out bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"reset"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want validation error")
	}
	if !strings.Contains(err.Error(), "--progress or --reseed") {
		t.Fatalf("Execute() error = %v, want scope guidance", err)
	}
	if _, statErr := os.Stat(dbPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("user.db should not be created, stat error = %v", statErr)
	}
}

func createLegacyDoctorDB(t *testing.T, dbPath string) {
	t.Helper()

	ctx := context.Background()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	if _, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_migrations (
version TEXT PRIMARY KEY,
applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`); err != nil {
		t.Fatalf("create schema_migrations: %v", err)
	}

	for _, migration := range []string{
		"001_init.sql",
		"002_indexes.sql",
		"003_words_pos_rank.sql",
		"004_words_source.sql",
	} {
		sqlBytes, err := projectassets.Embedded.ReadFile("migrations/" + migration)
		if err != nil {
			t.Fatalf("ReadFile(%s) error = %v", migration, err)
		}
		if _, err := db.ExecContext(ctx, string(sqlBytes)); err != nil {
			t.Fatalf("apply %s: %v", migration, err)
		}
		if _, err := db.ExecContext(ctx, `INSERT INTO schema_migrations (version) VALUES (?)`, migration); err != nil {
			t.Fatalf("record %s: %v", migration, err)
		}
	}

	if _, err := db.ExecContext(ctx, `
INSERT INTO words (id, lemma, pos, meaning_ja, level, frequency_rank, distractor_group, source)
VALUES
	(1, 'adopt', 'verb', '採用する', 'core-1', 100, 'basic-verb-action', ?),
	(2, 'apply', 'verb', '応募する', 'core-1', 120, 'basic-verb-action', ?),
	(3, 'cancel', 'verb', '取り消す', 'core-1', 140, 'basic-verb-action', ?),
	(4, 'deliver', 'verb', '届ける', 'core-1', 160, 'basic-verb-action', ?)
`, store.WordSourceCore, store.WordSourceCore, store.WordSourceCore, store.WordSourceCore); err != nil {
		t.Fatalf("insert words: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO app_meta (key, value)
VALUES ('dict_version', ?)
`, dict.CoreWordsVersion); err != nil {
		t.Fatalf("insert dict_version: %v", err)
	}

	if _, err := db.ExecContext(ctx, `
INSERT INTO sessions (id, started_at, mode, total_questions, answered_questions, status)
VALUES (?, CURRENT_TIMESTAMP, ?, 1, 0, ?)
`, "legacy-active", store.ModeLearn, store.SessionStatusActive); err != nil {
		t.Fatalf("insert legacy session: %v", err)
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO session_items (session_id, ordinal, word_id, kind, status)
VALUES (?, 1, ?, ?, ?)
`, "legacy-active", int64(1), store.ItemKindNew, store.ItemStatusPending); err != nil {
		t.Fatalf("insert legacy session item: %v", err)
	}
}

func TestResetCommandProgressClearsLearningHistory(t *testing.T) {
	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, "user.db")
	entries := resetTestEntries()

	seedResetFixture(t, dataDir, entries, dict.CoreWordsVersion)
	t.Setenv("EITANGO_DATA_DIR", dataDir)

	var out bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"reset", "--progress"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), i18n.T(i18n.CLIResetHeader)) {
		t.Fatalf("reset output = %q, want header", out.String())
	}
	if got := mustCountSQLiteTable(t, dbPath, "words"); got != len(entries) {
		t.Fatalf("words after progress reset = %d, want %d", got, len(entries))
	}
	if got := mustCountSQLiteTable(t, dbPath, "sessions"); got != 0 {
		t.Fatalf("sessions after progress reset = %d, want 0", got)
	}
	if got := mustCountSQLiteTable(t, dbPath, "session_items"); got != 0 {
		t.Fatalf("session_items after progress reset = %d, want 0", got)
	}
	if got := mustCountSQLiteTable(t, dbPath, "reviews"); got != 0 {
		t.Fatalf("reviews after progress reset = %d, want 0", got)
	}
	if got := mustCountSQLiteTable(t, dbPath, "progress"); got != 0 {
		t.Fatalf("progress after progress reset = %d, want 0", got)
	}
	if version := mustMetaValue(t, dbPath, "dict_version"); version != dict.CoreWordsVersion {
		t.Fatalf("dict_version after progress reset = %q, want %q", version, dict.CoreWordsVersion)
	}
}

func TestResetCommandReseedReloadsEmbeddedCoreWords(t *testing.T) {
	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, "user.db")

	seedResetFixture(t, dataDir, resetTestEntries(), dict.CoreWordsVersion)
	t.Setenv("EITANGO_DATA_DIR", dataDir)

	var out bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"reset", "--reseed"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !strings.Contains(out.String(), dict.CoreWordsVersion) {
		t.Fatalf("reset output = %q, want reseed summary with dict version", out.String())
	}

	coreWords, err := dict.LoadCoreWords()
	if err != nil {
		t.Fatalf("LoadCoreWords() error = %v", err)
	}
	if got := mustCountSQLiteTable(t, dbPath, "words"); got != len(coreWords) {
		t.Fatalf("words after reseed = %d, want %d", got, len(coreWords))
	}
	if got := mustCountSQLiteTable(t, dbPath, "sessions"); got != 0 {
		t.Fatalf("sessions after reseed = %d, want 0", got)
	}
	if got := mustCountSQLiteTable(t, dbPath, "reviews"); got != 0 {
		t.Fatalf("reviews after reseed = %d, want 0", got)
	}
	if got := mustCountSQLiteTable(t, dbPath, "progress"); got != 0 {
		t.Fatalf("progress after reseed = %d, want 0", got)
	}
	if version := mustMetaValue(t, dbPath, "dict_version"); version != dict.CoreWordsVersion {
		t.Fatalf("dict_version after reseed = %q, want %q", version, dict.CoreWordsVersion)
	}
}

func seedResetFixture(t *testing.T, dataDir string, entries []dict.Entry, version string) {
	t.Helper()

	ctx := context.Background()
	st, err := store.Open(ctx, filepath.Join(dataDir, "user.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		_ = st.Close()
	}()

	if err := st.Migrate(ctx); err != nil {
		t.Fatalf("Migrate() error = %v", err)
	}
	if err := st.SeedWords(ctx, entries, version); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	words, err := st.ListNewWords(ctx, 10, nil)
	if err != nil {
		t.Fatalf("ListNewWords() error = %v", err)
	}
	record, _, err := st.CreateSession(ctx, store.ModeLearn, store.AnswerModeChoice, []store.SessionItemPlan{
		{WordID: words[0].ID, Kind: store.ItemKindNew},
	})
	if err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}
	if _, _, err := st.SaveAnswer(ctx, store.ReviewEvent{
		SessionID:      record.ID,
		ItemOrdinal:    1,
		WordID:         words[0].ID,
		Kind:           store.ItemKindNew,
		SelectedChoice: 1,
		CorrectChoice:  1,
		IsCorrect:      true,
		Rating:         srs.Good,
		AnsweredAt:     time.Now().UTC(),
		ResponseMS:     750,
	}); err != nil {
		t.Fatalf("SaveAnswer() error = %v", err)
	}
}

func resetTestEntries() []dict.Entry {
	return []dict.Entry{
		{Lemma: "accept", Pos: "verb", MeaningJA: "受け入れる", Level: "core-1", FrequencyRank: 100, DistractorGroup: "basic-verb-action"},
		{Lemma: "avoid", Pos: "verb", MeaningJA: "避ける", Level: "core-1", FrequencyRank: 120, DistractorGroup: "basic-verb-action"},
		{Lemma: "budget", Pos: "noun", MeaningJA: "予算", Level: "core-1", FrequencyRank: 140, DistractorGroup: "basic-noun-business"},
	}
}

func mustCountSQLiteTable(t *testing.T, dbPath, table string) int {
	t.Helper()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	return count
}

func mustMetaValue(t *testing.T, dbPath, key string) string {
	t.Helper()

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	defer func() {
		_ = db.Close()
	}()

	var value string
	if err := db.QueryRow("SELECT value FROM app_meta WHERE key = ?", key).Scan(&value); err != nil {
		t.Fatalf("load app_meta %s: %v", key, err)
	}
	return value
}

type mainStubUpdateService struct {
	checkNowResult  updatecheck.Result
	checkNowErr     error
	checkNowCalls   int
	checkNowVersion string
}

func (s *mainStubUpdateService) Check(context.Context, string) (updatecheck.Result, error) {
	return updatecheck.Result{}, nil
}

func (s *mainStubUpdateService) CheckNow(_ context.Context, currentVersion string) (updatecheck.Result, error) {
	s.checkNowCalls++
	s.checkNowVersion = currentVersion
	return s.checkNowResult, s.checkNowErr
}
