// Package tokenbucket 提供基于内存的令牌桶限流器实现（V2 2.1，V3.6 跟随 ratelimit 迁移）。
//
// 该实现零外部依赖（不依赖 golang.org/x/time），适合单机部署。
// 分布式场景下，应用层应自行实现基于 Redis 的限流器。
//
// 使用示例：
//
//	import (
//	    ruleflow "github.com/vpptu/ruleflow/pkg/ruleflow/core"
//	    tb "github.com/vpptu/ruleflow/pkg/ruleflow/contrib/ratelimit/tokenbucket"
//	)
//
//	// 共享一个全局限流器：每秒 1000 个令牌，桶容量 2000
//	lim := tb.NewGlobalLimiter(1000, 2000)
//	// 或者按 key 维度（每个 key 独立桶）
//	lim := tb.NewPerKeyLimiter(100, 100)
//
//	engine := ruleflow.NewEngine(ruleflow.WithLimiter(lim))
package tokenbucket

import (
	"context"
	"sync"
	"time"

	"github.com/vpptu/ruleflow/pkg/ruleflow/core/contract"
)

// TokenBucket 内存令牌桶（线程安全）
type TokenBucket struct {
	mu         sync.Mutex
	rate       float64   // 每秒填充令牌数
	burst      float64   // 桶容量
	tokens     float64   // 当前令牌数
	lastRefill time.Time // 上次填充时间
}

// NewTokenBucket 构造令牌桶
//   - rate:  每秒填充的令牌数
//   - burst: 桶容量（最大令牌数）
func NewTokenBucket(rate, burst float64) *TokenBucket {
	if rate <= 0 {
		rate = 1
	}
	if burst <= 0 {
		burst = rate
	}
	return &TokenBucket{
		rate:       rate,
		burst:      burst,
		tokens:     burst, // 初始满桶
		lastRefill: time.Now(),
	}
}

// allow 尝试消费 1 个令牌
func (tb *TokenBucket) allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.rate
	if tb.tokens > tb.burst {
		tb.tokens = tb.burst
	}
	tb.lastRefill = now

	if tb.tokens >= 1 {
		tb.tokens--
		return true
	}
	return false
}

// wait 阻塞等待直到拿到 1 个令牌或 ctx 取消
func (tb *TokenBucket) wait(ctx context.Context) error {
	for {
		if tb.allow() {
			return nil
		}
		// 计算到下一个令牌的等待时间
		tb.mu.Lock()
		wait := time.Duration((1 - tb.tokens) / tb.rate * float64(time.Second))
		tb.mu.Unlock()
		if wait <= 0 {
			wait = time.Millisecond
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}
}

// ─────────────────────────────────────────────
//  GlobalLimiter — 全局单桶
// ─────────────────────────────────────────────

// GlobalLimiter 所有 key 共享同一个令牌桶
type GlobalLimiter struct {
	bucket *TokenBucket
}

// NewGlobalLimiter 构造全局限流器
func NewGlobalLimiter(rate, burst float64) *GlobalLimiter {
	return &GlobalLimiter{bucket: NewTokenBucket(rate, burst)}
}

// Allow 实现 core.Limiter
func (g *GlobalLimiter) Allow(_ string) bool { return g.bucket.allow() }

// Wait 实现 core.Limiter
func (g *GlobalLimiter) Wait(ctx context.Context, _ string) error { return g.bucket.wait(ctx) }

// 确保实现 contract.Limiter 接口
var _ contract.Limiter = (*GlobalLimiter)(nil)

// ─────────────────────────────────────────────
//  PerKeyLimiter — 每 key 独立桶
// ─────────────────────────────────────────────

// PerKeyLimiter 每个 key 维护独立的令牌桶。
// 适用于"按规则限流"、"按设备限流"等场景。
type PerKeyLimiter struct {
	mu       sync.RWMutex
	buckets  map[string]*TokenBucket
	rate     float64
	burst    float64
	idleTTL  time.Duration
	lastGC   time.Time
	gcPeriod time.Duration
}

// NewPerKeyLimiter 构造按 key 限流的限流器
//   - rate:  每秒填充令牌数（每个 key 独立）
//   - burst: 桶容量
func NewPerKeyLimiter(rate, burst float64) *PerKeyLimiter {
	return &PerKeyLimiter{
		buckets:  make(map[string]*TokenBucket),
		rate:     rate,
		burst:    burst,
		idleTTL:  5 * time.Minute,
		gcPeriod: 1 * time.Minute,
		lastGC:   time.Now(),
	}
}

// WithIdleTTL 配置空闲桶的 GC TTL
func (l *PerKeyLimiter) WithIdleTTL(d time.Duration) *PerKeyLimiter {
	l.idleTTL = d
	return l
}

// WithGCPeriod 配置 GC 周期（仅测试/特殊场景使用；默认 1 分钟）
func (l *PerKeyLimiter) WithGCPeriod(d time.Duration) *PerKeyLimiter {
	l.gcPeriod = d
	return l
}

// getOrCreate 获取或创建 key 对应的桶
func (l *PerKeyLimiter) getOrCreate(key string) *TokenBucket {
	l.mu.RLock()
	if tb, ok := l.buckets[key]; ok {
		l.mu.RUnlock()
		return tb
	}
	l.mu.RUnlock()

	l.mu.Lock()
	defer l.mu.Unlock()
	if tb, ok := l.buckets[key]; ok {
		return tb
	}
	tb := NewTokenBucket(l.rate, l.burst)
	l.buckets[key] = tb
	return tb
}

// Allow 实现 core.Limiter
func (l *PerKeyLimiter) Allow(key string) bool {
	l.maybeGC()
	return l.getOrCreate(key).allow()
}

// Wait 实现 core.Limiter
func (l *PerKeyLimiter) Wait(ctx context.Context, key string) error {
	l.maybeGC()
	return l.getOrCreate(key).wait(ctx)
}

// maybeGC 定期清理空闲桶
func (l *PerKeyLimiter) maybeGC() {
	l.mu.RLock()
	last := l.lastGC
	l.mu.RUnlock()
	if time.Since(last) < l.gcPeriod {
		return
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	if time.Since(l.lastGC) < l.gcPeriod {
		return
	}
	l.lastGC = time.Now()

	// 清理最后 refill 时间早于 TTL 的桶
	cutoff := time.Now().Add(-l.idleTTL)
	for k, tb := range l.buckets {
		tb.mu.Lock()
		if tb.lastRefill.Before(cutoff) {
			delete(l.buckets, k)
		}
		tb.mu.Unlock()
	}
}

// Size 返回当前 key 数（仅用于监控）
func (l *PerKeyLimiter) Size() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.buckets)
}

// 确保实现 contract.Limiter 接口
var _ contract.Limiter = (*PerKeyLimiter)(nil)

// ─────────────────────────────────────────────
//  兼容旧 API
// ─────────────────────────────────────────────

// NewTokenBucketLimiter 等价于 NewGlobalLimiter
func NewTokenBucketLimiter(rate, burst float64) *GlobalLimiter {
	return NewGlobalLimiter(rate, burst)
}

// NewPerKeyTokenBucketLimiter 等价于 NewPerKeyLimiter
func NewPerKeyTokenBucketLimiter(rate, burst float64) *PerKeyLimiter {
	return NewPerKeyLimiter(rate, burst)
}
