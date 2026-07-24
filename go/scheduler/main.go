/**
 * TigerWallet Task Scheduler
 * High-Load Distributed Go Implementation
 */

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type Task struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Schedule  string    `json:"schedule"` // cron
	Endpoint  string    `json:"endpoint"`
	LastRun   int64     `json:"last_run"`
	NextRun   int64     `json:"next_run"`
	Status    string    `json:"status"`
	Retries   int       `json:"retries"`
}

type Execution struct {
	ID        string    `json:"id"`
	TaskID    string    `json:"task_id"`
	StartedAt int64     `json:"started_at"`
	EndedAt   int64     `json:"ended_at"`
	Status    string    `json:"status"`
	Output    string    `json:"output"`
	Error     string    `json:"error"`
}

type Scheduler struct {
	tasks      map[string]*Task
	executions []Execution
	mu         sync.RWMutex
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		tasks:      make(map[string]*Task),
		executions: make([]Execution, 0),
	}
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

	// Simulate task execution
	time.Sleep(100 * time.Millisecond)

	exec.Status = "completed"
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
		"status": "healthy",
		"tasks": len(s.tasks),
		"executions": len(s.executions),
	})
}

func main() {
	log.Println("Starting TigerWallet Task Scheduler...")
	NewScheduler().Run()
}
