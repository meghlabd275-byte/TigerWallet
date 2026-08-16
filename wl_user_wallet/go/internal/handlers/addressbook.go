package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/tigerwallet/wl-user-wallet/internal/middleware"
)

// GET /address-book — list the caller's saved contacts (real PG).
func (s *Svc) ListAddressBook(c *gin.Context) {
	contacts, err := s.store.ListContacts(c.Request.Context(), middleware.UserID(c))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"contacts": contacts})
}

// POST /address-book — create a contact (real PG insert).
func (s *Svc) CreateAddressBook(c *gin.Context) {
	var req struct {
		Name    string `json:"name" binding:"required"`
		Address string `json:"address" binding:"required"`
		ChainID int64  `json:"chain_id"`
		Note    string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id, err := s.store.CreateContact(c.Request.Context(), middleware.UserID(c), req.Name, req.Address, req.ChainID, req.Note)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name, "address": req.Address, "chain_id": req.ChainID})
}

// PUT /address-book/:id — update a contact (real PG update).
func (s *Svc) UpdateAddressBook(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req struct {
		Name    string `json:"name" binding:"required"`
		Address string `json:"address" binding:"required"`
		ChainID int64  `json:"chain_id"`
		Note    string `json:"note"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	chainID, _ := strconv.ParseInt(c.Query("chain_id"), 10, 64)
	if req.ChainID == 0 {
		req.ChainID = chainID
	}
	rows, err := s.store.UpdateContact(c.Request.Context(), middleware.UserID(c), id, req.Name, req.Address, req.ChainID, req.Note)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "contact not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "updated": rows})
}

// DELETE /address-book/:id — delete a contact (real PG delete).
func (s *Svc) DeleteAddressBook(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	rows, err := s.store.DeleteContact(c.Request.Context(), middleware.UserID(c), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if rows == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "contact not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "deleted": rows})
}
