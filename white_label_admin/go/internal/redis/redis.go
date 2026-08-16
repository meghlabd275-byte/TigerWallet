package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
	"github.com/tigerwallet/white-label-admin/internal/config"
)

// RedisClient is the shared feature-flag store client for the white-label
// admin backend. Admin backends WRITE flag state to Redis; downstream services
// READ it. Respects app separation: only a shared Redis namespace, no code
// imports cross between admin apps and wallet apps.
type RedisClient struct {
	Client *redis.Client
	Ctx    context.Context
}

func NewRedisClient(cfg *config.Config) (*RedisClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	cli := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	ctx := context.Background()
	if err := cli.Ping(ctx).Err(); err != nil {
		// Non-fatal: backend still boots; flag publish fails closed.
		return &RedisClient{Client: cli, Ctx: ctx}, nil
	}
	return &RedisClient{Client: cli, Ctx: ctx}, nil
}

func (r *RedisClient) Close() error {
	if r == nil || r.Client == nil {
		return nil
	}
	return r.Client.Close()
}
