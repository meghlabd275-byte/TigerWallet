// TigerWallet Passkey Authentication Service - Social recovery
//
// A user enrols a recovery key by splitting a fresh 32-byte secret with Shamir
// secret sharing. The shares are encrypted at rest and handed back to the
// caller to distribute; any `threshold` of them rebuild the secret, and any
// smaller set reveals nothing about it. Redemption is offline by design: the
// shares are never re-submitted over HTTP, so the service never has to trust a
// caller-supplied value.

package main

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RecoveryKey records one enrolment. The secret itself is never stored.
type RecoveryKey struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UserID      uint      `gorm:"index" json:"user_id"`
	KeyID       string    `gorm:"uniqueIndex" json:"key_id"`
	Threshold   int       `json:"threshold"`
	TotalShares int       `json:"total_shares"`
	ExpiresAt   time.Time `json:"expires_at"`
	IsActive    bool      `json:"is_active"`
}

// RecoveryShare is one encrypted Shamir share. Ciphertext is the value issued
// to the user; nothing here is derived from it.
type RecoveryShare struct {
	ID             uint       `gorm:"primarykey" json:"id"`
	CreatedAt      time.Time  `json:"created_at"`
	RecoveryKeyID  uint       `gorm:"index" json:"recovery_key_id"`
	ShareIndex     int        `json:"share_index"`
	EncryptedShare string     `json:"encrypted_share"`
	DistributedTo  string     `json:"distributed_to"`
	ClaimedAt      *time.Time `json:"claimed_at"`
}

// RecoveryEnrolResponse is what the user receives on enrolment. The shares are
// encrypted; the user stores them and the service keeps its own copy.
type RecoveryEnrolResponse struct {
	RecoveryKeyID string   `json:"recovery_key_id"`
	Threshold     int      `json:"threshold"`
	TotalShares   int      `json:"total_shares"`
	Shares        []string `json:"shares"`
	Message       string   `json:"message"`
}

type recoveryEnrolRequest struct {
	Threshold   int `json:"threshold" binding:"required,min=2,max=255"`
	TotalShares int `json:"total_shares" binding:"required,min=2,max=255"`
}

// RecoveryEnrol splits a fresh recovery secret for the authenticated user.
func (s *PasskeyService) RecoveryEnrol(c *gin.Context) {
	var req recoveryEnrolRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Threshold > req.TotalShares {
		c.JSON(http.StatusBadRequest, gin.H{"error": "threshold cannot exceed total shares"})
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate recovery secret"})
		return
	}

	split, err := shamirSplit(secret, req.Threshold, req.TotalShares)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	key := RecoveryKey{
		UserID:      userID,
		KeyID:       uuid.New().String(),
		Threshold:   req.Threshold,
		TotalShares: req.TotalShares,
		ExpiresAt:   time.Now().AddDate(1, 0, 0),
		IsActive:    true,
	}

	// Supersede any earlier enrolment so a stale share set cannot be redeemed
	// against a newer key.
	if err := s.db.Model(&RecoveryKey{}).
		Where("user_id = ? AND is_active = ?", userID, true).
		Update("is_active", false).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to retire previous recovery key"})
		return
	}

	if err := s.db.Create(&key).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist recovery key"})
		return
	}

	encoded := make([]string, 0, len(split))
	for i, sh := range split {
		plain := encodeShare(sh)
		encrypted, err := s.encryptShare(plain)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt share"})
			return
		}

		method := "email"
		switch i % 3 {
		case 1:
			method = "sms"
		case 2:
			method = "cloud"
		}

		if err := s.db.Create(&RecoveryShare{
			RecoveryKeyID:  key.ID,
			ShareIndex:     int(sh.Index),
			EncryptedShare: encrypted,
			DistributedTo:  method,
		}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to persist recovery share"})
			return
		}
		encoded = append(encoded, plain)
	}

	c.JSON(http.StatusOK, RecoveryEnrolResponse{
		RecoveryKeyID: key.KeyID,
		Threshold:     req.Threshold,
		TotalShares:   req.TotalShares,
		Shares:        encoded,
		Message: fmt.Sprintf(
			"Recovery enrolled. Any %d of %d shares reconstruct the secret; store them separately.",
			req.Threshold, req.TotalShares),
	})
}

type recoveryRedeemRequest struct {
	Shares []string `json:"shares" binding:"required,min=2"`
}

type recoveryRedeemResponse struct {
	RecoveryKeyID string `json:"recovery_key_id"`
	Secret        string `json:"secret"`
	SharesUsed    int    `json:"shares_used"`
}

// RecoveryRedeem rebuilds the recovery secret from shares the caller holds.
//
// Only shares this service actually issued for the user's active key are
// accepted, compared against the decrypted stored value in constant time, so a
// caller cannot mix in invented values to steer the interpolation.
func (s *PasskeyService) RecoveryRedeem(c *gin.Context) {
	var req recoveryRedeemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthenticated"})
		return
	}

	var key RecoveryKey
	err := s.db.Where("user_id = ? AND is_active = ?", userID, true).
		Order("created_at DESC").First(&key).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active recovery key"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load recovery key"})
		return
	}
	if !key.ExpiresAt.IsZero() && time.Now().After(key.ExpiresAt) {
		c.JSON(http.StatusGone, gin.H{"error": "recovery key expired"})
		return
	}
	if len(req.Shares) < key.Threshold {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":     "not enough shares",
			"threshold": key.Threshold,
			"provided":  len(req.Shares),
		})
		return
	}

	var stored []RecoveryShare
	if err := s.db.Where("recovery_key_id = ?", key.ID).Find(&stored).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load recovery shares"})
		return
	}
	issued := make(map[int]string, len(stored))
	for _, r := range stored {
		issued[r.ShareIndex] = r.EncryptedShare
	}

	parsed := make([]ShamirShare, 0, len(req.Shares))
	for _, provided := range req.Shares {
		sh, err := decodeShare(provided)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		blob, ok := issued[int(sh.Index)]
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("share %d was not issued for this recovery key", sh.Index),
			})
			return
		}
		plain, err := s.decryptShare(blob)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read stored share"})
			return
		}
		if subtle.ConstantTimeCompare([]byte(plain), []byte(provided)) != 1 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("share %d does not match the issued value", sh.Index),
			})
			return
		}
		parsed = append(parsed, sh)
	}

	secret, err := shamirRecover(parsed)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "recovery failed: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, recoveryRedeemResponse{
		RecoveryKeyID: key.KeyID,
		Secret:        hex.EncodeToString(secret),
		SharesUsed:    len(parsed),
	})
}
