/**
 * TigerWallet Red Packets Service
 *
 * Lucky money/gift distribution service.
 * Built with Go for high-load distributed operations.
 */

package redpackets

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/google/uuid"
)

// RedPacket represents a red packet
type RedPacket struct {
	ID              string `json:"id"`
	SenderID        string `json:"sender_id"`
	SenderAddress   string `json:"sender_address"`
	TokenAddress    string `json:"token_address"`
	ChainID         uint64 `json:"chain_id"`
	TotalAmount     string `json:"total_amount"`
	Quantity        int    `json:"quantity"`
	RemainingAmount string `json:"remaining_amount"`
	RemainingQty    int    `json:"remaining_qty"`
	ClaimType       string `json:"claim_type"` // random, fixed
	Password        string `json:"password"`
	Message         string `json:"message"`
	ExpiredAt       int64  `json:"expired_at"`
	Status          string `json:"status"`
	TxHash          string `json:"tx_hash"`
	CreatedAt       int64  `json:"created_at"`
}

// RedPacketClaim represents a claim
type RedPacketClaim struct {
	ID             string `json:"id"`
	PacketID       string `json:"packet_id"`
	ClaimerID      string `json:"claimer_id"`
	ClaimerAddress string `json:"claimer_address"`
	Amount         string `json:"amount"`
	ClaimTxHash    string `json:"claim_tx_hash"`
	ClaimedAt      int64  `json:"claimed_at"`
}

// RedPacketService manages red packet operations
type RedPacketService struct {
	mu      sync.RWMutex
	packets map[string]*RedPacket
	claims  map[string]*RedPacketClaim
}

var (
	redPacketService     *RedPacketService
	redPacketServiceOnce sync.Once
)

func GetRedPacketService() *RedPacketService {
	redPacketServiceOnce.Do(func() {
		redPacketService = &RedPacketService{
			packets: make(map[string]*RedPacket),
			claims:  make(map[string]*RedPacketClaim),
		}
	})
	return redPacketService
}

func (s *RedPacketService) CreateRedPacket(ctx context.Context, packet *RedPacket) (*RedPacket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if packet.Quantity <= 0 {
		return nil, fmt.Errorf("quantity must be > 0")
	}
	if packet.TotalAmount == "" {
		return nil, fmt.Errorf("total amount required")
	}

	// Validate amount >= quantity (minimum 1 per packet)
	total, _ := new(big.Int).SetString(packet.TotalAmount, 10)
	minAmount := big.NewInt(int64(packet.Quantity))
	if total.Cmp(minAmount) < 0 {
		return nil, fmt.Errorf("total amount must be >= quantity")
	}

	packet.ID = "redpacket_" + uuid.New().String()
	packet.RemainingAmount = packet.TotalAmount
	packet.RemainingQty = packet.Quantity
	packet.Status = "active"
	packet.CreatedAt = time.Now().Unix()

	s.packets[packet.ID] = packet
	return packet, nil
}

func (s *RedPacketService) GetRedPacket(ctx context.Context, packetID string) (*RedPacket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	packet, exists := s.packets[packetID]
	if !exists {
		return nil, fmt.Errorf("red packet not found")
	}
	return packet, nil
}

func (s *RedPacketService) GetRedPacketByTx(ctx context.Context, txHash string) (*RedPacket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, packet := range s.packets {
		if packet.TxHash == txHash {
			return packet, nil
		}
	}
	return nil, fmt.Errorf("red packet not found")
}

func (s *RedPacketService) Claim(ctx context.Context, packetID, claimerID, claimerAddress, password string) (*RedPacketClaim, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	packet, exists := s.packets[packetID]
	if !exists {
		return nil, fmt.Errorf("red packet not found")
	}

	if packet.Status != "active" {
		return nil, fmt.Errorf("red packet not active")
	}

	// Check expiry
	if time.Now().Unix() > packet.ExpiredAt {
		packet.Status = "expired"
		return nil, fmt.Errorf("red packet expired")
	}

	// Check password
	if packet.Password != "" && packet.Password != password {
		return nil, fmt.Errorf("invalid password")
	}

	// Check remaining quantity
	if packet.RemainingQty <= 0 {
		return nil, fmt.Errorf("all claims distributed")
	}

	// Check if already claimed
	for _, claim := range s.claims {
		if claim.PacketID == packetID && claim.ClaimerID == claimerID {
			return nil, fmt.Errorf("already claimed")
		}
	}

	// Calculate amount based on claim type
	var amount string
	if packet.ClaimType == "fixed" {
		// Divide equally
		remaining, _ := new(big.Int).SetString(packet.RemainingAmount, 10)
		amount = new(big.Int).Div(remaining, big.NewInt(int64(packet.RemainingQty))).String()
	} else {
		// Random amount (simplified - in production use proper random distribution)
		remaining, _ := new(big.Int).SetString(packet.RemainingAmount, 10)
		if packet.RemainingQty == 1 {
			amount = remaining.String()
		} else {
			// Random between 1 and remaining/qty * 2
			avg := new(big.Int).Div(remaining, big.NewInt(int64(packet.RemainingQty)))
			max := new(big.Int).Mul(avg, big.NewInt(2))
			amount = new(big.Int).Mod(big.NewInt(time.Now().UnixNano()), max).String()
			amt, ok := new(big.Int).SetString(amount, 10)
			if !ok || amt.Cmp(big.NewInt(0)) <= 0 {
				amount = "1"
			}
		}
	}

	// Ensure amount > 0
	amountInt, _ := new(big.Int).SetString(amount, 10)
	if amountInt.Cmp(big.NewInt(0)) <= 0 {
		amount = "1"
	}

	// Create claim
	claim := &RedPacketClaim{
		ID:             "claim_" + uuid.New().String(),
		PacketID:       packetID,
		ClaimerID:      claimerID,
		ClaimerAddress: claimerAddress,
		Amount:         amount,
		ClaimedAt:      time.Now().Unix(),
	}

	s.claims[claim.ID] = claim

	// Update packet
	remainingAmt, _ := new(big.Int).SetString(packet.RemainingAmount, 10)
	remainingAmt.Sub(remainingAmt, amountInt)
	packet.RemainingAmount = remainingAmt.String()
	packet.RemainingQty--

	if packet.RemainingQty <= 0 {
		packet.Status = "completed"
	}

	return claim, nil
}

func (s *RedPacketService) GetPacketClaims(ctx context.Context, packetID string) ([]*RedPacketClaim, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*RedPacketClaim, 0)
	for _, claim := range s.claims {
		if claim.PacketID == packetID {
			result = append(result, claim)
		}
	}
	return result, nil
}

func (s *RedPacketService) GetUserClaims(ctx context.Context, userID string) ([]*RedPacketClaim, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*RedPacketClaim, 0)
	for _, claim := range s.claims {
		if claim.ClaimerID == userID {
			result = append(result, claim)
		}
	}
	return result, nil
}

// GetSentPackets returns all red packets created by the given user.
func (s *RedPacketService) GetSentPackets(ctx context.Context, userID string) ([]*RedPacket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*RedPacket, 0)
	for _, packet := range s.packets {
		if packet.SenderID == userID {
			result = append(result, packet)
		}
	}
	return result, nil
}

// GetReceivedPackets returns all red packets the given user has claimed.
func (s *RedPacketService) GetReceivedPackets(ctx context.Context, userID string) ([]*RedPacket, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*RedPacket, 0)
	for _, claim := range s.claims {
		if claim.ClaimerID == userID {
			if packet, ok := s.packets[claim.PacketID]; ok {
				result = append(result, packet)
			}
		}
	}
	return result, nil
}

func (r *RedPacket) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
