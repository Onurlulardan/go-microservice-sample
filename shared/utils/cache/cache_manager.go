package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"

	"forgecrud-backend/shared/config"
)

type CacheManager struct {
	client *redis.Client
	ctx    context.Context
}

type PermissionCacheData struct {
	HasPermission bool                   `json:"has_permission"`
	UserID        uint                   `json:"user_id"`
	Resource      string                 `json:"resource"`
	Action        string                 `json:"action"`
	FoundAt       string                 `json:"found_at"` // "user", "role", "organization"
	CachedAt      time.Time              `json:"cached_at"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

var (
	globalCacheManager *CacheManager
	DefaultTTL         = 30 * time.Minute
	UserPermissionTTL  = 15 * time.Minute
	RolePermissionTTL  = 1 * time.Hour
	OrgPermissionTTL   = 2 * time.Hour
)

// InitCacheManager initializes the global cache manager
func InitCacheManager() error {
	cfg := config.GetConfig()

	redisDB, err := strconv.Atoi(cfg.RedisDB)
	if err != nil {
		log.Printf("❌ Invalid Redis DB number: %s, using default 0", cfg.RedisDB)
		redisDB = 0
	}

	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPassword,
		DB:       redisDB,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %v", err)
	}

	globalCacheManager = &CacheManager{
		client: client,
		ctx:    ctx,
	}

	log.Printf("✅ Redis Cache Manager initialized successfully - %s:%s DB:%d",
		cfg.RedisHost, cfg.RedisPort, redisDB)

	return nil
}

// GetCacheManager returns the global cache manager instance
func GetCacheManager() *CacheManager {
	if globalCacheManager == nil {
		if err := InitCacheManager(); err != nil {
			log.Printf("❌ Failed to initialize cache manager: %v", err)
			return nil
		}
	}
	return globalCacheManager
}

// GeneratePermissionKey generates a cache key for permission
func GeneratePermissionKey(userID uint, resource, action string) string {
	return fmt.Sprintf("perm:user:%d:res:%s:act:%s", userID, resource, action)
}

// GenerateUserPermissionsKey generates a cache key for all user permissions
func GenerateUserPermissionsKey(userID uint) string {
	return fmt.Sprintf("perm:user:%d:*", userID)
}

// GenerateRolePermissionsKey generates a cache key for role permissions
func GenerateRolePermissionsKey(roleID uint) string {
	return fmt.Sprintf("perm:role:%d:*", roleID)
}

// GenerateOrgPermissionsKey generates a cache key for organization permissions
func GenerateOrgPermissionsKey(orgID uint) string {
	return fmt.Sprintf("perm:org:%d:*", orgID)
}

// SetPermissionCache caches a permission check result
func (cm *CacheManager) SetPermissionCache(userID uint, resource, action string, data *PermissionCacheData) error {
	if cm == nil || cm.client == nil {
		return fmt.Errorf("cache manager not initialized")
	}

	key := GeneratePermissionKey(userID, resource, action)

	// Set TTL based on where the permission was found
	var ttl time.Duration
	switch data.FoundAt {
	case "user":
		ttl = UserPermissionTTL
	case "role":
		ttl = RolePermissionTTL
	case "organization":
		ttl = OrgPermissionTTL
	default:
		ttl = DefaultTTL
	}

	// Add timestamp
	data.CachedAt = time.Now()

	// Serialize data
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %v", err)
	}

	// Set in Redis
	err = cm.client.Set(cm.ctx, key, jsonData, ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache: %v", err)
	}

	log.Printf("🔄 Permission cached: %s (TTL: %v, FoundAt: %s)", key, ttl, data.FoundAt)
	return nil
}

// GetPermissionCache retrieves a cached permission check result
func (cm *CacheManager) GetPermissionCache(userID uint, resource, action string) (*PermissionCacheData, bool) {
	if cm == nil || cm.client == nil {
		return nil, false
	}

	key := GeneratePermissionKey(userID, resource, action)

	result, err := cm.client.Get(cm.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			log.Printf("🔍 Cache miss: %s", key)
			return nil, false
		}
		log.Printf("❌ Cache error: %v", err)
		return nil, false
	}

	var data PermissionCacheData
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		log.Printf("❌ Failed to unmarshal cache data: %v", err)
		return nil, false
	}

	log.Printf("✅ Cache hit: %s (Age: %v)", key, time.Since(data.CachedAt))
	return &data, true
}

// InvalidateUserPermissions invalidates all permissions for a user
func (cm *CacheManager) InvalidateUserPermissions(userID uint) error {
	if cm == nil || cm.client == nil {
		return fmt.Errorf("cache manager not initialized")
	}

	pattern := fmt.Sprintf("perm:user:%d:*", userID)
	return cm.invalidateByPattern(pattern)
}

// InvalidateRolePermissions invalidates all permissions for a role
func (cm *CacheManager) InvalidateRolePermissions(roleID uint) error {
	if cm == nil || cm.client == nil {
		return fmt.Errorf("cache manager not initialized")
	}

	pattern := fmt.Sprintf("perm:role:%d:*", roleID)
	return cm.invalidateByPattern(pattern)
}

// InvalidateOrgPermissions invalidates all permissions for an organization
func (cm *CacheManager) InvalidateOrgPermissions(orgID uint) error {
	if cm == nil || cm.client == nil {
		return fmt.Errorf("cache manager not initialized")
	}

	pattern := fmt.Sprintf("perm:org:%d:*", orgID)
	return cm.invalidateByPattern(pattern)
}

// InvalidateSpecificPermission invalidates a specific permission
func (cm *CacheManager) InvalidateSpecificPermission(userID uint, resource, action string) error {
	if cm == nil || cm.client == nil {
		return fmt.Errorf("cache manager not initialized")
	}

	key := GeneratePermissionKey(userID, resource, action)
	err := cm.client.Del(cm.ctx, key).Err()
	if err != nil {
		return fmt.Errorf("failed to delete cache key %s: %v", key, err)
	}

	log.Printf("🗑️  Cache invalidated: %s", key)
	return nil
}

// InvalidateAllPermissions invalidates all permission caches
func (cm *CacheManager) InvalidateAllPermissions() error {
	if cm == nil || cm.client == nil {
		return fmt.Errorf("cache manager not initialized")
	}

	pattern := "perm:*"
	return cm.invalidateByPattern(pattern)
}

// invalidateByPattern invalidates cache entries matching a pattern
func (cm *CacheManager) invalidateByPattern(pattern string) error {
	iter := cm.client.Scan(cm.ctx, 0, pattern, 0).Iterator()
	var keys []string

	for iter.Next(cm.ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan keys: %v", err)
	}

	if len(keys) > 0 {
		err := cm.client.Del(cm.ctx, keys...).Err()
		if err != nil {
			return fmt.Errorf("failed to delete keys: %v", err)
		}
		log.Printf("🗑️  Cache invalidated: %d keys matching pattern '%s'", len(keys), pattern)
	} else {
		log.Printf("🔍 No cache keys found for pattern: %s", pattern)
	}

	return nil
}

// GetCacheStats returns cache statistics
func (cm *CacheManager) GetCacheStats() (map[string]interface{}, error) {
	if cm == nil || cm.client == nil {
		return nil, fmt.Errorf("cache manager not initialized")
	}

	// Get Redis info
	info, err := cm.client.Info(cm.ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis info: %v", err)
	}

	// Count permission keys
	iter := cm.client.Scan(cm.ctx, 0, "perm:*", 0).Iterator()
	keyCount := 0
	for iter.Next(cm.ctx) {
		keyCount++
	}

	stats := map[string]interface{}{
		"total_permission_keys": keyCount,
		"redis_info":            info,
		"cache_manager_active":  true,
	}

	return stats, nil
}

// TestConnection tests the Redis connection
func (cm *CacheManager) TestConnection() error {
	if cm == nil || cm.client == nil {
		return fmt.Errorf("cache manager not initialized")
	}

	// Test basic operations
	testKey := "test:connection"
	testValue := "connection_test_ok"

	// Set test value
	err := cm.client.Set(cm.ctx, testKey, testValue, time.Minute).Err()
	if err != nil {
		return fmt.Errorf("failed to set test value: %v", err)
	}

	// Get test value
	result, err := cm.client.Get(cm.ctx, testKey).Result()
	if err != nil {
		return fmt.Errorf("failed to get test value: %v", err)
	}

	if result != testValue {
		return fmt.Errorf("test value mismatch: expected %s, got %s", testValue, result)
	}

	// Delete test value
	err = cm.client.Del(cm.ctx, testKey).Err()
	if err != nil {
		return fmt.Errorf("failed to delete test value: %v", err)
	}

	log.Println("✅ Redis connection test passed")
	return nil
}

// Close closes the cache manager connection
func (cm *CacheManager) Close() error {
	if cm != nil && cm.client != nil {
		return cm.client.Close()
	}
	return nil
}
