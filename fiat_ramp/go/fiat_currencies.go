package main

import (
	"sync"
)

// Supported fiat currencies
type FiatCurrency struct {
	Code         string  `json:"code"`
	Name         string  `json:"name"`
	Symbol       string  `json:"symbol"`
	Decimals     int     `json:"decimals"`
	MinAmount    float64 `json:"minAmount"`
	MaxAmount    float64 `json:"maxAmount"`
	Supported    bool    `json:"supported"`
}

type FiatCurrencyService struct {
	mu          sync.RWMutex
	currencies  map[string]*FiatCurrency
}

func NewFiatCurrencyService() *FiatCurrencyService {
	svc := &FiatCurrencyService{
		currencies: make(map[string]*FiatCurrency),
	}
	svc.initCurrencies()
	return svc
}

func (s *FiatCurrencyService) initCurrencies() {
	currencies := []*FiatCurrency{
		// Major Currencies
		{Code: "USD", Name: "US Dollar", Symbol: "$", Decimals: 2, MinAmount: 10, MaxAmount: 100000, Supported: true},
		{Code: "EUR", Name: "Euro", Symbol: "€", Decimals: 2, MinAmount: 10, MaxAmount: 100000, Supported: true},
		{Code: "GBP", Name: "British Pound", Symbol: "£", Decimals: 2, MinAmount: 10, MaxAmount: 100000, Supported: true},
		{Code: "JPY", Name: "Japanese Yen", Symbol: "¥", Decimals: 0, MinAmount: 1000, MaxAmount: 10000000, Supported: true},
		{Code: "CHF", Name: "Swiss Franc", Symbol: "Fr", Decimals: 2, MinAmount: 10, MaxAmount: 100000, Supported: true},
		{Code: "CAD", Name: "Canadian Dollar", Symbol: "C$", Decimals: 2, MinAmount: 10, MaxAmount: 100000, Supported: true},
		{Code: "AUD", Name: "Australian Dollar", Symbol: "A$", Decimals: 2, MinAmount: 10, MaxAmount: 100000, Supported: true},
		
		// European
		{Code: "SEK", Name: "Swedish Krona", Symbol: "kr", Decimals: 2, MinAmount: 100, MaxAmount: 100000, Supported: true},
		{Code: "NOK", Name: "Norwegian Krone", Symbol: "kr", Decimals: 2, MinAmount: 100, MaxAmount: 100000, Supported: true},
		{Code: "DKK", Name: "Danish Krone", Symbol: "kr", Decimals: 2, MinAmount: 100, MaxAmount: 100000, Supported: true},
		{Code: "PLN", Name: "Polish Zloty", Symbol: "zł", Decimals: 2, MinAmount: 50, MaxAmount: 50000, Supported: true},
		{Code: "CZK", Name: "Czech Koruna", Symbol: "Kč", Decimals: 2, MinAmount: 250, MaxAmount: 50000, Supported: true},
		{Code: "HUF", Name: "Hungarian Forint", Symbol: "Ft", Decimals: 0, MinAmount: 3500, MaxAmount: 3500000, Supported: true},
		
		// Asia Pacific
		{Code: "KRW", Name: "South Korean Won", Symbol: "₩", Decimals: 0, MinAmount: 10000, MaxAmount: 100000000, Supported: true},
		{Code: "CNY", Name: "Chinese Yuan", Symbol: "¥", Decimals: 2, MinAmount: 70, MaxAmount: 700000, Supported: true},
		{Code: "HKD", Name: "Hong Kong Dollar", Symbol: "HK$", Decimals: 2, MinAmount: 80, MaxAmount: 800000, Supported: true},
		{Code: "SGD", Name: "Singapore Dollar", Symbol: "S$", Decimals: 2, MinAmount: 15, MaxAmount: 150000, Supported: true},
		{Code: "NZD", Name: "New Zealand Dollar", Symbol: "NZ$", Decimals: 2, MinAmount: 15, MaxAmount: 150000, Supported: true},
		{Code: "INR", Name: "Indian Rupee", Symbol: "₹", Decimals: 2, MinAmount: 750, MaxAmount: 750000, Supported: true},
		{Code: "MYR", Name: "Malaysian Ringgit", Symbol: "RM", Decimals: 2, MinAmount: 40, MaxAmount: 400000, Supported: true},
		{Code: "THB", Name: "Thai Baht", Symbol: "฿", Decimals: 2, MinAmount: 350, MaxAmount: 3500000, Supported: true},
		{Code: "IDR", Name: "Indonesian Rupiah", Symbol: "Rp", Decimals: 0, MinAmount: 150000, MaxAmount: 1500000000, Supported: true},
		{Code: "PHP", Name: "Philippine Peso", Symbol: "₱", Decimals: 2, MinAmount: 550, MaxAmount: 5500000, Supported: true},
		{Code: "VND", Name: "Vietnamese Dong", Symbol: "₫", Decimals: 0, MinAmount: 230000, MaxAmount: 2300000000, Supported: true},
		{Code: "TWD", Name: "Taiwan Dollar", Symbol: "NT$", Decimals: 2, MinAmount: 300, MaxAmount: 3000000, Supported: true},
		
		// Latin America
		{Code: "BRL", Name: "Brazilian Real", Symbol: "R$", Decimals: 2, MinAmount: 50, MaxAmount: 500000, Supported: true},
		{Code: "MXN", Name: "Mexican Peso", Symbol: "$", Decimals: 2, MinAmount: 200, MaxAmount: 2000000, Supported: true},
		{Code: "ARS", Name: "Argentine Peso", Symbol: "$", Decimals: 2, MinAmount: 9000, MaxAmount: 90000000, Supported: true},
		{Code: "CLP", Name: "Chilean Peso", Symbol: "$", Decimals: 0, MinAmount: 9000, MaxAmount: 90000000, Supported: true},
		{Code: "COP", Name: "Colombian Peso", Symbol: "$", Decimals: 0, MinAmount: 40000, MaxAmount: 400000000, Supported: true},
		{Code: "PEN", Name: "Peruvian Sol", Symbol: "S/", Decimals: 2, MinAmount: 40, MaxAmount: 400000, Supported: true},
		
		// Middle East & Africa
		{Code: "AED", Name: "UAE Dirham", Symbol: "د.إ", Decimals: 2, MinAmount: 40, MaxAmount: 400000, Supported: true},
		{Code: "SAR", Name: "Saudi Riyal", Symbol: "﷼", Decimals: 2, MinAmount: 40, MaxAmount: 400000, Supported: true},
		{Code: "ILS", Name: "Israeli Shekel", Symbol: "₪", Decimals: 2, MinAmount: 40, MaxAmount: 400000, Supported: true},
		{Code: "TRY", Name: "Turkish Lira", Symbol: "₺", Decimals: 2, MinAmount: 300, MaxAmount: 3000000, Supported: true},
		{Code: "ZAR", Name: "South African Rand", Symbol: "R", Decimals: 2, MinAmount: 200, MaxAmount: 2000000, Supported: true},
		{Code: "NGN", Name: "Nigerian Naira", Symbol: "₦", Decimals: 2, MinAmount: 15000, MaxAmount: 150000000, Supported: true},
		{Code: "EGP", Name: "Egyptian Pound", Symbol: "£", Decimals: 2, MinAmount: 500, MaxAmount: 5000000, Supported: true},
		{Code: "PKR", Name: "Pakistani Rupee", Symbol: "₨", Decimals: 0, MinAmount: 3000, MaxAmount: 30000000, Supported: true},
		{Code: "BDT", Name: "Bangladeshi Taka", Symbol: "৳", Decimals: 2, MinAmount: 1100, MaxAmount: 11000000, Supported: true},
	}

	for _, c := range currencies {
		s.currencies[c.Code] = c
	}
}

func (s *FiatCurrencyService) GetCurrency(code string) (*FiatCurrency, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if c, ok := s.currencies[code]; ok {
		return c, nil
	}
	return nil, nil
}

func (s *FiatCurrencyService) ListCurrencies() []*FiatCurrency {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var result []*FiatCurrency
	for _, c := range s.currencies {
		if c.Supported {
			result = append(result, c)
		}
	}
	return result
}