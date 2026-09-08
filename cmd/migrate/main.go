// 数据库迁移命令：migrate [up|down]
//
//	up   应用全部未执行的迁移（默认）
//	down 回滚最近一条迁移
package main

import (
	"fmt"
	"os"

	"github.com/yantao-Wang/f1-guide/internal/config"
	"github.com/yantao-Wang/f1-guide/pkg/db"
)

func main() {
	cfg := config.Load()

	action := "up"
	if len(os.Args) > 1 {
		action = os.Args[1]
	}

	var err error
	switch action {
	case "up":
		err = db.MigrateUp(cfg.DatabaseURL)
	case "down":
		err = db.MigrateDown(cfg.DatabaseURL)
	default:
		fmt.Fprintf(os.Stderr, "用法: migrate [up|down]\n")
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "迁移失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("迁移完成:", action)
}
