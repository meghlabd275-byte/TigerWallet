package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/wl-user-wallet/internal/middleware"
)

// GET /devices — list the caller's registered devices (real PG).
func (s *Svc) ListDevices(c *gin.Context) {
	devs, err := s.store.ListDevices(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"devices": devs})
}

// POST /devices — register a new device (real PG insert).
func (s *Svc) RegisterDevice(c *gin.Context) {
	var req struct {
		Name string `json:"name" binding:"required"`
		Type string `json:"type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := s.store.RegisterDevice(c.Request.Context(), middleware.UserID(c), req.Name, req.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name, "type": req.Type, "status": "offline"})
}

// POST /devices/:id/sync — mark a device synced/online (real PG update).
func (s *Svc) SyncDevice(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	rows, err := s.store.SyncDevice(c.Request.Context(), middleware.UserID(c), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "status": "online", "synced": rows})
}

// DELETE /devices/:id — unregister a device (real PG delete).
func (s *Svc) DeleteDevice(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	rows, err := s.store.DeleteDevice(c.Request.Context(), middleware.UserID(c), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "device not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "deleted": rows})
}
