package main

// Tests for the DApp submission and review persistence layer. These run
// without a Redis server: they assert the fail-closed contract, which is what
// prevents a submission or review from being acknowledged when nothing was
// stored.

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestStoreFailsClosedWithoutBackend(t *testing.T) {
	ctx := context.Background()
	store := &Store{}

	if err := store.SaveSubmission(ctx, DAppSubmission{ID: "x"}); !errors.Is(err, ErrStoreUnavailable) {
		t.Errorf("SaveSubmission error = %v, want ErrStoreUnavailable", err)
	}
	if _, err := store.GetSubmission(ctx, "x"); !errors.Is(err, ErrStoreUnavailable) {
		t.Errorf("GetSubmission error = %v, want ErrStoreUnavailable", err)
	}
	if _, err := store.ListSubmissions(ctx, 10); !errors.Is(err, ErrStoreUnavailable) {
		t.Errorf("ListSubmissions error = %v, want ErrStoreUnavailable", err)
	}
	if err := store.SaveReview(ctx, Review{ID: "x"}); !errors.Is(err, ErrStoreUnavailable) {
		t.Errorf("SaveReview error = %v, want ErrStoreUnavailable", err)
	}
	if _, err := store.GetReviews(ctx, "uniswap"); !errors.Is(err, ErrStoreUnavailable) {
		t.Errorf("GetReviews error = %v, want ErrStoreUnavailable", err)
	}
}

func TestServiceFailsClosedWithoutStore(t *testing.T) {
	svc := &DAppStoreService{dapps: map[string]*DApp{}}
	svc.Initialize()
	ctx := context.Background()

	// Submitting must not return an id when the record cannot be stored: a
	// caller holding an id believes the submission exists.
	id, err := svc.SubmitDApp(ctx, DApp{Name: "Example", AppURL: "https://example.org"}, "user-1")
	if err == nil {
		t.Fatalf("SubmitDApp returned id %q with no store configured", id)
	}
	if id != "" {
		t.Errorf("SubmitDApp returned a non-empty id %q alongside an error", id)
	}

	if err := svc.AddReview(ctx, Review{DAppID: "uniswap", UserID: "user-1", Rating: 5}); !errors.Is(err, ErrStoreUnavailable) {
		t.Errorf("AddReview error = %v, want ErrStoreUnavailable", err)
	}
	if _, err := svc.GetReviews(ctx, "uniswap"); !errors.Is(err, ErrStoreUnavailable) {
		t.Errorf("GetReviews error = %v, want ErrStoreUnavailable", err)
	}
}

func TestSubmitDAppValidatesInput(t *testing.T) {
	svc := &DAppStoreService{dapps: map[string]*DApp{}, store: &Store{}}
	ctx := context.Background()

	cases := []struct {
		name string
		dapp DApp
		user string
	}{
		{"missing name", DApp{AppURL: "https://example.org"}, "user-1"},
		{"missing app url", DApp{Name: "Example"}, "user-1"},
		{"missing submitter", DApp{Name: "Example", AppURL: "https://example.org"}, ""},
		{"blank name", DApp{Name: "   ", AppURL: "https://example.org"}, "user-1"},
	}
	for _, tc := range cases {
		if _, err := svc.SubmitDApp(ctx, tc.dapp, tc.user); err == nil {
			t.Errorf("%s: SubmitDApp accepted invalid input", tc.name)
		}
	}
}

func TestValidateReview(t *testing.T) {
	valid := Review{DAppID: "uniswap", UserID: "user-1", Rating: 4, Comment: "good"}
	if err := ValidateReview(&valid); err != nil {
		t.Errorf("a valid review was rejected: %v", err)
	}

	invalid := []struct {
		name   string
		review Review
	}{
		{"missing dapp", Review{UserID: "u", Rating: 3}},
		{"missing user", Review{DAppID: "uniswap", Rating: 3}},
		{"rating too low", Review{DAppID: "uniswap", UserID: "u", Rating: 0}},
		{"rating too high", Review{DAppID: "uniswap", UserID: "u", Rating: 6}},
		{"rating negative", Review{DAppID: "uniswap", UserID: "u", Rating: -1}},
		{"comment too long", Review{DAppID: "uniswap", UserID: "u", Rating: 3, Comment: strings.Repeat("x", 4001)}},
	}
	for _, tc := range invalid {
		if err := ValidateReview(&tc.review); err == nil {
			t.Errorf("%s: ValidateReview accepted an invalid review", tc.name)
		}
	}
}

func TestAddReviewRejectsUnknownDApp(t *testing.T) {
	svc := &DAppStoreService{dapps: map[string]*DApp{}, store: &Store{}}
	svc.Initialize()

	// "no-such-dapp" is not part of the catalog, so the review must be rejected
	// before it reaches storage.
	err := svc.AddReview(context.Background(), Review{
		DAppID: "no-such-dapp", UserID: "user-1", Rating: 5,
	})
	if err == nil {
		t.Fatal("AddReview accepted a review for an unknown DApp")
	}
	if errors.Is(err, ErrStoreUnavailable) {
		t.Fatalf("unknown-DApp review was rejected by the store rather than validation: %v", err)
	}
}

func TestNormalizeRedisURL(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"redis://localhost:6379", "redis://localhost:6379"},
		{"localhost:6379", "redis://localhost:6379"},
		{"", "redis://localhost:6379"},
		{"  redis://cache:6379  ", "redis://cache:6379"},
		{"rediss://secure:6380", "rediss://secure:6380"},
	}
	for _, tc := range cases {
		if got := normalizeRedisURL(tc.in); got != tc.want {
			t.Errorf("normalizeRedisURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestNewStoreRejectsInvalidURL(t *testing.T) {
	if _, err := NewStore(context.Background(), "://not a url"); err == nil {
		t.Error("NewStore accepted a malformed Redis URL")
	}
}

func TestSubmissionIDsAreUnique(t *testing.T) {
	// Distinct submissions must not collide on id, since the id is the only
	// handle a submitter has for the record.
	svc := &DAppStoreService{dapps: map[string]*DApp{}, store: nil}
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		// With no store the call errors before producing an id, so the id
		// generation path is exercised through the catalog instead.
		id := newSubmissionID()
		if id == "" {
			t.Fatal("newSubmissionID returned an empty id")
		}
		if seen[id] {
			t.Fatalf("duplicate submission id %q", id)
		}
		seen[id] = true
	}
	_ = svc
}
