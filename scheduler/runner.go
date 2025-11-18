package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

type TaskRunner interface {
	Run(ctx context.Context, task Task) error
}

type HTTPTaskRunner struct {
	BaseURL string
	Client  *http.Client
	Logger  *zap.SugaredLogger
}

func NewHTTPTaskRunner(baseURL string, client *http.Client, logger *zap.SugaredLogger) *HTTPTaskRunner {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	return &HTTPTaskRunner{
		BaseURL: strings.TrimRight(baseURL, "/"),
		Client:  client,
		Logger:  logger,
	}
}

func (r *HTTPTaskRunner) Run(ctx context.Context, task Task) error {
	if r.BaseURL == "" {
		return fmt.Errorf("base URL is empty")
	}
	limit := task.DataCountLimit
	if limit <= 0 {
		limit = 50
	}
	sleepDuration := time.Duration(task.RunSleepMicro) * time.Microsecond
	lastID := 0
	for {
		rows, err := r.fetchData(ctx, task.Command, limit, lastID)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
		minID := 0
		for _, row := range rows {
			if err := r.sendUpdate(ctx, task.Command, row); err != nil {
				r.Logger.Warnf("update failed for task %s: %v", task.Command, err)
			}
			if id, ok := extractID(row); ok {
				if minID == 0 || id < minID {
					minID = id
				}
			}
			if sleepDuration > 0 {
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(sleepDuration):
				}
			}
		}
		if minID == 0 {
			return nil
		}
		lastID = minID
	}
}

type getDataResponse struct {
	Status  string                   `json:"status"`
	Message string                   `json:"message"`
	Data    []map[string]interface{} `json:"data"`
}

func (r *HTTPTaskRunner) fetchData(ctx context.Context, command string, limit, lastID int) ([]map[string]interface{}, error) {
	endpoint := fmt.Sprintf("%s/timer/getdata/command/%s/limit/%d/last_id/%d", r.BaseURL, url.PathEscape(command), limit, lastID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	resp, err := r.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("getdata request failed: %s", resp.Status)
	}
	var gd getDataResponse
	if err := json.Unmarshal(body, &gd); err != nil {
		return nil, err
	}
	if strings.ToUpper(gd.Status) != "SUCCESS" {
		return nil, fmt.Errorf("getdata failed: %s", gd.Message)
	}
	return gd.Data, nil
}

func (r *HTTPTaskRunner) sendUpdate(ctx context.Context, command string, payload map[string]interface{}) error {
	dataBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	encoded := url.QueryEscape(string(dataBytes))
	endpoint := fmt.Sprintf("%s/timer/update/command/%s/data/%s", r.BaseURL, url.PathEscape(command), encoded)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	resp, err := r.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update request failed: %s %s", resp.Status, string(body))
	}
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return err
	}
	if status, _ := result["status"].(string); strings.ToUpper(status) != "SUCCESS" {
		return fmt.Errorf("update failed: %v", result["message"])
	}
	return nil
}

func extractID(row map[string]interface{}) (int, bool) {
	if row == nil {
		return 0, false
	}
	switch v := row["id"].(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case int64:
		return int(v), true
	case json.Number:
		i, err := v.Int64()
		if err != nil {
			return 0, false
		}
		return int(i), true
	case string:
		val, err := strconv.Atoi(v)
		if err != nil {
			return 0, false
		}
		return val, true
	default:
		return 0, false
	}
}
