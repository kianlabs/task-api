package worker

import (
	"context"
	"log/slog"
	"time"

	"task-api/internal/model"
)

type ReminderSource interface {
	DueReminders(ctx context.Context) ([]model.ReminderCount, error)
}

type ReminderWorker struct {
	source   ReminderSource
	interval time.Duration
	logger   *slog.Logger
}

func NewReminderWorker(source ReminderSource, interval time.Duration, logger *slog.Logger) *ReminderWorker {
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &ReminderWorker{source: source, interval: interval, logger: logger}
}

func (w *ReminderWorker) Run(ctx context.Context) {
	w.logger.Info("reminder worker started", "interval", w.interval.String())
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Jalankan sekali saat start agar scan pertama tidak tertunda satu interval.
	w.runOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("reminder worker stopped")
			return
		case <-ticker.C:
			w.runOnce(ctx)
		}
	}
}

func (w *ReminderWorker) runOnce(ctx context.Context) {
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	reminders, err := w.source.DueReminders(runCtx)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		w.logger.Error("reminder scan failed", "error", err)
		return
	}

	for _, r := range reminders {
		w.logger.Warn("task needs attention",
			"user_id", r.UserID,
			"overdue", r.Overdue,
			"due_today", r.DueToday,
		)
	}

	w.logger.Info("reminder scan finished", "users_notified", len(reminders))
}
