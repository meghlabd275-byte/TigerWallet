/**
 * TigerWallet Task Scheduler
 * High-Load Distributed Go Implementation
 */

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Task struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"` // optional handler key for registered handlers
	Schedule string `json:"schedule"` // cron
	Endpoint string `json:"endpoint"` // optional URL to invoke
	LastRun  int64  `json:"last_run"`
	NextRun  int64  `json:"next_run"`
	Status   string `json:"status"`
	Retries  int    `json:"retries"`
}

type Execution struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	StartedAt int64  `json:"started_at"`
	EndedAt   int64  `json:"ended_at"`
	Status    string `json:"status"`
	Output    string `json:"output"`
	Error     string `json:"error"`
}

type Scheduler struct {
	tasks      map[string]*Task
	executions []Execution
	handlers   map[string]func(*Task) (string, error)
	mu         sync.RWMutex
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		tasks:      make(map[string]*Task),
		executions: make([]Execution, 0),
		handlers:   make(map[string]func(*Task) (string, error)),
	}
}

// RegisterHandler associates a task type with the real handler invoked when
// the scheduler runs a task of that type. Handlers return their output and
// any error; the execution status reflects the real result.
func (s *Scheduler) RegisterHandler(taskType string, fn func(*Task) (string, error)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[taskType] = fn
}

func (s *Scheduler) Run() error {
	go s.runScheduler()

	mux := http.NewServeMux()
	mux.HandleFunc("/tasks", s.handleTasks)
	mux.HandleFunc("/executions", s.handleExecutions)
	mux.HandleFunc("/health", s.handleHealth)

	log.Println("Scheduler starting on :8095")
	return http.ListenAndServe(":8095", mux)
}

func (s *Scheduler) runScheduler() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().UnixMilli()

		s.mu.Lock()
		for _, task := range s.tasks {
			if task.Status == "active" && now >= task.NextRun {
				go s.executeTask(task)
				task.LastRun = now
				task.NextRun = now + 60000 // Next run in 1 min
			}
		}
		s.mu.Unlock()
	}
}

func (s *Scheduler) executeTask(task *Task) {
	exec := Execution{
		ID:        fmt.Sprintf("exec_%d", time.Now().UnixNano()),
		TaskID:    task.ID,
		StartedAt: time.Now().UnixMilli(),
		Status:    "running",
	}

	s.mu.Lock()
	s.executions = append(s.executions, exec)
	s.mu.Unlock()

	// Execute the task for real: prefer a registered handler keyed by task
	// type; otherwise, if the task carries an endpoint URL, invoke it via
	// HTTP. If neither is available the execution is marked "no_handler"
	// rather than pretending to succeed.
	output, execErr, status := s.runTask(task)

	exec.Status = status
	exec.Output = output
	if execErr != nil {
		exec.Error = execErr.Error()
	}
	exec.EndedAt = time.Now().UnixMilli()

	s.mu.Lock()
	for i := range s.executions {
		if s.executions[i].ID == exec.ID {
			s.executions[i] = exec
			break
		}
	}
	s.mu.Unlock()
}

// runTask performs the real work for a task and returns (output, err, status).
func (s *Scheduler) runTask(task *Task) (string, error, string) {
	// Registered handler takes precedence.
	s.mu.RLock()
	handler, hasHandler := s.handlers[task.Type]
	s.mu.RUnlock()
	if hasHandler {
		out, err := handler(task)
		if err != nil {
			return out, err, "failed"
		}
		return out, nil, "completed"
	}

	// Fall back to invoking the task's endpoint URL, if any.
	if strings.TrimSpace(task.Endpoint) != "" {
		return s.invokeEndpoint(task)
	}

	// Nothing to do: record honestly rather than faking success.
	return "", nil, "no_handler"
}

// invokeEndpoint POSTs the task JSON to the configured endpoint URL and
// reports the real outcome.
func (s *Scheduler) invokeEndpoint(task *Task) (string, error, string) {
	body, err := json.Marshal(task)
	if err != nil {
		return "", err, "failed"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, task.Endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err, "failed"
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err, "failed"
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	out := strings.TrimSpace(string(respBody))
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return out, nil, "completed"
	}
	return out, fmt.Errorf("endpoint returned status %d", resp.StatusCode), "failed"
}

func (s *Scheduler) handleTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var task Task
		json.NewDecoder(r.Body).Decode(&task)
		task.ID = fmt.Sprintf("task_%d", time.Now().UnixNano())
		task.Status = "active"
		task.NextRun = time.Now().UnixMilli()

		s.mu.Lock()
		s.tasks[task.ID] = &task
		s.mu.Unlock()

		json.NewEncoder(w).Encode(task)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	json.NewEncoder(w).Encode(s.tasks)
}

func (s *Scheduler) handleExecutions(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	json.NewEncoder(w).Encode(s.executions)
}

func (s *Scheduler) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "healthy",
		"tasks":      len(s.tasks),
		"executions": len(s.executions),
	})
}

func main() {
	log.Println("Starting TigerWallet Task Scheduler...")
	NewScheduler().Run()
}
