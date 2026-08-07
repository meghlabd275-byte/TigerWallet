/**
 * Redis Cache Layer
 * 
 * High-performance distributed caching for wallet services
 */

package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/tigerwallet/wallet-services/internal/config"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("component", "cache")

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(cfg config.RedisConfig) (*RedisClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr(),
		Password:     cfg.Password,
		DB:           cfg.Database,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: 10,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  10 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.Infof("Connected to Redis: %s", cfg.Addr())
	return &RedisClient{client: client}, nil
}

func (r *RedisClient) Close() error {
	return r.client.Close()
}

// Key prefixes for different data types
const (
	PrefixUser          = "user:"
	PrefixWallet        = "wallet:"
	PrefixBalance       = "balance:"
	PrefixPrice         = "price:"
	PrefixGas           = "gas:"
	PrefixNonce         = "nonce:"
	PrefixRateLimit     = "ratelimit:"
	PrefixSession       = "session:"
	PrefixVerification  = "verification:"
	PrefixOTP           = "otp:"
	PrefixLock          = "lock:"
	PrefixBlock         = "block:"
	PrefixToken         = "token:"
)

// Set stores a value with expiration
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		var err error
		data, err = json.Marshal(v)
		if err != nil {
			return fmt.Errorf("failed to marshal value: %w", err)
		}
	}
	return r.client.Set(ctx, key, data, expiration).Err()
}

// Get retrieves a value
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		return "", err
	}
	return val, nil
}

// GetBytes retrieves a value as bytes
func (r *RedisClient) GetBytes(ctx context.Context, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}
	return val, nil
}

// GetStruct retrieves and unmarshals a struct
func (r *RedisClient) GetStruct(ctx context.Context, key string, dest interface{}) error {
	data, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// Delete removes a key
func (r *RedisClient) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}

// Exists checks if a key exists
func (r *RedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	return r.client.Exists(ctx, keys...).Result()
}

// Expire sets expiration on a key
func (r *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return r.client.Expire(ctx, key, expiration).Err()
}

// TTL gets remaining time to live
func (r *RedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	return r.client.TTL(ctx, key).Result()
}

// Incr increments a counter
func (r *RedisClient) Incr(ctx context.Context, key string) (int64, error) {
	return r.client.Incr(ctx, key).Result()
}

// IncrBy increments by a value
func (r *RedisClient) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return r.client.IncrBy(ctx, key, value).Result()
}

// SetNX sets a key only if it doesn't exist (for distributed locks)
func (r *RedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	var data []byte
	switch v := value.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		var err error
		data, err = json.Marshal(v)
		if err != nil {
			return false, err
		}
	}
	return r.client.SetNX(ctx, key, data, expiration).Result()
}

// Hash operations
func (r *RedisClient) HSet(ctx context.Context, key string, values map[string]interface{}) error {
	return r.client.HSet(ctx, key, values).Err()
}

func (r *RedisClient) HGet(ctx context.Context, key, field string) (string, error) {
	return r.client.HGet(ctx, key, field).Result()
}

func (r *RedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return r.client.HGetAll(ctx, key).Result()
}

func (r *RedisClient) HDel(ctx context.Context, key string, fields ...string) error {
	return r.client.HDel(ctx, key, fields...).Err()
}

func (r *RedisClient) HIncrBy(ctx context.Context, key, field string, incr int64) (int64, error) {
	return r.client.HIncrBy(ctx, key, field, incr).Result()
}

// List operations
func (r *RedisClient) LPush(ctx context.Context, key string, values ...interface{}) error {
	return r.client.LPush(ctx, key, values...).Err()
}

func (r *RedisClient) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return r.client.LRange(ctx, key, start, stop).Result()
}

func (r *RedisClient) LRem(ctx context.Context, key string, count int64, value interface{}) error {
	return r.client.LRem(ctx, key, count, value).Err()
}

// Sorted set operations
func (r *RedisClient) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	return r.client.ZAdd(ctx, key, members...).Err()
}

func (r *RedisClient) ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	return r.client.ZRangeByScore(ctx, key, opt).Result()
}

func (r *RedisClient) ZRem(ctx context.Context, key string, members ...interface{}) error {
	return r.client.ZRem(ctx, key, members...).Err()
}

// Pub/Sub
func (r *RedisClient) Publish(ctx context.Context, channel string, message interface{}) error {
	var data []byte
	switch v := message.(type) {
	case string:
		data = []byte(v)
	case []byte:
		data = v
	default:
		var err error
		data, err = json.Marshal(v)
		if err != nil {
			return err
		}
	}
	return r.client.Publish(ctx, channel, data).Err()
}

func (r *RedisClient) Subscribe(ctx context.Context, channel string) *redis.PubSub {
	return r.client.Subscribe(ctx, channel)
}

// Pipeline executes multiple commands
func (r *RedisClient) Pipeline(fn func(redis.Pipeliner) error) error {
	pipe := r.client.Pipeline()
	if err := fn(pipe); err != nil {
		return err
	}
	_, err := pipe.Exec(ctx)
	return err
}

// Transaction executes multiple commands atomically
func (r *RedisClient) TxPipelined(fn func(redis.Pipeliner) error) error {
	pipe := r.client.TxPipeline()
	if err := fn(pipe); err != nil {
		return err
	}
	_, err := pipe.Exec(ctx)
	return err
}

// Specialized cache methods

// CacheUser caches user data
func (r *RedisClient) CacheUser(ctx context.Context, userID string, data interface{}, ttl time.Duration) error {
	return r.Set(ctx, PrefixUser+userID, data, ttl)
}

// GetCachedUser retrieves cached user
func (r *RedisClient) GetCachedUser(ctx context.Context, userID string, dest interface{}) error {
	return r.GetStruct(ctx, PrefixUser+userID, dest)
}

// InvalidateUserCache removes user from cache
func (r *RedisClient) InvalidateUserCache(ctx context.Context, userID string) error {
	return r.Delete(ctx, PrefixUser+userID)
}

// CacheWallet caches wallet data
func (r *RedisClient) CacheWallet(ctx context.Context, walletID string, data interface{}, ttl time.Duration) error {
	return r.Set(ctx, PrefixWallet+walletID, data, ttl)
}

// GetCachedWallet retrieves cached wallet
func (r *RedisClient) GetCachedWallet(ctx context.Context, walletID string, dest interface{}) error {
	return r.GetStruct(ctx, PrefixWallet+walletID, dest)
}

// CachePrice caches token price
func (r *RedisClient) CachePrice(ctx context.Context, tokenAddress, chainType string, price float64, ttl time.Duration) error {
	key := fmt.Sprintf("%s%s:%s", PrefixPrice, chainType, tokenAddress)
	return r.Set(ctx, key, price, ttl)
}

// GetCachedPrice retrieves cached price
func (r *RedisClient) GetCachedPrice(ctx context.Context, tokenAddress, chainType string) (float64, error) {
	key := fmt.Sprintf("%s%s:%s", PrefixPrice, chainType, tokenAddress)
	val, err := r.Get(ctx, key)
	if err != nil {
		return 0, err
	}
	var price float64
	if err := json.Unmarshal([]byte(val), &price); err != nil {
		return 0, err
	}
	return price, nil
}

// CacheGasPrice caches gas price
func (r *RedisClient) CacheGasPrice(ctx context.Context, chainType string, data interface{}, ttl time.Duration) error {
	return r.Set(ctx, PrefixGas+chainType, data, ttl)
}

// GetCachedGasPrice retrieves cached gas price
func (r *RedisClient) GetCachedGasPrice(ctx context.Context, chainType string, dest interface{}) error {
	return r.GetStruct(ctx, PrefixGas+chainType, dest)
}

// Rate limiting
func (r *RedisClient) CheckRateLimit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	rateKey := PrefixRateLimit + key
	
	count, err := r.client.Incr(ctx, rateKey).Result()
	if err != nil {
		return false, err
	}
	
	if count == 1 {
		r.client.Expire(ctx, rateKey, window)
	}
	
	return count <= int64(limit), nil
}

// AcquireLock acquires a distributed lock
func (r *RedisClient) AcquireLock(ctx context.Context, lockKey string, ttl time.Duration) (bool, error) {
	return r.SetNX(ctx, PrefixLock+lockKey, "1", ttl)
}

// ReleaseLock releases a distributed lock
func (r *RedisClient) ReleaseLock(ctx context.Context, lockKey string) error {
	return r.Delete(ctx, PrefixLock+lockKey)
}
