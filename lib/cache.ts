import NodeCache from 'node-cache';

// Main cache for API responses and computed data
export const cache = new NodeCache({
  stdTTL: 60,        // Default 60 seconds
  checkperiod: 120,  // Check for expired keys every 2 minutes
  useClones: false,  // Don't clone values (faster, but mutable)
  maxKeys: 10000,    // Prevent unbounded growth
});

// Cache key generators
export const CK = {
  sessions: () => 'sessions:all',
  stats: (range: string, date?: string) => `stats:${range}:${date || 'today'}`,
  groupDetail: (chatroomId: string, date: string) => `group:${chatroomId}:${date}`,
  mentions: () => 'mentions:count',
  links: (date: string) => `links:${date}`,
  topics: (date: string) => `topics:${date}`,
  intelligence: (date: string) => `intelligence:${date}`,
  larkChats: () => 'lark:chats',
  larkFilter: () => 'lark:filter',
  search: (q: string) => `search:${q}`,
} as const;

/** Wrap a function with caching */
export async function withCache<T>(
  key: string,
  ttl: number,
  fn: () => Promise<T>,
): Promise<T> {
  const cached = cache.get<T>(key);
  if (cached !== undefined) return cached;
  const result = await fn();
  cache.set(key, result, ttl);
  return result;
}

/** Wrap a sync function with caching */
export function withCacheSync<T>(
  key: string,
  ttl: number,
  fn: () => T,
): T {
  const cached = cache.get<T>(key);
  if (cached !== undefined) return cached;
  const result = fn();
  cache.set(key, result, ttl);
  return result;
}

/** Invalidate cache keys by pattern (simple prefix match) */
export function invalidateCache(pattern: string): void {
  const keys = cache.keys();
  for (const key of keys) {
    if (key.startsWith(pattern)) {
      cache.del(key);
    }
  }
}

/** Get cache stats for monitoring */
export function cacheStats() {
  return {
    keys: cache.keys().length,
    hits: cache.getStats().hits,
    misses: cache.getStats().misses,
    vsize: cache.getStats().vsize,
  };
}
