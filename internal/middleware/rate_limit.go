package middleware

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"app/internal/cache"
	"app/internal/errs"
	"app/internal/logger"

	"github.com/gin-gonic/gin"
)

const (
	// Bursts are counted per IP. A whole office sits behind one NAT address, so this
	// has to fit several people signing in at once and only catch scripted hammering.
	loginBurstWindow = time.Minute
	loginBurstLimit  = 60

	// Rejected credentials are counted per email+IP: one person retrying a forgotten
	// password must not lock every colleague behind the same address out.
	loginFailureWindow = time.Hour
	loginFailureLimit  = 40

	// loginStrikeWindow is how long a served block is remembered for escalation,
	// counted from the moment that block ends.
	loginStrikeWindow = 24 * time.Hour

	// Only this much of the body is buffered to read the email out of it. Login
	// payloads are a few hundred bytes; anything larger is passed through untouched.
	authBodyPeekLimit = 8 << 10

	// Emails end up inside cache keys, so an oversized one is truncated.
	authEmailKeyMaxLen = 190
)

// loginBlockLadder is the lockout length per strike, where a strike is one block this
// identity has already collected. The first lockout is deliberately short: someone who
// cannot recall which password they used should not lose their afternoon over it. Only
// sustained guessing, loginFailureLimit fresh failures per rung, reaches a full day.
var loginBlockLadder = []time.Duration{
	5 * time.Minute,
	15 * time.Minute,
	time.Hour,
	6 * time.Hour,
	24 * time.Hour,
}

// authSuccessKey marks a request that issued tokens.
const authSuccessKey = "auth_rate_limit_success"

// MarkAuthSuccess lets AuthRateLimit forget everything it has accumulated for this
// identity, including the escalation ladder, once login actually completed.
func MarkAuthSuccess(c *gin.Context) {
	c.Set(authSuccessKey, true)
}

// AuthRateLimit throttles the login endpoint.
//
// ClientIP must be trustworthy for this to mean anything: the router has to declare
// its trusted proxies (see internal/server), otherwise a client can forge
// X-Forwarded-For and get a fresh bucket per request.
func AuthRateLimit(c cache.Cache, log *logger.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if c == nil {
			ctx.Next()
			return
		}

		reqCtx := ctx.Request.Context()
		ip := ctx.ClientIP()
		identity := authIdentity(peekAuthEmail(ctx), ip)

		if until, blocked := blockedUntil(reqCtx, c, identity); blocked {
			rejectRateLimited(ctx, log, identity, "blocked", time.Until(until))
			return
		}

		burst, burstResetsAt := bumpWindow(reqCtx, c, burstKey("login", ip), loginBurstWindow)
		if burst > loginBurstLimit {
			rejectRateLimited(ctx, log, ip, "burst", time.Until(burstResetsAt))
			return
		}

		ctx.Next()

		switch {
		case ctx.GetBool(authSuccessKey):
			clearAuthFailures(reqCtx, c, identity)
		case ctx.Writer.Status() == http.StatusUnauthorized:
			recordAuthFailure(reqCtx, c, identity)
		}
	}
}

// RateLimit caps how often a single IP may call the routes it guards: limit requests
// per window, counted under scope. It only stops hammering and keeps no failure counter,
// so use it for endpoints such as token refresh or "send me a reset link".
func RateLimit(c cache.Cache, log *logger.Logger, scope string, limit int, window time.Duration) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if c == nil {
			ctx.Next()
			return
		}

		ip := ctx.ClientIP()
		count, resetsAt := bumpWindow(ctx.Request.Context(), c, burstKey(scope, ip), window)
		if count > limit {
			rejectRateLimited(ctx, log, ip, scope+"_burst", time.Until(resetsAt))
			return
		}

		ctx.Next()
	}
}

func recordAuthFailure(ctx context.Context, c cache.Cache, identity string) {
	failures, _ := bumpWindow(ctx, c, failureKey(identity), loginFailureWindow)
	if failures < loginFailureLimit {
		return
	}

	block := blockDuration(bumpStrike(ctx, c, identity))
	_ = c.Set(ctx, blockKey(identity), time.Now().Add(block), block)

	// The failure counter goes away with the block that it armed. Keeping it would
	// leave the identity one failure short of the next block the moment this one
	// expires, which is how a 15 minute lockout used to stretch into hours.
	_ = c.Forget(ctx, failureKey(identity))
}

func clearAuthFailures(ctx context.Context, c cache.Cache, identity string) {
	_ = c.Forget(ctx, failureKey(identity))
	_ = c.Forget(ctx, blockKey(identity))
	_ = c.Forget(ctx, strikeKey(identity))
}

func blockDuration(strikes int) time.Duration {
	if strikes < 1 {
		strikes = 1
	}
	if strikes > len(loginBlockLadder) {
		strikes = len(loginBlockLadder)
	}
	return loginBlockLadder[strikes-1]
}

// bumpStrike advances the escalation ladder and returns the new strike count. The
// window slides on every strike, which is safe here because a strike costs
// loginFailureLimit fresh failures; it still decays loginStrikeWindow after the block
// it armed has ended, or immediately once the login succeeds.
func bumpStrike(ctx context.Context, c cache.Cache, identity string) int {
	now := time.Now()

	var w windowCounter
	if err := c.Get(ctx, strikeKey(identity), &w); err != nil || !now.Before(w.ExpiresAt) {
		w = windowCounter{}
	}
	w.Count++
	w.ExpiresAt = now.Add(blockDuration(w.Count) + loginStrikeWindow)

	_ = c.Set(ctx, strikeKey(identity), w, time.Until(w.ExpiresAt))

	return w.Count
}

func blockedUntil(ctx context.Context, c cache.Cache, identity string) (time.Time, bool) {
	var until time.Time
	if err := c.Get(ctx, blockKey(identity), &until); err != nil {
		return time.Time{}, false
	}
	return until, time.Now().Before(until)
}

// windowCounter carries its own expiry so the window is fixed. Relying on the cache
// TTL and rewriting it on every hit turns "N per minute" into "N ever, as long as you
// keep trying", which blocks people who are merely slow rather than scripted.
type windowCounter struct {
	Count     int       `json:"count"`
	ExpiresAt time.Time `json:"expires_at"`
}

// bumpWindow returns the new count and the moment the window rolls over.
func bumpWindow(ctx context.Context, c cache.Cache, key string, window time.Duration) (int, time.Time) {
	now := time.Now()

	var w windowCounter
	if err := c.Get(ctx, key, &w); err != nil || !now.Before(w.ExpiresAt) {
		w = windowCounter{ExpiresAt: now.Add(window)}
	}
	w.Count++

	ttl := time.Until(w.ExpiresAt)
	if ttl <= 0 {
		ttl = time.Second
	}
	_ = c.Set(ctx, key, w, ttl)

	return w.Count, w.ExpiresAt
}

// peekAuthEmail reads the email out of the JSON body and restores the body for the
// handler. An unparsable or oversized body yields "", which falls back to IP-only
// keying rather than failing the request.
func peekAuthEmail(ctx *gin.Context) string {
	if ctx.Request.Body == nil {
		return ""
	}

	buf, err := io.ReadAll(io.LimitReader(ctx.Request.Body, authBodyPeekLimit+1))
	if err != nil {
		ctx.Request.Body = io.NopCloser(bytes.NewReader(buf))
		return ""
	}

	if len(buf) > authBodyPeekLimit {
		ctx.Request.Body = io.NopCloser(io.MultiReader(bytes.NewReader(buf), ctx.Request.Body))
		return ""
	}
	ctx.Request.Body = io.NopCloser(bytes.NewReader(buf))

	var payload struct {
		Email string `json:"email"`
	}
	if json.Unmarshal(buf, &payload) != nil {
		return ""
	}

	email := strings.ToLower(strings.TrimSpace(payload.Email))
	if len(email) > authEmailKeyMaxLen {
		email = email[:authEmailKeyMaxLen]
	}
	return email
}

func authIdentity(email, ip string) string {
	if email == "" {
		return ip
	}
	return email + "|" + ip
}

// rejectRateLimited answers 429 and states how long the wait is, so the caller can
// tell "wait a minute" apart from a day-long lockout. The wording lives in the client.
func rejectRateLimited(ctx *gin.Context, log *logger.Logger, identity, reason string, retryAfter time.Duration) {
	seconds := int(math.Ceil(retryAfter.Seconds()))
	if seconds < 1 {
		seconds = 1
	}

	if log != nil {
		log.WarnContext(ctx.Request.Context(), "Auth rate limit triggered",
			"identity", identity, "reason", reason,
			"retry_after_seconds", seconds, "path", ctx.FullPath())
	}

	ctx.Header("Retry-After", strconv.Itoa(seconds))
	errs.RespondWithError(ctx, errs.NewTooManyRequestsError(
		errs.ErrKeyAuthTooManyRequests, "Too many attempts. Please try again later.",
	).WithDetails(map[string]interface{}{"retry_after_seconds": seconds}))
	ctx.Abort()
}

func burstKey(scope, ip string) string  { return "auth:rl:burst:" + scope + ":" + ip }
func failureKey(identity string) string { return "auth:rl:fail:" + identity }
func blockKey(identity string) string   { return "auth:rl:block:" + identity }
func strikeKey(identity string) string  { return "auth:rl:strike:" + identity }
