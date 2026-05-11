package scanner

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type mockReleaseScanner struct {
	calls atomic.Int32
}

func (m *mockReleaseScanner) CheckReleases(_ context.Context) {
	m.calls.Add(1)
}

func TestScanner_Start_CallsCheckReleasesOnTick(t *testing.T) {
	svc := &mockReleaseScanner{}
	sc := New(svc, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sc.Start(ctx)

	time.Sleep(180 * time.Millisecond)
	cancel()

	got := svc.calls.Load()
	if got < 2 {
		t.Fatalf("expected at least 2 CheckReleases calls, got %d", got)
	}
}
