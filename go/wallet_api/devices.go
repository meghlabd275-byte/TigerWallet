package main

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// DeviceRecord is a connected device entry for multi-device sync.
type DeviceRecord struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	LastSync  int64  `json:"lastSync"`
	CreatedAt int64  `json:"createdAt"`
}

// handleListDevices returns the authenticated user's connected devices.
func handleListDevices(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, name, device_type, status, COALESCE(extract(epoch from last_sync)::bigint, 0),
		        extract(epoch from created_at)::bigint
		 FROM devices WHERE user_id=$1 ORDER BY created_at DESC`, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []DeviceRecord{}
	for rows.Next() {
		var r DeviceRecord
		if err := rows.Scan(&r.ID, &r.Name, &r.Type, &r.Status, &r.LastSync, &r.CreatedAt); err != nil {
			continue
		}
		out = append(out, r)
	}
	c.JSON(http.StatusOK, gin.H{"devices": out, "count": len(out)})
}

// handleRegisterDevice registers a new device for the authenticated user.
func handleRegisterDevice(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var req struct {
		Name string `json:"name" binding:"required"`
		Type string `json:"type" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name and type required"})
		return
	}
	id := uuid.New()
	_, err = store.PG.Exec(c.Request.Context(),
		`INSERT INTO devices (id, user_id, name, device_type, status) VALUES ($1, $2, $3, $4, 'offline')`,
		id, userUUID, req.Name, req.Type)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "insert failed"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name, "type": req.Type, "status": "offline"})
}

// handleSyncDevice marks a device as synced + online. Real multi-device sync
// (encrypted state transfer) is handled by the multi_device_sync library; this
// endpoint records the sync event + timestamp in PostgreSQL.
func handleSyncDevice(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	deviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device id"})
		return
	}
	now := time.Now()
	_, err = store.PG.Exec(c.Request.Context(),
		`UPDATE devices SET status='online', last_sync=$1 WHERE id=$2 AND user_id=$3`,
		now, deviceID, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "sync failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": deviceID, "status": "online", "last_sync": now.Unix()})
}

// handleDeleteDevice removes a device from the user's device list.
func handleDeleteDevice(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	deviceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid device id"})
		return
	}
	_, err = store.PG.Exec(c.Request.Context(),
		`DELETE FROM devices WHERE id=$1 AND user_id=$2`, deviceID, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": deviceID})
}
