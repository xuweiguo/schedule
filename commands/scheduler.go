package commands

import (
	"context"
	"net/http"
	"time"

	"github.com/mix-go/xcli/flag"
	"github.com/mix-go/xutil/xenv"

	"scheule/di"
	"scheule/scheduler"
)

type SchedulerCommand struct{}

func (t *SchedulerCommand) Main() {
	ctx := context.Background()
	logger := di.Zap()
	db := di.Gorm()

	repo := scheduler.NewModelsRepo(db)
	refreshSec := xenv.Getenv("TASK_CACHE_REFRESH_SECOND").Int64(10)
	if refreshSec <= 0 {
		refreshSec = 10
	}
	cache := scheduler.NewTaskCache(repo, logger, time.Duration(refreshSec)*time.Second)

	baseURL := flag.Match("base").String(xenv.Getenv("TASK_API_BASE_URL").String())
	if baseURL == "" {
		logger.Fatalf("TASK_API_BASE_URL must be set via env or --base option")
	}
	timeout := xenv.Getenv("TASK_HTTP_TIMEOUT_SECOND").Int64(30)
	if timeout <= 0 {
		timeout = 30
	}
	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
	runner := scheduler.NewHTTPTaskRunner(baseURL, client, logger)

	sched := scheduler.NewScheduler(cache, repo, runner, logger)
	sched.Start(ctx)

	select {}
}
