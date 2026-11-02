package worker

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"task-api/internal/model"
)

type stubSource struct {
	calls  atomic.Int32
	result []model.ReminderCount
	err    error
}

func (s *stubSource) DueReminders(ctx context.Context) ([]model.ReminderCount, error) {
	s.calls.Add(1)
	if s.err != nil {
		return nil, s.err
	}
	return s.result, nil
}

func discardLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

func TestReminderWorkerRunOnce(t *testing.T) {
	source := &stubSource{result: []model.ReminderCount{{UserID: 1, Overdue: 2, DueToday: 1}}}
	worker := NewReminderWorker(source, time.Minute, discardLogger())

	worker.runOnce(context.Background())

	if got := source.calls.Load(); got != 1 {
		t.Errorf("expected 1 call, got %d", got)
	}
}

func TestReminderWorkerContinuesAfterError(t *testing.T) {
	source := &stubSource{err: errors.New("db down")}
	worker := NewReminderWorker(source, time.Minute, discardLogger())

	worker.runOnce(context.Background())
	worker.runOnce(context.Background())

	if got := source.calls.Load(); got != 2 {
		t.Errorf("expected worker to retry on next tick, got %d calls", got)
	}
}

func TestReminderWorkerStopsOnContextCancel(t *testing.T) {
	source := &stubSource{}
	worker := NewReminderWorker(source, 10*time.Millisecond, discardLogger())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		worker.Run(ctx)
		close(done)
	}()

	time.Sleep(30 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not stop after context cancellation")
	}
}
