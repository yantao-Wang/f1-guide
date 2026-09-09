package middleware

import (
	"sync"
	"time"
)

// LoginLimiter 登录失败限流：窗口内失败超过上限即拒绝。内存实现（单实例部署假设）。
type LoginLimiter struct {
	mu       sync.Mutex
	failures map[string][]time.Time // key: 客户端 IP
	max      int
	window   time.Duration
	now      func() time.Time // 注入时钟（测试用）
}

// NewLoginLimiter 创建限流器。
func NewLoginLimiter(max int, window time.Duration) *LoginLimiter {
	return &LoginLimiter{
		failures: make(map[string][]time.Time),
		max:      max,
		window:   window,
		now:      time.Now,
	}
}

// Allowed 返回该 key 是否还可以尝试登录（窗口内失败数 < max）。
func (l *LoginLimiter) Allowed(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.recent(key)) < l.max
}

// RecordFailure 记录一次失败并清理窗口外旧记录。
func (l *LoginLimiter) RecordFailure(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.failures[key] = append(l.recent(key), l.now())
}

// recent 返回窗口内的失败记录（调用方需持锁）。
func (l *LoginLimiter) recent(key string) []time.Time {
	cutoff := l.now().Add(-l.window)
	records := l.failures[key]
	kept := records[:0]
	for _, t := range records {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	l.failures[key] = kept
	return kept
}
