package wxgate

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/anchel/wechat-official-account-admin/internal/cache"
	"github.com/anchel/wechat-official-account-admin/internal/logger"
	"go.uber.org/zap"
)

// CacheItem 内存缓存项
type CacheItem struct {
	Value      interface{}
	ExpireTime time.Time
}

// TokenCacheItem 访问令牌结构体
type TokenCacheItem struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	ExpiresIn    int       `json:"expires_in"`
	ExpireTime   time.Time `json:"expire_time"`
	AppID        string    `json:"app_id"`
	OpenID       string    `json:"openid,omitempty"`
	Scope        string    `json:"scope,omitempty"`
	UnionID      string    `json:"unionid,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// TicketCacheItem JSAPI 票据结构体
type TicketCacheItem struct {
	Ticket     string    `json:"ticket"`
	ExpiresIn  int       `json:"expires_in"`
	ExpireTime time.Time `json:"expire_time"`
	AppID      string    `json:"app_id"`
	Type       string    `json:"type"`
	CreatedAt  time.Time `json:"created_at"`
}

// GateCacheManager 高并发两级缓存管理器（L1 本地内存 + L2 Redis，带防穿透单飞锁）
type GateCacheManager struct {
	l1Cache    map[string]*CacheItem
	mutex      sync.RWMutex
	singleLock map[string]*sync.Mutex
	lockMutex  sync.Mutex
}

var (
	defaultCacheManager *GateCacheManager
	cacheOnce           sync.Once
)

func GetCacheManager() *GateCacheManager {
	cacheOnce.Do(func() {
		cm := &GateCacheManager{
			l1Cache:    make(map[string]*CacheItem),
			singleLock: make(map[string]*sync.Mutex),
		}
		go cm.cleanupLoop()
		defaultCacheManager = cm
	})
	return defaultCacheManager
}

func (cm *GateCacheManager) getLock(key string) *sync.Mutex {
	cm.lockMutex.Lock()
	defer cm.lockMutex.Unlock()
	l, exists := cm.singleLock[key]
	if !exists {
		l = &sync.Mutex{}
		cm.singleLock[key] = l
	}
	return l
}

// Set 设置 L1 内存缓存
func (cm *GateCacheManager) Set(key string, value interface{}, ttl time.Duration) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.l1Cache[key] = &CacheItem{
		Value:      value,
		ExpireTime: time.Now().Add(ttl),
	}
}

// Get 获取 L1 内存缓存
func (cm *GateCacheManager) Get(key string) (interface{}, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	item, exists := cm.l1Cache[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(item.ExpireTime) {
		go cm.Delete(key)
		return nil, false
	}

	return item.Value, true
}

// Delete 删除缓存
func (cm *GateCacheManager) Delete(key string) {
	cm.mutex.Lock()
	delete(cm.l1Cache, key)
	cm.mutex.Unlock()

	if cache.RDB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = cache.RDB.Del(ctx, "wxgate:"+key).Err()
	}
}

// 定时清理过期缓存
func (cm *GateCacheManager) cleanupLoop() {
	ticker := time.NewTicker(3 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		cm.mutex.Lock()
		now := time.Now()
		for k, v := range cm.l1Cache {
			if now.After(v.ExpireTime) {
				delete(cm.l1Cache, k)
			}
		}
		cm.mutex.Unlock()
	}
}

// GetOrFetchClientAccessToken 获取全局 access_token（防穿透单飞合并）
func (cm *GateCacheManager) GetOrFetchClientAccessToken(ctx context.Context, appID string, fetcher func() (token string, expiresIn int, err error)) (string, error) {
	cacheKey := fmt.Sprintf("client_token:%s", appID)

	// 1. 尝试读 L1 内存缓存
	if val, ok := cm.Get(cacheKey); ok {
		if tc, ok := val.(*TokenCacheItem); ok && time.Now().Before(tc.ExpireTime) {
			return tc.AccessToken, nil
		}
	}

	// 2. 尝试读 L2 Redis
	if cache.RDB != nil {
		val, err := cache.RDB.Get(ctx, "wxgate:"+cacheKey).Result()
		if err == nil && val != "" {
			var tc TokenCacheItem
			if err := json.Unmarshal([]byte(val), &tc); err == nil && time.Now().Before(tc.ExpireTime) {
				cm.Set(cacheKey, &tc, time.Until(tc.ExpireTime))
				return tc.AccessToken, nil
			}
		}
	}

	// 3. 防击穿互斥锁 (SingleFlight 模式)
	mu := cm.getLock(cacheKey)
	mu.Lock()
	defer mu.Unlock()

	// Double check
	if val, ok := cm.Get(cacheKey); ok {
		if tc, ok := val.(*TokenCacheItem); ok && time.Now().Before(tc.ExpireTime) {
			return tc.AccessToken, nil
		}
	}

	logger.Info("[WXGate] Fetching fresh client access token from WeChat", zap.String("appid", appID))
	token, expiresIn, err := fetcher()
	if err != nil {
		return "", err
	}

	// 提前 300 秒刷新
	safeTTL := time.Duration(expiresIn-300) * time.Second
	if safeTTL <= 0 {
		safeTTL = 60 * time.Second
	}

	tc := &TokenCacheItem{
		AccessToken: token,
		ExpiresIn:   expiresIn,
		ExpireTime:  time.Now().Add(safeTTL),
		AppID:       appID,
		CreatedAt:   time.Now(),
	}

	// 存入 L1 内存
	cm.Set(cacheKey, tc, safeTTL)

	// 存入 L2 Redis
	if cache.RDB != nil {
		bytes, _ := json.Marshal(tc)
		_ = cache.RDB.Set(ctx, "wxgate:"+cacheKey, string(bytes), safeTTL).Err()
	}

	return token, nil
}

// GetOrFetchJsapiTicket 获取 JSSDK ticket（防穿透单飞合并）
func (cm *GateCacheManager) GetOrFetchJsapiTicket(ctx context.Context, appID string, fetcher func() (ticket string, expiresIn int, err error)) (string, error) {
	cacheKey := fmt.Sprintf("jsapi_ticket:%s", appID)

	// 1. 尝试读 L1
	if val, ok := cm.Get(cacheKey); ok {
		if tc, ok := val.(*TicketCacheItem); ok && time.Now().Before(tc.ExpireTime) {
			return tc.Ticket, nil
		}
	}

	// 2. 尝试读 L2
	if cache.RDB != nil {
		val, err := cache.RDB.Get(ctx, "wxgate:"+cacheKey).Result()
		if err == nil && val != "" {
			var tc TicketCacheItem
			if err := json.Unmarshal([]byte(val), &tc); err == nil && time.Now().Before(tc.ExpireTime) {
				cm.Set(cacheKey, &tc, time.Until(tc.ExpireTime))
				return tc.Ticket, nil
			}
		}
	}

	// 3. 锁防击穿
	mu := cm.getLock(cacheKey)
	mu.Lock()
	defer mu.Unlock()

	// Double check
	if val, ok := cm.Get(cacheKey); ok {
		if tc, ok := val.(*TicketCacheItem); ok && time.Now().Before(tc.ExpireTime) {
			return tc.Ticket, nil
		}
	}

	logger.Info("[WXGate] Fetching fresh jsapi ticket from WeChat", zap.String("appid", appID))
	ticket, expiresIn, err := fetcher()
	if err != nil {
		return "", err
	}

	safeTTL := time.Duration(expiresIn-300) * time.Second
	if safeTTL <= 0 {
		safeTTL = 60 * time.Second
	}

	tc := &TicketCacheItem{
		Ticket:     ticket,
		ExpiresIn:  expiresIn,
		ExpireTime: time.Now().Add(safeTTL),
		AppID:      appID,
		Type:       "jsapi",
		CreatedAt:  time.Now(),
	}

	cm.Set(cacheKey, tc, safeTTL)
	if cache.RDB != nil {
		bytes, _ := json.Marshal(tc)
		_ = cache.RDB.Set(ctx, "wxgate:"+cacheKey, string(bytes), safeTTL).Err()
	}

	return ticket, nil
}

// ClearAppAllCache 清除指定公众号所有凭证缓存
func (cm *GateCacheManager) ClearAppAllCache(appID string) {
	cm.Delete(fmt.Sprintf("client_token:%s", appID))
	cm.Delete(fmt.Sprintf("jsapi_ticket:%s", appID))
	cm.Delete(fmt.Sprintf("oauth_token:%s", appID))
}

// GetStatus 获取缓存状态统计
func (cm *GateCacheManager) GetStatus() map[string]interface{} {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	return map[string]interface{}{
		"l1_memory_items": len(cm.l1Cache),
		"l2_redis_online": cache.RDB != nil,
	}
}
