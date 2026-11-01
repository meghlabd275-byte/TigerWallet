package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

const govTestSecret = "governance-test-secret"

// balanceStub serves the canonical balance shape so voting power can be read
// without a live wallet backend.
func balanceStub(t *testing.T, balances map[string]string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		addr := r.URL.Query().Get("address")
		raw, ok := balances[addr]
		if !ok {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"balance": "0", "balance_wei": "0"})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"balance": raw, "balance_wei": raw})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func newGovServer(t *testing.T, balances map[string]string) (*gin.Engine, *GovernanceService, func(sub, role string) string) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	stub := balanceStub(t, balances)
	cfg := &Config{
		Port:           "0",
		VotingPeriod:   time.Hour,
		JWTSecret:      govTestSecret,
		VotingPowerURL: stub.URL,
	}
	svc := NewGovernanceService(cfg)
	svc.Initialize()
	// Give the seeded DAO a governance token so voting power can be resolved.
	dao, _ := svc.GetDAO("tiger-dao")
	dao.Token.Address = "0xTIGER000000000000000000000000000000000001"
	dao.Threshold = 100
	dao.Quorum = 50

	h := NewHandler(svc)

	mint := func(sub, role string) string {
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": sub, "role": role, "exp": time.Now().Add(time.Hour).Unix(),
		})
		s, err := tok.SignedString([]byte(govTestSecret))
		if err != nil {
			t.Fatal(err)
		}
		return s
	}

	r := gin.New()
	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })
	api := r.Group("/api/v1")
	api.Use(authMiddleware(cfg.JWTSecret))
	{
		api.GET("/daos", h.GetDAOs)
		api.GET("/daos/:id", h.GetDAO)
		api.POST("/proposals", h.CreateProposal)
		api.GET("/proposals", h.GetProposals)
		api.GET("/proposals/:id", h.GetProposal)
		api.POST("/proposals/:id/votes", h.CastVote)
		api.GET("/proposals/:id/votes", h.GetVotes)
		api.POST("/proposals/:id/execute", h.ExecuteProposal)
		api.POST("/delegate", h.Delegate)
		api.GET("/daos/:id/treasury", h.GetTreasury)
	}
	return r, svc, mint
}

func govReq(t *testing.T, r *gin.Engine, method, path, token string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// TestGovernanceAnonymousRejected proves the API is no longer open.
func TestGovernanceAnonymousRejected(t *testing.T) {
	r, _, _ := newGovServer(t, nil)
	for _, tc := range []struct{ m, p string }{
		{http.MethodGet, "/api/v1/daos"},
		{http.MethodPost, "/api/v1/proposals"},
		{http.MethodPost, "/api/v1/proposals/x/votes"},
		{http.MethodPost, "/api/v1/proposals/x/execute"},
		{http.MethodPost, "/api/v1/delegate"},
	} {
		if w := govReq(t, r, tc.m, tc.p, "", nil); w.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s without token returned %d, want 401", tc.m, tc.p, w.Code)
		}
	}
}

// TestGovernanceFailsClosedWithoutSecret ensures a misconfigured deployment
// does not serve unauthenticated traffic.
func TestGovernanceFailsClosedWithoutSecret(t *testing.T) {
	gin.SetMode(gin.TestMode)
	svc := NewGovernanceService(&Config{})
	h := NewHandler(svc)
	r := gin.New()
	api := r.Group("/api/v1")
	api.Use(authMiddleware(""))
	api.GET("/daos", h.GetDAOs)

	if w := govReq(t, r, http.MethodGet, "/api/v1/daos", "anything", nil); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing secret returned %d, want 503", w.Code)
	}
}

// TestExecuteRejectedWithoutVotes is the core regression guard: a proposal with
// no votes must not be executable.
func TestExecuteRejectedWithoutVotes(t *testing.T) {
	r, svc, mint := newGovServer(t, nil)
	token := mint("proposer", "user")

	w := govReq(t, r, http.MethodPost, "/api/v1/proposals", token, map[string]interface{}{
		"daoId": "tiger-dao", "title": "Drain treasury", "description": "x",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("create proposal returned %d: %s", w.Code, w.Body.String())
	}
	var created struct {
		ProposalID string `json:"proposalId"`
	}
	json.Unmarshal(w.Body.Bytes(), &created)

	// Force the voting window closed so only the tally is under test.
	p := svc.GetProposal(created.ProposalID)
	p.EndBlock = uint64(time.Now().Add(-time.Hour).Unix())

	ex := govReq(t, r, http.MethodPost, "/api/v1/proposals/"+created.ProposalID+"/execute", token, nil)
	if ex.Code == http.StatusOK {
		t.Fatalf("proposal executed with zero votes: %s", ex.Body.String())
	}
	if p.Status != StatusDefeated {
		t.Fatalf("proposal status is %q after a failed execution, want %q", p.Status, StatusDefeated)
	}
}

// TestVoteWeightCannotBeSelfAsserted proves a voter cannot inflate their power
// through the request body.
func TestVoteWeightCannotBeSelfAsserted(t *testing.T) {
	r, svc, mint := newGovServer(t, map[string]string{"0xrealvoter": "10"})

	token := mint("0xrealvoter", "user")
	w := govReq(t, r, http.MethodPost, "/api/v1/proposals", token, map[string]interface{}{
		"daoId": "tiger-dao", "title": "P", "description": "d",
	})
	var created struct {
		ProposalID string `json:"proposalId"`
	}
	json.Unmarshal(w.Body.Bytes(), &created)

	// The body claims a weight of 999999; the real balance is 10.
	vote := govReq(t, r, http.MethodPost, "/api/v1/proposals/"+created.ProposalID+"/votes", token,
		map[string]interface{}{"support": true, "weight": 999999})
	if vote.Code != http.StatusOK {
		t.Fatalf("vote returned %d: %s", vote.Code, vote.Body.String())
	}

	p := svc.GetProposal(created.ProposalID)
	if p.ForVotes != 10 {
		t.Fatalf("ForVotes is %d, want 10 (the real balance); the body weight was trusted", p.ForVotes)
	}
}

// TestVoterWithoutBalanceRejected proves a zero-balance address cannot vote.
func TestVoterWithoutBalanceRejected(t *testing.T) {
	r, _, mint := newGovServer(t, map[string]string{"0xrich": "1000"})

	token := mint("0xpoor", "user")
	w := govReq(t, r, http.MethodPost, "/api/v1/proposals", token, map[string]interface{}{
		"daoId": "tiger-dao", "title": "P", "description": "d",
	})
	var created struct {
		ProposalID string `json:"proposalId"`
	}
	json.Unmarshal(w.Body.Bytes(), &created)

	vote := govReq(t, r, http.MethodPost, "/api/v1/proposals/"+created.ProposalID+"/votes", token,
		map[string]interface{}{"support": true})
	if vote.Code == http.StatusOK {
		t.Fatal("a zero-balance address was allowed to vote")
	}
}

// TestDoubleVoteRejected proves one address votes once per proposal.
func TestDoubleVoteRejected(t *testing.T) {
	r, _, mint := newGovServer(t, map[string]string{"0xvoter": "500"})
	token := mint("0xvoter", "user")

	w := govReq(t, r, http.MethodPost, "/api/v1/proposals", token, map[string]interface{}{
		"daoId": "tiger-dao", "title": "P", "description": "d",
	})
	var created struct {
		ProposalID string `json:"proposalId"`
	}
	json.Unmarshal(w.Body.Bytes(), &created)

	path := "/api/v1/proposals/" + created.ProposalID + "/votes"
	if w := govReq(t, r, http.MethodPost, path, token, map[string]interface{}{"support": true}); w.Code != http.StatusOK {
		t.Fatalf("first vote returned %d", w.Code)
	}
	if w := govReq(t, r, http.MethodPost, path, token, map[string]interface{}{"support": true}); w.Code == http.StatusOK {
		t.Fatal("a second vote from the same address was accepted")
	}
}

// TestQuorumEnforced proves a proposal below quorum cannot execute.
func TestQuorumEnforced(t *testing.T) {
	r, svc, mint := newGovServer(t, map[string]string{"0xsmall": "10"})
	token := mint("0xsmall", "user")

	w := govReq(t, r, http.MethodPost, "/api/v1/proposals", token, map[string]interface{}{
		"daoId": "tiger-dao", "title": "P", "description": "d",
	})
	var created struct {
		ProposalID string `json:"proposalId"`
	}
	json.Unmarshal(w.Body.Bytes(), &created)

	govReq(t, r, http.MethodPost, "/api/v1/proposals/"+created.ProposalID+"/votes", token,
		map[string]interface{}{"support": true})

	p := svc.GetProposal(created.ProposalID)
	p.EndBlock = uint64(time.Now().Add(-time.Hour).Unix())

	ex := govReq(t, r, http.MethodPost, "/api/v1/proposals/"+created.ProposalID+"/execute", token, nil)
	if ex.Code == http.StatusOK {
		t.Fatalf("proposal executed below quorum: %s", ex.Body.String())
	}
}

// TestExecuteRejectedBeforeVotingEnds proves a proposer cannot pass a proposal
// immediately.
func TestExecuteRejectedBeforeVotingEnds(t *testing.T) {
	r, _, mint := newGovServer(t, map[string]string{"0xwhale": "100000"})
	token := mint("0xwhale", "user")

	w := govReq(t, r, http.MethodPost, "/api/v1/proposals", token, map[string]interface{}{
		"daoId": "tiger-dao", "title": "P", "description": "d",
	})
	var created struct {
		ProposalID string `json:"proposalId"`
	}
	json.Unmarshal(w.Body.Bytes(), &created)

	govReq(t, r, http.MethodPost, "/api/v1/proposals/"+created.ProposalID+"/votes", token,
		map[string]interface{}{"support": true})

	// VotingPeriod is one hour, so execution must still be refused.
	ex := govReq(t, r, http.MethodPost, "/api/v1/proposals/"+created.ProposalID+"/execute", token, nil)
	if ex.Code == http.StatusOK {
		t.Fatalf("proposal executed before the voting period ended: %s", ex.Body.String())
	}
}

// TestProposalPassesWithQuorumAndThreshold proves the happy path still works.
func TestProposalPassesWithQuorumAndThreshold(t *testing.T) {
	r, svc, mint := newGovServer(t, map[string]string{"0xwhale": "100000"})
	token := mint("0xwhale", "user")

	w := govReq(t, r, http.MethodPost, "/api/v1/proposals", token, map[string]interface{}{
		"daoId": "tiger-dao", "title": "P", "description": "d",
	})
	var created struct {
		ProposalID string `json:"proposalId"`
	}
	json.Unmarshal(w.Body.Bytes(), &created)

	govReq(t, r, http.MethodPost, "/api/v1/proposals/"+created.ProposalID+"/votes", token,
		map[string]interface{}{"support": true})

	p := svc.GetProposal(created.ProposalID)
	p.EndBlock = uint64(time.Now().Add(-time.Hour).Unix())

	ex := govReq(t, r, http.MethodPost, "/api/v1/proposals/"+created.ProposalID+"/execute", token, nil)
	if ex.Code != http.StatusOK {
		t.Fatalf("valid proposal failed to execute: %s", ex.Body.String())
	}
	if p.Status != StatusExecuted {
		t.Fatalf("status is %q, want executed", p.Status)
	}
}

// TestDelegateCannotImpersonate proves the delegator comes from the token.
func TestDelegateCannotImpersonate(t *testing.T) {
	r, _, mint := newGovServer(t, nil)
	token := mint("0xattacker", "user")

	// The body claims to act for 0xvictim; the caller is 0xattacker.
	w := govReq(t, r, http.MethodPost, "/api/v1/delegate", token, map[string]interface{}{
		"delegator": "0xvictim",
		"delegate":  "0xdelegatee",
	})
	if w.Code != http.StatusOK {
		t.Fatalf("delegate returned %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Delegation Delegate `json:"delegation"`
	}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Delegation.Delegator != "0xattacker" {
		t.Fatalf("delegator recorded as %q, want the authenticated caller", resp.Delegation.Delegator)
	}
}

// TestSelfDelegationRejected covers the trivial misconfiguration.
func TestSelfDelegationRejected(t *testing.T) {
	r, _, mint := newGovServer(t, nil)
	w := govReq(t, r, http.MethodPost, "/api/v1/delegate", mint("0xsame", "user"),
		map[string]interface{}{"delegate": "0xsame"})
	if w.Code == http.StatusOK {
		t.Fatal("self delegation was accepted")
	}
}
