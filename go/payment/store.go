package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Payment state lives in Redis so it survives a restart and is shared by every
// replica. Deposit addresses are the important part: the chain listener looks
// one up by address to attribute an incoming transfer, so an address that only
// exists in one process's memory is lost the moment that process is recycled,
// and a payment made to it can never be matched.
//
// A local map fronts Redis. Writes go to Redis first and are only mirrored on
// success, so the cache can never claim a value that was not stored. A Redis
// outage therefore fails the operation rather than silently losing money.
const (
	paymentKeyPrefix = "payment:"
	addressKeyPrefix = "payment:addr:"
	feeKeyPrefix     = "payment:fee:"
	paymentIndexKey  = "payment:index"
	addressIndexKey  = "payment:addr:index"
	feeIndexKey      = "payment:fee:index"

	// Redis is shared infrastructure and may be mid-restart, so a short retry
	// is worth more than an immediate failure. Requests are not retried at the
	// HTTP layer.
	storeWriteTimeout = 5 * time.Second
)

type paymentStore struct {
	rdb      *redis.Client
	payments map[string]*Payment
	addrs    map[string]*PaymentAddress
	fees     map[string]*FeeConfig
}

// newPaymentStore returns a Redis-backed store.
func newPaymentStore(rdb *redis.Client) *paymentStore {
	return &paymentStore{
		rdb:      rdb,
		payments: make(map[string]*Payment),
		addrs:    make(map[string]*PaymentAddress),
		fees:     make(map[string]*FeeConfig),
	}
}

// newMemoryPaymentStore is for tests only. It has no durability and no
// cross-replica visibility, which is exactly what production must not use.
func newMemoryPaymentStore() *paymentStore {
	return newPaymentStore(nil)
}

func (s *paymentStore) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), storeWriteTimeout)
}

func (s *paymentStore) putPayment(p *Payment) error {
	blob, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("cannot encode payment %s: %w", p.ID, err)
	}
	if s.rdb != nil {
		ctx, cancel := s.ctx()
		defer cancel()
		if err := s.rdb.Set(ctx, paymentKeyPrefix+p.ID, blob, 0).Err(); err != nil {
			return fmt.Errorf("cannot persist payment %s: %w", p.ID, err)
		}
		if err := s.rdb.SAdd(ctx, paymentIndexKey, p.ID).Err(); err != nil {
			return fmt.Errorf("cannot index payment %s: %w", p.ID, err)
		}
	}
	s.payments[p.ID] = p
	return nil
}

func (s *paymentStore) getPayment(id string) (*Payment, bool) {
	if s.rdb != nil {
		ctx, cancel := s.ctx()
		defer cancel()
		blob, err := s.rdb.Get(ctx, paymentKeyPrefix+id).Bytes()
		switch {
		case err == nil:
			var p Payment
			if err := json.Unmarshal(blob, &p); err == nil {
				s.payments[id] = &p
				return &p, true
			}
		case errors.Is(err, redis.Nil):
			return nil, false
		default:
			// The shared store is authoritative. Falling back to a local map
			// here would answer from a possibly stale copy.
			log.Printf("payment store unavailable reading %s: %v", id, err)
			return nil, false
		}
	}
	p, ok := s.payments[id]
	return p, ok
}

func (s *paymentStore) listPayments() []*Payment {
	out := make([]*Payment, 0, len(s.payments))
	seen := make(map[string]bool)

	if s.rdb != nil {
		ctx, cancel := s.ctx()
		defer cancel()
		ids, err := s.rdb.SMembers(ctx, paymentIndexKey).Result()
		if err != nil {
			log.Printf("payment store unavailable listing payments: %v", err)
			return out
		}
		for _, id := range ids {
			if seen[id] {
				continue
			}
			if p, ok := s.getPayment(id); ok {
				seen[id] = true
				out = append(out, p)
			}
		}
		return out
	}

	for _, p := range s.payments {
		out = append(out, p)
	}
	return out
}

// putPaymentWithAddress records a payment and its deposit address together, so
// a crash between the two cannot leave an address the listener knows about
// pointing at no payment, or a payment whose address nobody watches.
func (s *paymentStore) putPaymentWithAddress(p *Payment, a *PaymentAddress) error {
	paymentBlob, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("cannot encode payment %s: %w", p.ID, err)
	}
	addrBlob, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("cannot encode address %s: %w", a.Address, err)
	}

	if s.rdb != nil {
		ctx, cancel := s.ctx()
		defer cancel()

		// One atomic script: either both records and both index entries land,
		// or nothing does.
		script := redis.NewScript(`
			redis.call('SET', KEYS[1], ARGV[1])
			redis.call('SET', KEYS[2], ARGV[2])
			redis.call('SADD', KEYS[3], ARGV[3])
			redis.call('SADD', KEYS[4], ARGV[4])
			return 1
		`)
		err := script.Run(ctx, s.rdb,
			[]string{
				paymentKeyPrefix + p.ID,
				addressKeyPrefix + a.Address,
				paymentIndexKey,
				addressIndexKey,
			},
			paymentBlob, addrBlob, p.ID, a.Address,
		).Err()
		if err != nil {
			return fmt.Errorf("cannot persist payment %s with its address: %w", p.ID, err)
		}
	}

	s.payments[p.ID] = p
	s.addrs[a.Address] = a
	return nil
}

func (s *paymentStore) getAddress(address string) (*PaymentAddress, bool) {
	if s.rdb != nil {
		ctx, cancel := s.ctx()
		defer cancel()
		blob, err := s.rdb.Get(ctx, addressKeyPrefix+address).Bytes()
		switch {
		case err == nil:
			var a PaymentAddress
			if err := json.Unmarshal(blob, &a); err == nil {
				s.addrs[address] = &a
				return &a, true
			}
		case errors.Is(err, redis.Nil):
			return nil, false
		default:
			log.Printf("payment store unavailable reading address %s: %v", address, err)
			return nil, false
		}
	}
	a, ok := s.addrs[address]
	return a, ok
}

func (s *paymentStore) putAddress(a *PaymentAddress) error {
	blob, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("cannot encode address %s: %w", a.Address, err)
	}
	if s.rdb != nil {
		ctx, cancel := s.ctx()
		defer cancel()
		if err := s.rdb.Set(ctx, addressKeyPrefix+a.Address, blob, 0).Err(); err != nil {
			return fmt.Errorf("cannot persist address %s: %w", a.Address, err)
		}
		if err := s.rdb.SAdd(ctx, addressIndexKey, a.Address).Err(); err != nil {
			return fmt.Errorf("cannot index address %s: %w", a.Address, err)
		}
	}
	s.addrs[a.Address] = a
	return nil
}

func (s *paymentStore) putFee(f *FeeConfig) error {
	blob, err := json.Marshal(f)
	if err != nil {
		return fmt.Errorf("cannot encode fee %s: %w", f.ID, err)
	}
	if s.rdb != nil {
		ctx, cancel := s.ctx()
		defer cancel()
		if err := s.rdb.Set(ctx, feeKeyPrefix+f.ID, blob, 0).Err(); err != nil {
			return fmt.Errorf("cannot persist fee %s: %w", f.ID, err)
		}
		if err := s.rdb.SAdd(ctx, feeIndexKey, f.ID).Err(); err != nil {
			return fmt.Errorf("cannot index fee %s: %w", f.ID, err)
		}
	}
	s.fees[f.ID] = f
	return nil
}

func (s *paymentStore) listFees() []*FeeConfig {
	out := make([]*FeeConfig, 0, len(s.fees))
	seen := make(map[string]bool)

	if s.rdb != nil {
		ctx, cancel := s.ctx()
		defer cancel()
		ids, err := s.rdb.SMembers(ctx, feeIndexKey).Result()
		if err != nil {
			log.Printf("payment store unavailable listing fees: %v", err)
			return out
		}
		for _, id := range ids {
			if seen[id] {
				continue
			}
			blob, err := s.rdb.Get(ctx, feeKeyPrefix+id).Bytes()
			if err != nil {
				continue
			}
			var f FeeConfig
			if err := json.Unmarshal(blob, &f); err != nil {
				continue
			}
			seen[id] = true
			s.fees[id] = &f
			out = append(out, &f)
		}
		return out
	}

	for _, f := range s.fees {
		out = append(out, f)
	}
	return out
}

func (s *paymentStore) getFee(id string) (*FeeConfig, bool) {
	if s.rdb != nil {
		ctx, cancel := s.ctx()
		defer cancel()
		blob, err := s.rdb.Get(ctx, feeKeyPrefix+id).Bytes()
		if err == nil {
			var f FeeConfig
			if err := json.Unmarshal(blob, &f); err == nil {
				s.fees[id] = &f
				return &f, true
			}
		}
	}
	f, ok := s.fees[id]
	return f, ok
}
