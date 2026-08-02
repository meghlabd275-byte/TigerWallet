package red_packet_service

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ============================================================================
// Types
// ============================================================================

type PacketType string

const (
	PacketTypeRandom PacketType = "random"
	PacketTypeFixed PacketType = "fixed"
)

type RedPacket struct {
	ID              string      `json:"id"`
	Sender          string      `json:"sender"`
	SenderAddress   string      `json:"senderAddress"`
	Token           string      `json:"token"`
	Amount          float64     `json:"amount"`
	TotalCount      int         `json:"totalCount"`
	ReceivedCount   int         `json:"receivedCount"`
	RemainingAmount float64     `json:"remainingAmount"`
	RemainingCount  int         `json:"remainingCount"`
	Message         string      `json:"message"`
	Type            PacketType  `json:"type"`
	Link            string      `json:"link"`
	Status          string      `json:"status"`
	CreateTime      time.Time   `json:"createTime"`
	ExpireTime      time.Time   `json:"expireTime"`
}

type ClaimRecord struct {
	PacketID    string    `json:"packetId"`
	Claimer     string    `json:"claimer"`
	ClaimerAddr string    `json:"claimerAddr"`
	Amount      float64   `json:"amount"`
	ClaimTime   time.Time `json:"claimTime"`
}

type RedPacketService struct {
	mu      sync.RWMutex
	packets map[string]*RedPacket
	claims  map[string][]ClaimRecord
}

func NewRedPacketService() *RedPacketService {
	return &RedPacketService{
		packets: make(map[string]*RedPacket),
		claims:  make(map[string][]ClaimRecord),
	}
}

func (rps *RedPacketService) CreatePacket(sender, senderAddress, token string, amount float64, totalCount int, packetType PacketType, message string) (*RedPacket, error) {
	rps.mu.Lock()
	defer rps.mu.Unlock()

	if amount <= 0 || totalCount <= 0 {
		return nil, fmt.Errorf("invalid amount or count")
	}

	packet := &RedPacket{
		ID:              fmt.Sprintf("rp-%d", time.Now().Unix()),
		Sender:          sender,
		SenderAddress:   senderAddress,
		Token:           token,
		Amount:          amount,
		TotalCount:      totalCount,
		ReceivedCount:   0,
		RemainingAmount: amount,
		RemainingCount:  totalCount,
		Message:         message,
		Type:            packetType,
		Link:            fmt.Sprintf("https://tigerwallet.com/redpacket/claim/%s", fmt.Sprintf("rp-%d", time.Now().Unix())),
		Status:          "active",
		CreateTime:      time.Now(),
		ExpireTime:      time.Now().Add(24 * time.Hour),
	}

	rps.packets[packet.ID] = packet
	rps.packets[packet.Link] = packet

	return packet, nil
}

func (rps *RedPacketService) GetPacket(packetID string) (*RedPacket, error) {
	rps.mu.RLock()
	defer rps.mu.RUnlock()

	packet, ok := rps.packets[packetID]
	if !ok {
		return nil, fmt.Errorf("packet not found: %s", packetID)
	}
	return packet, nil
}

func (rps *RedPacketService) ClaimPacket(packetID, claimer, claimerAddr string) (*ClaimRecord, error) {
	rps.mu.Lock()
	defer rps.mu.Unlock()

	packet, ok := rps.packets[packetID]
	if !ok {
		return nil, fmt.Errorf("packet not found: %s", packetID)
	}

	if packet.Status != "active" {
		return nil, fmt.Errorf("packet is not active")
	}

	if packet.RemainingCount <= 0 {
		return nil, fmt.Errorf("all packets have been claimed")
	}

	var claimAmount float64
	if packet.Type == PacketTypeRandom {
		// Random amount distribution
		if packet.RemainingCount == 1 {
			claimAmount = packet.RemainingAmount
		} else {
			max := packet.RemainingAmount * 2 / float64(packet.RemainingCount)
			claimAmount = rand.Float64() * max
			if claimAmount > packet.RemainingAmount {
				claimAmount = packet.RemainingAmount / 2
			}
		}
	} else {
		// Fixed amount
		claimAmount = packet.Amount / float64(packet.TotalCount)
	}

	claim := &ClaimRecord{
		PacketID:    packetID,
		Claimer:     claimer,
		ClaimerAddr: claimerAddr,
		Amount:      claimAmount,
		ClaimTime:   time.Now(),
	}

	rps.claims[packetID] = append(rps.claims[packetID], *claim)

	packet.ReceivedCount++
	packet.RemainingCount--
	packet.RemainingAmount -= claimAmount

	if packet.RemainingCount <= 0 {
		packet.Status = "completed"
	}

	return claim, nil
}

func (rps *RedPacketService) GetPacketClaims(packetID string) []ClaimRecord {
	rps.mu.RLock()
	defer rps.mu.RUnlock()
	return rps.claims[packetID]
}

func (rps *RedPacketService) GetUserPackets(userID string) []*RedPacket {
	rps.mu.RLock()
	defer rps.mu.RUnlock()

	var packets []*RedPacket
	for _, packet := range rps.packets {
		if packet.Sender == userID {
			packets = append(packets, packet)
		}
	}
	return packets
}

func (rps *RedPacketService) ExpirePackets() {
	rps.mu.Lock()
	defer rps.mu.Unlock()

	now := time.Now()
	for _, packet := range rps.packets {
		if packet.Status == "active" && now.After(packet.ExpireTime) {
			packet.Status = "expired"
		}
	}
}

func (rps *RedPacketService) DeletePacket(packetID string) error {
	rps.mu.Lock()
	defer rps.mu.Unlock()

	if _, ok := rps.packets[packetID]; !ok {
		return fmt.Errorf("packet not found: %s", packetID)
	}

	delete(rps.packets, packetID)
	return nil
}

func (rps *RedPacketService) ToJSON() (string, error) {
	rps.mu.RLock()
	defer rps.mu.RUnlock()

	data := struct {
		Packets map[string]*RedPacket `json:"packets"`
		Claims  map[string][]ClaimRecord `json:"claims"`
	}{
		Packets: rps.packets,
		Claims:  rps.claims,
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}
