package logger

import (
	"context"
	"log/slog"
	"testing"
)

func TestNewLevels(t *testing.T) {
	cases := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"info":  slog.LevelInfo,
		"warn":  slog.LevelWarn,
		"error": slog.LevelError,
		"INFO":  slog.LevelInfo, // 大小写不敏感
		"bogus": slog.LevelInfo, // 非法值回退 info
		"":      slog.LevelInfo, // 空值回退 info
	}

	for in, want := range cases {
		got := New(in).Handler()

		if !got.Enabled(context.Background(), want) {
			t.Errorf("New(%q): level %v 消息未启用", in, want)
		}
		// 比目标低一级的消息应被过滤（debug 已是最低，跳过）
		if want > slog.LevelDebug && got.Enabled(context.Background(), want-4) {
			t.Errorf("New(%q): 低于 %v 的消息不应启用", in, want)
		}
	}
}
