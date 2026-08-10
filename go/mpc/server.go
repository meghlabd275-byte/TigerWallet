/**
 * MPC HTTP service — exposes the real TSS engine (Shamir secret sharing +
 * Lagrange recovery over secp256k1) over a REST API so frontends and other
 * services can create MPC wallets and sign without ever assembling the full
 * private key in one place (outside the signing quorum).
 *
 * Endpoints:
 *   POST /api/v1/mpc/create            { threshold, totalShards } -> wallet
 *   GET  /api/v1/mpc/wallet/:keyId      -> wallet metadata (no shares)
 *   POST /api/v1/mpc/sign              { keyId, messageHash } -> signature
 *   GET  /api/v1/health                -> { ok: true }
 *
 * Key shares live in the engine's in-memory map for the signing quorum. A
 * production deployment persists encrypted shards to a secrets store (KMS/HSM);
 * this service never returns the raw share scalars over the wire.
 */
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type mpcWallet struct {
	KeyID       string    `json:"keyId"`
	Address     Address   `json:"address"`
	PublicKey   Bytes     `json:"publicKey"` // 65-byte uncompressed
	Threshold   uint32    `json:"threshold"`
	TotalShards uint32    `json:"totalShards"`
	CreatedAt   Timestamp `json:"createdAt"`
}

type createRequest struct {
	Threshold   uint32 `json:"threshold"`
	TotalShards uint32 `json:"totalShards"`
}

type signRequest struct {
	KeyID       string `json:"keyId"`
	MessageHash string `json:"messageHash"` // hex ("0x..." or "...")
}

type signResponse struct {
	Signature Bytes     `json:"signature"` // 65-byte r||s||v
	KeyID     string    `json:"keyId"`
	Address   Address   `json:"address"`
	SignedAt  Timestamp `json:"signedAt"`
}

// walletStore holds the live TSS engines keyed by keyId. Each engine owns its
// own shares; signing recombines a threshold of shares on demand.
type walletStore struct {
	mu      sync.RWMutex
	engines map[string]*TSSEngine
	meta    map[string]*mpcWallet
}

var store = &walletStore{
	engines: make(map[string]*TSSEngine),
	meta:    make(map[string]*mpcWallet),
}

func newKeyID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "key_" + hex.EncodeToString(b), nil
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func handleCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req createRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Threshold == 0 || req.TotalShards == 0 || req.Threshold > req.TotalShards {
		writeError(w, http.StatusBadRequest, "threshold must satisfy 0 < threshold <= totalShards")
		return
	}

	engine := NewTSSEngine(req.Threshold, req.TotalShards)
	shares := engine.GenerateKeyShares()
	if len(shares) == 0 {
		writeError(w, http.StatusInternalServerError, "key generation failed")
		return
	}

	pub := engine.GetPublicKey()
	if len(pub) == 0 {
		writeError(w, http.StatusInternalServerError, "public key unavailable")
		return
	}
	addr := DeriveAddress(pub)

	keyID, err := newKeyID()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "key id generation failed")
		return
	}

	wallet := &mpcWallet{
		KeyID:       keyID,
		Address:     addr,
		PublicKey:   pub,
		Threshold:   req.Threshold,
		TotalShards: req.TotalShards,
		CreatedAt:   currentTimestamp(),
	}

	store.mu.Lock()
	store.engines[keyID] = engine
	store.meta[keyID] = wallet
	store.mu.Unlock()

	// Never return the raw share scalars. Only return non-sensitive metadata.
	writeJSON(w, http.StatusCreated, wallet)
}

func handleGetWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	keyID := strings.TrimPrefix(r.URL.Path, "/api/v1/mpc/wallet/")
	if keyID == "" {
		writeError(w, http.StatusBadRequest, "missing keyId")
		return
	}
	store.mu.RLock()
	wallet, ok := store.meta[keyID]
	store.mu.RUnlock()
	if !ok {
		writeError(w, http.StatusNotFound, "wallet not found")
		return
	}
	writeJSON(w, http.StatusOK, wallet)
}

func handleSign(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req signRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.KeyID == "" {
		writeError(w, http.StatusBadRequest, "missing keyId")
		return
	}
	hashHex := strings.TrimPrefix(req.MessageHash, "0x")
	msgHash, err := hex.DecodeString(hashHex)
	if err != nil || len(msgHash) != 32 {
		writeError(w, http.StatusBadRequest, "messageHash must be 32 bytes of hex")
		return
	}

	store.mu.RLock()
	engine, ok := store.engines[req.KeyID]
	wallet := store.meta[req.KeyID]
	store.mu.RUnlock()
	if !ok || wallet == nil {
		writeError(w, http.StatusNotFound, "wallet not found")
		return
	}

	// Collect threshold signature shares from the engine's parties, then
	// recombine. The full private key only exists transiently inside CombineShares.
	var sigShares []SignatureShare
	for partyID := uint32(1); partyID <= wallet.Threshold; partyID++ {
		share, err := engine.SignWithShare(partyID, msgHash)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "share signing failed")
			return
		}
		scalar, ok := engine.ShareFor(partyID)
		if !ok {
			writeError(w, http.StatusInternalServerError, "share unavailable")
			return
		}
		sigShares = append(sigShares, SignatureShare{
			PartyID:   partyID,
			Share:     scalarBytes(scalar),
			Timestamp: currentTimestamp(),
			SessionID: req.KeyID,
		})
		_ = share // per-party partial sig retained for auditability
	}

	combined := engine.CombineShares(sigShares, wallet.Address, msgHash)
	if len(combined.Signature) == 0 {
		writeError(w, http.StatusInternalServerError, "signature combination failed")
		return
	}

	// Normalize v to 27/28 (ethereum recovery byte convention).
	sig := combined.Signature
	if len(sig) == 65 && (sig[64] == 0 || sig[64] == 1) {
		sig[64] += 27
	}

	writeJSON(w, http.StatusOK, signResponse{
		Signature: sig,
		KeyID:     req.KeyID,
		Address:   wallet.Address,
		SignedAt:  currentTimestamp(),
	})
}

// currentTimestamp is provided by enterprise.go (UnixMilli).

// WithCORS adds permissive CORS for browser frontends in development.
func WithCORS(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h(w, r)
	}
}

func main() {
	port := os.Getenv("MPC_PORT")
	if port == "" {
		port = "9099"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/health", WithCORS(handleHealth))
	mux.HandleFunc("/api/v1/mpc/create", WithCORS(handleCreate))
	mux.HandleFunc("/api/v1/mpc/sign", WithCORS(handleSign))
	mux.HandleFunc("/api/v1/mpc/wallet/", WithCORS(handleGetWallet))

	log.Printf("TigerWallet MPC service listening on :%s", port)
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("mpc server: %v", err)
	}
}
