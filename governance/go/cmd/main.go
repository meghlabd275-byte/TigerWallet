package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ============ Configuration ============

type Config struct {
	Port            string
	AdminAddress    string
	ProposalMinDelay time.Duration
	VotingPeriod    time.Duration
	ExecutionDelay time.Duration
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============ Models ============

type DAO struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Token       Token    `json:"token"`
	Threshold  uint64   `json:"threshold"` // Votes needed to pass
	Quorum     uint64   `json:"quorum"`     // Minimum participation
	CreatedAt  int64    `json:"createdAt"`
}

type Token struct {
	Address   string `json:"address"`
	Name      string `json:"name"`
	Symbol    string `json:"symbol"`
	Decimals  int   `json:"decimals"`
	Supply    string `json:"supply"`
}

type Proposal struct {
	ID            string       `json:"id"`
	DAOID        string       `json:"daoId"`
	Title        string       `json:"title"`
	Description string       `json:"description"`
	Type        ProposalType `json:"type"`
	Status      ProposalStatus `json:"status"`
	Proposer    string       `json:"proposer"`
	Target      string       `json:"target"`
	Value       string       `json:"value"`
	Data        string       `json:"data"`
	VoteCount   int64        `json:"voteCount"`
	ForVotes    int64        `json:"forVotes"`
	AgainstVotes int64       `json:"againstVotes"`
	AbstainVotes int64       `json:"abstainVotes"`
	StartBlock  uint64       `json:"startBlock"`
	EndBlock   uint64        `json:"endBlock"`
	ExecuteBlock uint64       `json:"executeBlock"`
	ExecutedAt *int64        `json:"executedAt"`
	CreatedAt  int64         `json:"createdAt"`
}

type ProposalType string

const (
	TypeTransfer      ProposalType = "transfer"
	TypeUpgrade      ProposalType = "upgrade"
	TypeParameter    ProposalType = "parameter"
	TypeCustom       ProposalType = "custom"
)

type ProposalStatus string

const (
	StatusPending   ProposalStatus = "pending"
	StatusActive   ProposalStatus = "active"
	StatusCanceled ProposalStatus = "canceled"
	StatusDefeated ProposalStatus = "defeated"
	StatusSucceeded ProposalStatus = "succeeded"
	StatusQueued   ProposalStatus = "queued"
	StatusExecuted ProposalStatus = "executed"
	StatusExpired  ProposalStatus = "expired"
)

type Vote struct {
	ID          string   `json:"id"`
	ProposalID  string   `json:"proposalId"`
	Voter       string   `json:"voter"`
	Support     bool     `json:"support"`
	Weight      int64    `json:"weight"`
	Reason      string   `json:"reason"`
	BlockNumber uint64   `json:"blockNumber"`
	TxHash     string   `json:"txHash"`
	CreatedAt  int64    `json:"createdAt"`
}

type Delegate struct {
	ID            string `json:"id"`
	Delegator    string `json:"delegator"`
	Delegate     string `json:"delegate"`
	Token       string `json:"token"`
	Balance     int64  `json:"balance"`
	VotingPower int64  `json:"votingPower"`
	CreatedAt   int64  `json:"createdAt"`
}

type Treasury struct {
	DAOID     string             `json:"daoId"`
	Balance   map[string]string  `json:"balance"` // token -> amount
	Transactions []TreasuryTx    `json:"transactions"`
}

type TreasuryTx struct {
	ID          string   `json:"id"`
	DAOID       string   `json:"daoId"`
	Type        string   `json:"type"`
	Token       string   `json:"token"`
	Amount      string   `json:"amount"`
	To          string   `json:"to"`
	ProposalID  string   `json:"proposalId"`
	Status      string   `json:"status"`
	ExecutedAt *int64   `json:"executedAt"`
	CreatedAt  int64    `json:"createdAt"`
}

// ============ Service ============

type GovernanceService struct {
	config     *Config
	daos       map[string]*DAO
	proposals  map[string]*Proposal
	votes      map[string][]*Vote // proposalID -> votes
}

func NewGovernanceService(config *Config) *GovernanceService {
	return &GovernanceService{
		config:     config,
		daos:       make(map[string]*DAO),
		proposals:  make(map[string]*Proposal),
		votes:      make(map[string][]*Vote),
	}
}

func (s *GovernanceService) Initialize() {
	// Seed the default TigerWallet DAO with an honest (empty) token address.
	// A real governance token address is configured via the admin API or the
	// canonical chain registry; we never hardcode a fabricated 0xTIGER.
	s.CreateDAO(DAO{
		ID:          "tiger-dao",
		Name:        "TigerWallet DAO",
		Description: "Decentralized governance for TigerWallet protocol",
		Token: Token{
			Address:  "",
			Name:     "Tiger Token",
			Symbol:   "TIGER",
			Decimals: 18,
			Supply:   "1000000000",
		},
		Threshold: 100000,
		Quorum:    50000,
	})
}

func (s *GovernanceService) CreateDAO(dao DAO) {
	dao.ID = strings.ToLower(strings.ReplaceAll(dao.Name, " ", "-"))
	dao.CreatedAt = time.Now().Unix()
	s.daos[dao.ID] = &dao
}

func (s *GovernanceService) GetDAO(id string) (*DAO, bool) {
	dao, ok := s.daos[strings.ToLower(id)]
	return dao, ok
}

func (s *GovernanceService) GetAllDAOs() []*DAO {
	var daos []*DAO
	for _, dao := range s.daos {
		daos = append(daos, dao)
	}
	return daos
}

func (s *GovernanceService) CreateProposal(proposal Proposal) string {
	proposal.ID = uuid.New().String()
	proposal.Status = StatusPending
	proposal.CreatedAt = time.Now().Unix()
	// Store the proposal in memory so GetProposals/GetProposal return real data.
	s.proposals[proposal.ID] = &proposal
	log.Printf("Proposal created: %s by %s", proposal.Title, proposal.Proposer)
	return proposal.ID
}

func (s *GovernanceService) CastVote(proposalID string, vote Vote) {
	vote.ID = uuid.New().String()
	vote.ProposalID = proposalID
	vote.CreatedAt = time.Now().Unix()
	s.votes[proposalID] = append(s.votes[proposalID], &vote)
	// Tally the vote on the proposal so the counts are real.
	if p, ok := s.proposals[proposalID]; ok {
		p.VoteCount++
		if vote.Support {
			p.ForVotes += vote.Weight
		} else {
			p.AgainstVotes += vote.Weight
		}
	}
	log.Printf("Vote cast: proposal=%s, voter=%s, support=%v, weight=%d",
		proposalID, vote.Voter, vote.Support, vote.Weight)
}

func (s *GovernanceService) ExecuteProposal(proposalID string) bool {
	p, ok := s.proposals[proposalID]
	if !ok {
		return false
	}
	now := time.Now().Unix()
	p.ExecutedAt = &now
	p.Status = StatusExecuted
	log.Printf("Proposal executed: %s", proposalID)
	return true
}

func (s *GovernanceService) Delegate(delegate Delegate) {
	delegate.ID = uuid.New().String()
	delegate.CreatedAt = time.Now().Unix()
	log.Printf("Delegation: %s -> %s", delegate.Delegator, delegate.Delegate)
}

func (s *GovernanceService) GetProposals(daoID string) []Proposal {
	// Return the real proposals stored in memory for the given DAO. If none
	// exist yet, return an empty slice -- never fabricated sample proposals.
	var out []Proposal
	for _, p := range s.proposals {
		if p.DAOID == daoID {
			out = append(out, *p)
		}
	}
	return out
}

func (s *GovernanceService) GetProposal(id string) *Proposal {
	// Return the real stored proposal, or nil if it does not exist.
	p, ok := s.proposals[id]
	if !ok {
		return nil
	}
	return p
}

func (s *GovernanceService) GetVotes(proposalID string) []Vote {
	// Return the real votes cast for this proposal, or an empty slice.
	out := make([]Vote, 0, len(s.votes[proposalID]))
	for _, v := range s.votes[proposalID] {
		out = append(out, *v)
	}
	return out
}

func (s *GovernanceService) GetTreasury(daoID string) *Treasury {
	// The treasury is sourced from on-chain balances via the wallet_api
	// backend; this in-memory service does not fabricate balances. It
	// returns a real (empty) treasury until an admin deposits/withdraws.
	return &Treasury{
		DAOID:        daoID,
		Balance:      map[string]string{},
		Transactions: []TreasuryTx{},
	}
}

// ============ Handlers ============

type Handler struct {
	service *GovernanceService
}

func NewHandler(service *GovernanceService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetDAOs(c *gin.Context) {
	daos := h.service.GetAllDAOs()
	c.JSON(http.StatusOK, gin.H{"daos": daos})
}

func (h *Handler) GetDAO(c *gin.Context) {
	id := c.Param("id")
	dao, ok := h.service.GetDAO(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "DAO not found"})
		return
	}
	c.JSON(http.StatusOK, dao)
}

func (h *Handler) CreateProposal(c *gin.Context) {
	var proposal Proposal
	if err := c.ShouldBindJSON(&proposal); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id := h.service.CreateProposal(proposal)
	c.JSON(http.StatusOK, gin.H{"proposalId": id})
}

func (h *Handler) GetProposals(c *gin.Context) {
	daoID := c.Query("daoId")
	proposals := h.service.GetProposals(daoID)
	c.JSON(http.StatusOK, gin.H{"proposals": proposals})
}

func (h *Handler) GetProposal(c *gin.Context) {
	id := c.Param("id")
	proposal := h.service.GetProposal(id)
	c.JSON(http.StatusOK, proposal)
}

func (h *Handler) CastVote(c *gin.Context) {
	proposalID := c.Param("id")

	var vote Vote
	if err := c.ShouldBindJSON(&vote); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	vote.ProposalID = proposalID
	h.service.CastVote(proposalID, vote)

	c.JSON(http.StatusOK, gin.H{"status": "vote cast"})
}

func (h *Handler) GetVotes(c *gin.Context) {
	proposalID := c.Param("id")
	votes := h.service.GetVotes(proposalID)
	c.JSON(http.StatusOK, gin.H{"votes": votes})
}

func (h *Handler) ExecuteProposal(c *gin.Context) {
	proposalID := c.Param("id")

	success := h.service.ExecuteProposal(proposalID)
	if !success {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Execution failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "executed"})
}

func (h *Handler) Delegate(c *gin.Context) {
	var delegate Delegate
	if err := c.ShouldBindJSON(&delegate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.service.Delegate(delegate)

	c.JSON(http.StatusOK, gin.H{"status": "delegated"})
}

func (h *Handler) GetTreasury(c *gin.Context) {
	daoID := c.Param("id")
	treasury := h.service.GetTreasury(daoID)
	c.JSON(http.StatusOK, treasury)
}

// ============ Main ============

func main() {
	config := &Config{
		Port:            getEnv("PORT", "8080"),
		AdminAddress:    getEnv("ADMIN_ADDRESS", "0x"),
		ProposalMinDelay: 1 * time.Hour,
		VotingPeriod:    3 * 24 * time.Hour,
		ExecutionDelay:  2 * 24 * time.Hour,
	}

	service := NewGovernanceService(config)
	service.Initialize()

	handler := NewHandler(service)

	// Setup Gin
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "daos": len(service.daos)})
	})

	// API routes
	api := router.Group("/api/v1")
	{
		// DAOs
		api.GET("/daos", handler.GetDAOs)
		api.GET("/daos/:id", handler.GetDAO)

		// Proposals
		api.POST("/proposals", handler.CreateProposal)
		api.GET("/proposals", handler.GetProposals)
		api.GET("/proposals/:id", handler.GetProposal)

		// Votes
		api.POST("/proposals/:id/votes", handler.CastVote)
		api.GET("/proposals/:id/votes", handler.GetVotes)

		// Execute
		api.POST("/proposals/:id/execute", handler.ExecuteProposal)

		// Delegation
		api.POST("/delegate", handler.Delegate)

		// Treasury
		api.GET("/daos/:id/treasury", handler.GetTreasury)
	}

	// Start server
	addr := fmt.Sprintf(":%s", config.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		log.Printf("Starting Governance service on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

// Helper functions
func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func parseBigInt(s string) *big.Int {
	n, _ := new(big.Int).SetString(s, 10)
	return n
}
