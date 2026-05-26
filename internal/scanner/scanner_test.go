package scanner

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/release"
)

type mockPoller struct {
	calls  atomic.Int32
	onPoll func()
}

func (m *mockPoller) Poll(_ context.Context) {
	m.calls.Add(1)
	if m.onPoll != nil {
		m.onPoll()
	}
}

var _ release.Poller = (*mockPoller)(nil)

func TestScanner_Start_CallsPollOnTick(t *testing.T) {
	done := make(chan struct{}, 10)
	p := &mockPoller{
		onPoll: func() { done <- struct{}{} },
	}
	sc := New(p, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sc.Start(ctx)

	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatalf("Poll was not called in time (call %d)", i+1)
		}
	}
}
