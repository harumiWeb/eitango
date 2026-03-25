package store

import (
	"context"
	"testing"
	"time"

	"github.com/yourname/eitango/internal/dict"
)

func TestEmbeddedCoreWordsPassDoctorDiagnostics(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	st := newTestStore(t)
	entries, err := dict.LoadCoreWords()
	if err != nil {
		t.Fatalf("LoadCoreWords() error = %v", err)
	}
	if err := st.SeedWords(ctx, entries, dict.CoreWordsVersion); err != nil {
		t.Fatalf("SeedWords() error = %v", err)
	}

	snapshot, err := st.LoadHomeSnapshot(ctx)
	if err != nil {
		t.Fatalf("LoadHomeSnapshot() error = %v", err)
	}
	if snapshot.NewCount != len(entries) {
		t.Fatalf("NewCount = %d, want %d", snapshot.NewCount, len(entries))
	}

	report := st.RunDiagnostics(ctx)
	if report.HasIssues() {
		t.Fatalf("RunDiagnostics() reported issues on embedded core words: %+v", report.Checks)
	}
}
