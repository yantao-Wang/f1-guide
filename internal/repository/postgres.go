// Package repository 提供基于 PostgreSQL 的数据访问实现。
//
// 依赖方向：service → repository → domain。本包不依赖 service 与 handler。
// 接口由消费方（service）定义，本包只提供实现。
package repository

import (
	"errors"

	"github.com/jmoiron/sqlx"
)

// ErrNotFound 表示资源不存在。service 层将其映射为对外错误码。
var ErrNotFound = errors.New("repository: not found")

// Postgres 是 PostgreSQL 数据访问实现，持有连接池。
type Postgres struct {
	db *sqlx.DB
}

// New 创建 Postgres 数据访问层。
func New(db *sqlx.DB) *Postgres {
	return &Postgres{db: db}
}
