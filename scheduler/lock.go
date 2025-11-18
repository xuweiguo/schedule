package scheduler

import "sync"

type taskLock struct {
	locks sync.Map
}

func newTaskLock() *taskLock {
	return &taskLock{}
}

func (l *taskLock) TryLock(id int) bool {
	_, loaded := l.locks.LoadOrStore(id, struct{}{})
	return !loaded
}

func (l *taskLock) Unlock(id int) {
	l.locks.Delete(id)
}
