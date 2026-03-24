package srs

import (
	"testing"
	"time"
)

func TestUpdateGoodFromNew(t *testing.T) {
	now := time.Date(2026, 3, 24, 12, 0, 0, 0, time.UTC)
	updated := Update(DefaultProgress(), Good, now)

	if updated.State != "review" {
		t.Fatalf("State = %q, want review", updated.State)
	}
	if updated.IntervalDays != 3 {
		t.Fatalf("IntervalDays = %v, want 3", updated.IntervalDays)
	}
	if updated.DueAt == nil || !updated.DueAt.Equal(now.Add(72*time.Hour)) {
		t.Fatalf("DueAt = %v, want %v", updated.DueAt, now.Add(72*time.Hour))
	}
}
