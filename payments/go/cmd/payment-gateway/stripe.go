// stripe.go — real card tokenization, payment capture and refunds.
//
// The gateway previously minted a fake provider reference and marked payments
// completed without contacting anyone. Every function here calls the payment
// provider and returns its response; when the provider is not configured the
// call fails and the transaction stays pending, which is the recoverable state.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	// errProviderNotConfigured is returned when no provider key is set.
	errProviderNotConfigured = errors.New("payment provider is not configured (STRIPE_SECRET_KEY)")
	// errCardDeclined is returned when the provider refuses the card.
	errCardDeclined = errors.New("card was declined by the payment provider")
)

// providerTimeout bounds a single provider call.
const providerTimeout = 30 * time.Second

// StripeClient talks to the Stripe API.
type StripeClient struct {
	secretKey string
	baseURL   string
	client    *http.Client
}

// NewStripeClient builds a client from the environment. It returns an error
// when no secret key is configured so callers cannot proceed unconfigured.
func NewStripeClient() (*StripeClient, error) {
	secretKey := strings.TrimSpace(os.Getenv("STRIPE_SECRET_KEY"))
	if secretKey == "" {
		return nil, errProviderNotConfigured
	}

	baseURL := strings.TrimSpace(os.Getenv("STRIPE_API_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://api.stripe.com"
	}

	return &StripeClient{
		secretKey: secretKey,
		baseURL:   strings.TrimRight(baseURL, "/"),
		client:    &http.Client{Timeout: providerTimeout},
	}, nil
}

// stripeError is the error object Stripe returns on a failed call.
type stripeError struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// stripeEnvelope wraps every Stripe response.
type stripeEnvelope struct {
	ID      string          `json:"id"`
	Status  string          `json:"status"`
	Paid    bool            `json:"paid"`
	Object  string          `json:"object"`
	Error   *stripeError    `json:"error"`
	RawJSON json.RawMessage `json:"-"`
}

// call posts form values to an endpoint and decodes the response.
func (c *StripeClient) call(ctx context.Context, method, path string, form url.Values) (*stripeEnvelope, []byte, error) {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, nil, fmt.Errorf("build provider request: %w", err)
	}
	req.SetBasicAuth(c.secretKey, "")
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("call payment provider: %w", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, nil, fmt.Errorf("read provider response: %w", err)
	}

	var envelope stripeEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, payload, fmt.Errorf("decode provider response (status %d): %w", resp.StatusCode, err)
	}
	if envelope.Error != nil {
		return &envelope, payload, fmt.Errorf("%w: %s (%s)", errCardDeclined, envelope.Error.Message, envelope.Error.Code)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return &envelope, payload, fmt.Errorf("payment provider returned status %d", resp.StatusCode)
	}
	return &envelope, payload, nil
}

// TokenizeCard exchanges raw card details for a provider token.
//
// The raw number, expiry and CVC are sent straight to the provider and are never
// stored by this service: only the returned token and the display fields are
// persisted.
func (c *StripeClient) TokenizeCard(ctx context.Context, number, expMonth, expYear, cvc string) (string, error) {
	form := url.Values{}
	form.Set("card[number]", number)
	form.Set("card[exp_month]", expMonth)
	form.Set("card[exp_year]", expYear)
	form.Set("card[cvc]", cvc)

	envelope, _, err := c.call(ctx, http.MethodPost, "/v1/tokens", form)
	if err != nil {
		return "", err
	}
	if envelope.ID == "" {
		return "", errors.New("payment provider returned no token")
	}
	return envelope.ID, nil
}

// CreateCustomer creates a provider customer to attach payment methods to.
func (c *StripeClient) CreateCustomer(ctx context.Context, userID uint) (string, error) {
	form := url.Values{}
	form.Set("description", fmt.Sprintf("tigerwallet-user-%d", userID))

	envelope, _, err := c.call(ctx, http.MethodPost, "/v1/customers", form)
	if err != nil {
		return "", err
	}
	if envelope.ID == "" {
		return "", errors.New("payment provider returned no customer id")
	}
	return envelope.ID, nil
}

// AttachPaymentMethod attaches a tokenized card to a customer and returns the
// resulting payment method id.
func (c *StripeClient) AttachPaymentMethod(ctx context.Context, customerID, token string) (string, error) {
	form := url.Values{}
	form.Set("customer", customerID)

	envelope, _, err := c.call(ctx, http.MethodPost, "/v1/payment_methods/"+url.PathEscape(token)+"/attach", form)
	if err != nil {
		return "", err
	}
	if envelope.ID == "" {
		return "", errors.New("payment provider returned no payment method id")
	}
	return envelope.ID, nil
}

// CreateCharge captures amountMinor (in the currency's smallest unit) against a
// payment method. The returned reference and raw response come from the
// provider.
func (c *StripeClient) CreateCharge(ctx context.Context, amountMinor int64, currency, paymentMethodID, description string) (string, string, error) {
	if amountMinor <= 0 {
		return "", "", errors.New("charge amount must be positive")
	}

	form := url.Values{}
	form.Set("amount", fmt.Sprintf("%d", amountMinor))
	form.Set("currency", strings.ToLower(currency))
	form.Set("payment_method", paymentMethodID)
	form.Set("confirm", "true")
	form.Set("capture", "true")
	if description != "" {
		form.Set("description", description)
	}

	envelope, raw, err := c.call(ctx, http.MethodPost, "/v1/payment_intents", form)
	if err != nil {
		return "", string(raw), err
	}
	if envelope.ID == "" {
		return "", string(raw), errors.New("payment provider returned no payment reference")
	}
	return envelope.ID, string(raw), nil
}

// Refund issues a refund against a captured charge. An empty amount refunds the
// full charge.
func (c *StripeClient) Refund(ctx context.Context, providerRef string, amountMinor int64) (string, error) {
	form := url.Values{}
	form.Set("payment_intent", providerRef)
	if amountMinor > 0 {
		form.Set("amount", fmt.Sprintf("%d", amountMinor))
	}

	envelope, _, err := c.call(ctx, http.MethodPost, "/v1/refunds", form)
	if err != nil {
		return "", err
	}
	if envelope.ID == "" {
		return "", errors.New("payment provider returned no refund reference")
	}
	return envelope.ID, nil
}
