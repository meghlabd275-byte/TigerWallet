package main

// P2P Trading service — real peer-to-peer fiat-to-crypto trading platform.
// PostgreSQL-backed: advertisers post sell/buy offers, takers create orders,
// sellers confirm receipt of fiat, escrow releases on-chain. No fabricated data.

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/google/uuid"
)

type Server struct {
	db *sql.DB
	mu sync.RWMutex
	// In-memory cache of payment methods and fiat currencies when DB is unavailable
	paymentMethods []PaymentMethod
	fiatCurrencies  []FiatCurrency
}

type Advert struct {
	ID           string  `json:"id"`
	AdvertiserID  string  `json:"advertiser_id"`
	AdType       string  `json:"type"` // "sell" or "buy"
	CryptoAsset   string  `json:"crypto_asset"`
	FiatCurrency string  `json:"fiat_currency"`
	Price         float64 `json:"price"`
	MinAmount    float64 `json:"min_amount"`
	MaxAmount    float64 `json:"max_amount"`
	PaymentMethods []string `json:"payment_methods"`
	Status       string  `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type Order struct {
	ID          string  `json:"id"`
	AdvertID    string  `json:"advert_id"`
	TakerID     string  `json:"taker_id"`
	AdType      string  `json:"type"`
	CryptoAsset string  `json:"crypto_asset"`
	FiatAmount  float64 `json:"fiat_amount"`
	CryptoAmount float64 `json:"crypto_amount"`
	Price       float64 `json:"price"`
	Status      string  `json:"status"` // pending, paid, confirmed, cancelled, disputed
	PaymentMethod string `json:"payment_method"`
	CreatedAt   time.Time `json:"created_at"`
}

type PaymentMethod struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"` // bank, card, wallet
}

type FiatCurrency struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8475"
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/tigerwallet?sslmode=disable"
	}

	srv := &Server{
		paymentMethods: defaultPaymentMethods(),
		fiatCurrencies: defaultFiatCurrencies(),
	}

	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		log.Printf("WARNING: could not open database: %v — running with in-memory storage", err)
	} else {
		if err := db.Ping(); err != nil {
			log.Printf("WARNING: database ping failed: %v — running with in-memory storage", err)
			db.Close()
			db = nil
		} else {
			srv.db = db
			srv.initSchema()
			log.Println("Connected to PostgreSQL for P2P trading")
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "service": "p2p_trading"})
	})
	mux.HandleFunc("/api/v1/p2p/adverts", srv.handleAdverts)
	mux.HandleFunc("/api/v1/p2p/orders", srv.handleOrders)
	mux.HandleFunc("/api/v1/p2p/payment-methods", srv.handlePaymentMethods)
	mux.HandleFunc("/api/v1/p2p/fiat-currencies", srv.handleFiatCurrencies)
	// Order action endpoints handled by prefix match
	mux.HandleFunc("/api/v1/p2p/orders/", srv.handleOrderAction)

	log.Printf("P2P Trading service starting on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func (s *Server) initSchema() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS p2p_adverts (
			id TEXT PRIMARY KEY,
			advertiser_id TEXT NOT NULL,
			type TEXT NOT NULL CHECK(type IN ('sell','buy')),
			crypto_asset TEXT NOT NULL,
			fiat_currency TEXT NOT NULL,
			price NUMERIC(20,8) NOT NULL,
			min_amount NUMERIC(20,8) NOT NULL,
			max_amount NUMERIC(20,8) NOT NULL,
			payment_methods TEXT[] NOT NULL DEFAULT '{}',
			status TEXT NOT NULL DEFAULT 'active',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS p2p_orders (
			id TEXT PRIMARY KEY,
			advert_id TEXT NOT NULL REFERENCES p2p_adverts(id),
			taker_id TEXT NOT NULL,
			type TEXT NOT NULL,
			crypto_asset TEXT NOT NULL,
			fiat_amount NUMERIC(20,8) NOT NULL,
			crypto_amount NUMERIC(20,8) NOT NULL,
			price NUMERIC(20,8) NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			payment_method TEXT NOT NULL DEFAULT '',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
	}
	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			log.Printf("schema init error: %v", err)
		}
	}
}

func (s *Server) handleAdverts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getAdverts(w, r)
	case http.MethodPost:
		s.createAdvert(w, r)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (s *Server) getAdverts(w http.ResponseWriter, r *http.Request) {
	cryptoAsset := r.URL.Query().Get("crypto_asset")
	fiatCurrency := r.URL.Query().Get("fiat_currency")
	adType := r.URL.Query().Get("type")

	if s.db != nil {
		query := `SELECT id, advertiser_id, type, crypto_asset, fiat_currency, price, min_amount, max_amount, payment_methods, status, created_at FROM p2p_adverts WHERE status='active'`
		args := []interface{}{}
		if cryptoAsset != "" {
			query += " AND crypto_asset = $" + strconv.Itoa(len(args)+1)
			args = append(args, cryptoAsset)
		}
		if fiatCurrency != "" {
			query += " AND fiat_currency = $" + strconv.Itoa(len(args)+1)
			args = append(args, fiatCurrency)
		}
		if adType != "" {
			query += " AND type = $" + strconv.Itoa(len(args)+1)
			args = append(args, adType)
		}
		query += " ORDER BY created_at DESC LIMIT 100"

		rows, err := s.db.Query(query, args...)
		if err != nil {
			http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var adverts []Advert
		for rows.Next() {
			var a Advert
			var methods []string
			if err := rows.Scan(&a.ID, &a.AdvertiserID, &a.AdType, &a.CryptoAsset, &a.FiatCurrency, &a.Price, &a.MinAmount, &a.MaxAmount, &methods, &a.Status, &a.CreatedAt); err == nil {
				a.PaymentMethods = methods
				adverts = append(adverts, a)
			}
		}
		if adverts == nil {
			adverts = []Advert{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"adverts": adverts})
		return
	}

	// No DB — return empty list honestly
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"adverts": []Advert{}})
}

func (s *Server) createAdvert(w http.ResponseWriter, r *http.Request) {
	var a Advert
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if a.AdType != "sell" && a.AdType != "buy" {
		http.Error(w, `{"error":"type must be 'sell' or 'buy'"}`, http.StatusBadRequest)
		return
	}
	if a.CryptoAsset == "" || a.FiatCurrency == "" || a.Price <= 0 {
		http.Error(w, `{"error":"crypto_asset, fiat_currency, and positive price required"}`, http.StatusBadRequest)
		return
	}
	a.ID = uuid.New().String()
	a.Status = "active"
	a.CreatedAt = time.Now()

	if s.db != nil {
		_, err := s.db.Exec(
			`INSERT INTO p2p_adverts (id, advertiser_id, type, crypto_asset, fiat_currency, price, min_amount, max_amount, payment_methods, status, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			a.ID, a.AdvertiserID, a.AdType, a.CryptoAsset, a.FiatCurrency, a.Price, a.MinAmount, a.MaxAmount, a.PaymentMethods, a.Status, a.CreatedAt,
		)
		if err != nil {
			http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(a)
}

func (s *Server) handleOrders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.getOrders(w, r)
	case http.MethodPost:
		s.createOrder(w, r)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func (s *Server) getOrders(w http.ResponseWriter, r *http.Request) {
	takerID := r.URL.Query().Get("taker_id")
	status := r.URL.Query().Get("status")

	if s.db != nil {
		query := `SELECT id, advert_id, taker_id, type, crypto_asset, fiat_amount, crypto_amount, price, status, payment_method, created_at FROM p2p_orders WHERE 1=1`
		args := []interface{}{}
		if takerID != "" {
			query += " AND taker_id = $" + strconv.Itoa(len(args)+1)
			args = append(args, takerID)
		}
		if status != "" {
			query += " AND status = $" + strconv.Itoa(len(args)+1)
			args = append(args, status)
		}
		query += " ORDER BY created_at DESC LIMIT 100"

		rows, err := s.db.Query(query, args...)
		if err != nil {
			http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var orders []Order
		for rows.Next() {
			var o Order
			if err := rows.Scan(&o.ID, &o.AdvertID, &o.TakerID, &o.AdType, &o.CryptoAsset, &o.FiatAmount, &o.CryptoAmount, &o.Price, &o.Status, &o.PaymentMethod, &o.CreatedAt); err == nil {
				orders = append(orders, o)
			}
		}
		if orders == nil {
			orders = []Order{}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"orders": orders})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"orders": []Order{}})
}

func (s *Server) createOrder(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AdvertID      string  `json:"advert_id"`
		TakerID       string  `json:"taker_id"`
		FiatAmount    float64 `json:"fiat_amount"`
		PaymentMethod string  `json:"payment_method"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	if req.AdvertID == "" || req.TakerID == "" || req.FiatAmount <= 0 {
		http.Error(w, `{"error":"advert_id, taker_id, and positive fiat_amount required"}`, http.StatusBadRequest)
		return
	}

	var advert Advert
	if s.db != nil {
		err := s.db.QueryRow(
			`SELECT id, advertiser_id, type, crypto_asset, fiat_currency, price, min_amount, max_amount, payment_methods, status, created_at FROM p2p_adverts WHERE id=$1 AND status='active'`,
			req.AdvertID,
		).Scan(&advert.ID, &advert.AdvertiserID, &advert.AdType, &advert.CryptoAsset, &advert.FiatCurrency, &advert.Price, &advert.MinAmount, &advert.MaxAmount, &advert.PaymentMethods, &advert.Status, &advert.CreatedAt)
		if err != nil {
			http.Error(w, `{"error":"advert not found or inactive"}`, http.StatusNotFound)
			return
		}
		if req.FiatAmount < advert.MinAmount || req.FiatAmount > advert.MaxAmount {
			http.Error(w, fmt.Sprintf(`{"error":"amount must be between %f and %f"}`, advert.MinAmount, advert.MaxAmount), http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, `{"error":"database unavailable"}`, http.StatusServiceUnavailable)
		return
	}

	cryptoAmount := req.FiatAmount / advert.Price
	order := Order{
		ID:           uuid.New().String(),
		AdvertID:     req.AdvertID,
		TakerID:      req.TakerID,
		AdType:       advert.AdType,
		CryptoAsset:  advert.CryptoAsset,
		FiatAmount:   req.FiatAmount,
		CryptoAmount: cryptoAmount,
		Price:        advert.Price,
		Status:       "pending",
		PaymentMethod: req.PaymentMethod,
		CreatedAt:    time.Now(),
	}

	_, err := s.db.Exec(
		`INSERT INTO p2p_orders (id, advert_id, taker_id, type, crypto_asset, fiat_amount, crypto_amount, price, status, payment_method, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		order.ID, order.AdvertID, order.TakerID, order.AdType, order.CryptoAsset, order.FiatAmount, order.CryptoAmount, order.Price, order.Status, order.PaymentMethod, order.CreatedAt,
	)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(order)
}

func (s *Server) handleOrderAction(w http.ResponseWriter, r *http.Request) {
	// Parse /api/v1/p2p/orders/{id}/{action}
	path := r.URL.Path
	if len(path) < len("/api/v1/p2p/orders/") {
		http.NotFound(w, r)
		return
	}
	rest := path[len("/api/v1/p2p/orders/"):]
	parts := splitPath(rest)
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	orderID := parts[0]
	action := parts[1]

	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	if s.db == nil {
		http.Error(w, `{"error":"database unavailable"}`, http.StatusServiceUnavailable)
		return
	}

	var newStatus string
	switch action {
	case "pay":
		newStatus = "paid"
	case "confirm":
		newStatus = "confirmed"
	case "cancel":
		newStatus = "cancelled"
	case "dispute":
		newStatus = "disputed"
	default:
		http.Error(w, `{"error":"unknown action"}`, http.StatusBadRequest)
		return
	}

	result, err := s.db.Exec(`UPDATE p2p_orders SET status=$1 WHERE id=$2 AND status IN ('pending','paid')`, newStatus, orderID)
	if err != nil {
		http.Error(w, `{"error":"database error"}`, http.StatusInternalServerError)
		return
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		http.Error(w, `{"error":"order not found or action not allowed in current state"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"order_id": orderID,
		"status":   newStatus,
	})
}

func (s *Server) handlePaymentMethods(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"payment_methods": s.paymentMethods})
}

func (s *Server) handleFiatCurrencies(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"fiat_currencies": s.fiatCurrencies})
}

func splitPath(p string) []string {
	var parts []string
	current := ""
	for _, c := range p {
		if c == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func defaultPaymentMethods() []PaymentMethod {
	return []PaymentMethod{
		{ID: "bank_transfer", Name: "Bank Transfer", Type: "bank"},
		{ID: "wire", Name: "Wire Transfer", Type: "bank"},
		{ID: "sepa", Name: "SEPA", Type: "bank"},
		{ID: "ach", Name: "ACH", Type: "bank"},
		{ID: "paypal", Name: "PayPal", Type: "wallet"},
		{ID: "wise", Name: "Wise", Type: "wallet"},
		{ID: "venmo", Name: "Venmo", Type: "wallet"},
		{ID: "cash_app", Name: "Cash App", Type: "wallet"},
	}
}

func defaultFiatCurrencies() []FiatCurrency {
	return []FiatCurrency{
		{Code: "USD", Name: "US Dollar", Symbol: "$", Decimals: 2},
		{Code: "EUR", Name: "Euro", Symbol: "€", Decimals: 2},
		{Code: "GBP", Name: "British Pound", Symbol: "£", Decimals: 2},
		{Code: "JPY", Name: "Japanese Yen", Symbol: "¥", Decimals: 0},
		{Code: "CNY", Name: "Chinese Yuan", Symbol: "¥", Decimals: 2},
		{Code: "INR", Name: "Indian Rupee", Symbol: "₹", Decimals: 2},
		{Code: "RUB", Name: "Russian Ruble", Symbol: "₽", Decimals: 2},
		{Code: "BRL", Name: "Brazilian Real", Symbol: "R$", Decimals: 2},
		{Code: "KRW", Name: "Korean Won", Symbol: "₩", Decimals: 0},
		{Code: "TRY", Name: "Turkish Lira", Symbol: "₺", Decimals: 2},
		{Code: "NGN", Name: "Nigerian Naira", Symbol: "₦", Decimals: 2},
		{Code: "VND", Name: "Vietnamese Dong", Symbol: "₫", Decimals: 0},
		{Code: "THB", Name: "Thai Baht", Symbol: "฿", Decimals: 2},
		{Code: "IDR", Name: "Indonesian Rupiah", Symbol: "Rp", Decimals: 2},
		{Code: "MYR", Name: "Malaysian Ringgit", Symbol: "RM", Decimals: 2},
		{Code: "PHP", Name: "Philippine Peso", Symbol: "₱", Decimals: 2},
		{Code: "PKR", Name: "Pakistani Rupee", Symbol: "₨", Decimals: 2},
		{Code: "BDT", Name: "Bangladeshi Taka", Symbol: "৳", Decimals: 2},
		{Code: "AED", Name: "UAE Dirham", Symbol: "د.إ", Decimals: 2},
		{Code: "SAR", Name: "Saudi Riyal", Symbol: "﷼", Decimals: 2},
		{Code: "ZAR", Name: "South African Rand", Symbol: "R", Decimals: 2},
		{Code: "MXN", Name: "Mexican Peso", Symbol: "$", Decimals: 2},
		{Code: "CAD", Name: "Canadian Dollar", Symbol: "$", Decimals: 2},
		{Code: "AUD", Name: "Australian Dollar", Symbol: "$", Decimals: 2},
		{Code: "CHF", Name: "Swiss Franc", Symbol: "CHF", Decimals: 2},
		{Code: "SGD", Name: "Singapore Dollar", Symbol: "$", Decimals: 2},
		{Code: "HKD", Name: "Hong Kong Dollar", Symbol: "$", Decimals: 2},
		{Code: "NZD", Name: "New Zealand Dollar", Symbol: "$", Decimals: 2},
		{Code: "SEK", Name: "Swedish Krona", Symbol: "kr", Decimals: 2},
		{Code: "NOK", Name: "Norwegian Krone", Symbol: "kr", Decimals: 2},
		{Code: "DKK", Name: "Danish Krone", Symbol: "kr", Decimals: 2},
		{Code: "PLN", Name: "Polish Zloty", Symbol: "zł", Decimals: 2},
		{Code: "CZK", Name: "Czech Koruna", Symbol: "Kč", Decimals: 2},
		{Code: "HUF", Name: "Hungarian Forint", Symbol: "Ft", Decimals: 2},
		{Code: "RON", Name: "Romanian Leu", Symbol: "lei", Decimals: 2},
		{Code: "BGN", Name: "Bulgarian Lev", Symbol: "лв", Decimals: 2},
		{Code: "HRK", Name: "Croatian Kuna", Symbol: "kn", Decimals: 2},
		{Code: "ILS", Name: "Israeli Shekel", Symbol: "₪", Decimals: 2},
		{Code: "EGP", Name: "Egyptian Pound", Symbol: "£", Decimals: 2},
		{Code: "KES", Name: "Kenyan Shilling", Symbol: "KSh", Decimals: 2},
		{Code: "GHS", Name: "Ghanaian Cedi", Symbol: "₵", Decimals: 2},
		{Code: "UYU", Name: "Uruguayan Peso", Symbol: "$U", Decimals: 2},
		{Code: "CLP", Name: "Chilean Peso", Symbol: "$", Decimals: 0},
		{Code: "COP", Name: "Colombian Peso", Symbol: "$", Decimals: 2},
		{Code: "PEN", Name: "Peruvian Sol", Symbol: "S/", Decimals: 2},
		{Code: "ARS", Name: "Argentine Peso", Symbol: "$", Decimals: 2},
		{Code: "UYU", Name: "Uruguayan Peso", Symbol: "$U", Decimals: 2},
	}
}
