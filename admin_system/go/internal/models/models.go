package models

import (
	"time"

	"github.com/google/uuid"
)

type SystemUser struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	Username     string     `json:"username"`
	PasswordHash string     `json:"-"`
	FirstName    *string    `json:"first_name,omitempty"`
	LastName     *string    `json:"last_name,omitempty"`
	Role         string     `json:"role"`
	Status       string     `json:"status"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type SystemConfig struct {
	Key          string     `json:"key"`
	Value        string     `json:"value"`
	ValueType    string     `json:"value_type"`
	Description  *string    `json:"description,omitempty"`
	IsSecret     bool       `json:"is_secret"`
	UpdatedBy    *uuid.UUID `json:"updated_by,omitempty"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type SystemBackup struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	FilePath    *string    `json:"file_path,omitempty"`
	FileSize    *int64     `json:"file_size,omitempty"`
	Status      string     `json:"status"`
	CreatedBy   *uuid.UUID `json:"created_by,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

type SystemLog struct {
	ID         uuid.UUID              `json:"id"`
	Level      string                 `json:"level"`
	Message    string                 `json:"message"`
	Component  *string               `json:"component,omitempty"`
	UserID     *uuid.UUID            `json:"user_id,omitempty"`
	IPAddress  *string               `json:"ip_address,omitempty"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  time.Time              `json:"created_at"`
}

type SystemMetric struct {
	ID          uuid.UUID              `json:"id"`
	MetricType  string                 `json:"metric_type"`
	MetricName  string                 `json:"metric_name"`
	Value       float64               `json:"value"`
	Unit        *string               `json:"unit,omitempty"`
	Tags        map[string]interface{} `json:"tags"`
	RecordedAt  time.Time              `json:"recorded_at"`
}

type SystemAlert struct {
	ID              uuid.UUID  `json:"id"`
	Title           string     `json:"title"`
	Message         string     `json:"message"`
	Severity        string     `json:"severity"`
	Status          string     `json:"status"`
	Source          *string    `json:"source,omitempty"`
	AcknowledgedBy  *uuid.UUID `json:"acknowledged_by,omitempty"`
	AcknowledgedAt  *time.Time `json:"acknowledged_at,omitempty"`
	ResolvedBy      *uuid.UUID `json:"resolved_by,omitempty"`
	ResolvedAt      *time.Time `json:"resolved_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

type MonitoringData struct {
	ID            uuid.UUID  `json:"id"`
	ResourceType  string     `json:"resource_type"`
	MetricName    string     `json:"metric_name"`
	Value         float64    `json:"value"`
	Unit          *string    `json:"unit,omitempty"`
	Hostname      *string    `json:"hostname,omitempty"`
	RecordedAt    time.Time  `json:"recorded_at"`
}

type NetworkStats struct {
	ID              uuid.UUID  `json:"id"`
	InterfaceName   *string    `json:"interface_name,omitempty"`
	BytesSent       *int64     `json:"bytes_sent,omitempty"`
	BytesReceived   *int64     `json:"bytes_received,omitempty"`
	PacketsSent     *int64     `json:"packets_sent,omitempty"`
	PacketsReceived *int64     `json:"packets_received,omitempty"`
	ErrorsIn        *int64     `json:"errors_in,omitempty"`
	ErrorsOut       *int64     `json:"errors_out,omitempty"`
	RecordedAt      time.Time  `json:"recorded_at"`
}

type ProcessInfo struct {
	ID            uuid.UUID  `json:"id"`
	PID           int        `json:"pid"`
	Name          string     `json:"name"`
	User          *string    `json:"user,omitempty"`
	CPUPercent    *float64   `json:"cpu_percent,omitempty"`
	MemoryPercent *float64   `json:"memory_percent,omitempty"`
	Status        *string    `json:"status,omitempty"`
	StartedAt     *time.Time `json:"started_at,omitempty"`
	RecordedAt    time.Time  `json:"recorded_at"`
}

// Request/Response DTOs
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int64        `json:"expires_in"`
	User         *SystemUser  `json:"user"`
}

type SystemInfo struct {
	Hostname     string    `json:"hostname"`
	OS           string    `json:"os"`
	Arch         string    `json:"arch"`
	CPUCores     int       `json:"cpu_cores"`
	MemoryTotal  uint64    `json:"memory_total"`
	Uptime       uint64    `json:"uptime"`
	LoadAverage  []float64 `json:"load_average"`
}

type SystemStatus struct {
	Status      string            `json:"status"`
	CPU         float64           `json:"cpu"`
	Memory      float64           `json:"memory"`
	Disk        float64           `json:"disk"`
	Network     map[string]int64  `json:"network"`
	Processes   int               `json:"processes"`
	Connections int               `json:"connections"`
}

type PaginationParams struct {
	Page     int `form:"page" binding:"omitempty,min=1"`
	PageSize int `form:"page_size" binding:"omitempty,min=1,max=100"`
}

type PaginatedResponse struct {
	Total       int         `json:"total"`
	Page        int         `json:"page"`
	PageSize    int         `json:"page_size"`
	TotalPages  int         `json:"total_pages"`
	Data        interface{} `json:"data"`
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
