package submitter

import (
	"context"
	"errors"
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/make-software/casper-go-sdk/v2/rpc"
	"github.com/make-software/casper-go-sdk/v2/types"
	"github.com/make-software/casper-go-sdk/v2/types/key"
)

// fakePutter is a programmable txPutter for unit-testing the retry
// wrapper without touching a real Casper node.
type fakePutter struct {
	calls   atomic.Int32
	program []programStep
}

type programStep struct {
	err    error
	result rpc.PutTransactionResult
}

func (f *fakePutter) PutTransactionV1(ctx context.Context, tx types.TransactionV1) (rpc.PutTransactionResult, error) {
	idx := int(f.calls.Add(1)) - 1
	if idx >= len(f.program) {
		return rpc.PutTransactionResult{}, errors.New("fakePutter: unexpected extra call")
	}
	step := f.program[idx]
	return step.result, step.err
}

// successResult returns a PutTransactionResult with the given hex hash.
func successResult(hexHash string) rpc.PutTransactionResult {
	var h key.Hash
	// Fill deterministically so ToHex() returns something predictable —
	// exact content is irrelevant for the retry-loop assertions.
	copy(h[:], []byte(hexHash))
	return rpc.PutTransactionResult{
		TransactionHash: types.TransactionHash{TransactionV1: &h},
	}
}

// newTestSubmitter builds a CasperSubmitter with a mock RPC client and
// small backoffs so tests stay fast. Keys/chain are not exercised —
// retry runs at the transport layer, above signing.
func newTestSubmitter(t *testing.T, client txPutter) *CasperSubmitter {
	t.Helper()
	return &CasperSubmitter{
		client: client,
		retry: retryConfig{
			maxAttempts:    3,
			initialBackoff: 1 * time.Millisecond,
			backoffFactor:  2.0,
		},
	}
}

// -------- isRetryableRPCError --------

func TestIsRetryableRPCError_NilIsFalse(t *testing.T) {
	if isRetryableRPCError(nil) {
		t.Fatalf("nil error must not be retryable")
	}
}

func TestIsRetryableRPCError_ContextDeadlineIsRetryable(t *testing.T) {
	if !isRetryableRPCError(context.DeadlineExceeded) {
		t.Fatalf("context.DeadlineExceeded must be retryable")
	}
}

func TestIsRetryableRPCError_NetTimeoutIsRetryable(t *testing.T) {
	// net.OpError wrapping net.DNSError with IsTimeout=true satisfies
	// the net.Error interface with Timeout()==true.
	dnsErr := &net.DNSError{Err: "timeout", IsTimeout: true}
	err := &net.OpError{Op: "read", Net: "tcp", Err: dnsErr}
	if !isRetryableRPCError(err) {
		t.Fatalf("net.Error with Timeout()==true must be retryable")
	}
}

func TestIsRetryableRPCError_TransientMarkers(t *testing.T) {
	cases := []string{
		"connection reset by peer",
		"connection refused",
		"broken pipe",
		"read: EOF",
		"i/o timeout",
		"upstream returned HTTP status 502 Bad Gateway",
		"upstream returned HTTP status 503 Service Unavailable",
		"upstream returned HTTP status 504 Gateway Timeout",
		"upstream returned HTTP status 500 Internal Server Error",
		"put transaction: TCP timeout after 30s",
	}
	for _, msg := range cases {
		if !isRetryableRPCError(errors.New(msg)) {
			t.Errorf("expected %q to be retryable", msg)
		}
	}
}

func TestIsRetryableRPCError_TerminalMarkers(t *testing.T) {
	// 4xx / validation-style errors must NOT be retried: retrying a
	// bad payload just wastes attempts and delays the real error.
	cases := []string{
		"invalid transaction: bad signature",
		"HTTP status 400 Bad Request",
		"HTTP status 401 Unauthorized",
		"HTTP status 404 Not Found",
		"HTTP status 422 Unprocessable Entity",
		"unknown entry point",
	}
	for _, msg := range cases {
		if isRetryableRPCError(errors.New(msg)) {
			t.Errorf("expected %q to be terminal (not retryable)", msg)
		}
	}
}

// -------- putWithRetry — happy path --------

func TestPutWithRetry_HappyPathSucceedsOnce(t *testing.T) {
	client := &fakePutter{
		program: []programStep{
			{err: nil, result: successResult("first-try-ok")},
		},
	}
	s := newTestSubmitter(t, client)

	res, err := s.putWithRetry(context.Background(), types.TransactionV1{}, "submit_proof")
	if err != nil {
		t.Fatalf("expected success, got err=%v", err)
	}
	if got := client.calls.Load(); got != 1 {
		t.Fatalf("expected exactly 1 RPC call on happy path, got %d", got)
	}
	if res.TransactionHash.TransactionV1 == nil {
		t.Fatalf("expected non-nil TransactionHash.TransactionV1 on success")
	}
}

// -------- putWithRetry — retry path (fails twice, succeeds on 3rd) --------

func TestPutWithRetry_RetriesTransientTwiceThenSucceeds(t *testing.T) {
	client := &fakePutter{
		program: []programStep{
			{err: errors.New("connection reset by peer")},
			{err: errors.New("HTTP status 503 Service Unavailable")},
			{err: nil, result: successResult("third-try-ok")},
		},
	}
	s := newTestSubmitter(t, client)

	res, err := s.putWithRetry(context.Background(), types.TransactionV1{}, "submit_proof")
	if err != nil {
		t.Fatalf("expected eventual success after 2 transient failures, got err=%v", err)
	}
	if got := client.calls.Load(); got != 3 {
		t.Fatalf("expected exactly 3 RPC calls (2 fails + 1 success), got %d", got)
	}
	if res.TransactionHash.TransactionV1 == nil {
		t.Fatalf("expected non-nil TransactionHash on eventual success")
	}
}

// -------- putWithRetry — terminal error stops retries immediately --------

func TestPutWithRetry_TerminalErrorNotRetried(t *testing.T) {
	client := &fakePutter{
		program: []programStep{
			{err: errors.New("HTTP status 400 Bad Request: invalid transaction")},
		},
	}
	s := newTestSubmitter(t, client)

	_, err := s.putWithRetry(context.Background(), types.TransactionV1{}, "submit_proof")
	if err == nil {
		t.Fatalf("expected terminal error to surface, got success")
	}
	if got := client.calls.Load(); got != 1 {
		t.Fatalf("terminal error must stop after 1 call, got %d", got)
	}
}

// -------- putWithRetry — max attempts exhausted --------

func TestPutWithRetry_ExhaustsMaxAttempts(t *testing.T) {
	client := &fakePutter{
		program: []programStep{
			{err: errors.New("i/o timeout")},
			{err: errors.New("i/o timeout")},
			{err: errors.New("i/o timeout")},
		},
	}
	s := newTestSubmitter(t, client)

	_, err := s.putWithRetry(context.Background(), types.TransactionV1{}, "submit_proof")
	if err == nil {
		t.Fatalf("expected error after exhausting %d attempts", s.retry.maxAttempts)
	}
	if got := client.calls.Load(); got != int32(s.retry.maxAttempts) {
		t.Fatalf("expected %d RPC calls at exhaustion, got %d", s.retry.maxAttempts, got)
	}
}

// -------- putWithRetry — context cancellation stops retry loop --------

func TestPutWithRetry_ContextCancellationAborts(t *testing.T) {
	client := &fakePutter{
		program: []programStep{
			{err: errors.New("connection reset by peer")},
			// second attempt should never be reached because ctx is
			// cancelled during the backoff wait.
			{err: nil, result: successResult("should-not-see")},
		},
	}
	// Slow enough backoff that the first cancel wins.
	s := &CasperSubmitter{
		client: client,
		retry: retryConfig{
			maxAttempts:    3,
			initialBackoff: 100 * time.Millisecond,
			backoffFactor:  2.0,
		},
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := s.putWithRetry(ctx, types.TransactionV1{}, "submit_proof")
	if err == nil {
		t.Fatalf("expected context-cancelled error, got success")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if got := client.calls.Load(); got != 1 {
		t.Fatalf("expected exactly 1 RPC call before cancel, got %d", got)
	}
}
