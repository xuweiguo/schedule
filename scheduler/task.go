package scheduler

import "time"

type Task struct {
	ID             int
	Name           string
	Command        string
	RunExpr        string
	IsEnable       int
	Status         int
	DataCountLimit int
	RunSleepMicro  int
	TryTimesLimit  int
	RunWay         int
	LastStartTime  time.Time
}

type compiledTask struct {
	Task Task
	Expr *CronExprCompiled
}
