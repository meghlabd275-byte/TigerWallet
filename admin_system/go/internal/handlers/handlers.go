package handlers

import (
	"net/http"

	"admin_system/internal/models"
	"admin_system/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuthHandler struct {
	authService *services.AuthService
}

func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	resp, err := h.authService.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	resp, err := h.authService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	userID, _ := c.Get("user_id")
	h.authService.Logout(c.Request.Context(), userID.(uuid.UUID))
	c.JSON(http.StatusOK, gin.H{"message": "Logged out"})
}

// System Handler
type SystemHandler struct {
	systemService *services.SystemService
}

func NewSystemHandler(systemService *services.SystemService) *SystemHandler {
	return &SystemHandler{systemService: systemService}
}

func (h *SystemHandler) GetInfo(c *gin.Context) {
	info, err := h.systemService.GetInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get system info"})
		return
	}
	c.JSON(http.StatusOK, info)
}

func (h *SystemHandler) GetStatus(c *gin.Context) {
	status, err := h.systemService.GetStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get system status"})
		return
	}
	c.JSON(http.StatusOK, status)
}

func (h *SystemHandler) Restart(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Restart initiated"})
}

func (h *SystemHandler) Shutdown(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "Shutdown initiated"})
}

// Monitoring Handler
type MonitoringHandler struct {
	monitoringService *services.MonitoringService
}

func NewMonitoringHandler(monitoringService *services.MonitoringService) *MonitoringHandler {
	return &MonitoringHandler{monitoringService: monitoringService}
}

func (h *MonitoringHandler) GetMetrics(c *gin.Context) {
	metrics, err := h.monitoringService.GetMetrics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get metrics"})
		return
	}
	c.JSON(http.StatusOK, metrics)
}

func (h *MonitoringHandler) GetResources(c *gin.Context) {
	resources, err := h.monitoringService.GetResources()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get resources"})
		return
	}
	c.JSON(http.StatusOK, resources)
}

func (h *MonitoringHandler) GetProcesses(c *gin.Context) {
	processes, err := h.monitoringService.GetProcesses()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get processes"})
		return
	}
	c.JSON(http.StatusOK, processes)
}

func (h *MonitoringHandler) GetNetworkStats(c *gin.Context) {
	stats, err := h.monitoringService.GetNetworkStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get network stats"})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// Config Handler
type ConfigHandler struct {
	configService *services.ConfigService
}

func NewConfigHandler(configService *services.ConfigService) *ConfigHandler {
	return &ConfigHandler{configService: configService}
}

func (h *ConfigHandler) GetAll(c *gin.Context) {
	configs, err := h.configService.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get configs"})
		return
	}
	c.JSON(http.StatusOK, configs)
}

func (h *ConfigHandler) Get(c *gin.Context) {
	key := c.Param("key")
	config, err := h.configService.Get(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Config not found"})
		return
	}
	c.JSON(http.StatusOK, config)
}

func (h *ConfigHandler) Update(c *gin.Context) {
	key := c.Param("key")
	var req struct {
		Value string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	userID, _ := c.Get("user_id")
	if err := h.configService.Update(c.Request.Context(), key, req.Value, userID.(uuid.UUID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update config"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Config updated"})
}

func (h *ConfigHandler) Delete(c *gin.Context) {
	key := c.Param("key")
	if err := h.configService.Delete(c.Request.Context(), key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete config"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Config deleted"})
}

// Backup Handler
type BackupHandler struct {
	backupService *services.BackupService
}

func NewBackupHandler(backupService *services.BackupService) *BackupHandler {
	return &BackupHandler{backupService: backupService}
}

func (h *BackupHandler) List(c *gin.Context) {
	var params models.PaginationParams
	c.ShouldBindQuery(&params)
	resp, err := h.backupService.List(c.Request.Context(), &params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list backups"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *BackupHandler) Create(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
		Type string `json:"type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	userID, _ := c.Get("user_id")
	backup, err := h.backupService.Create(c.Request.Context(), req.Name, req.Type, userID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create backup"})
		return
	}
	c.JSON(http.StatusCreated, backup)
}

func (h *BackupHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	backup, err := h.backupService.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Backup not found"})
		return
	}
	c.JSON(http.StatusOK, backup)
}

func (h *BackupHandler) Restore(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := h.backupService.Restore(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to restore backup"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Restore initiated"})
}

func (h *BackupHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := h.backupService.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete backup"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Backup deleted"})
}

// Log Handler
type LogHandler struct {
	logService *services.LogService
}

func NewLogHandler(logService *services.LogService) *LogHandler {
	return &LogHandler{logService: logService}
}

func (h *LogHandler) List(c *gin.Context) {
	var params models.PaginationParams
	c.ShouldBindQuery(&params)
	level := c.Query("level")
	resp, err := h.logService.List(c.Request.Context(), &params, level)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list logs"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *LogHandler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	log, err := h.logService.Get(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Log not found"})
		return
	}
	c.JSON(http.StatusOK, log)
}

func (h *LogHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	if err := h.logService.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete log"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Log deleted"})
}

func (h *LogHandler) DeleteOld(c *gin.Context) {
	var req struct {
		Days int `json:"days" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	if err := h.logService.DeleteOld(c.Request.Context(), req.Days); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete old logs"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Old logs deleted"})
}

// Metrics Handler
type MetricsHandler struct {
	metricsService *services.MetricsService
}

func NewMetricsHandler(metricsService *services.MetricsService) *MetricsHandler {
	return &MetricsHandler{metricsService: metricsService}
}

func (h *MetricsHandler) Get(c *gin.Context) {
	metrics, err := h.metricsService.Get(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get metrics"})
		return
	}
	c.JSON(http.StatusOK, metrics)
}

func (h *MetricsHandler) GetByType(c *gin.Context) {
	metricType := c.Param("type")
	metrics, err := h.metricsService.GetByType(c.Request.Context(), metricType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get metrics"})
		return
	}
	c.JSON(http.StatusOK, metrics)
}

// Alert Handler
type AlertHandler struct {
	alertService *services.AlertService
}

func NewAlertHandler(alertService *services.AlertService) *AlertHandler {
	return &AlertHandler{alertService: alertService}
}

func (h *AlertHandler) List(c *gin.Context) {
	var params models.PaginationParams
	c.ShouldBindQuery(&params)
	status := c.Query("status")
	resp, err := h.alertService.List(c.Request.Context(), &params, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list alerts"})
		return
	}
	c.JSON(http.StatusOK, resp)
}

func (h *AlertHandler) Create(c *gin.Context) {
	var alert models.SystemAlert
	if err := c.ShouldBindJSON(&alert); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	created, err := h.alertService.Create(c.Request.Context(), &alert)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create alert"})
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *AlertHandler) Acknowledge(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	userID, _ := c.Get("user_id")
	if err := h.alertService.Acknowledge(c.Request.Context(), id, userID.(uuid.UUID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to acknowledge alert"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Alert acknowledged"})
}

func (h *AlertHandler) Resolve(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	userID, _ := c.Get("user_id")
	if err := h.alertService.Resolve(c.Request.Context(), id, userID.(uuid.UUID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resolve alert"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Alert resolved"})
}

func GetAdminStats(c *gin.Context) {
	stats := map[string]interface{}{
		"total_admins":  0,
		"active_sessions": 0,
		"system_health": "healthy",
	}
	c.JSON(http.StatusOK, stats)
}
