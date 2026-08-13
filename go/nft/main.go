package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/tigerwallet/nft/nft"
)

func main() {
	svc := nft.NewService()
	// Register built-in marketplaces so list/buy operations are operational.
	svc.RegisterMarketplace("opensea", nft.NewOpenSeaMarketplace(""))
	svc.RegisterMarketplace("magiceden", nft.NewMagicEdenMarketplace(""))

	// The route set includes both "/api/v1/nft/collections/{id}" and
	// "/api/v1/nft/{address}/tokens". These two wildcard patterns overlap
	// (both match "/api/v1/nft/collections/tokens") and neither is more
	// specific, so net/http's ServeMux refuses to register them together.
	// We therefore mount a single subtree handler per method and dispatch
	// manually, preserving the exact route semantics from the spec.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler("nft"))
	mux.HandleFunc("GET /api/v1/nft/", nftGetHandler(svc))
	mux.HandleFunc("POST /api/v1/nft/", nftPostHandler(svc))

	addr := ":" + port("8471")
	log.Printf("nft service listening on %s", addr)
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

const nftPrefix = "/api/v1/nft/"

// nftGetHandler dispatches the GET routes under /api/v1/nft/.
func nftGetHandler(svc *nft.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tail := strings.TrimPrefix(r.URL.Path, nftPrefix)
		segments := splitPath(tail)

		switch {
		case len(segments) == 1 && segments[0] == "collections":
			writeError(w, http.StatusNotImplemented,
				"not implemented: no method to enumerate all collections")
		case len(segments) == 2 && segments[0] == "collections":
			addr := segments[1]
			collection, err := svc.GetCollection(r.Context(), addr)
			if err != nil {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, collection)
		case len(segments) == 2 && segments[1] == "tokens":
			writeError(w, http.StatusNotImplemented,
				"not implemented: no method to enumerate tokens by owner (address="+segments[0]+")")
		case len(segments) == 1 && segments[0] == "listings":
			collection := r.URL.Query().Get("collection")
			listings, err := svc.GetListings(r.Context(), collection)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, listings)
		default:
			writeError(w, http.StatusNotFound, "no route for GET "+r.URL.Path)
		}
	}
}

// nftPostHandler dispatches the POST routes under /api/v1/nft/.
func nftPostHandler(svc *nft.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tail := strings.TrimPrefix(r.URL.Path, nftPrefix)
		segments := splitPath(tail)

		switch {
		case len(segments) == 1 && segments[0] == "list":
			var req struct {
				Marketplace string       `json:"marketplace"`
				Listing     *nft.Listing `json:"listing"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			if req.Marketplace == "" {
				req.Marketplace = "opensea"
			}
			if err := svc.CreateListing(r.Context(), req.Marketplace, req.Listing); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		case len(segments) == 1 && segments[0] == "buy":
			var req struct {
				Marketplace string `json:"marketplace"`
				ListingID   string `json:"listingId"`
				Buyer       string `json:"buyer"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			if req.Marketplace == "" {
				req.Marketplace = "opensea"
			}
			if err := svc.FillListing(r.Context(), req.Marketplace, req.ListingID, req.Buyer); err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		default:
			writeError(w, http.StatusNotFound, "no route for POST "+r.URL.Path)
		}
	}
}

// splitPath splits the part of the path after the API prefix into non-empty
// segments.
func splitPath(tail string) []string {
	if tail == "" {
		return nil
	}
	tail = strings.Trim(tail, "/")
	if tail == "" {
		return nil
	}
	return strings.Split(tail, "/")
}

func port(def string) string {
	if p := os.Getenv("PORT"); p != "" {
		return p
	}
	return def
}

func healthHandler(service string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": service})
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
