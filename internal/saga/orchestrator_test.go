package saga_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Dashulya-coder/CaseTaskNotifier/internal/saga"
)

func action(log *[]string, name string, err error) func(context.Context) error {
	return func(context.Context) error {
		*log = append(*log, "do:"+name)
		return err
	}
}

func compensation(log *[]string, name string, err error) func(context.Context) error {
	return func(context.Context) error {
		*log = append(*log, "undo:"+name)
		return err
	}
}

func TestRun_AllStepsSucceed(t *testing.T) {
	var log []string

	err := saga.Run(context.Background(),
		saga.Step{Name: "a", Action: action(&log, "a", nil), Compensation: compensation(&log, "a", nil)},
		saga.Step{Name: "b", Action: action(&log, "b", nil), Compensation: compensation(&log, "b", nil)},
		saga.Step{Name: "c", Action: action(&log, "c", nil)},
	)

	require.NoError(t, err)
	assert.Equal(t, []string{"do:a", "do:b", "do:c"}, log)
}

func TestRun_FailureCompensatesCompletedInReverse(t *testing.T) {
	var log []string
	boom := errors.New("boom")

	err := saga.Run(context.Background(),
		saga.Step{Name: "a", Action: action(&log, "a", nil), Compensation: compensation(&log, "a", nil)},
		saga.Step{Name: "b", Action: action(&log, "b", nil), Compensation: compensation(&log, "b", nil)},
		saga.Step{Name: "c", Action: action(&log, "c", boom), Compensation: compensation(&log, "c", nil)},
	)

	require.ErrorIs(t, err, boom)
	assert.Equal(t, []string{"do:a", "do:b", "do:c", "undo:b", "undo:a"}, log)
}

func TestRun_FirstStepFailureRunsNoCompensation(t *testing.T) {
	var log []string
	boom := errors.New("boom")

	err := saga.Run(context.Background(),
		saga.Step{Name: "a", Action: action(&log, "a", boom), Compensation: compensation(&log, "a", nil)},
		saga.Step{Name: "b", Action: action(&log, "b", nil), Compensation: compensation(&log, "b", nil)},
	)

	require.ErrorIs(t, err, boom)
	assert.Equal(t, []string{"do:a"}, log)
}

func TestRun_CompensationErrorDoesNotStopOthers(t *testing.T) {
	var log []string
	boom := errors.New("boom")

	err := saga.Run(context.Background(),
		saga.Step{Name: "a", Action: action(&log, "a", nil), Compensation: compensation(&log, "a", nil)},
		saga.Step{Name: "b", Action: action(&log, "b", nil), Compensation: compensation(&log, "b", errors.New("undo failed"))},
		saga.Step{Name: "c", Action: action(&log, "c", boom)},
	)

	require.ErrorIs(t, err, boom)
	assert.Equal(t, []string{"do:a", "do:b", "do:c", "undo:b", "undo:a"}, log)
}
