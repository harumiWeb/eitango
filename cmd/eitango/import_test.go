package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/yourname/eitango/internal/store"
)

func TestImportCommandImportsCSVAndUsesDefaultSource(t *testing.T) {
	dataDir := t.TempDir()
	csvPath := filepath.Join(dataDir, "travel-pack.csv")
	if err := os.WriteFile(csvPath, []byte(strings.Join([]string{
		"lemma,meaning_ja,pos,level,frequency_rank,distractor_group,example_en,example_ja",
		"coordinate,調整する,verb,toeic700,4200,import-verb,They coordinate each release.,彼らは各リリースを調整する。",
		"draft,下書きする,verb,toeic700,4300,import-verb,Please draft the reply.,返信の下書きをしてください。",
	}, "\n")), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("EITANGO_DATA_DIR", dataDir)

	var out bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"import", "--file", csvPath, "--format", "csv"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	output := out.String()
	if !strings.Contains(output, "source: import:travel-pack") {
		t.Fatalf("import output = %q, want derived source", output)
	}
	if !strings.Contains(output, "inserted words: 2") {
		t.Fatalf("import output = %q, want inserted count", output)
	}

	st, err := store.Open(context.Background(), filepath.Join(dataDir, "user.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		_ = st.Close()
	}()

	snapshots, err := st.ListExportWordSnapshots(context.Background())
	if err != nil {
		t.Fatalf("ListExportWordSnapshots() error = %v", err)
	}
	imported := 0
	coordinateRank := 0
	draftRank := 0
	for _, snapshot := range snapshots {
		if snapshot.Word.Source == "import:travel-pack" {
			imported++
			switch snapshot.Word.Lemma {
			case "coordinate":
				coordinateRank = snapshot.Word.FrequencyRank
			case "draft":
				draftRank = snapshot.Word.FrequencyRank
			}
		}
	}
	if imported != 2 {
		t.Fatalf("imported snapshot count = %d, want 2", imported)
	}
	if coordinateRank != 4200 || draftRank != 4300 {
		t.Fatalf("imported frequency ranks = (%d, %d), want (4200, 4300)", coordinateRank, draftRank)
	}
}

func TestImportCommandImportsJSONL(t *testing.T) {
	dataDir := t.TempDir()
	jsonlPath := filepath.Join(dataDir, "business-pack.jsonl")
	if err := os.WriteFile(jsonlPath, []byte(strings.Join([]string{
		`{"lemma":"coordinate","pos":"verb","meaning_ja":"調整する","level":"toeic700","frequency_rank":4200,"distractor_group":"import-verb"}`,
		`{"lemma":"budget","pos":"noun","meaning_ja":"予算","level":"toeic700","frequency_rank":4300,"distractor_group":"import-noun"}`,
	}, "\n")), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv("EITANGO_DATA_DIR", dataDir)

	var out bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"import", "--file", jsonlPath, "--format", "jsonl"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "source: import:business-pack") {
		t.Fatalf("import output = %q, want derived JSONL source", output)
	}
	if !strings.Contains(output, "inserted words: 2") {
		t.Fatalf("import output = %q, want inserted count", output)
	}
}

func TestImportCommandRejectsUnsupportedFormatBeforeOpeningDB(t *testing.T) {
	dataDir := t.TempDir()
	csvPath := filepath.Join(dataDir, "travel-pack.csv")
	if err := os.WriteFile(csvPath, []byte("lemma,meaning_ja\ncoordinate,調整する\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	dbPath := filepath.Join(dataDir, "user.db")
	t.Setenv("EITANGO_DATA_DIR", dataDir)

	var out bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"import", "--file", csvPath, "--format", "json"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want format validation error")
	}
	if got := err.Error(); got != "eitango import only supports --format csv or jsonl" {
		t.Fatalf("Execute() error = %q, want format guidance", got)
	}
	if _, statErr := os.Stat(dbPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("user.db should not be created, stat error = %v", statErr)
	}
}

func TestResolveImportFormatNormalizesCaseAndWhitespace(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{Use: "eitango import"}
	cmd.Flags().String("format", "", "")
	if err := cmd.Flags().Set("format", " CSV "); err != nil {
		t.Fatalf("Set(format) error = %v", err)
	}

	format, err := resolveImportFormat(cmd)
	if err != nil {
		t.Fatalf("resolveImportFormat() error = %v", err)
	}
	if format != formatCSV {
		t.Fatalf("resolveImportFormat() = %q, want %q", format, formatCSV)
	}
}
