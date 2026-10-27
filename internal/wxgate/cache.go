package wxgate

import "github.com/anchel/wechat-official-account-admin/internal/cache"

type CacheItem = cache.CacheItem
type TokenCacheItem = cache.TokenCacheItem
type TicketCacheItem = cache.TicketCacheItem
type GateCacheManager = cache.GateCacheManager

func GetCacheManager() *cache.GateCacheManager {
	return cache.GetGateCacheManager()
}
