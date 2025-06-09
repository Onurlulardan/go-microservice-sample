package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// RateLimit - For IP and User limit info
type RateLimit struct {
	Count      int
	ResetAt    time.Time
	LastAccess time.Time
	Blocked    bool
	BlockUntil time.Time
}

// RateLimiter - Rate limitin Manager
type RateLimiter struct {
	store       map[string]*RateLimit
	mutex       sync.RWMutex
	cleanupTime time.Duration
}

// RateLimitConfig - Rate limiter configurations
type RateLimitConfig struct {
	MaxRequests   int
	TimeWindow    time.Duration
	BlockDuration time.Duration
}

// NewRateLimiter - Creates a new RateLimiter instance
func NewRateLimiter(cleanupTime time.Duration) *RateLimiter {
	limiter := &RateLimiter{
		store:       make(map[string]*RateLimit),
		cleanupTime: cleanupTime,
	}

	go limiter.cleanup()

	return limiter
}

// cleanup - Remove old records
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(rl.cleanupTime)
	defer ticker.Stop()

	for range ticker.C {
		rl.mutex.Lock()
		now := time.Now()
		for key, limit := range rl.store {
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

	if !exists {
		rl.store[key] = &RateLimit{
			Count:      1,
			ResetAt:    now.Add(config.TimeWindow),
			LastAccess: now,
			Blocked:    false,
		}
		return true
	}

	if limit.Blocked {
		if now.After(limit.BlockUntil) {
			limit.Blocked = false
			limit.Count = 1
			limit.ResetAt = now.Add(config.TimeWindow)
			limit.LastAccess = now
			return true
		}
		return false
	}

	if now.After(limit.ResetAt) {
		limit.Count = 1
		limit.ResetAt = now.Add(config.TimeWindow)
		limit.LastAccess = now
		return true
	}

	if limit.Count >= config.MaxRequests {
		limit.Blocked = true
		limit.BlockUntil = now.Add(config.BlockDuration)
		limit.LastAccess = now
		return false
	}

	limit.Count++
	limit.LastAccess = now
	return true
}

// RateLimitMiddleware - General rate limiting middleware
func (rl *RateLimiter) RateLimitMiddleware(config RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		key := clientIP

		if !rl.isAllowed(key, config) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Too many requests",
				"message": "Rate limit exceeded. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// LoginRateLimitMiddleware - Loing endpoint rate limiting middleware
func (rl *RateLimiter) LoginRateLimitMiddleware(config RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// IP adresini al
		clientIP := c.ClientIP()
		key := "login:" + clientIP

		if !rl.isAllowed(key, config) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Too many login attempts",
				"message": "Too many login attempts. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RegistrationRateLimitMiddleware - Registration endpoint rate limiting middleware
func (rl *RateLimiter) RegistrationRateLimitMiddleware(config RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		key := "register:" + clientIP

		if !rl.isAllowed(key, config) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Too many registration attempts",
				"message": "Too many registration attempts. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// PasswordResetRateLimitMiddleware - Password reset endpoint rate limiting middleware
func (rl *RateLimiter) PasswordResetRateLimitMiddleware(config RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		key := "password-reset:" + clientIP

		if !rl.isAllowed(key, config) {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Too many password reset attempts",
				"message": "Too many password reset attempts. Please try again later.",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
