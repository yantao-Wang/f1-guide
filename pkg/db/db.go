// Package db 提供数据库连接与迁移。
package db

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"
)

// Open 建立 PostgreSQL 连接池并验证连通性（启动即失败，快速暴露配置问题）。
func Open(databaseURL string) (*sqlx.DB, error) {
	db, err := sqlx.Open("pgx", databaseURL)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
