// F1-Guide API 服务入口。
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yantao-Wang/f1-guide/internal/config"
	"github.com/yantao-Wang/f1-guide/internal/handler"
	"github.com/yantao-Wang/f1-guide/internal/repository"
	"github.com/yantao-Wang/f1-guide/internal/router"
	"github.com/yantao-Wang/f1-guide/internal/service"
	"github.com/yantao-Wang/f1-guide/pkg/db"
	"github.com/yantao-Wang/f1-guide/pkg/logger"
)

// 服务器超时参数（暂为常量，配置体系复杂化后移入 config）。
const (
	readTimeout  = 10 * time.Second
	writeTimeout = 30 * time.Second
	idleTimeout  = 60 * time.Second
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.LogLevel)

	// 数据层启动即连接：数据库不可用时快速失败，暴露配置问题。
	pg, err := db.Open(cfg.DatabaseURL)
	if err != nil {
		log.Error("connect database failed", "error", err)
		os.Exit(1)
	}
	defer func() { _ = pg.Close() }()

	repo := repository.New(pg)
	svc := service.NewContent(repo, repo, repo)

	srv := &http.Server{
		Addr:         cfg.HTTPAddr,
		Handler:      router.New(log, handler.NewHealth(log), handler.NewContent(svc)),
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	go func() {
		log.Info("server starting", "addr", cfg.HTTPAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	// 优雅退出：收到 SIGINT/SIGTERM 后等待存量请求处理完毕。
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Info("server shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("shutdown failed", "error", err)
	}
}
