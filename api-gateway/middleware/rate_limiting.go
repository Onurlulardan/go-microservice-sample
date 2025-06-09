package middleware

import (
	"net/http"
	"sync"
	"time"

	"forgecrud-backend/shared/config"

	"github.com/gin-gonic/gin"
)

// RateLimit - Rate limit info for IP addresses
type RateLimit struct {
	Count      int
	ResetAt    time.Time
	LastAccess time.Time
	Blocked    bool
	BlockUntil time.Time
}

// RateLimiter - Global rate limiter for API Gateway
type RateLimiter struct {
	store       map[string]*RateLimit
	mutex       sync.RWMutex
	cleanupTime time.Duration
}

// RateLimitConfig - Rate limiter configuration
type RateLimitConfig struct {
	MaxRequests   int
	TimeWindow    time.Duration
	BlockDuration time.Duration
}

// NewRateLimitConfig - Creates a new RateLimitConfig from environment variables
func NewRateLimitConfig() RateLimitConfig {
	cfg := config.GetConfig()

	return RateLimitConfig{
		MaxRequests:   cfg.GetRateLimitMaxRequests(),
		TimeWindow:    time.Duration(cfg.GetRateLimitTimeWindowSeconds()) * time.Second,
		BlockDuration: time.Duration(cfg.GetRateLimitBlockDurationMinutes()) * time.Minute,
	}
}

// NewRateLimiter - Creates a new RateLimiter instance
func NewRateLimiter(cleanupTime time.Duration) *RateLimiter {
	limiter := &RateLimiter{
		store:       make(map[string]*RateLimit),
		cleanupTime: cleanupTime,
	}

	// Start cleanup goroutine
	go limiter.cleanup()

	return limiter
}

// cleanup - Remove old records periodically
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupTime)
	defer ticker.Stop()

	for range ticker.C {
		rl.mutex.Lock()
		now := time.Now()
		for key, limit := range rl.store {
			// Remove entries older than 24 hours
			if now.Sub(limit.LastAccess) > 24*time.Hour {
				delete(rl.store, key)
			}
		}
		rl.mutex.Unlock()
	}
}

// isAllowed - Checks if the request is allowed based on rate limiting
func (rl *RateLimiter) isAllowed(key string, config RateLimitConfig) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	limit, exists := rl.store[key]

	// First request from this key
	if !exists {
		rl.store[key] = &RateLimit{
			Count:      1,
			ResetAt:    now.Add(config.TimeWindow),
			LastAccess: now,
			Blocked:    false,
		}
		return true
	}

	// Check if currently blocked
	if limit.Blocked {
		if now.After(limit.BlockUntil) {
			// Block period expired, reset
			limit.Blocked = false
			limit.Count = 1
			limit.ResetAt = now.Add(config.TimeWindow)
			limit.LastAccess = now
			return true
		}
		return false // Still blocked
	}

	// Reset window if time expired
	if now.After(limit.ResetAt) {
		limit.Count = 1
		limit.ResetAt = now.Add(config.TimeWindow)
		limit.LastAccess = now
		return true
	}

	// Check if limit exceeded
	if limit.Count >= config.MaxRequests {
		limit.Blocked = true
		limit.BlockUntil = now.Add(config.BlockDuration)
		limit.LastAccess = now
		return false
	}

	// Allow request and increment count
	limit.Count++
	limit.LastAccess = now
	return true
}

// GlobalRateLimitMiddleware - Global rate limiting for all API Gateway requests
func (rl *RateLimiter) GlobalRateLimitMiddleware(config RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		key := "global:" + clientIP

		if !rl.isAllowed(key, config) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":       "Rate limit exceeded",
				"message":     "Too many requests from this IP. Please try again later.",
				"retry_after": config.BlockDuration.Seconds(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
