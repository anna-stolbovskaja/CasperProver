package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Webhook subsystem — outbound event delivery to caller-registered URLs.
//
// Design:
//
//   * A caller POSTs to /v1/webhooks with a target URL, an optional
//     shared secret (used to sign payloads with HMAC-SHA256), and a
//     list of event kinds it wants to receive. The server assigns an
//     opaque id; the caller can DELETE /v1/webhooks/{id} to
//     unregister and GET /v1/webhooks to list its own registrations.
//
//   * When something interesting happens inside the engine
//     (proof.verified, proof.anchored, slash.executed,
//     governance.executed, …), a handler calls
//     Server.emitWebhookEvent(ctx, kind, payload). That call is
//     non-blocking: the event is enqueued and a background worker
//     fans it out to every subscriber that opted into the kind.
//
//   * Delivery is best-effort with exponential backoff (1s → 2s → 4s
//     … capped at 5 minutes) up to eight tries. Each attempt sends
//     the signed payload, records the last outcome on the
//     registration, and — if all eight attempts fail — moves the
//     event to an in-memory dead-letter list that operators can
//     inspect via GET /v1/webhooks/dead-letters.
//
//   * Signature contract (documented on the OpenAPI schema too):
//         X-CP-Signature = "sha256=" + hex(HMAC-SHA256(secret, body))
//         X-CP-Timestamp = RFC3339 timestamp of dispatch attempt
//         X-CP-Event     = event kind (e.g. "proof.verified")
//         X-CP-Delivery  = per-attempt UUID-ish id (subscription + seq)
//     Receivers MUST verify both the signature and that the timestamp
//     is fresh (we recommend a 5-minute window) to defeat replay.
//
//   * This is intentionally in-memory: single-node, non-durable, TTL
//     for dead-letters is 24 hours. A durable variant belongs in
//     Postgres and is tracked in KNOWN_LIMITATIONS.md.

// Recognized event kinds. Emit-side code MUST use one of these
// constants so a typo becomes a compile error rather than a silent
// misroute.
const (
	EventProofVerified     = "proof.verified"
	EventProofAnchored     = "proof.anchored"
	EventSlashExecuted     = "slash.executed"
	EventGovernanceExecute = "governance.executed"
	EventKYCGranted        = "kyc.granted"
	EventKYCRevoked        = "kyc.revoked"
)

// KnownWebhookEvents is exposed so the OpenAPI generator and admin
// UIs can enumerate what a caller may subscribe to without a
// hand-maintained duplicate list.
var KnownWebhookEvents = []string{
	EventProofVerified,
	EventProofAnchored,
	EventSlashExecuted,
	EventGovernanceExecute,
	EventKYCGranted,
	EventKYCRevoked,
}

// webhookSubscription is a caller-registered target.
type webhookSubscription struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Events    []string  `json:"events"`
	CreatedAt time.Time `json:"created_at"`
	OwnerKey  string    `json:"-"` // hash of API key that created it — never returned
	// secret is the shared HMAC secret. Not marshalled in list responses.
	secret string
	// Rolling delivery stats — inspectable via GET /v1/webhooks/{id}.
	Attempts       int       `json:"attempts"`
	Deliveries     int       `json:"deliveries"`
	Failures       int       `json:"failures"`
	LastAttemptAt  time.Time `json:"last_attempt_at,omitempty"`
	LastStatusCode int       `json:"last_status_code,omitempty"`
	LastError      string    `json:"last_error,omitempty"`
}

// webhookEvent is one dispatch unit — one (subscription, event)
// pairing that the worker will retry until success or dead-lettered.
type webhookEvent struct {
	SubID     string          `json:"sub_id"`
	Kind      string          `json:"kind"`
	Body      json.RawMessage `json:"body"`
	Attempts  int             `json:"attempts"`
	NextTryAt time.Time       `json:"next_try_at"`
	LastError string          `json:"last_error,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// deadLetter is the archived tail of an event that ran out of retries.
type deadLetter struct {
	Event    webhookEvent `json:"event"`
	FailedAt time.Time    `json:"failed_at"`
	URL      string       `json:"url"`
}

// webhookStore holds registrations, the pending queue, and dead
// letters. The zero value is not usable — call newWebhookStore.
type webhookStore struct {
	mu       sync.RWMutex
	subs     map[string]*webhookSubscription
	queue    []*webhookEvent
	dead     []*deadLetter
	stopping chan struct{}
	client   *http.Client
	now      func() time.Time // injectable for tests
}

const (
	webhookMaxAttempts     = 8
	webhookMaxBackoff      = 5 * time.Minute
	webhookInitialBackoff  = time.Second
	webhookDeadLetterLimit = 1000
	webhookRequestTimeout  = 10 * time.Second
)

func newWebhookStore() *webhookStore {
	s := &webhookStore{
		subs:     make(map[string]*webhookSubscription),
		stopping: make(chan struct{}),
		client:   &http.Client{Timeout: webhookRequestTimeout},
		now:      time.Now,
	}
	return s
}

// register creates a new subscription.
func (s *webhookStore) register(ownerKeyHash, url, secret string, events []string) (*webhookSubscription, error) {
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") {
		return nil, errors.New("url must start with http:// or https://")
	}
	if len(events) == 0 {
		return nil, errors.New("events list must not be empty")
	}
	// Validate every event kind so a typo fails registration rather
	// than silently subscribing to a phantom kind.
	for _, e := range events {
		if !isKnownEvent(e) {
			return nil, fmt.Errorf("unknown event kind %q", e)
		}
	}
	id := newWebhookID(s.now())
	sub := &webhookSubscription{
		ID:        id,
		URL:       url,
		Events:    append([]string(nil), events...),
		CreatedAt: s.now(),
		OwnerKey:  ownerKeyHash,
		secret:    secret,
	}
	s.mu.Lock()
	s.subs[id] = sub
	s.mu.Unlock()
	return sub, nil
}

// unregister removes a subscription. Enforced against ownerKeyHash
// so caller A cannot delete caller B's webhooks.
func (s *webhookStore) unregister(id, ownerKeyHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sub, ok := s.subs[id]
	if !ok {
		return errors.New("not found")
	}
	if sub.OwnerKey != ownerKeyHash {
		return errors.New("not found") // deliberately opaque
	}
	delete(s.subs, id)
	return nil
}

// list returns every subscription owned by ownerKeyHash. The order is
// insertion-time-ish (map iteration then sort by CreatedAt).
func (s *webhookStore) list(ownerKeyHash string) []*webhookSubscription {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*webhookSubscription, 0, len(s.subs))
	for _, sub := range s.subs {
		if sub.OwnerKey == ownerKeyHash {
			// Shallow copy so we never leak the secret via the API.
			cp := *sub
			cp.secret = ""
			out = append(out, &cp)
		}
	}
	// Stable ordering — smallest CreatedAt first.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].CreatedAt.Before(out[j-1].CreatedAt); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// enqueue fans one event out to every matching subscription. Called
// by handlers via Server.emitWebhookEvent.
func (s *webhookStore) enqueue(kind string, body []byte) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	added := 0
	now := s.now()
	for _, sub := range s.subs {
		if !subscribedTo(sub.Events, kind) {
			continue
		}
		s.queue = append(s.queue, &webhookEvent{
			SubID:     sub.ID,
			Kind:      kind,
			Body:      append(json.RawMessage(nil), body...),
			NextTryAt: now,
			CreatedAt: now,
		})
		added++
	}
	return added
}

// deliverOnce processes every event whose NextTryAt has passed.
// Returns the number of events attempted. Idempotent and safe to
// call from the worker loop or a test.
func (s *webhookStore) deliverOnce(ctx context.Context) int {
	s.mu.Lock()
	now := s.now()
	pending := make([]*webhookEvent, 0)
	remaining := make([]*webhookEvent, 0, len(s.queue))
	for _, ev := range s.queue {
		if !ev.NextTryAt.After(now) {
			pending = append(pending, ev)
		} else {
			remaining = append(remaining, ev)
		}
	}
	s.queue = remaining
	// Snapshot subs so we can release the lock during delivery.
	subCopy := make(map[string]*webhookSubscription, len(s.subs))
	for k, v := range s.subs {
		subCopy[k] = v
	}
	s.mu.Unlock()

	attempted := 0
	for _, ev := range pending {
		sub, ok := subCopy[ev.SubID]
		if !ok {
			// Subscription vanished (unregistered between enqueue and now).
			continue
		}
		attempted++
		s.deliverEvent(ctx, sub, ev)
	}
	return attempted
}

// deliverEvent sends one dispatch attempt and mutates the event with
// the outcome — either finalized (dropped from queue), retried
// (pushed back with backoff), or dead-lettered.
func (s *webhookStore) deliverEvent(ctx context.Context, sub *webhookSubscription, ev *webhookEvent) {
	sig := signWebhook(sub.secret, ev.Body)
	deliveryID := fmt.Sprintf("%s-%d", sub.ID, ev.Attempts+1)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sub.URL, bytes.NewReader(ev.Body))
	if err != nil {
		s.recordFailure(sub, ev, fmt.Sprintf("build request: %v", err), 0)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CP-Event", ev.Kind)
	req.Header.Set("X-CP-Delivery", deliveryID)
	req.Header.Set("X-CP-Timestamp", s.now().UTC().Format(time.RFC3339))
	if sig != "" {
		req.Header.Set("X-CP-Signature", sig)
	}
	resp, err := s.client.Do(req)
	if err != nil {
		s.recordFailure(sub, ev, err.Error(), 0)
		return
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		s.recordSuccess(sub, ev, resp.StatusCode)
		return
	}
	s.recordFailure(sub, ev, fmt.Sprintf("http %d", resp.StatusCode), resp.StatusCode)
}

func (s *webhookStore) recordSuccess(sub *webhookSubscription, ev *webhookEvent, code int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sub.Attempts++
	sub.Deliveries++
	sub.LastAttemptAt = s.now()
	sub.LastStatusCode = code
	sub.LastError = ""
}

func (s *webhookStore) recordFailure(sub *webhookSubscription, ev *webhookEvent, msg string, code int) {
	s.mu.Lock()
	sub.Attempts++
	sub.Failures++
	sub.LastAttemptAt = s.now()
	sub.LastStatusCode = code
	sub.LastError = msg
	ev.Attempts++
	ev.LastError = msg
	if ev.Attempts >= webhookMaxAttempts {
		dl := &deadLetter{Event: *ev, FailedAt: s.now(), URL: sub.URL}
		s.dead = append(s.dead, dl)
		if len(s.dead) > webhookDeadLetterLimit {
			s.dead = s.dead[len(s.dead)-webhookDeadLetterLimit:]
		}
		s.mu.Unlock()
		return
	}
	// Exponential backoff: 1s, 2s, 4s, 8s, 16s, 32s, 64s, 128s — capped.
	backoff := webhookInitialBackoff << (ev.Attempts - 1)
	if backoff > webhookMaxBackoff || backoff < 0 {
		backoff = webhookMaxBackoff
	}
	ev.NextTryAt = s.now().Add(backoff)
	s.queue = append(s.queue, ev)
	s.mu.Unlock()
}

func (s *webhookStore) deadLetters() []*deadLetter {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*deadLetter, len(s.dead))
	copy(out, s.dead)
	return out
}

// runWorker delivers events on a tick until the store's stopping
// channel closes. Callers spawn it once at server start.
func (s *webhookStore) runWorker(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopping:
			return
		case <-t.C:
			s.deliverOnce(ctx)
		}
	}
}

// signWebhook computes the HMAC-SHA256 header value. An empty secret
// disables signing (the header is omitted, and the receiver must not
// require it) — useful for local demos where the whole path is
// localhost.
func signWebhook(secret string, body []byte) string {
	if secret == "" {
		return ""
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// VerifyWebhookSignature is exposed for SDKs and tests: given the
// shared secret, the raw body, and the received header value, return
// true if the signature matches. Uses subtle.ConstantTimeCompare to
// avoid timing leaks.
func VerifyWebhookSignature(secret string, body []byte, header string) bool {
	if secret == "" || header == "" {
		return false
	}
	want := signWebhook(secret, body)
	return subtle.ConstantTimeCompare([]byte(want), []byte(header)) == 1
}

func isKnownEvent(kind string) bool {
	for _, e := range KnownWebhookEvents {
		if e == kind {
			return true
		}
	}
	return false
}

func subscribedTo(list []string, kind string) bool {
	for _, e := range list {
		if e == kind || e == "*" {
			return true
		}
	}
	return false
}

// newWebhookID mints a short, sortable-ish subscription id. Format:
// "wh_" + hex(6 bytes of unix nanos xor'd) — collision-resistant
// enough for a demo and human-readable.
func newWebhookID(now time.Time) string {
	nanos := now.UnixNano()
	b := make([]byte, 6)
	for i := 0; i < 6; i++ {
		b[i] = byte(nanos >> (i * 8))
	}
	// Fold in a per-call counter via a monotonic bump so two calls in
	// the same nanosecond can't collide.
	webhookIDCounter.mu.Lock()
	webhookIDCounter.n++
	c := webhookIDCounter.n
	webhookIDCounter.mu.Unlock()
	b[0] ^= byte(c)
	b[1] ^= byte(c >> 8)
	return "wh_" + hex.EncodeToString(b)
}

var webhookIDCounter = struct {
	mu sync.Mutex
	n  uint64
}{}
