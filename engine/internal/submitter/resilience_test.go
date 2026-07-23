package submitter

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/make-software/casper-go-sdk/v2/rpc"
)

// fakeQuerier is a GlobalStateQuerier test double whose behavior is driven
// by a caller-supplied function, with an atomic call counter.
type fakeQuerier struct {
	calls int32
	fn    func(callNum int32) (rpc.QueryGlobalStateResult, error)
}

func (f *fakeQuerier) QueryLatestGlobalState(ctx context.Context, key string, path []string) (rpc.QueryGlobalStateResult, error) {
	n := atomic.AddInt32(&f.calls, 1)
	return f.fn(n)
}

func fastRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 4,
		BaseDelay:   1 * time.Millisecond,
		MaxDelay:    5 * time.Millisecond,
		Multiplier:  2.0,
	}
}

func TestQueryGlobalState_SucceedsFirstTry(t *testing.T) {
	f := &fakeQuerier{fn: func(n int32) (rpc.QueryGlobalStateResult, error) {
		return rpc.QueryGlobalStateResult{ApiVersion: "1.0"}, nil
	}}
	q := NewResilientQuerier(f, fastRetryConfig(), DefaultCircuitBreakerConfig())

	res, err := q.QueryGlobalState(context.Background(), "hash-key", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.ApiVersion != "1.0" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if f.calls != 1 {
		t.Fatalf("expected 1 call, got %d", f.calls)
	}
	if q.State() != "closed" {
		t.Fatalf("expected closed circuit, got %s", q.State())
	}
}

func TestQueryGlobalState_RetriesThenSucceeds(t *testing.T) {
	f := &fakeQuerier{fn: func(n int32) (rpc.QueryGlobalStateResult, error) {
		if n < 3 {
			return rpc.QueryGlobalStateResult{}, errors.New("transient rpc error")
		}
		return rpc.QueryGlobalStateResult{ApiVersion: "ok"}, nil
	}}
	q := NewResilientQuerier(f, fastRetryConfig(), DefaultCircuitBreakerConfig())

	res, err := q.QueryGlobalState(context.Background(), "k", nil)
	if err != nil {
		t.Fatalf("expected eventual success, got: %v", err)
	}
	if res.ApiVersion != "ok" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if f.calls != 3 {
		t.Fatalf("expected 3 attempts, got %d", f.calls)
	}
	// A successful call should reset the breaker to closed.
	if q.State() != "closed" {
		t.Fatalf("expected closed circuit after success, got %s", q.State())
	}
}

func TestQueryGlobalState_ExhaustsRetriesAndFails(t *testing.T) {
	wantErr := errors.New("node unreachable")
	f := &fakeQuerier{fn: func(n int32) (rpc.QueryGlobalStateResult, error) {
		return rpc.QueryGlobalStateResult{}, wantErr
	}}
	cfg := fastRetryConfig()
	q := NewResilientQuerier(f, cfg, DefaultCircuitBreakerConfig())

	_, err := q.QueryGlobalState(context.Background(), "k", nil)
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}
	if f.calls != int32(cfg.MaxAttempts) {
		t.Fatalf("expected exactly %d attempts, got %d", cfg.MaxAttempts, f.calls)
	}
}

func TestCircuitBreaker_OpensAfterConsecutiveFailures(t *testing.T) {
	f := &fakeQuerier{fn: func(n int32) (rpc.QueryGlobalStateResult, error) {
		return rpc.QueryGlobalStateResult{}, errors.New("boom")
	}}
	retry := RetryConfig{MaxAttempts: 1, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond, Multiplier: 1}
	cb := CircuitBreakerConfig{FailureThreshold: 3, OpenTimeout: time.Hour}
	q := NewResilientQuerier(f, retry, cb)

	for i := 0; i < 3; i++ {
		if _, err := q.QueryGlobalState(context.Background(), "k", nil); err == nil {
			t.Fatalf("iteration %d: expected failure", i)
		}
	}
	if q.State() != "open" {
		t.Fatalf("expected circuit to be open after %d consecutive failures, got %s", cb.FailureThreshold, q.State())
	}

	// The next call must be rejected immediately (fail fast), without
	// invoking the underlying querier again.
	callsBefore := f.calls
	_, err := q.QueryGlobalState(context.Background(), "k", nil)
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}
	if f.calls != callsBefore {
		t.Fatalf("expected no underlying call while circuit is open, calls went from %d to %d", callsBefore, f.calls)
	}
}

func TestCircuitBreaker_HalfOpenProbeRecovers(t *testing.T) {
	failing := true
	f := &fakeQuerier{fn: func(n int32) (rpc.QueryGlobalStateResult, error) {
		if failing {
			return rpc.QueryGlobalStateResult{}, errors.New("still down")
		}
		return rpc.QueryGlobalStateResult{ApiVersion: "recovered"}, nil
	}}
	retry := RetryConfig{MaxAttempts: 1, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond, Multiplier: 1}
	cb := CircuitBreakerConfig{FailureThreshold: 2, OpenTimeout: 20 * time.Millisecond}
	q := NewResilientQuerier(f, retry, cb)

	for i := 0; i < 2; i++ {
		_, _ = q.QueryGlobalState(context.Background(), "k", nil)
	}
	if q.State() != "open" {
		t.Fatalf("expected open circuit, got %s", q.State())
	}

	// Still within the open window: rejected without a call.
	if _, err := q.QueryGlobalState(context.Background(), "k", nil); !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen while cooling down, got %v", err)
	}

	// Wait past OpenTimeout so the breaker allows a half-open probe.
	time.Sleep(cb.OpenTimeout + 10*time.Millisecond)
	failing = false

	res, err := q.QueryGlobalState(context.Background(), "k", nil)
	if err != nil {
		t.Fatalf("expected half-open probe to succeed, got %v", err)
	}
	if res.ApiVersion != "recovered" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if q.State() != "closed" {
		t.Fatalf("expected circuit closed after successful probe, got %s", q.State())
	}
}

func TestCircuitBreaker_HalfOpenProbeFailureReopens(t *testing.T) {
	f := &fakeQuerier{fn: func(n int32) (rpc.QueryGlobalStateResult, error) {
		return rpc.QueryGlobalStateResult{}, errors.New("still down")
	}}
	retry := RetryConfig{MaxAttempts: 1, BaseDelay: time.Millisecond, MaxDelay: time.Millisecond, Multiplier: 1}
	cb := CircuitBreakerConfig{FailureThreshold: 2, OpenTimeout: 15 * time.Millisecond}
	q := NewResilientQuerier(f, retry, cb)

	for i := 0; i < 2; i++ {
		_, _ = q.QueryGlobalState(context.Background(), "k", nil)
	}

	time.Sleep(cb.OpenTimeout + 5*time.Millisecond)

	// Probe call: underlying querier still fails, breaker must reopen.
	if _, err := q.QueryGlobalState(context.Background(), "k", nil); err == nil {
		t.Fatal("expected probe failure")
	}
	if q.State() != "open" {
		t.Fatalf("expected circuit reopened after failed probe, got %s", q.State())
	}
}

func TestQueryGlobalState_ContextCancelDuringBackoff(t *testing.T) {
	f := &fakeQuerier{fn: func(n int32) (rpc.QueryGlobalStateResult, error) {
		return rpc.QueryGlobalStateResult{}, errors.New("fail")
	}}
	retry := RetryConfig{MaxAttempts: 5, BaseDelay: 50 * time.Millisecond, MaxDelay: time.Second, Multiplier: 2}
	q := NewResilientQuerier(f, retry, DefaultCircuitBreakerConfig())

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := q.QueryGlobalState(ctx, "k", nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
}

func TestBackoffDelay_ExponentialAndCapped(t *testing.T) {
	r := RetryConfig{BaseDelay: 100 * time.Millisecond, MaxDelay: 1 * time.Second, Multiplier: 2}
	if d := r.backoffDelay(1); d != 100*time.Millisecond {
		t.Fatalf("attempt 1: expected 100ms, got %v", d)
	}
	if d := r.backoffDelay(2); d != 200*time.Millisecond {
		t.Fatalf("attempt 2: expected 200ms, got %v", d)
	}
	if d := r.backoffDelay(3); d != 400*time.Millisecond {
		t.Fatalf("attempt 3: expected 400ms, got %v", d)
	}
	if d := r.backoffDelay(10); d != r.MaxDelay {
		t.Fatalf("large attempt should cap at MaxDelay, got %v", d)
	}
	if d := r.backoffDelay(0); d != 0 {
		t.Fatalf("attempt 0 should be 0, got %v", d)
	}
}
