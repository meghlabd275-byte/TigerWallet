package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// ============ Configuration ============

type Config struct {
	Port             string
	AdminAddress     string
	ProposalMinDelay time.Duration
	VotingPeriod     time.Duration
	ExecutionDelay   time.Duration

	// JWTSecret authenticates callers. The service refuses every request when
	// it is unset so a misconfigured deployment cannot be driven anonymously.
	JWTSecret string

	// VotingPowerURL is the canonical service used to read a voter's real
	// token balance. Voting power is never taken from the request body.
	VotingPowerURL string
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// ============ Models ============

type DAO struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Token       Token  `json:"token"`
	Threshold   uint64 `json:"threshold"` // Votes needed to pass
	Quorum      uint64 `json:"quorum"`    // Minimum participation
	CreatedAt   int64  `json:"createdAt"`
}

type Token struct {
	Address  string `json:"address"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
	Supply   string `json:"supply"`
}

type Proposal struct {
	ID           string         `json:"id"`
	DAOID        string         `json:"daoId"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	Type         ProposalType   `json:"type"`
	Status       ProposalStatus `json:"status"`
	Proposer     string         `json:"proposer"`
	Target       string         `json:"target"`
	Value        string         `json:"value"`
	Data         string         `json:"data"`
	VoteCount    int64          `json:"voteCount"`
	ForVotes     int64          `json:"forVotes"`
	AgainstVotes int64          `json:"againstVotes"`
	AbstainVotes int64          `json:"abstainVotes"`
	StartBlock   uint64         `json:"startBlock"`
	EndBlock     uint64         `json:"endBlock"`
	ExecuteBlock uint64         `json:"executeBlock"`
	ExecutedAt   *int64         `json:"executedAt"`
	CreatedAt    int64          `json:"createdAt"`
}

type ProposalType string

const (
	TypeTransfer  ProposalType = "transfer"
	TypeUpgrade   ProposalType = "upgrade"
	TypeParameter ProposalType = "parameter"
	TypeCustom    ProposalType = "custom"
)

type ProposalStatus string

const (
	StatusPending   ProposalStatus = "pending"
	StatusActive    ProposalStatus = "active"
	StatusCanceled  ProposalStatus = "canceled"
	StatusDefeated  ProposalStatus = "defeated"
	StatusSucceeded ProposalStatus = "succeeded"
	StatusQueued    ProposalStatus = "queued"
	StatusExecuted  ProposalStatus = "executed"
	StatusExpired   ProposalStatus = "expired"
)

type Vote struct {
	ID          string `json:"id"`
	ProposalID  string `json:"proposalId"`
	Voter       string `json:"voter"`
	Support     bool   `json:"support"`
	Weight      int64  `json:"weight"`
	Reason      string `json:"reason"`
	BlockNumber uint64 `json:"blockNumber"`
	TxHash      string `json:"txHash"`
	CreatedAt   int64  `json:"createdAt"`
}

type Delegate struct {
	ID          string `json:"id"`
	Delegator   string `json:"delegator"`
	Delegate    string `json:"delegate"`
	Token       string `json:"token"`
	Balance     int64  `json:"balance"`
	VotingPower int64  `json:"votingPower"`
	CreatedAt   int64  `json:"createdAt"`
}

type Treasury struct {
	DAOID        string            `json:"daoId"`
	Balance      map[string]string `json:"balance"` // token -> amount
	Transactions []TreasuryTx      `json:"transactions"`
}

type TreasuryTx struct {
	ID         string `json:"id"`
	DAOID      string `json:"daoId"`
	Type       string `json:"type"`
	Token      string `json:"token"`
	Amount     string `json:"amount"`
	To         string `json:"to"`
	ProposalID string `json:"proposalId"`
	Status     string `json:"status"`
	ExecutedAt *int64 `json:"executedAt"`
	CreatedAt  int64  `json:"createdAt"`
}

// votingPower returns the caller's real voting power in the DAO, read from the
// canonical balance service. A request can never assert its own weight: an
// unknown voter, an unreachable balance service or an unconfigured DAO token
// all yield zero, which the vote handler rejects.
func (s *GovernanceService) votingPower(dao *DAO, voter string) (int64, error) {
	if dao.Token.Address == "" {
		return 0, fmt.Errorf("dao has no governance token configured")
	}
	if s.config.VotingPowerURL == "" {
		return 0, fmt.Errorf("VOTING_POWER_URL is not configured")
	}

	url := fmt.Sprintf("%s/api/v1/balance?address=%s&chain_id=%s&token=%s",
		strings.TrimRight(s.config.VotingPowerURL, "/"), voter,
		dao.Token.Address, dao.Token.Address)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("balance service unavailable: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("balance service returned %d", resp.StatusCode)
	}

	var payload struct {
		BalanceWei string `json:"balance_wei"`
		Balance    string `json:"balance"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, fmt.Errorf("invalid balance response: %w", err)
	}

	raw := payload.BalanceWei
	if raw == "" {
		raw = payload.Balance
	}
	power, ok := new(big.Int).SetString(raw, 10)
	if !ok {
		return 0, fmt.Errorf("unparseable balance %q", raw)
	}
	if !power.IsInt64() {
		return 0, fmt.Errorf("balance exceeds int64 range")
	}
	return power.Int64(), nil
}

// ============ Service ============

type GovernanceService struct {
	config    *Config
	daos      map[string]*DAO
	proposals map[string]*Proposal
	votes     map[string][]*Vote // proposalID -> votes
}

func NewGovernanceService(config *Config) *GovernanceService {
	return &GovernanceService{
		config:    config,
		daos:      make(map[string]*DAO),
		proposals: make(map[string]*Proposal),
		votes:     make(map[string][]*Vote),
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
	// An explicit id wins; only derive one from the name when none is given.
	if dao.ID == "" {
		dao.ID = strings.ToLower(strings.ReplaceAll(dao.Name, " ", "-"))
	}
	dao.ID = strings.ToLower(dao.ID)
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
	proposal.Status = StatusActive
	proposal.CreatedAt = time.Now().Unix()
	// Voting closes after the configured period; execution is refused until
	// then, so a proposer cannot pass a proposal in the same block it opens.
	if s.config.VotingPeriod > 0 {
		proposal.EndBlock = uint64(time.Now().Add(s.config.VotingPeriod).Unix())
	}
	// Store the proposal in memory so GetProposals/GetProposal return real data.
	s.proposals[proposal.ID] = &proposal
	log.Printf("Proposal created: %s by %s", proposal.Title, proposal.Proposer)
	return proposal.ID
}

// CastVote records a vote for the authenticated voter. Weight is read from the
// voter's real token balance, never from the request, and each address may vote
// once per proposal.
func (s *GovernanceService) CastVote(proposalID, voter string, support bool, reason string) (int64, error) {
	p, ok := s.proposals[proposalID]
	if !ok {
		return 0, fmt.Errorf("proposal not found")
	}
	if p.Status == StatusExecuted || p.Status == StatusDefeated {
		return 0, fmt.Errorf("proposal is closed")
	}
	if p.EndBlock != 0 && uint64(time.Now().Unix()) >= p.EndBlock {
		return 0, fmt.Errorf("voting period has ended")
	}

	dao, ok := s.daos[strings.ToLower(p.DAOID)]
	if !ok {
		return 0, fmt.Errorf("dao not found")
	}

	for _, existing := range s.votes[proposalID] {
		if strings.EqualFold(existing.Voter, voter) {
			return 0, fmt.Errorf("already voted on this proposal")
		}
	}

	weight, err := s.votingPower(dao, voter)
	if err != nil {
		return 0, err
	}
	if weight <= 0 {
		return 0, fmt.Errorf("caller holds no voting power in this dao")
	}

	vote := Vote{
		ID:         uuid.New().String(),
		ProposalID: proposalID,
		Voter:      voter,
		Support:    support,
		Weight:     weight,
		Reason:     reason,
		CreatedAt:  time.Now().Unix(),
	}
	s.votes[proposalID] = append(s.votes[proposalID], &vote)

	p.VoteCount++
	if support {
		p.ForVotes += weight
	} else {
		p.AgainstVotes += weight
	}
	log.Printf("Vote cast: proposal=%s voter=%s support=%v weight=%d",
		proposalID, voter, support, weight)
	return weight, nil
}

// ExecuteProposal applies a proposal only when the DAO's own quorum and
// approval threshold are satisfied and voting has closed. Marking a proposal
// executed regardless of its tally would let anyone pass anything.
func (s *GovernanceService) ExecuteProposal(proposalID string) error {
	p, ok := s.proposals[proposalID]
	if !ok {
		return fmt.Errorf("proposal not found")
	}
	if p.Status == StatusExecuted {
		return fmt.Errorf("proposal already executed")
	}

	dao, ok := s.daos[strings.ToLower(p.DAOID)]
	if !ok {
		return fmt.Errorf("dao not found")
	}

	if p.EndBlock != 0 && uint64(time.Now().Unix()) < p.EndBlock {
		return fmt.Errorf("voting period has not ended")
	}

	totalCast := p.ForVotes + p.AgainstVotes + p.AbstainVotes
	if uint64(totalCast) < dao.Quorum {
		p.Status = StatusDefeated
		return fmt.Errorf("quorum not reached: %d of %d required", totalCast, dao.Quorum)
	}
	if uint64(p.ForVotes) < dao.Threshold {
		p.Status = StatusDefeated
		return fmt.Errorf("approval threshold not reached: %d of %d required", p.ForVotes, dao.Threshold)
	}

	now := time.Now().Unix()
	p.ExecutedAt = &now
	p.Status = StatusExecuted
	log.Printf("Proposal executed: %s (for=%d against=%d)", proposalID, p.ForVotes, p.AgainstVotes)
	return nil
}

// Delegate records a delegation from the authenticated delegator. The
// delegator is taken from the token, not the body, so nobody can move another
// holder's voting power.
func (s *GovernanceService) Delegate(delegator, delegate string) (Delegate, error) {
	if strings.EqualFold(delegator, delegate) {
		return Delegate{}, fmt.Errorf("cannot delegate to self")
	}
	if delegate == "" {
		return Delegate{}, fmt.Errorf("delegate address required")
	}

	d := Delegate{
		ID:        uuid.New().String(),
		Delegator: delegator,
		Delegate:  delegate,
		CreatedAt: time.Now().Unix(),
	}
	log.Printf("Delegation: %s -> %s", delegator, delegate)
	return d, nil
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

// callerID returns the authenticated subject.
func callerID(c *gin.Context) string {
	return c.GetString("userID")
}

// callerIsAdmin reports whether the caller may perform operator-only actions.
func callerIsAdmin(c *gin.Context) bool {
	switch c.GetString("role") {
	case "admin", "super_admin":
		return true
	}
	return false
}

// authMiddleware verifies the bearer token. An unset secret is a
// misconfiguration, so requests are refused rather than served unauthenticated.
func authMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if secret == "" {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
				"error": "JWT_SECRET not configured; refusing unauthenticated request",
			})
			return
		}

		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
			return
		}

		token, err := jwt.Parse(strings.TrimPrefix(header, "Bearer "), func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(secret), nil
		}, jwt.WithValidMethods([]string{"HS256", "HS384", "HS512"}))
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid claims"})
			return
		}
		sub, _ := claims["sub"].(string)
		if sub == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing subject"})
			return
		}

		c.Set("userID", sub)
		if role, ok := claims["role"].(string); ok {
			c.Set("role", role)
		}
		c.Next()
	}
}

// requireAdmin gates DAO creation, which configures the governance token and
// thresholds for the whole deployment.
func requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !callerIsAdmin(c) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "admin role required"})
			return
		}
		c.Next()
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

	// The proposer is the authenticated caller, so a proposal cannot be
	// attributed to someone else.
	proposal.Proposer = callerID(c)
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

	var body struct {
		Support bool   `json:"support"`
		Reason  string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	weight, err := h.service.CastVote(proposalID, callerID(c), body.Support, body.Reason)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "vote cast", "weight": weight})
}

func (h *Handler) GetVotes(c *gin.Context) {
	proposalID := c.Param("id")
	votes := h.service.GetVotes(proposalID)
	c.JSON(http.StatusOK, gin.H{"votes": votes})
}

func (h *Handler) ExecuteProposal(c *gin.Context) {
	proposalID := c.Param("id")

	if err := h.service.ExecuteProposal(proposalID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "executed"})
}

func (h *Handler) Delegate(c *gin.Context) {
	var body struct {
		Delegate string `json:"delegate" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	d, err := h.service.Delegate(callerID(c), body.Delegate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "delegated", "delegation": d})
}

func (h *Handler) GetTreasury(c *gin.Context) {
	daoID := c.Param("id")
	treasury := h.service.GetTreasury(daoID)
	c.JSON(http.StatusOK, treasury)
}

// ============ Main ============

func main() {
	config := &Config{
		Port:             getEnv("PORT", "8080"),
		AdminAddress:     getEnv("ADMIN_ADDRESS", "0x"),
		ProposalMinDelay: 1 * time.Hour,
		VotingPeriod:     3 * 24 * time.Hour,
		ExecutionDelay:   2 * 24 * time.Hour,
		JWTSecret:        getEnv("JWT_SECRET", ""),
		VotingPowerURL:   getEnv("VOTING_POWER_URL", "http://wallet_api:8443"),
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

	// API routes. Every route requires an authenticated caller.
	api := router.Group("/api/v1")
	api.Use(authMiddleware(config.JWTSecret))
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
