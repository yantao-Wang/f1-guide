package middleware

import (
	"testing"
	"time"
)

func TestLimiterWindow(t *testing.T) {
	l := NewLoginLimiter(5, 15*time.Minute)
	key := "1.2.3.4"

	// 4 次失败后第 5 次尝试仍放行
	for i := 0; i < 4; i++ {
		l.RecordFailure(key)
	}
	if !l.Allowed(key) {
		t.Fatal("4 次失败后应仍放行")
	}
	// 第 5 次失败后，第 6 次尝试被拒绝
	l.RecordFailure(key)
	if l.Allowed(key) {
		t.Fatal("5 次失败后应拒绝")
	}
}

func TestLimiterRecoveryAfterWindow(t *testing.T) {
	l := NewLoginLimiter(2, 15*time.Minute)
	key := "1.2.3.4"

	l.RecordFailure(key)
	l.RecordFailure(key)
	if l.Allowed(key) {
		t.Fatal("窗口内 2 次失败应拒绝")
	}

	// 时钟拨过窗口期：旧记录过期，重新放行
	l.now = func() time.Time { return time.Now().Add(16 * time.Minute) }
	if !l.Allowed(key) {
		t.Fatal("窗口过期后应重新放行")
	}
	// 窗口过期后的第 1 次新失败仍放行，第 2 次拒绝
	l.RecordFailure(key)
	if !l.Allowed(key) {
		t.Fatal("新窗口内 1 次失败应仍放行")
	}
	l.RecordFailure(key)
	if l.Allowed(key) {
		t.Fatal("新窗口内 2 次失败应拒绝")
	}
}

func TestLimiterKeyIsolation(t *testing.T) {
	l := NewLoginLimiter(1, 15*time.Minute)
	l.RecordFailure("1.1.1.1")
	if l.Allowed("1.1.1.1") {
		t.Fatal("同 key 应被限流")
	}
	if !l.Allowed("2.2.2.2") {
		t.Fatal("不同 key 不应互相影响")
	}
}
