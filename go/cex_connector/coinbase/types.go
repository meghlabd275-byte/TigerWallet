package main

import "time"

// Shared CEX connector types used by the Coinbase connector demo.

type Side string

const (
	BUY  Side = "BUY"
	SELL Side = "SELL"
)

type OrderType string

const (
	LIMIT  OrderType = "LIMIT"
	MARKET OrderType = "MARKET"
)

type OrderStatus string

const (
	ORDER_NEW      OrderStatus = "NEW"
	ORDER_FILLED   OrderStatus = "FILLED"
	ORDER_CANCELED OrderStatus = "CANCELED"
)

type Order struct {
	OrderID     string      `json:"orderId"`
	Symbol      string      `json:"symbol"`
	Side        Side        `json:"side"`
	Type        OrderType   `json:"type"`
	Price       float64     `json:"price"`
	OriginalQty float64     `json:"originalQty"`
	ExecutedQty float64     `json:"executedQty"`
	Status      OrderStatus `json:"status"`
	CreateTime  int64       `json:"createTime"`
}

type Balance struct {
	Asset string  `json:"asset"`
	Free  float64 `json:"free"`
	Total float64 `json:"total"`
}

type RateLimiter struct {
	rate     int
	interval time.Duration
}

func NewRateLimiter(rate int, interval time.Duration) *RateLimiter {
	return &RateLimiter{rate: rate, interval: interval}
}
