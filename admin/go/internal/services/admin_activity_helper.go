package services

import (
	"encoding/json"

	"github.com/tigerwallet/admin/internal/models"
	"github.com/tigerwallet/admin/pkg/database"
)

// logAdminActivity records an admin activity audit entry. It mirrors the
// helper of the same name defined in the handlers package so that service
// code can audit privileged operations through the shared PostgresDB wrapper.
func logAdminActivity(db *database.PostgresDB, adminID uint, action, resource, resourceID, details, ip, userAgent string) {
	detailsJSON, _ := json.Marshal(details)
	activity := models.AdminActivity{
		AdminID:    adminID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Details:    detailsJSON,
		IPAddress:  ip,
		UserAgent:  userAgent,
		Status:     "success",
	}
	db.Create(&activity)
}
