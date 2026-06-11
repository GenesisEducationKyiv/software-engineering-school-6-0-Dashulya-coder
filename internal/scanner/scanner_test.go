package scanner

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockPoller struct {
	mock.Mock
}

func (m *mockPoller) Poll(ctx context.Context) {
	m.Called(ctx)
}

func TestScanner_Start_CallsPollOnTick(t *testing.T) {
	done := make(chan struct{}, 10)
	p := new(mockPoller)
	p.On("Poll", mock.Anything).
		Run(func(_ mock.Arguments) { done <- struct{}{} }).
		Return()

	sc := New(p, 10*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sc.Start(ctx)

	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			require.Fail(t, fmt.Sprintf("Poll was not called in time (call %d)", i+1))
		}
	}

	p.AssertExpectations(t)
}
