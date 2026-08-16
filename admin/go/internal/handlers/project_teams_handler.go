/**
 * TigerWallet Admin - Project Teams Handler
 * Governance records only — no fund movement. Admins manage project teams
 * (coin/token listing project teams) and their members.
 */

package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ProjectTeamsHandler struct {
	db *gorm.DB
}

func NewProjectTeamsHandler(db *gorm.DB) *ProjectTeamsHandler {
	return &ProjectTeamsHandler{db: db}
}

// ProjectTeam mirrors the admin_project_teams governance table.
type ProjectTeam struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	TeamID      string    `gorm:"uniqueIndex;not null" json:"team_id"`
	Name        string    `gorm:"not null" json:"name"`
	ProjectType string    `json:"project_type"`
	TokenSymbol string    `gorm:"index" json:"token_symbol"`
	ChainID     *int64    `json:"chain_id"`
	Status      string    `gorm:"not null;default:'active';index" json:"status"`
	Website     string    `json:"website"`
	Email       string    `json:"email"`
	Metadata    string    `json:"metadata"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (ProjectTeam) TableName() string { return "admin_project_teams" }

// ProjectTeamMember mirrors the admin_project_team_members governance table.
type ProjectTeamMember struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	TeamID    string    `gorm:"index;not null" json:"team_id"`
	UserID    *string   `gorm:"index" json:"user_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `gorm:"not null;default:'active'" json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (ProjectTeamMember) TableName() string { return "admin_project_team_members" }

func (h *ProjectTeamsHandler) List(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	offset := (page - 1) * limit

	query := h.db.Model(&ProjectTeam{})
	if status != "" && status != "all" {
		query = query.Where("status = ?", status)
	}

	var items []ProjectTeam
	if err := query.Offset(offset).Limit(limit).Order("created_at DESC").Find(&items).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"project_teams": items})
}

func (h *ProjectTeamsHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var item ProjectTeam
	if err := h.db.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project team not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"project_team": item})
}

func (h *ProjectTeamsHandler) Create(c *gin.Context) {
	var item ProjectTeam
	if err := c.ShouldBindJSON(&item); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if item.Status == "" {
		item.Status = "active"
	}
	if err := h.db.Create(&item).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"project_team": item})
}

func (h *ProjectTeamsHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var item ProjectTeam
	if err := h.db.First(&item, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project team not found"})
		return
	}
	var input ProjectTeam
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.db.Model(&item).Updates(input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"project_team": item})
}

func (h *ProjectTeamsHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	result := h.db.Delete(&ProjectTeam{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "project team not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "project team deleted"})
}

// UpdateStatus sets project-team status (start/stop/pause/resume — governance record only).
func (h *ProjectTeamsHandler) UpdateStatus(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	result := h.db.Model(&ProjectTeam{}).Where("id = ?", id).Update("status", req.Status)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "project team not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "status updated", "status": req.Status})
}

// --- Members ---

func (h *ProjectTeamsHandler) GetMembers(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var team ProjectTeam
	if err := h.db.First(&team, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project team not found"})
		return
	}
	var members []ProjectTeamMember
	if err := h.db.Where("team_id = ?", team.TeamID).Order("created_at DESC").Find(&members).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"members": members})
}

func (h *ProjectTeamsHandler) AddMember(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var team ProjectTeam
	if err := h.db.First(&team, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "project team not found"})
		return
	}
	var member ProjectTeamMember
	if err := c.ShouldBindJSON(&member); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	member.TeamID = team.TeamID
	if member.Status == "" {
		member.Status = "active"
	}
	if err := h.db.Create(&member).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"member": member})
}

func (h *ProjectTeamsHandler) RemoveMember(c *gin.Context) {
	memberID, err := strconv.ParseUint(c.Param("memberId"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid member id"})
		return
	}
	result := h.db.Delete(&ProjectTeamMember{}, memberID)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "member not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "member removed"})
}
