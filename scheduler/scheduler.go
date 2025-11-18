package scheduler

import (
	"context"
	"time"

	"go.uber.org/zap"
)

type Scheduler struct {
	cache  *TaskCache
	repo   TaskRepository
	runner TaskRunner
	lock   *taskLock
	logger *zap.SugaredLogger
}

func NewScheduler(cache *TaskCache, repo TaskRepository, runner TaskRunner, logger *zap.SugaredLogger) *Scheduler {
	return &Scheduler{
		cache:  cache,
		repo:   repo,
		runner: runner,
		lock:   newTaskLock(),
		logger: logger,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	s.cache.Start(ctx)
	go s.loop(ctx)
}

func (s *Scheduler) loop(ctx context.Context) {
	timer := time.NewTimer(s.untilNextMinute())
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			s.dispatch(ctx, time.Now())
			timer.Reset(time.Minute)
		}
	}
}

func (s *Scheduler) untilNextMinute() time.Duration {
	now := time.Now()
	next := now.Truncate(time.Minute).Add(time.Minute)
	return next.Sub(now)
}

func (s *Scheduler) dispatch(ctx context.Context, now time.Time) {
	tasks := s.cache.Snapshot()
	for _, ct := range tasks {
		task := ct.Task
		if task.IsEnable != 0 {
			continue
		}
		if task.Status == 1 {
			continue
		}
		if ct.Expr == nil || !ct.Expr.Match(now) {
			continue
		}
		if !s.lock.TryLock(task.ID) {
			continue
		}
		if err := s.repo.MarkRunning(ctx, task.ID, now); err != nil {
			s.logger.Errorf("mark running failed for task %d: %v", task.ID, err)
			s.lock.Unlock(task.ID)
			continue
		}
		go s.executeTask(ctx, task)
	}
}

func (s *Scheduler) executeTask(ctx context.Context, task Task) {
	defer s.lock.Unlock(task.ID)
	if err := s.runner.Run(ctx, task); err != nil {
		s.logger.Errorf("task %d execute failed: %v", task.ID, err)
		if err2 := s.repo.MarkStatus(ctx, task.ID, -1); err2 != nil {
			s.logger.Errorf("mark status failed for task %d: %v", task.ID, err2)
		}
		return
	}
	if err := s.repo.MarkStatus(ctx, task.ID, 0); err != nil {
		s.logger.Errorf("mark success status failed for task %d: %v", task.ID, err)
	}
}
