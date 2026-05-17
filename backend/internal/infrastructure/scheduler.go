package infrastructure

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
    cron *cron.Cron
    log  *slog.Logger
}

// Job is the function signature for scheduled tasks.
type Job func(ctx context.Context) error

// NewScheduler creates a new scheduler with Istanbul timezone.
func NewScheduler(log *slog.Logger) *Scheduler {
    loc, err := time.LoadLocation("Europe/Istanbul")
    if err != nil {
        loc = time.UTC
    }

    c := cron.New(
        cron.WithLocation(loc),
        cron.WithChain(
            cron.Recover(cron.DefaultLogger),
        ),
    )

    return &Scheduler{cron: c, log: log}
}

// Schedule schedules a job at a specific time (HH:MM format, 24-hour).
// Example: "08:00" runs daily at 08:00.
func (s *Scheduler) Schedule(schedule string, job Job) error {
    timeOfDay, err := time.Parse("15:04", schedule)
    if err != nil {
        return fmt.Errorf("invalid schedule %q: expected HH:MM", schedule)
    }

    spec := fmt.Sprintf("%d %d * * *", timeOfDay.Minute(), timeOfDay.Hour())

    _, err = s.cron.AddFunc(spec, func() {
        ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
        defer cancel()

        s.log.Info("job started", "time", schedule)
        if err := job(ctx); err != nil {
            s.log.Error("job failed", "error", err)
            return
        }
        s.log.Info("job completed")
    })

    return err
}

// Start begins the scheduler (blocks until Stop is called).
func (s *Scheduler) Start() {
    s.cron.Start()
}

// Stop halts the scheduler and waits for running jobs.
func (s *Scheduler) Stop() context.Context {
    return s.cron.Stop()
}