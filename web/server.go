package web

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

//go:embed static/index.html
var indexHTMLFS embed.FS

// AdminServer exposes a minimal Layui based UI for viewing and creating tasks.
type AdminServer struct {
	db     *gorm.DB
	logger *zap.SugaredLogger
}

func NewAdminServer(db *gorm.DB, logger *zap.SugaredLogger) *AdminServer {
	return &AdminServer{db: db, logger: logger}
}

func (s *AdminServer) Start(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/api/tasks", s.handleTasks)

	s.logger.Infof("Layui admin listening on %s", addr)
	server := &http.Server{Addr: addr, Handler: s.logMiddleware(mux)}
	return server.ListenAndServe()
}

func (s *AdminServer) logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Infow("http request",
			"method", r.Method,
			"path", r.URL.Path,
			"duration", time.Since(start).String(),
		)
	})
}

func (s *AdminServer) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data, err := indexHTMLFS.ReadFile("static/index.html")
	if err != nil {
		s.writeJSONError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}

func (s *AdminServer) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleListTasks(w, r)
	case http.MethodPost:
		s.handleCreateTask(w, r)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *AdminServer) handleListTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.listTasks(r.Context())
	if err != nil {
		s.writeJSONError(w, http.StatusInternalServerError, err)
		return
	}
	resp := struct {
		Code  int         `json:"code"`
		Msg   string      `json:"msg"`
		Count int         `json:"count"`
		Data  []taskEntry `json:"data"`
	}{
		Code:  0,
		Msg:   "",
		Count: len(tasks),
		Data:  tasks,
	}
	s.writeJSON(w, http.StatusOK, resp)
}

func (s *AdminServer) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeJSONError(w, http.StatusBadRequest, err)
		return
	}
	if err := req.Validate(); err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]interface{}{
			"code": 1,
			"msg":  err.Error(),
		})
		return
	}
	entry, err := s.createTask(r.Context(), &req)
	if err != nil {
		s.writeJSONError(w, http.StatusInternalServerError, err)
		return
	}
	s.writeJSON(w, http.StatusCreated, map[string]interface{}{
		"code": 0,
		"msg":  "任务已创建",
		"data": entry,
	})
}

func (s *AdminServer) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		s.logger.Errorf("write json error: %v", err)
	}
}

func (s *AdminServer) writeJSONError(w http.ResponseWriter, status int, err error) {
	s.logger.Error(err)
	msg := err.Error()
	if status == http.StatusInternalServerError {
		msg = "内部错误"
	}
	s.writeJSON(w, status, map[string]interface{}{
		"code": 1,
		"msg":  msg,
	})
}

type taskRecord struct {
	ID             int        `gorm:"column:id"`
	Name           string     `gorm:"column:name"`
	Task           string     `gorm:"column:task"`
	RunTime        string     `gorm:"column:run_time"`
	RunTimeRegular string     `gorm:"column:run_time_regular"`
	DataCountLimit int        `gorm:"column:data_count_limit"`
	RunSleepMicro  int        `gorm:"column:run_sleep_micro_second"`
	Status         int        `gorm:"column:status"`
	IsEnable       int        `gorm:"column:is_enable"`
	Description    string     `gorm:"column:description"`
	TryTimesLimit  int        `gorm:"column:try_times_limit"`
	RunWay         int        `gorm:"column:runWay"`
	LastStartTime  *time.Time `gorm:"column:last_start_time"`
}

func (taskRecord) TableName() string {
	return "tasks"
}

type taskEntry struct {
	ID             int    `json:"id"`
	Name           string `json:"name"`
	Command        string `json:"command"`
	RunExpr        string `json:"run_expr"`
	Status         int    `json:"status"`
	IsEnable       int    `json:"is_enable"`
	DataCountLimit int    `json:"data_count_limit"`
	RunSleepMicro  int    `json:"run_sleep_micro_second"`
	TryTimesLimit  int    `json:"try_times_limit"`
	RunWay         int    `json:"run_way"`
	Description    string `json:"description"`
	LastStartTime  string `json:"last_start_time"`
}

func (s *AdminServer) listTasks(ctx context.Context) ([]taskEntry, error) {
	var records []taskRecord
	if err := s.db.WithContext(ctx).Order("id DESC").Find(&records).Error; err != nil {
		return nil, err
	}
	entries := make([]taskEntry, 0, len(records))
	for _, r := range records {
		entries = append(entries, toTaskEntry(r))
	}
	return entries, nil
}

func toTaskEntry(r taskRecord) taskEntry {
	entry := taskEntry{
		ID:             r.ID,
		Name:           r.Name,
		Command:        r.Task,
		RunExpr:        r.RunTimeRegular,
		Status:         r.Status,
		IsEnable:       r.IsEnable,
		DataCountLimit: r.DataCountLimit,
		RunSleepMicro:  r.RunSleepMicro,
		TryTimesLimit:  r.TryTimesLimit,
		RunWay:         r.RunWay,
		Description:    r.Description,
	}
	if r.LastStartTime != nil && !r.LastStartTime.IsZero() {
		entry.LastStartTime = r.LastStartTime.Format("2006-01-02 15:04:05")
	}
	return entry
}

type createTaskRequest struct {
	Name           string `json:"name"`
	Command        string `json:"command"`
	RunExpr        string `json:"run_expr"`
	DataCountLimit int    `json:"data_count_limit"`
	RunSleepMicro  int    `json:"run_sleep_micro_second"`
	TryTimesLimit  int    `json:"try_times_limit"`
	RunWay         int    `json:"run_way"`
	Description    string `json:"description"`
}

func (r *createTaskRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return errors.New("任务名称不能为空")
	}
	if strings.TrimSpace(r.Command) == "" {
		return errors.New("Command 不能为空")
	}
	if strings.TrimSpace(r.RunExpr) == "" {
		return errors.New("Cron 表达式不能为空")
	}
	return nil
}

func (s *AdminServer) createTask(ctx context.Context, req *createTaskRequest) (taskEntry, error) {
	record := taskRecord{
		Name:           strings.TrimSpace(req.Name),
		Task:           strings.TrimSpace(req.Command),
		RunTimeRegular: strings.TrimSpace(req.RunExpr),
		RunTime:        strings.TrimSpace(req.RunExpr),
		DataCountLimit: req.DataCountLimit,
		RunSleepMicro:  req.RunSleepMicro,
		TryTimesLimit:  req.TryTimesLimit,
		RunWay:         req.RunWay,
		Description:    strings.TrimSpace(req.Description),
		Status:         0,
		IsEnable:       0,
	}
	if err := s.db.WithContext(ctx).Create(&record).Error; err != nil {
		return taskEntry{}, fmt.Errorf("create task: %w", err)
	}
	return toTaskEntry(record), nil
}
