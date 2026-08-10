package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AddressBookRecord mirrors the address_book table.
type AddressBookRecord struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Name      string    `json:"name"`
	Address   string    `json:"address"`
	ChainID   int64     `json:"chain_id"`
	Note      string    `json:"note"`
	CreatedAt int64     `json:"created_at"`
}

func handleListContacts(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	rows, err := store.PG.Query(c.Request.Context(),
		`SELECT id, user_id, name, address, chain_id, COALESCE(note,''), extract(epoch from created_at)::bigint FROM address_book WHERE user_id=$1 ORDER BY created_at DESC`, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	defer rows.Close()
	out := []AddressBookRecord{}
	for rows.Next() {
		var r AddressBookRecord
		if err := rows.Scan(&r.ID, &r.UserID, &r.Name, &r.Address, &r.ChainID, &r.Note, &r.CreatedAt); err != nil {
			continue
		}
		out = append(out, r)
	}
	c.JSON(http.StatusOK, gin.H{"contacts": out, "count": len(out)})
}

type createContactReq struct {
	Name    string `json:"name" binding:"required"`
	Address string `json:"address" binding:"required"`
	ChainID int64  `json:"chain_id"`
	Note    string `json:"note"`
}

func handleCreateContact(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	var req createContactReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	id := uuid.New()
	_, err = store.PG.Exec(c.Request.Context(),
		`INSERT INTO address_book (id, user_id, name, address, chain_id, note) VALUES ($1,$2,$3,$4,$5,$6)`,
		id, userUUID, req.Name, req.Address, req.ChainID, req.Note)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot save contact"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name, "address": req.Address, "chain_id": req.ChainID})
}

func handleUpdateContact(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var req createContactReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tag, err := store.PG.Exec(c.Request.Context(),
		`UPDATE address_book SET name=$1, address=$2, chain_id=$3, note=$4 WHERE id=$5 AND user_id=$6`,
		req.Name, req.Address, req.ChainID, req.Note, id, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "contact not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "updated": true})
}

func handleDeleteContact(c *gin.Context) {
	uid := c.GetString("userID")
	userUUID, err := uuid.Parse(uid)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user"})
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	tag, err := store.PG.Exec(c.Request.Context(), `DELETE FROM address_book WHERE id=$1 AND user_id=$2`, id, userUUID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete failed"})
		return
	}
	if tag.RowsAffected() == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "contact not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"id": id, "deleted": true})
}
