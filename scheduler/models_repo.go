package scheduler

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type ModelsRepo struct {
	DB *gorm.DB
}

func NewModelsRepo(db *gorm.DB) *ModelsRepo {
	return &ModelsRepo{DB: db}
}

type taskModel struct {
	ID             int       `gorm:"column:id"`
	Name           string    `gorm:"column:name"`
	Task           string    `gorm:"column:task"`
	LastStartTime  time.Time `gorm:"column:last_start_time"`
	RunSleepMicro  int       `gorm:"column:run_sleep_micro_second"`
	TryTimesLimit  int       `gorm:"column:try_times_limit"`
	DataCountLimit int       `gorm:"column:data_count_limit"`
	Status         int       `gorm:"column:status"`
	IsEnable       int       `gorm:"column:is_enable"`
	Description    string    `gorm:"column:description"`
	ServerID       int       `gorm:"column:server_id"`
	HaveRunning    int       `gorm:"column:have_running"`
	RunTime        string    `gorm:"column:run_time"`
	RunTimeRegular string    `gorm:"column:run_time_regular"`
	RunWay         int       `gorm:"column:runWay"`
	IsEnabledNew   int       `gorm:"column:is_enabled_new"`
}

func (taskModel) TableName() string {
	return "tasks"
}

func (r *ModelsRepo) LoadEnabledTasks(ctx context.Context) ([]Task, error) {
	var models []taskModel
	if err := r.DB.WithContext(ctx).Where("is_enable = ?", 0).Find(&models).Error; err != nil {
		return nil, err
	}
	tasks := make([]Task, 0, len(models))
	for _, m := range models {
		tasks = append(tasks, Task{
			ID:             m.ID,
			Name:           m.Name,
			Command:        m.Task,
			RunExpr:        m.RunTimeRegular,
			IsEnable:       m.IsEnable,
			Status:         m.Status,
			DataCountLimit: m.DataCountLimit,
			RunSleepMicro:  m.RunSleepMicro,
			TryTimesLimit:  m.TryTimesLimit,
			RunWay:         m.RunWay,
			LastStartTime:  m.LastStartTime,
		})
	}
	return tasks, nil
}

func (r *ModelsRepo) MarkRunning(ctx context.Context, id int, t time.Time) error {
	return r.DB.WithContext(ctx).Table("tasks").Where("id = ?", id).Updates(map[string]interface{}{
		"status":          1,
		"last_start_time": t,
	}).Error
}

func (r *ModelsRepo) MarkStatus(ctx context.Context, id int, status int) error {
	return r.DB.WithContext(ctx).Table("tasks").Where("id = ?", id).Update("status", status).Error
}
