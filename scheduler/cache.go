package scheduler

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

type TaskCache struct {
	repo            TaskRepository
	logger          *zap.SugaredLogger
	refreshInterval time.Duration

	mu    sync.RWMutex
	tasks []compiledTask
}

func NewTaskCache(repo TaskRepository, logger *zap.SugaredLogger, refreshInterval time.Duration) *TaskCache {
	if refreshInterval <= 0 {
		refreshInterval = 10 * time.Second
	}
	return &TaskCache{
		repo:            repo,
		logger:          logger,
		refreshInterval: refreshInterval,
	}
}

func (c *TaskCache) Start(ctx context.Context) {
	go c.loop(ctx)
	c.refresh(ctx)
}

func (c *TaskCache) loop(ctx context.Context) {
	ticker := time.NewTicker(c.refreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.refresh(ctx)
		}
	}
}

func (c *TaskCache) refresh(ctx context.Context) {
	tasks, err := c.repo.LoadEnabledTasks(ctx)
	if err != nil {
		c.logger.Errorf("load tasks failed: %v", err)
		return
	}
	compiled := make([]compiledTask, 0, len(tasks))
	for _, task := range tasks {
		expr, err := CompileCronExpr(task.RunExpr)
		if err != nil {
			c.logger.Warnf("compile cron failed for task %d(%s): %v", task.ID, task.Name, err)
			continue
		}
		compiled = append(compiled, compiledTask{Task: task, Expr: expr})
	}
	c.mu.Lock()
	c.tasks = compiled
	c.mu.Unlock()
}

func (c *TaskCache) Snapshot() []compiledTask {
	c.mu.RLock()
	defer c.mu.RUnlock()
	copyTasks := make([]compiledTask, len(c.tasks))
	copy(copyTasks, c.tasks)
	return copyTasks
}
