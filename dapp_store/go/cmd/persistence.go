package main

// persistence.go — Redis-backed storage for DApp submissions and reviews.
//
// Submissions and reviews are durable: a submission returns an id that can be
// read back, and a review is stored before it is acknowledged. When Redis is
// unreachable every write and read returns an error rather than a plausible
// empty result, so a client is never told a review was accepted when nothing
// was stored.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// ErrStoreUnavailable is returned when the backing store cannot be reached.
var ErrStoreUnavailable = errors.New("dapp store backend is unavailable")

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("record not found")

const (
	submissionKeyPrefix = "dappstore:submission:"
	submissionIndexKey  = "dappstore:submissions"
	reviewKeyPrefix     = "dappstore:review:"
	dappReviewIndex     = "dappstore:dappreviews:"
	reviewIndexKey      = "dappstore:reviews"
	storeTimeout        = 5 * time.Second
)

// Store persists submissions and reviews.
type Store struct {
	rdb *redis.Client
}

// NewStore connects to Redis and verifies the connection so a misconfigured
// endpoint fails at startup rather than on the first user request.
func NewStore(ctx context.Context, redisURL string) (*Store, error) {
	opts, err := redis.ParseURL(normalizeRedisURL(redisURL))
	if err != nil {
		return nil, fmt.Errorf("invalid Redis URL: %w", err)
	}
	client := redis.NewClient(opts)
	pingCtx, cancel := context.WithTimeout(ctx, storeTimeout)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		return nil, fmt.Errorf("connect to Redis: %w", err)
	}
	return &Store{rdb: client}, nil
}

// Close releases the Redis connection.
func (s *Store) Close() error {
	if s.rdb == nil {
		return nil
	}
	return s.rdb.Close()
}

// normalizeRedisURL accepts either a full redis:// URL or a bare host:port.
func normalizeRedisURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "redis://localhost:6379"
	}
	if strings.Contains(raw, "://") {
		return raw
	}
	return "redis://" + raw
}

// SaveSubmission stores a submission and indexes it.
func (s *Store) SaveSubmission(ctx context.Context, sub DAppSubmission) error {
	if s.rdb == nil {
		return ErrStoreUnavailable
	}
	payload, err := json.Marshal(sub)
	if err != nil {
		return fmt.Errorf("encode submission: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, storeTimeout)
	defer cancel()

	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, submissionKeyPrefix+sub.ID, payload, 0)
	pipe.ZAdd(ctx, submissionIndexKey, redis.Z{
		Score:  float64(sub.SubmittedAt),
		Member: sub.ID,
	})
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("store submission: %w", err)
	}
	return nil
}

// GetSubmission returns a stored submission.
func (s *Store) GetSubmission(ctx context.Context, id string) (*DAppSubmission, error) {
	if s.rdb == nil {
		return nil, ErrStoreUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, storeTimeout)
	defer cancel()

	raw, err := s.rdb.Get(ctx, submissionKeyPrefix+id).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read submission: %w", err)
	}

	var sub DAppSubmission
	if err := json.Unmarshal(raw, &sub); err != nil {
		return nil, fmt.Errorf("decode submission: %w", err)
	}
	return &sub, nil
}

// ListSubmissions returns the most recent submissions, newest first.
func (s *Store) ListSubmissions(ctx context.Context, limit int64) ([]DAppSubmission, error) {
	if s.rdb == nil {
		return nil, ErrStoreUnavailable
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	ctx, cancel := context.WithTimeout(ctx, storeTimeout)
	defer cancel()

	ids, err := s.rdb.ZRevRange(ctx, submissionIndexKey, 0, limit-1).Result()
	if err != nil {
		return nil, fmt.Errorf("list submissions: %w", err)
	}
	if len(ids) == 0 {
		return []DAppSubmission{}, nil
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = submissionKeyPrefix + id
	}
	values, err := s.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("read submissions: %w", err)
	}

	out := make([]DAppSubmission, 0, len(values))
	for _, v := range values {
		raw, ok := v.(string)
		if !ok {
			// A missing value means the record expired between the index read
			// and the fetch; it is skipped rather than reported as corrupt.
			continue
		}
		var sub DAppSubmission
		if err := json.Unmarshal([]byte(raw), &sub); err != nil {
			continue
		}
		out = append(out, sub)
	}
	return out, nil
}

// SaveReview stores a review and indexes it both globally and per DApp.
func (s *Store) SaveReview(ctx context.Context, review Review) error {
	if s.rdb == nil {
		return ErrStoreUnavailable
	}
	payload, err := json.Marshal(review)
	if err != nil {
		return fmt.Errorf("encode review: %w", err)
	}

	ctx, cancel := context.WithTimeout(ctx, storeTimeout)
	defer cancel()

	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, reviewKeyPrefix+review.ID, payload, 0)
	pipe.ZAdd(ctx, reviewIndexKey, redis.Z{
		Score:  float64(review.CreatedAt),
		Member: review.ID,
	})
	pipe.ZAdd(ctx, dappReviewIndex+review.DAppID, redis.Z{
		Score:  float64(review.CreatedAt),
		Member: review.ID,
	})
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("store review: %w", err)
	}
	return nil
}

// GetReviews returns the reviews for a DApp, newest first.
func (s *Store) GetReviews(ctx context.Context, dappID string) ([]Review, error) {
	if s.rdb == nil {
		return nil, ErrStoreUnavailable
	}
	ctx, cancel := context.WithTimeout(ctx, storeTimeout)
	defer cancel()

	ids, err := s.rdb.ZRevRange(ctx, dappReviewIndex+dappID, 0, -1).Result()
	if err != nil {
		return nil, fmt.Errorf("list reviews: %w", err)
	}
	if len(ids) == 0 {
		return []Review{}, nil
	}

	keys := make([]string, len(ids))
	for i, id := range ids {
		keys[i] = reviewKeyPrefix + id
	}
	values, err := s.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		return nil, fmt.Errorf("read reviews: %w", err)
	}

	out := make([]Review, 0, len(values))
	for _, v := range values {
		raw, ok := v.(string)
		if !ok {
			continue
		}
		var review Review
		if err := json.Unmarshal([]byte(raw), &review); err != nil {
			continue
		}
		out = append(out, review)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt > out[j].CreatedAt })
	return out, nil
}

// ReviewSummary returns the number of reviews and their mean rating for a DApp.
// The mean is computed from the stored reviews, so the reported rating always
// matches the reviews a user can read.
func (s *Store) ReviewSummary(ctx context.Context, dappID string) (int, float64, error) {
	reviews, err := s.GetReviews(ctx, dappID)
	if err != nil {
		return 0, 0, err
	}
	if len(reviews) == 0 {
		return 0, 0, nil
	}
	total := 0
	for _, r := range reviews {
		total += r.Rating
	}
	return len(reviews), float64(total) / float64(len(reviews)), nil
}

// newSubmissionID returns a fresh identifier for a submission.
func newSubmissionID() string {
	return uuid.New().String()
}

// ValidateReview checks that a submitted review is well formed.
func ValidateReview(review *Review) error {
	if strings.TrimSpace(review.DAppID) == "" {
		return errors.New("dappId is required")
	}
	if strings.TrimSpace(review.UserID) == "" {
		return errors.New("userId is required")
	}
	if review.Rating < 1 || review.Rating > 5 {
		return errors.New("rating must be between 1 and 5")
	}
	if len(review.Comment) > 4000 {
		return errors.New("comment is too long")
	}
	return nil
}
