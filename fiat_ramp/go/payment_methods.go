package main

import (
	"sync"
)

// Payment method types
type PaymentMethod struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	SupportedCurrencies []string `json:"supportedCurrencies"`
	MinAmount   float64  `json:"minAmount"`
	MaxAmount   float64  `json:"maxAmount"`
	FeePercent  float64  `json:"feePercent"`
	FeeFixed   float64  `json:"feeFixed"`
	ProcessingTime string  `json:"processingTime"`
	Enabled    bool     `json:"enabled"`
}

type PaymentMethodService struct {
	mu     sync.RWMutex
	methods map[string]*PaymentMethod
}

func NewPaymentMethodService() *PaymentMethodService {
	svc := &PaymentMethodService{
		methods: make(map[string]*PaymentMethod),
	}
	svc.initMethods()
	return svc
}

func (s *PaymentMethodService) initMethods() {
	methods := []*PaymentMethod{
		// Credit/Debit Cards
		{
			ID: "credit_card", Name: "Credit Card", Type: "card",
			Description: "Visa, Mastercard, American Express",
			SupportedCurrencies: []string{"USD", "EUR", "GBP"},
			MinAmount: 10, MaxAmount: 50000,
			FeePercent: 2.99, FeeFixed: 0.30,
			ProcessingTime: "instant", Enabled: true,
		},
		{
			ID: "debit_card", Name: "Debit Card", Type: "card",
			Description: "Visa Electron, Mastercard Debit",
			SupportedCurrencies: []string{"USD", "EUR", "GBP"},
			MinAmount: 10, MaxAmount: 25000,
			FeePercent: 2.99, FeeFixed: 0.30,
			ProcessingTime: "instant", Enabled: true,
		},
		
		// Bank Transfers
		{
			ID: "sepa", Name: "SEPA Bank Transfer", Type: "bank",
			Description: "European bank transfer (EUR)",
			SupportedCurrencies: []string{"EUR"},
			MinAmount: 50, MaxAmount: 500000,
			FeePercent: 0, FeeFixed: 0,
			ProcessingTime: "1-2 business days", Enabled: true,
		},
		{
			ID: "swift", Name: "SWIFT International Wire", Type: "bank",
			Description: "International wire transfer",
			SupportedCurrencies: []string{"USD", "EUR", "GBP", "CHF"},
			MinAmount: 1000, MaxAmount: 1000000,
			FeePercent: 0.01, FeeFixed: 25,
			ProcessingTime: "2-5 business days", Enabled: true,
		},
		{
			ID: "ach", Name: "ACH Bank Transfer", Type: "bank",
			Description: "US bank transfer (USD)",
			SupportedCurrencies: []string{"USD"},
			MinAmount: 10, MaxAmount: 100000,
			FeePercent: 0, FeeFixed: 0,
			ProcessingTime: "1-3 business days", Enabled: true,
		},
		{
			ID: "faster_payments", Name: "Faster Payments", Type: "bank",
			Description: "UK bank transfer (GBP)",
			SupportedCurrencies: []string{"GBP"},
			MinAmount: 10, MaxAmount: 100000,
			FeePercent: 0, FeeFixed: 0,
			ProcessingTime: "same day", Enabled: true,
		},
		
		// E-Wallets
		{
			ID: "apple_pay", Name: "Apple Pay", Type: "wallet",
			Description: "Apple Pay",
			SupportedCurrencies: []string{"USD", "EUR", "GBP"},
			MinAmount: 5, MaxAmount: 10000,
			FeePercent: 1.5, FeeFixed: 0,
			ProcessingTime: "instant", Enabled: true,
		},
		{
			ID: "google_pay", Name: "Google Pay", Type: "wallet",
			Description: "Google Pay",
			SupportedCurrencies: []string{"USD", "EUR", "GBP"},
			MinAmount: 5, MaxAmount: 10000,
			FeePercent: 1.5, FeeFixed: 0,
			ProcessingTime: "instant", Enabled: true,
		},
		{
			ID: "paypal", Name: "PayPal", Type: "wallet",
			Description: "PayPal",
			SupportedCurrencies: []string{"USD", "EUR", "GBP"},
			MinAmount: 5, MaxAmount: 10000,
			FeePercent: 3.5, FeeFixed: 0,
			ProcessingTime: "instant", Enabled: true,
		},
		{
			ID: "skrill", Name: "Skrill", Type: "wallet",
			Description: "Skrill e-wallet",
			SupportedCurrencies: []string{"USD", "EUR", "GBP"},
			MinAmount: 5, MaxAmount: 10000,
			FeePercent: 3.5, FeeFixed: 0,
			ProcessingTime: "instant", Enabled: true,
		},
		{
			ID: "neteller", Name: "NETELLER", Type: "wallet",
			Description: "NETELLER e-wallet",
			SupportedCurrencies: []string{"USD", "EUR", "GBP"},
			MinAmount: 5, MaxAmount: 10000,
			FeePercent: 3.5, FeeFixed: 0,
			ProcessingTime: "instant", Enabled: true,
		},
		
		// Regional Methods
		{
			ID: "poli", Name: "POLi", Type: "regional",
			Description: "Australia/New Zealand bank transfer",
			SupportedCurrencies: []string{"AUD", "NZD"},
			MinAmount: 10, MaxAmount: 10000,
			FeePercent: 0, FeeFixed: 0,
			ProcessingTime: "instant", Enabled: true,
		},
		{
			ID: "bancontact", Name: "Bancontact", Type: "regional",
			Description: "Belgium bank transfer",
			SupportedCurrencies: []string{"EUR"},
			MinAmount: 10, MaxAmount: 10000,
			FeePercent: 0, FeeFixed: 0,
			ProcessingTime: "instant", Enabled: true,
		},
		{
			ID: "ideal", Name: "iDEAL", Type: "regional",
			Description: "Netherlands bank transfer",
			SupportedCurrencies: []string{"EUR"},
			MinAmount: 10, MaxAmount: 10000,
			FeePercent: 0, FeeFixed: 0,
			ProcessingTime: "instant", Enabled: true,
		},
		{
			ID: "giropay", Name: "Giropay", Type: "regional",
			Description: "Germany bank transfer",
			SupportedCurrencies: []string{"EUR"},
			MinAmount: 10, MaxAmount: 10000,
			FeePercent: 0, FeeFixed: 0,
			ProcessingTime: "instant", Enabled: true,
		},
		{
			ID: "sofort", Name: "Sofort", Type: "regional",
			Description: "Europe bank transfer",
			SupportedCurrencies: []string{"EUR"},
			MinAmount: 10, MaxAmount: 10000,
			FeePercent: 0, FeeFixed: 0,
			ProcessingTime: "1-2 business days", Enabled: true,
		},
		{
			ID: "blik", Name: "BLIK", Type: "regional",
			Description: "Poland mobile payment",
			SupportedCurrencies: []string{"PLN"},
			MinAmount: 10, MaxAmount: 5000,
			FeePercent: 0, FeeFixed: 0,
			ProcessingTime: "instant", Enabled: true,
		},
		{
			ID: "upi", Name: "UPI", Type: "regional",
			Description: "India instant payment",
			SupportedCurrencies: []string{"INR"},
			MinAmount: 100, MaxAmount: 100000,
			FeePercent: 0, FeeFixed: 0,
			ProcessingTime: "instant", Enabled: true,
		},
		{
			ID: "pix", Name: "PIX", Type: "regional",
			Description: "Brazil instant payment",
			SupportedCurrencies: []string{"BRL"},
			MinAmount: 10, MaxAmount: 50000,
			FeePercent: 0, FeeFixed: 0,
			ProcessingTime: "instant", Enabled: true,
		},
		
		// Crypto-native
		{
			ID: "crypto_wallet", Name: "Crypto Wallet", Type: "crypto",
			Description: "Direct crypto transfer",
			SupportedCurrencies: []string{"BTC", "ETH", "USDT", "USDC"},
			MinAmount: 50, MaxAmount: 1000000,
			FeePercent: 0.5, FeeFixed: 0,
			ProcessingTime: "10-30 minutes", Enabled: true,
		},
	}

	for _, m := range methods {
		s.methods[m.ID] = m
	}
}

func (s *PaymentMethodService) GetMethod(id string) (*PaymentMethod, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if m, ok := s.methods[id]; ok {
		return m, nil
	}
	return nil, nil
}

func (s *PaymentMethodService) ListMethods() []*PaymentMethod {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var result []*PaymentMethod
	for _, m := range s.methods {
		if m.Enabled {
			result = append(result, m)
		}
	}
	return result
}

func (s *PaymentMethodService) GetMethodsByCurrency(currency string) []*PaymentMethod {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var result []*PaymentMethod
	for _, m := range s.methods {
		if !m.Enabled {
			continue
		}
		for _, c := range m.SupportedCurrencies {
			if c == currency {
				result = append(result, m)
				break
			}
		}
	}
	return result
}