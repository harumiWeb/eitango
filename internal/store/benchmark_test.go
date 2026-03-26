package store

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/yourname/eitango/internal/dict"
)

func BenchmarkSeedWordsEmbeddedCore(b *testing.B) {
	ctx := context.Background()
	entries, err := dict.LoadCoreWords()
	if err != nil {
		b.Fatalf("LoadCoreWords() error = %v", err)
	}

	baseDir := b.TempDir()
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		dbPath := filepath.Join(baseDir, fmt.Sprintf("seed-%d.db", i))
		st, err := Open(ctx, dbPath)
		if err != nil {
			b.Fatalf("Open() error = %v", err)
		}
		if err := st.Migrate(ctx); err != nil {
			_ = st.Close()
			b.Fatalf("Migrate() error = %v", err)
		}
		if err := st.SeedWords(ctx, entries, dict.CoreWordsVersion); err != nil {
			_ = st.Close()
			b.Fatalf("SeedWords() error = %v", err)
		}
		if err := st.Close(); err != nil {
			b.Fatalf("Close() error = %v", err)
		}
	}
}
