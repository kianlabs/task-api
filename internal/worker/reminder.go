package worker

import (
	"context"
	"log/slog"
	"time"

	"task-api/internal/model"
)

// ReminderSource adalah dependensi minimal worker agar mudah diuji.
type ReminderSource interface {
	DueReminders(ctx context.Context) ([]model.ReminderCount, error)
}

// ReminderWorker memindai task yang jatuh tempo secara berkala dan menulis
// ringkasannya ke log terstruktur.
//
// Redis sengaja tidak dipakai: beban kerjanya kecil, satu instance, dan belum
// ada kebutuhan koordinasi antar-proses yang terukur.
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

// Run berjalan sampai ctx dibatalkan, lalu keluar dengan bersih.
func (w *ReminderWorker) Run(ctx context.Context) {
	w.logger.Info("reminder worker started", "interval", w.interval.String())
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Jalankan sekali saat start agar tidak menunggu interval pertama.
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
	// Batasi durasi satu siklus agar worker tidak menggantung.
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
