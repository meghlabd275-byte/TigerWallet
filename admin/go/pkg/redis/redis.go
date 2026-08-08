package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tigerwallet/admin/internal/config"

	"github.com/redis/go-redis/v9"
)

// RedisClient holds the Redis connection
type RedisClient struct {
	Client *redis.Client
	Ctx    context.Context
}

// NewRedisClient creates a new Redis client
func NewRedisClient(cfg *config.Config) (*RedisClient, error) {
	ctx := context.Background()

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.RedisHost, cfg.RedisPort),
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		PoolSize:     cfg.RedisPoolSize,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
	})

	// Test connection
	_, err := client.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{
		Client: client,
		Ctx:    ctx,
	}, nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.Client.Close()
}

// Set sets a string value with expiration
func (r *RedisClient) Set(key string, value interface{}, expiration time.Duration) error {
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
			return err
		}
	}

	return r.Client.Set(r.Ctx, key, data, expiration).Err()
}

// Get gets a string value
func (r *RedisClient) Get(key string) (string, error) {
	result, err := r.Client.Get(r.Ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", nil
		}
		return "", err
	}
	return result, nil
}

// GetStruct gets a struct value
func (r *RedisClient) GetStruct(key string, dest interface{}) error {
	data, err := r.Client.Get(r.Ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil
		}
		return err
	}
	return json.Unmarshal(data, dest)
}

// Delete deletes a key
func (r *RedisClient) Delete(keys ...string) error {
	return r.Client.Del(r.Ctx, keys...).Err()
}

// Exists checks if a key exists
func (r *RedisClient) Exists(keys ...string) (int64, error) {
	return r.Client.Exists(r.Ctx, keys...).Result()
}

// Expire sets expiration on a key
func (r *RedisClient) Expire(key string, expiration time.Duration) error {
	return r.Client.Expire(r.Ctx, key, expiration).Err()
}

// TTL gets the time to live of a key
func (r *RedisClient) TTL(key string) (time.Duration, error) {
	return r.Client.TTL(r.Ctx, key).Result()
}

// Incr increments a counter
func (r *RedisClient) Incr(key string) (int64, error) {
	return r.Client.Incr(r.Ctx, key).Result()
}

// Decr decrements a counter
func (r *RedisClient) Decr(key string) (int64, error) {
	return r.Client.Decr(r.Ctx, key).Result()
}

// SetNX sets a value only if the key doesn't exist
func (r *RedisClient) SetNX(key string, value interface{}, expiration time.Duration) (bool, error) {
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

	return r.Client.SetNX(r.Ctx, key, data, expiration).Result()
}

// ZAdd adds a member to a sorted set
func (r *RedisClient) ZAdd(key string, members ...redis.Z) error {
	return r.Client.ZAdd(r.Ctx, key, members...).Err()
}

// ZRangeByScore gets members by score range
func (r *RedisClient) ZRangeByScore(key string, opt redis.ZRangeBy) ([]string, error) {
	return r.Client.ZRangeByScore(r.Ctx, key, &opt).Result()
}

// ZRem removes members from a sorted set
func (r *RedisClient) ZRem(key string, members ...interface{}) error {
	return r.Client.ZRem(r.Ctx, key, members...).Err()
}

// HSet sets a hash field
func (r *RedisClient) HSet(key string, values ...interface{}) error {
	return r.Client.HSet(r.Ctx, key, values...).Err()
}

// HGet gets a hash field
func (r *RedisClient) HGet(key, field string) (string, error) {
	return r.Client.HGet(r.Ctx, key, field).Result()
}

// HGetAll gets all hash fields
func (r *RedisClient) HGetAll(key string) (map[string]string, error) {
	return r.Client.HGetAll(r.Ctx, key).Result()
}

// HDel deletes hash fields
func (r *RedisClient) HDel(key string, fields ...string) error {
	return r.Client.HDel(r.Ctx, key, fields...).Err()
}

// HIncrBy increments a hash field
func (r *RedisClient) HIncrBy(key, field string, increment int64) (int64, error) {
	return r.Client.HIncrBy(r.Ctx, key, field, increment).Result()
}

// SAdd adds members to a set
func (r *RedisClient) SAdd(key string, members ...interface{}) error {
	return r.Client.SAdd(r.Ctx, key, members...).Err()
}

// SMembers gets all set members
func (r *RedisClient) SMembers(key string) ([]string, error) {
	return r.Client.SMembers(r.Ctx, key).Result()
}

// SIsMember checks if a member exists
func (r *RedisClient) SIsMember(key string, member interface{}) (bool, error) {
	return r.Client.SIsMember(r.Ctx, key, member).Result()
}

// SRem removes members from a set
func (r *RedisClient) SRem(key string, members ...interface{}) error {
	return r.Client.SRem(r.Ctx, key, members...).Err()
}

// Publish publishes to a channel
func (r *RedisClient) Publish(channel string, message interface{}) error {
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

	return r.Client.Publish(r.Ctx, channel, data).Err()
}

// Subscribe subscribes to a channel
func (r *RedisClient) Subscribe(channel string) *redis.PubSub {
	return r.Client.Subscribe(r.Ctx, channel)
}

// Pipeline executes multiple commands in a pipeline
func (r *RedisClient) Pipeline(commands ...func(redis.Pipeliner)) error {
	pipe := r.Client.Pipeline()

	for _, cmd := range commands {
		cmd(pipe)
	}

	_, err := pipe.Exec(r.Ctx)
	return err
}

// Lock acquires a distributed lock
func (r *RedisClient) Lock(key, value string, expiration time.Duration) (bool, error) {
	return r.Client.SetNX(r.Ctx, "lock:"+key, value, expiration).Result()
}

// Unlock releases a distributed lock
func (r *RedisClient) Unlock(key, value string) error {
	script := redis.NewScript(`
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`)

	keys := []string{"lock:" + key}
	_, err := script.Run(r.Ctx, r.Client, keys, value).Result()
	return err
}

// Cache helpers

// CacheUserSession caches a user session
func (r *RedisClient) CacheUserSession(sessionID string, data interface{}, expiration time.Duration) error {
	return r.Set(fmt.Sprintf("session:%s", sessionID), data, expiration)
}

// GetUserSession gets a cached user session
func (r *RedisClient) GetUserSession(sessionID string, dest interface{}) error {
	return r.GetStruct(fmt.Sprintf("session:%s", sessionID), dest)
}

// DeleteUserSession deletes a cached user session
func (r *RedisClient) DeleteUserSession(sessionID string) error {
	return r.Delete(fmt.Sprintf("session:%s", sessionID))
}

// Rate limiting

// IncrRateLimit increments rate limit counter
func (r *RedisClient) IncrRateLimit(key string, window time.Duration) (int64, error) {
	pipe := r.Client.Pipeline()

	incr := pipe.Incr(r.Ctx, key)
	pipe.Expire(r.Ctx, key, window)

	_, err := pipe.Exec(r.Ctx)
	if err != nil {
		return 0, err
	}

	return incr.Val(), nil
}

// GetRateLimit gets rate limit count
func (r *RedisClient) GetRateLimit(key string) (int64, error) {
	val, err := r.Client.Get(r.Ctx, key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}
