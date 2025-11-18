package scheduler

import (
    "context"
    "time"
)

type TaskRepository interface {
    LoadEnabledTasks(ctx context.Context) ([]Task, error)
    MarkRunning(ctx context.Context, id int, t time.Time) error
    MarkStatus(ctx context.Context, id int, status int) error
}
