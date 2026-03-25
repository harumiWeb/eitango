package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yourname/eitango/internal/store"
)

func TestImportCommandImportsCSVAndUsesDefaultSource(t *testing.T) {
	dataDir := t.TempDir()
	csvPath := filepath.Join(dataDir, "travel-pack.csv")
	if err := os.WriteFile(csvPath, []byte(strings.Join([]string{
		"lemma,meaning_ja,pos,level,distractor_group,example_en,example_ja",
		"coordinate,調整する,verb,toeic700,import-verb,They coordinate each release.,彼らは各リリースを調整する。",
		"draft,下書きする,verb,toeic700,import-verb,Please draft the reply.,返信の下書きをしてください。",
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
	for _, snapshot := range snapshots {
		if snapshot.Word.Source == "import:travel-pack" {
			imported++
		}
	}
	if imported != 2 {
		t.Fatalf("imported snapshot count = %d, want 2", imported)
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
	if got := err.Error(); got != "eitango import only supports --format csv" {
		t.Fatalf("Execute() error = %q, want format guidance", got)
	}
	if _, statErr := os.Stat(dbPath); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("user.db should not be created, stat error = %v", statErr)
	}
}
