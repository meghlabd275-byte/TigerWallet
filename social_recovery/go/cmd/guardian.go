// Guardian invitation and verification.
//
// A guardian is only marked confirmed after presenting the one-time code that
// was delivered to the contact address on the invitation. The code is stored as
// a salted hash with an expiry and an attempt budget, and the comparison is
// constant time. When no delivery channel is configured the invitation is
// reported as undelivered and the guardian cannot be confirmed, so an
// unconfigured deployment never leaves an open takeover path.

package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/smtp"
	"strings"
	"time"
)

// guardianCodeTTL bounds how long an invitation code stays valid.
const guardianCodeTTL = 72 * time.Hour

// guardianMaxAttempts bounds how many wrong codes an invitation accepts before
// it is invalidated.
const guardianMaxAttempts = 5

// guardianVerification holds the pending confirmation state for one guardian.
type guardianVerification struct {
	WalletID   string    `json:"walletId"`
	GuardianID string    `json:"guardianId"`
	CodeHash   string    `json:"codeHash"`
	Salt       string    `json:"salt"`
	Attempts   int       `json:"attempts"`
	ExpiresAt  time.Time `json:"expiresAt"`
	CreatedAt  time.Time `json:"createdAt"`
}

// guardianCodeKey names the Redis entry holding a guardian's pending code.
func guardianCodeKey(walletID, guardianID string) string {
	return fmt.Sprintf("guardian:code:%s:%s", walletID, guardianID)
}

// generateGuardianCode returns a cryptographically random six-digit code.
func generateGuardianCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("generate guardian code: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// hashCode returns the salted SHA-256 hash of a code.
func hashCode(code, salt string) string {
	sum := sha256.Sum256([]byte(salt + ":" + code))
	return hex.EncodeToString(sum[:])
}

// issueGuardianCode creates and stores a fresh confirmation code and returns the
// plaintext code so it can be delivered to the guardian.
func (s *SocialRecoveryService) issueGuardianCode(ctx context.Context, walletID, guardianID string) (string, error) {
	if s.redis == nil {
		return "", fmt.Errorf("guardian verification requires Redis, which is not configured")
	}

	code, err := generateGuardianCode()
	if err != nil {
		return "", err
	}

	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}
	salt := hex.EncodeToString(saltBytes)

	record := guardianVerification{
		WalletID:   walletID,
		GuardianID: guardianID,
		CodeHash:   hashCode(code, salt),
		Salt:       salt,
		Attempts:   0,
		ExpiresAt:  time.Now().Add(guardianCodeTTL),
		CreatedAt:  time.Now(),
	}

	data, err := json.Marshal(record)
	if err != nil {
		return "", err
	}
	if err := s.redis.Set(ctx, guardianCodeKey(walletID, guardianID), data, guardianCodeTTL).Err(); err != nil {
		return "", fmt.Errorf("store guardian code: %w", err)
	}
	return code, nil
}

// VerifyGuardian confirms a guardian after checking the delivered code.
//
// The code must match the stored hash, be unexpired, and be within the attempt
// budget. A failure never confirms the guardian.
func (s *SocialRecoveryService) VerifyGuardian(ctx context.Context, walletID, guardianID, code string) error {
	if s.redis == nil {
		return fmt.Errorf("guardian verification is unavailable: Redis is not configured")
	}

	raw, err := s.redis.Get(ctx, guardianCodeKey(walletID, guardianID)).Bytes()
	if err != nil {
		return fmt.Errorf("no pending verification code for this guardian; request a new invitation")
	}

	var record guardianVerification
	if err := json.Unmarshal(raw, &record); err != nil {
		return fmt.Errorf("pending verification record is unreadable")
	}

	if time.Now().After(record.ExpiresAt) {
		s.redis.Del(ctx, guardianCodeKey(walletID, guardianID))
		return fmt.Errorf("the verification code has expired; request a new invitation")
	}

	if record.Attempts >= guardianMaxAttempts {
		s.redis.Del(ctx, guardianCodeKey(walletID, guardianID))
		return fmt.Errorf("too many incorrect attempts; request a new invitation")
	}

	expected := record.CodeHash
	actual := hashCode(strings.TrimSpace(code), record.Salt)
	if subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) != 1 {
		record.Attempts++
		if data, err := json.Marshal(record); err == nil {
			s.redis.Set(ctx, guardianCodeKey(walletID, guardianID), data, time.Until(record.ExpiresAt))
		}
		return fmt.Errorf("the verification code is incorrect")
	}

	wallet, err := s.GetWallet(ctx, walletID)
	if err != nil {
		return err
	}

	for i, g := range wallet.Guardians {
		if g.ID != guardianID {
			continue
		}
		now := time.Now()
		wallet.Guardians[i].Confirmed = true
		wallet.Guardians[i].ConfirmedAt = &now
		wallet.UpdatedAt = now

		if data, err := json.Marshal(wallet); err == nil {
			s.redis.Set(ctx, fmt.Sprintf("wallet:%s", wallet.ID), data, 0)
		}
		// A code is single use.
		s.redis.Del(ctx, guardianCodeKey(walletID, guardianID))
		return nil
	}

	return fmt.Errorf("guardian not found")
}

// deliverGuardianInvitation sends the confirmation code to the guardian's
// contact address and reports whether delivery succeeded.
func (s *SocialRecoveryService) deliverGuardianInvitation(ctx context.Context, guardian Guardian, walletID, code string) error {
	switch {
	case guardian.Email != "" && s.config.SMTPHost != "":
		return s.sendInvitationEmail(guardian.Email, walletID, code)
	case guardian.Phone != "" && s.config.SMSWebhookURL != "":
		return s.sendInvitationSMS(ctx, guardian.Phone, walletID, code)
	default:
		return fmt.Errorf("no delivery channel is configured for this guardian (set SMTP_HOST for email or SMS_WEBHOOK_URL for SMS)")
	}
}

// sendInvitationEmail delivers the confirmation code over SMTP.
func (s *SocialRecoveryService) sendInvitationEmail(to, walletID, code string) error {
	host := strings.TrimSpace(s.config.SMTPHost)
	addr := fmt.Sprintf("%s:%d", host, s.config.SMTPPort)

	auth := smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, host)
	from := s.config.SMTPFrom
	if from == "" {
		from = s.config.SMTPUsername
	}
	if from == "" {
		return fmt.Errorf("SMTP_FROM or SMTP_USERNAME must be set to send invitations")
	}

	body := fmt.Sprintf(
		"Subject: TigerWallet guardian confirmation\r\n"+
			"From: %s\r\n"+
			"To: %s\r\n"+
			"MIME-Version: 1.0\r\n"+
			"Content-Type: text/plain; charset=UTF-8\r\n\r\n"+
			"You were invited as a guardian for wallet %s.\r\n\r\n"+
			"Your confirmation code is: %s\r\n\r\n"+
			"This code expires in %d hours. If you did not expect this invitation, ignore this message.\r\n",
		from, to, walletID, code, int(guardianCodeTTL.Hours()))

	if err := smtp.SendMail(addr, auth, from, []string{to}, []byte(body)); err != nil {
		return fmt.Errorf("send guardian invitation email: %w", err)
	}
	return nil
}

// sendInvitationSMS delivers the confirmation code through the configured
// messaging gateway.
func (s *SocialRecoveryService) sendInvitationSMS(ctx context.Context, to, walletID, code string) error {
	endpoint := strings.TrimSpace(s.config.SMSWebhookURL)
	if !strings.HasPrefix(endpoint, "https://") {
		return fmt.Errorf("SMS_WEBHOOK_URL must use https, got %q", endpoint)
	}

	payload, err := json.Marshal(map[string]string{
		"to":      to,
		"message": fmt.Sprintf("TigerWallet guardian code for wallet %s: %s", walletID, code),
	})
	if err != nil {
		return err
	}

	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.config.SMSWebhookToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.config.SMSWebhookToken)
	}

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("send guardian invitation SMS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("SMS gateway returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// deliverRecoveryNotice sends a recovery notification to one guardian.
func (s *SocialRecoveryService) deliverRecoveryNotice(ctx context.Context, guardian Guardian, wallet Wallet, request RecoveryRequest) error {
	message := fmt.Sprintf(
		"A recovery of wallet %s was requested to new owner %s. It completes at %s. If you did not approve this, cancel it.",
		wallet.Address, request.NewOwnerAddress, request.DelayEnd.Format(time.RFC3339))

	switch {
	case guardian.Email != "" && s.config.SMTPHost != "":
		return s.sendPlainEmail(guardian.Email, "TigerWallet recovery requested", message)
	case guardian.Phone != "" && s.config.SMSWebhookURL != "":
		return s.sendRawSMS(ctx, guardian.Phone, message)
	default:
		return fmt.Errorf("no delivery channel is configured for guardian %s", guardian.ID)
	}
}

// sendPlainEmail sends a plain-text message over SMTP.
func (s *SocialRecoveryService) sendPlainEmail(to, subject, message string) error {
	host := strings.TrimSpace(s.config.SMTPHost)
	from := s.config.SMTPFrom
	if from == "" {
		from = s.config.SMTPUsername
	}
	if from == "" {
		return fmt.Errorf("SMTP_FROM or SMTP_USERNAME must be set to send mail")
	}

	body := fmt.Sprintf("Subject: %s\r\nFrom: %s\r\nTo: %s\r\nMIME-Version: 1.0\r\n"+
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n%s\r\n", subject, from, to, message)

	auth := smtp.PlainAuth("", s.config.SMTPUsername, s.config.SMTPPassword, host)
	if err := smtp.SendMail(fmt.Sprintf("%s:%d", host, s.config.SMTPPort), auth, from, []string{to}, []byte(body)); err != nil {
		return fmt.Errorf("send mail: %w", err)
	}
	return nil
}

// sendRawSMS delivers an arbitrary message through the messaging gateway.
func (s *SocialRecoveryService) sendRawSMS(ctx context.Context, to, message string) error {
	endpoint := strings.TrimSpace(s.config.SMSWebhookURL)
	if !strings.HasPrefix(endpoint, "https://") {
		return fmt.Errorf("SMS_WEBHOOK_URL must use https, got %q", endpoint)
	}

	payload, err := json.Marshal(map[string]string{"to": to, "message": message})
	if err != nil {
		return err
	}

	reqCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if s.config.SMSWebhookToken != "" {
		req.Header.Set("Authorization", "Bearer "+s.config.SMSWebhookToken)
	}

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("send SMS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("SMS gateway returned HTTP %d", resp.StatusCode)
	}
	return nil
}


// sendGuardianInvitation issues a code and delivers it. Failures are logged: the
// guardian stays unconfirmed and the operator can retry.
func (s *SocialRecoveryService) sendGuardianInvitation(ctx context.Context, walletID string, guardian Guardian) {
	code, err := s.issueGuardianCode(ctx, walletID, guardian.ID)
	if err != nil {
		log.Printf("guardian invitation for %s could not be issued: %v", guardian.ID, err)
		return
	}
	if err := s.deliverGuardianInvitation(ctx, guardian, walletID, code); err != nil {
		log.Printf("guardian invitation for %s could not be delivered: %v", guardian.ID, err)
		return
	}
	log.Printf("guardian invitation delivered to %s", guardian.ID)
}

// notifyGuardiansOfRecovery alerts confirmed guardians that a recovery has been
// requested, so they can cancel it if they did not authorise it.
func (s *SocialRecoveryService) notifyGuardiansOfRecovery(wallet Wallet, request RecoveryRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, guardian := range wallet.Guardians {
		if !guardian.Confirmed {
			continue
		}
		if err := s.deliverRecoveryNotice(ctx, guardian, wallet, request); err != nil {
			log.Printf("recovery notice for guardian %s could not be delivered: %v", guardian.ID, err)
		}
	}
}
