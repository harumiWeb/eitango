package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yourname/eitango/internal/dict"
)

func TestValidateCommandValidatesEmbeddedCore(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"validate", "--embedded-core"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "kind: core") {
		t.Fatalf("validate output = %q, want core kind", output)
	}
	if !strings.Contains(output, "source: embedded-core") {
		t.Fatalf("validate output = %q, want embedded core source", output)
	}
	entries, err := dict.LoadCoreWords()
	if err != nil {
		t.Fatalf("LoadCoreWords() error = %v", err)
	}
	if !strings.Contains(output, fmt.Sprintf("entries: %d", len(entries))) {
		t.Fatalf("validate output = %q, want entry count", output)
	}
}

func TestValidateCommandValidatesImportCSVWithoutOpeningDB(t *testing.T) {
	dataDir := t.TempDir()
	csvPath := filepath.Join(dataDir, "travel-pack.csv")
	if err := os.WriteFile(csvPath, []byte(strings.Join([]string{
		"lemma,meaning_ja,pos,level,frequency_rank,distractor_group",
		"coordinate,調整する,verb,toeic700,4200,import-verb",
		"budget,予算,noun,toeic700,4300,import-noun",
	}, "\n")), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv("EITANGO_DATA_DIR", dataDir)

	var out bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"validate", "--file", csvPath, "--kind", "import"})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "kind: import") {
		t.Fatalf("validate output = %q, want import kind", output)
	}
	if !strings.Contains(output, "entries: 2") {
		t.Fatalf("validate output = %q, want entry count", output)
	}
	if _, statErr := os.Stat(filepath.Join(dataDir, "user.db")); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("user.db should not be created, stat error = %v", statErr)
	}
}

func TestValidateCommandRejectsMissingInput(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"validate"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want input guidance")
	}
	if got := err.Error(); got != "validate requires either --file or --embedded-core" {
		t.Fatalf("Execute() error = %q, want input guidance", got)
	}
}

func TestValidateCommandRejectsDuplicateImportRows(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	csvPath := filepath.Join(dataDir, "dupe.csv")
	if err := os.WriteFile(csvPath, []byte(strings.Join([]string{
		"lemma,meaning_ja,pos",
		"accept,受け入れる,verb",
		"accept,承諾する,verb",
	}, "\n")), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var out bytes.Buffer
	cmd := newRootCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"validate", "--file", csvPath, "--kind", "import"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("Execute() error = nil, want duplicate-row error")
	}
	if got := err.Error(); got != "import entry 2 (accept/verb) duplicates lemma/pos from entry 1" {
		t.Fatalf("Execute() error = %q, want duplicate-row guidance", got)
	}
}
