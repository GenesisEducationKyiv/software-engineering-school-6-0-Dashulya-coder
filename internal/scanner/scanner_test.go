package scanner

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/release"
)

type mockPoller struct {
	calls atomic.Int32
}

func (m *mockPoller) Poll(_ context.Context) {
	m.calls.Add(1)
}

var _ release.Poller = (*mockPoller)(nil)

func TestScanner_Start_CallsPollOnTick(t *testing.T) {
	p := &mockPoller{}
	sc := New(p, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sc.Start(ctx)

	time.Sleep(180 * time.Millisecond)
	cancel()

	got := p.calls.Load()
	if got < 2 {
		t.Fatalf("expected at least 2 Poll calls, got %d", got)
	}
}
