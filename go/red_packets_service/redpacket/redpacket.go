/**
 * TigerWallet Red Packets Service
 *
 * Lucky money/gift distribution service.
 * Built with Go for high-load distributed operations.
 * PostgreSQL-backed — all packets and claims are persisted.
 */

package redpackets

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
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
	// PasswordHash stores a bcrypt hash of the optional claim password. It is
	// NEVER serialized to JSON (json:"-") and is NEVER stored/compared in
	// plaintext.
	PasswordHash string `json:"-"`
	Message      string `json:"message"`
	ExpiredAt    int64  `json:"expired_at"`
	Status       string `json:"status"`
	TxHash       string `json:"tx_hash"`
	CreatedAt    int64  `json:"created_at"`
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

// RedPacketService manages red packet operations backed by PostgreSQL.
type RedPacketService struct {
	pg *pgxpool.Pool
}

var redPacketService *RedPacketService

func NewRedPacketService(pg *pgxpool.Pool) *RedPacketService {
	return &RedPacketService{pg: pg}
}

func GetRedPacketService() *RedPacketService {
	if redPacketService != nil {
		return redPacketService
	}
	return &RedPacketService{}
}

func SetRedPacketService(pg *pgxpool.Pool) {
	redPacketService = NewRedPacketService(pg)
}

const redPacketSchema = `
CREATE TABLE IF NOT EXISTS red_packets (
    id                TEXT PRIMARY KEY,
    sender_id         TEXT NOT NULL DEFAULT '',
    sender_address    TEXT NOT NULL DEFAULT '',
    token_address     TEXT NOT NULL DEFAULT '',
    chain_id          BIGINT NOT NULL DEFAULT 0,
    total_amount      TEXT NOT NULL DEFAULT '0',
    quantity          INTEGER NOT NULL DEFAULT 0,
    remaining_amount  TEXT NOT NULL DEFAULT '0',
    remaining_qty     INTEGER NOT NULL DEFAULT 0,
    claim_type        TEXT NOT NULL DEFAULT 'random',
    password_hash     TEXT NOT NULL DEFAULT '',
    message           TEXT NOT NULL DEFAULT '',
    expired_at        BIGINT NOT NULL DEFAULT 0,
    status            TEXT NOT NULL DEFAULT 'active',
    tx_hash           TEXT NOT NULL DEFAULT '',
    created_at        BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_red_packets_sender ON red_packets(sender_id);
CREATE INDEX IF NOT EXISTS idx_red_packets_txhash ON red_packets(tx_hash);
CREATE TABLE IF NOT EXISTS red_packet_claims (
    id              TEXT PRIMARY KEY,
    packet_id       TEXT NOT NULL REFERENCES red_packets(id) ON DELETE CASCADE,
    claimer_id      TEXT NOT NULL DEFAULT '',
    claimer_address TEXT NOT NULL DEFAULT '',
    amount          TEXT NOT NULL DEFAULT '0',
    claim_tx_hash   TEXT NOT NULL DEFAULT '',
    claimed_at      BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_red_packet_claims_packet ON red_packet_claims(packet_id);
CREATE INDEX IF NOT EXISTS idx_red_packet_claims_claimer ON red_packet_claims(claimer_id);
`

const packetSelectCols = `id,sender_id,sender_address,token_address,chain_id,total_amount,quantity,remaining_amount,remaining_qty,claim_type,password_hash,message,expired_at,status,tx_hash,created_at`

const claimSelectCols = `id,packet_id,claimer_id,claimer_address,amount,claim_tx_hash,claimed_at`

func (s *RedPacketService) Migrate(ctx context.Context) error {
	if s.pg == nil {
		return fmt.Errorf("database not configured")
	}
	_, err := s.pg.Exec(ctx, redPacketSchema)
	return err
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanPacket(row rowScanner) (*RedPacket, error) {
	var p RedPacket
	if err := row.Scan(&p.ID, &p.SenderID, &p.SenderAddress, &p.TokenAddress, &p.ChainID,
		&p.TotalAmount, &p.Quantity, &p.RemainingAmount, &p.RemainingQty, &p.ClaimType,
		&p.PasswordHash, &p.Message, &p.ExpiredAt, &p.Status, &p.TxHash, &p.CreatedAt); err != nil {
		return nil, err
	}
	return &p, nil
}

func scanClaim(row rowScanner) (*RedPacketClaim, error) {
	var c RedPacketClaim
	if err := row.Scan(&c.ID, &c.PacketID, &c.ClaimerID, &c.ClaimerAddress,
		&c.Amount, &c.ClaimTxHash, &c.ClaimedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *RedPacketService) CreateRedPacket(ctx context.Context, packet *RedPacket) (*RedPacket, error) {
	if s.pg == nil {
		return nil, fmt.Errorf("database not configured")
	}
	if packet.Quantity <= 0 {
		return nil, fmt.Errorf("quantity must be > 0")
	}
	if packet.TotalAmount == "" {
		return nil, fmt.Errorf("total amount required")
	}
	total, _ := new(big.Int).SetString(packet.TotalAmount, 10)
	minAmount := big.NewInt(int64(packet.Quantity))
	if total.Cmp(minAmount) < 0 {
		return nil, fmt.Errorf("total amount must be >= quantity")
	}

	plain := packet.PasswordHash
	packet.PasswordHash = ""
	if plain != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("failed to hash password: %v", err)
		}
		packet.PasswordHash = string(hash)
	}

	packet.ID = "redpacket_" + uuid.New().String()
	packet.RemainingAmount = packet.TotalAmount
	packet.RemainingQty = packet.Quantity
	packet.Status = "active"
	packet.CreatedAt = time.Now().Unix()

	_, err := s.pg.Exec(ctx, `INSERT INTO red_packets
		(id,sender_id,sender_address,token_address,chain_id,total_amount,quantity,remaining_amount,remaining_qty,claim_type,password_hash,message,expired_at,status,tx_hash,created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		packet.ID, packet.SenderID, packet.SenderAddress, packet.TokenAddress, packet.ChainID,
		packet.TotalAmount, packet.Quantity, packet.RemainingAmount, packet.RemainingQty, packet.ClaimType,
		packet.PasswordHash, packet.Message, packet.ExpiredAt, packet.Status, packet.TxHash, packet.CreatedAt)
	if err != nil {
		return nil, err
	}
	return packet, nil
}

func (s *RedPacketService) GetRedPacket(ctx context.Context, packetID string) (*RedPacket, error) {
	if s.pg == nil {
		return nil, fmt.Errorf("database not configured")
	}
	row := s.pg.QueryRow(ctx, `SELECT `+packetSelectCols+` FROM red_packets WHERE id=$1`, packetID)
	p, err := scanPacket(row)
	if err != nil {
		return nil, fmt.Errorf("red packet not found")
	}
	return p, nil
}

func (s *RedPacketService) GetRedPacketByTx(ctx context.Context, txHash string) (*RedPacket, error) {
	if s.pg == nil {
		return nil, fmt.Errorf("database not configured")
	}
	row := s.pg.QueryRow(ctx, `SELECT `+packetSelectCols+` FROM red_packets WHERE tx_hash=$1`, txHash)
	p, err := scanPacket(row)
	if err != nil {
		return nil, fmt.Errorf("red packet not found")
	}
	return p, nil
}

func (s *RedPacketService) Claim(ctx context.Context, packetID, claimerID, claimerAddress, password string) (*RedPacketClaim, error) {
	if s.pg == nil {
		return nil, fmt.Errorf("database not configured")
	}
	tx, err := s.pg.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `SELECT `+packetSelectCols+` FROM red_packets WHERE id=$1 FOR UPDATE`, packetID)
	p, err := scanPacket(row)
	if err != nil {
		return nil, fmt.Errorf("red packet not found")
	}
	if p.Status != "active" {
		return nil, fmt.Errorf("red packet not active")
	}
	if time.Now().Unix() > p.ExpiredAt {
		_, _ = tx.Exec(ctx, `UPDATE red_packets SET status='expired' WHERE id=$1`, packetID)
		_ = tx.Commit(ctx)
		return nil, fmt.Errorf("red packet expired")
	}
	if p.PasswordHash != "" {
		if password == "" || bcrypt.CompareHashAndPassword([]byte(p.PasswordHash), []byte(password)) != nil {
			return nil, fmt.Errorf("invalid password")
		}
	}
	if p.RemainingQty <= 0 {
		return nil, fmt.Errorf("all claims distributed")
	}

	var alreadyClaimed bool
	err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM red_packet_claims WHERE packet_id=$1 AND claimer_id=$2)`,
		packetID, claimerID).Scan(&alreadyClaimed)
	if err != nil {
		return nil, err
	}
	if alreadyClaimed {
		return nil, fmt.Errorf("already claimed")
	}

	var amount string
	if p.ClaimType == "fixed" {
		remaining, _ := new(big.Int).SetString(p.RemainingAmount, 10)
		amount = new(big.Int).Div(remaining, big.NewInt(int64(p.RemainingQty))).String()
	} else {
		remaining, _ := new(big.Int).SetString(p.RemainingAmount, 10)
		if p.RemainingQty == 1 {
			amount = remaining.String()
		} else {
			avg := new(big.Int).Div(remaining, big.NewInt(int64(p.RemainingQty)))
			max := new(big.Int).Mul(avg, big.NewInt(2))
			if max.Cmp(big.NewInt(1)) <= 0 {
				amount = "1"
			} else {
				rnd, err := rand.Int(rand.Reader, max)
				if err != nil || rnd.Cmp(big.NewInt(1)) <= 0 {
					amount = "1"
				} else {
					amount = rnd.String()
				}
			}
		}
	}

	amountInt, _ := new(big.Int).SetString(amount, 10)
	if amountInt == nil || amountInt.Cmp(big.NewInt(0)) <= 0 {
		amount = "1"
		amountInt = big.NewInt(1)
	}

	claim := &RedPacketClaim{
		ID:             "claim_" + uuid.New().String(),
		PacketID:       packetID,
		ClaimerID:      claimerID,
		ClaimerAddress: claimerAddress,
		Amount:         amount,
		ClaimedAt:      time.Now().Unix(),
	}
	if _, err := tx.Exec(ctx, `INSERT INTO red_packet_claims
		(id,packet_id,claimer_id,claimer_address,amount,claim_tx_hash,claimed_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		claim.ID, claim.PacketID, claim.ClaimerID, claim.ClaimerAddress, claim.Amount,
		claim.ClaimTxHash, claim.ClaimedAt); err != nil {
		return nil, err
	}

	remainingAmt, _ := new(big.Int).SetString(p.RemainingAmount, 10)
	remainingAmt.Sub(remainingAmt, amountInt)
	newQty := p.RemainingQty - 1
	newStatus := p.Status
	if newQty <= 0 {
		newStatus = "completed"
	}
	if _, err := tx.Exec(ctx, `UPDATE red_packets SET remaining_amount=$1, remaining_qty=$2, status=$3 WHERE id=$4`,
		remainingAmt.String(), newQty, newStatus, packetID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return claim, nil
}

func (s *RedPacketService) GetPacketClaims(ctx context.Context, packetID string) ([]*RedPacketClaim, error) {
	if s.pg == nil {
		return []*RedPacketClaim{}, fmt.Errorf("database not configured")
	}
	rows, err := s.pg.Query(ctx, `SELECT `+claimSelectCols+` FROM red_packet_claims WHERE packet_id=$1 ORDER BY claimed_at DESC`, packetID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]*RedPacketClaim, 0)
	for rows.Next() {
		c, err := scanClaim(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (s *RedPacketService) GetUserClaims(ctx context.Context, userID string) ([]*RedPacketClaim, error) {
	if s.pg == nil {
		return []*RedPacketClaim{}, fmt.Errorf("database not configured")
	}
	rows, err := s.pg.Query(ctx, `SELECT `+claimSelectCols+` FROM red_packet_claims WHERE claimer_id=$1 ORDER BY claimed_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]*RedPacketClaim, 0)
	for rows.Next() {
		c, err := scanClaim(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (s *RedPacketService) GetSentPackets(ctx context.Context, userID string) ([]*RedPacket, error) {
	if s.pg == nil {
		return []*RedPacket{}, fmt.Errorf("database not configured")
	}
	rows, err := s.pg.Query(ctx, `SELECT `+packetSelectCols+` FROM red_packets WHERE sender_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]*RedPacket, 0)
	for rows.Next() {
		p, err := scanPacket(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (s *RedPacketService) GetReceivedPackets(ctx context.Context, userID string) ([]*RedPacket, error) {
	if s.pg == nil {
		return []*RedPacket{}, fmt.Errorf("database not configured")
	}
	rows, err := s.pg.Query(ctx, `SELECT p.id,p.sender_id,p.sender_address,p.token_address,p.chain_id,p.total_amount,p.quantity,p.remaining_amount,p.remaining_qty,p.claim_type,p.password_hash,p.message,p.expired_at,p.status,p.tx_hash,p.created_at
		FROM red_packets p JOIN red_packet_claims c ON c.packet_id=p.id WHERE c.claimer_id=$1 ORDER BY c.claimed_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]*RedPacket, 0)
	for rows.Next() {
		p, err := scanPacket(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, rows.Err()
}

func (r *RedPacket) ToJSON() (string, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
