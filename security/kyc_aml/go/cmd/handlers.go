// HTTP handlers for the KYC/AML service.
//
// Every handler reads and writes PostgreSQL through Store. There is no fixture
// path: unknown records produce 404, a screening that could not run is reported
// as failed, and an application only reaches "approved" through an explicit
// reviewer decision.

package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// submitKYCHandler accepts a new application and records it as pending.
func submitKYCHandler(c *gin.Context) {
	var req struct {
		UserID             string `json:"user_id" binding:"required"`
		TenantID           string `json:"tenant_id"`
		Level              string `json:"level" binding:"required"`
		FirstName          string `json:"first_name" binding:"required"`
		LastName           string `json:"last_name" binding:"required"`
		DateOfBirth        string `json:"date_of_birth" binding:"required"`
		Nationality        string `json:"nationality" binding:"required"`
		CountryOfResidence string `json:"country_of_residence" binding:"required"`
		Address            string `json:"address" binding:"required"`
		City               string `json:"city" binding:"required"`
		State              string `json:"state"`
		PostalCode         string `json:"postal_code"`
		PhoneNumber        string `json:"phone_number"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	var tenantID *uuid.UUID
	if req.TenantID != "" {
		parsed, err := uuid.Parse(req.TenantID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tenant_id"})
			return
		}
		tenantID = &parsed
	}

	if !validKYCLevel(req.Level) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "level must be one of basic, advanced, institutional"})
		return
	}

	now := time.Now()
	application := &KYCApplication{
		ID:                 uuid.New(),
		UserID:             userID,
		TenantID:           tenantID,
		Level:              req.Level,
		Status:             "pending",
		FirstName:          strings.TrimSpace(req.FirstName),
		LastName:           strings.TrimSpace(req.LastName),
		DateOfBirth:        req.DateOfBirth,
		Nationality:        req.Nationality,
		CountryOfResidence: req.CountryOfResidence,
		Address:            req.Address,
		City:               req.City,
		State:              req.State,
		PostalCode:         req.PostalCode,
		PhoneNumber:        req.PhoneNumber,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := store.CreateApplication(c.Request.Context(), application); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not record the application"})
		return
	}

	store.LogAudit(c.Request.Context(), "kyc_application", &application.ID, "submitted", userID.String(), nil)

	c.JSON(http.StatusCreated, gin.H{
		"application": application,
		"message":     "KYC application submitted successfully",
	})
}

// getKYCStatusHandler returns the current status of a user's latest application.
func getKYCStatusHandler(c *gin.Context) {
	uid, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	application, err := store.GetLatestApplicationForUser(c.Request.Context(), uid)
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusOK, gin.H{
			"user_id": uid.String(),
			"status":  "not_started",
		})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not read the application"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user_id":        uid.String(),
		"application_id": application.ID.String(),
		"status":         application.Status,
		"level":          application.Level,
		"reviewed_at":    application.ReviewedAt,
		"updated_at":     application.UpdatedAt,
	})
}

// listApplicationsHandler lists applications filtered by status.
func listApplicationsHandler(c *gin.Context) {
	status := c.Query("status")
	if status != "" && !validKYCStatus(status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown status filter"})
		return
	}

	limit := parseIntQuery(c, "limit", 50)
	offset := parseIntQuery(c, "offset", 0)

	applications, total, err := store.ListApplications(c.Request.Context(), status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list applications"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"applications": applications,
		"total":        total,
		"limit":        limit,
		"offset":       offset,
	})
}

// getApplicationHandler returns one application by id.
func getApplicationHandler(c *gin.Context) {
	aid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid application ID"})
		return
	}

	application, err := store.GetApplication(c.Request.Context(), aid)
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not read the application"})
		return
	}

	c.JSON(http.StatusOK, application)
}

// reviewApplicationHandler moves an application into review.
func reviewApplicationHandler(c *gin.Context) {
	aid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid application ID"})
		return
	}

	var req struct {
		ReviewerID string `json:"reviewer_id" binding:"required"`
		Notes      string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reviewerID, err := uuid.Parse(req.ReviewerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reviewer_id"})
		return
	}

	application, err := store.UpdateApplicationStatus(c.Request.Context(), aid, "in_review", &reviewerID, nil)
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update the application"})
		return
	}

	store.LogAudit(c.Request.Context(), "kyc_application", &aid, "review_started", reviewerID.String(),
		map[string]interface{}{"notes": req.Notes})

	c.JSON(http.StatusOK, gin.H{"application": application, "message": "application marked for review"})
}

// approveApplicationHandler records an approval decision.
func approveApplicationHandler(c *gin.Context) {
	aid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid application ID"})
		return
	}

	var req struct {
		ReviewerID string `json:"reviewer_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reviewerID, err := uuid.Parse(req.ReviewerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reviewer_id"})
		return
	}

	// An application with a failed or missing AML screening cannot be approved.
	existing, err := store.GetApplication(c.Request.Context(), aid)
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not read the application"})
		return
	}
	if existing.AMLCheckPassed == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "the application has no completed AML screening"})
		return
	}
	if !*existing.AMLCheckPassed {
		c.JSON(http.StatusConflict, gin.H{"error": "the application did not pass AML screening"})
		return
	}

	application, err := store.UpdateApplicationStatus(c.Request.Context(), aid, "approved", &reviewerID, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not approve the application"})
		return
	}

	store.LogAudit(c.Request.Context(), "kyc_application", &aid, "approved", reviewerID.String(), nil)

	c.JSON(http.StatusOK, gin.H{"application": application, "message": "application approved", "status": "approved"})
}

// rejectApplicationHandler records a rejection with a reason.
func rejectApplicationHandler(c *gin.Context) {
	aid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid application ID"})
		return
	}

	var req struct {
		ReviewerID string `json:"reviewer_id" binding:"required"`
		Reason     string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reviewerID, err := uuid.Parse(req.ReviewerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reviewer_id"})
		return
	}

	application, err := store.UpdateApplicationStatus(c.Request.Context(), aid, "rejected", &reviewerID, &req.Reason)
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not reject the application"})
		return
	}

	store.LogAudit(c.Request.Context(), "kyc_application", &aid, "rejected", reviewerID.String(),
		map[string]interface{}{"reason": req.Reason})

	c.JSON(http.StatusOK, gin.H{"application": application, "message": "application rejected", "status": "rejected"})
}

// requestInfoHandler asks the applicant for more information.
func requestInfoHandler(c *gin.Context) {
	aid, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid application ID"})
		return
	}

	var req struct {
		ReviewerID string `json:"reviewer_id" binding:"required"`
		Message    string `json:"message" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reviewerID, err := uuid.Parse(req.ReviewerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reviewer_id"})
		return
	}

	application, err := store.UpdateApplicationStatus(c.Request.Context(), aid, "needs_more_info", &reviewerID, &req.Message)
	if errors.Is(err, ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not update the application"})
		return
	}

	store.LogAudit(c.Request.Context(), "kyc_application", &aid, "info_requested", reviewerID.String(),
		map[string]interface{}{"message": req.Message})

	c.JSON(http.StatusOK, gin.H{"application": application, "message": "additional information requested", "status": "needs_more_info"})
}

// uploadDocumentHandler records an uploaded document.
func uploadDocumentHandler(c *gin.Context) {
	var req struct {
		ApplicationID string `json:"application_id" binding:"required"`
		Type          string `json:"type" binding:"required"`
		Side          string `json:"side" binding:"required"`
		FileData      string `json:"file_data" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	applicationID, err := uuid.Parse(req.ApplicationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid application_id"})
		return
	}

	if _, err := store.GetApplication(c.Request.Context(), applicationID); err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "application not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not read the application"})
		return
	}

	hash, err := hashDocumentData(req.FileData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file_data must be valid base64"})
		return
	}

	url, err := storeDocumentFile(c.Request.Context(), applicationID, req.Type, req.Side, req.FileData)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "document storage is unavailable: " + err.Error()})
		return
	}

	document := &KYCDocument{
		ID:            uuid.New(),
		ApplicationID: applicationID,
		Type:          req.Type,
		Side:          req.Side,
		FileURL:       url,
		FileHash:      hash,
		Status:        "pending",
		CreatedAt:     time.Now(),
	}

	if err := store.CreateDocument(c.Request.Context(), document); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not record the document"})
		return
	}

	store.LogAudit(c.Request.Context(), "kyc_document", &document.ID, "uploaded", "", nil)

	c.JSON(http.StatusCreated, gin.H{
		"document": document,
		"message":  "document uploaded successfully",
	})
}

// getDocumentsHandler lists the documents for an application.
func getDocumentsHandler(c *gin.Context) {
	applicationID, err := uuid.Parse(c.Param("application_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid application_id"})
		return
	}

	documents, err := store.ListDocuments(c.Request.Context(), applicationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list documents"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"application_id": applicationID.String(), "documents": documents})
}

// deleteDocumentHandler removes a document.
func deleteDocumentHandler(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid document ID"})
		return
	}

	if err := store.DeleteDocument(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "document not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not delete the document"})
		return
	}

	store.LogAudit(c.Request.Context(), "kyc_document", &id, "deleted", "", nil)
	c.JSON(http.StatusOK, gin.H{"message": "document deleted"})
}

// runAMLChecksHandler screens an applicant and stores the result.
func runAMLChecksHandler(c *gin.Context) {
	var req struct {
		UserID        string `json:"user_id" binding:"required"`
		ApplicationID string `json:"application_id" binding:"required"`
		FirstName     string `json:"first_name" binding:"required"`
		LastName      string `json:"last_name" binding:"required"`
		Country       string `json:"country" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, err := uuid.Parse(req.UserID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}
	applicationID, err := uuid.Parse(req.ApplicationID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid application_id"})
		return
	}

	check, err := screener.runAMLCheck(c.Request.Context(), userID, applicationID, req.FirstName, req.LastName, req.Country)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not record the AML check"})
		return
	}

	store.LogAudit(c.Request.Context(), "aml_check", &check.ID, "screened", userID.String(),
		map[string]interface{}{"status": check.Status, "risk_score": check.RiskScore})

	message := "AML check completed"
	if check.Status != "completed" {
		message = "AML check could not be completed; the applicant is not cleared"
	}

	c.JSON(http.StatusOK, gin.H{"aml_check": check, "message": message})
}

// getAMLHistoryHandler lists the AML history for a user.
func getAMLHistoryHandler(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("user_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user_id"})
		return
	}

	history, err := store.ListAMLChecks(c.Request.Context(), userID, parseIntQuery(c, "limit", 50))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not list AML checks"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"user_id": userID.String(), "history": history})
}

// addToWatchlistHandler stores a watchlist record.
func addToWatchlistHandler(c *gin.Context) {
	var req struct {
		Name       string                 `json:"name" binding:"required"`
		EntityType string                 `json:"entity_type" binding:"required"`
		Country    string                 `json:"country"`
		ListType   string                 `json:"list_type" binding:"required"`
		RiskLevel  string                 `json:"risk_level" binding:"required"`
		Details    map[string]interface{} `json:"details"`
		AddedBy    string                 `json:"added_by"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !validWatchlistType(req.ListType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "list_type must be one of sanctions, pep, adverse_media, internal"})
		return
	}
	if !validRiskLevel(req.RiskLevel) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "risk_level must be one of low, medium, high"})
		return
	}

	now := time.Now()
	var addedBy uuid.UUID
	if raw := strings.TrimSpace(req.AddedBy); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid added_by"})
			return
		}
		addedBy = parsed
	}

	entry := &WatchlistEntry{
		ID:         uuid.New(),
		Name:       strings.TrimSpace(req.Name),
		EntityType: req.EntityType,
		Country:    req.Country,
		ListType:   req.ListType,
		RiskLevel:  req.RiskLevel,
		Details:    req.Details,
		AddedBy:    addedBy,
		AddedAt:    now,
		CreatedAt:  now,
	}

	if err := store.AddWatchlistEntry(c.Request.Context(), entry); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not add the watchlist entry"})
		return
	}

	store.LogAudit(c.Request.Context(), "watchlist_entry", &entry.ID, "added", "", nil)
	c.JSON(http.StatusCreated, gin.H{"entry": entry, "message": "added to watchlist"})
}

// removeFromWatchlistHandler deletes a watchlist record.
func removeFromWatchlistHandler(c *gin.Context) {
	var req struct {
		ID string `json:"id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := store.RemoveWatchlistEntry(c.Request.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "watchlist entry not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not remove the watchlist entry"})
		return
	}

	store.LogAudit(c.Request.Context(), "watchlist_entry", &id, "removed", "", nil)
	c.JSON(http.StatusOK, gin.H{"message": "removed from watchlist"})
}

// searchWatchlistHandler searches watchlist entries by name.
func searchWatchlistHandler(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "q is required"})
		return
	}

	results, err := store.SearchWatchlist(c.Request.Context(), query, parseIntQuery(c, "limit", 50))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not search the watchlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"query": query, "results": results, "total": len(results)})
}

// generateComplianceReportHandler aggregates compliance counts for a period.
func generateComplianceReportHandler(c *gin.Context) {
	start, end, ok := parseReportPeriod(c)
	if !ok {
		return
	}

	summary, err := store.ComplianceReport(c.Request.Context(), start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not build the compliance report"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"period":             gin.H{"start": start.Format(time.RFC3339), "end": end.Format(time.RFC3339)},
		"total_applications": summary.TotalApplications,
		"approved":           summary.Approved,
		"rejected":           summary.Rejected,
		"pending":            summary.Pending,
		"in_review":          summary.InReview,
		"needs_more_info":    summary.NeedsMoreInfo,
		"aml_checks": gin.H{
			"passed": summary.AMLChecksPassed,
			"failed": summary.AMLChecksFailed,
		},
		"generated_at": time.Now(),
	})
}

// generateAuditReportHandler aggregates audit actions for a period.
func generateAuditReportHandler(c *gin.Context) {
	start, end, ok := parseReportPeriod(c)
	if !ok {
		return
	}

	summary, err := store.AuditReport(c.Request.Context(), start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not build the audit report"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"period":        gin.H{"start": start.Format(time.RFC3339), "end": end.Format(time.RFC3339)},
		"total_actions": summary.TotalActions,
		"by_type":       summary.ByType,
		"by_actor":      summary.ByActor,
		"generated_at":  time.Now(),
	})
}

// parseReportPeriod reads start_date and end_date, defaulting to the last 30
// days. It writes an error response and returns false when the input is invalid.
func parseReportPeriod(c *gin.Context) (time.Time, time.Time, bool) {
	end := time.Now()
	start := end.AddDate(0, 0, -30)

	if raw := strings.TrimSpace(c.Query("start_date")); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			parsed, err = time.Parse("2006-01-02", raw)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "start_date must be RFC3339 or YYYY-MM-DD"})
				return start, end, false
			}
		}
		start = parsed
	}
	if raw := strings.TrimSpace(c.Query("end_date")); raw != "" {
		parsed, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			parsed, err = time.Parse("2006-01-02", raw)
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": "end_date must be RFC3339 or YYYY-MM-DD"})
				return start, end, false
			}
		}
		// A date-only end is inclusive of that whole day.
		if len(raw) == len("2006-01-02") {
			parsed = parsed.AddDate(0, 0, 1)
		}
		end = parsed
	}
	if !end.After(start) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "end_date must be after start_date"})
		return start, end, false
	}
	return start, end, true
}

// parseIntQuery reads an integer query parameter, falling back to def.
func parseIntQuery(c *gin.Context, key string, def int) int {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return def
	}
	var value int
	if _, err := fmt.Sscan(raw, &value); err != nil {
		return def
	}
	return value
}

// validKYCLevel reports whether a level is recognised.
func validKYCLevel(level string) bool {
	switch strings.ToLower(level) {
	case "basic", "advanced", "institutional":
		return true
	default:
		return false
	}
}

// validKYCStatus reports whether a status filter is recognised.
func validKYCStatus(status string) bool {
	switch status {
	case "pending", "in_review", "approved", "rejected", "needs_more_info":
		return true
	default:
		return false
	}
}

// validWatchlistType reports whether a watchlist type is recognised.
func validWatchlistType(listType string) bool {
	switch strings.ToLower(listType) {
	case "sanctions", "pep", "adverse_media", "internal":
		return true
	default:
		return false
	}
}

// validRiskLevel reports whether a risk level is recognised.
func validRiskLevel(level string) bool {
	switch strings.ToLower(level) {
	case "low", "medium", "high":
		return true
	default:
		return false
	}
}

// storeDocumentFile persists an uploaded document and returns its storage URL.
func storeDocumentFile(ctx context.Context, applicationID uuid.UUID, docType, side, fileData string) (string, error) {
	return documentStorage.Store(ctx, applicationID, docType, side, fileData)
}
