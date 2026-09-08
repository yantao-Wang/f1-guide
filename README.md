# 🏎️ 走近围场 · F1咨讯（F1-Guide）

从零开始，看懂F1的精彩。面向 F1 新手的中文咨讯与科普平台。

## 文档

- [项目研究报告](走近围场-F1咨讯-项目研究报告.md) —— 产品、内容、技术架构的完整设计（v1.0）
- [项目可行性方案](走近围场-F1咨讯-项目可行性方案.md) —— 决策文档：范围裁剪、排期、风险（v1.0）

## 技术栈

Go + chi + PostgreSQL + sqlx · OpenAPI 契约先行 · Go html/template 服务端渲染 · Docker · GitHub Actions CI

## 本地开发

```bash
# 1. 安装工具
make tools          # golangci-lint

# 2. 启动 PostgreSQL
make docker-up

# 3. 运行 API（:8080）
make run
```

## 质量门禁（CI 自动执行）

| 检查 | 工具 | 标准 |
| --- | --- | --- |
| 静态分析 | golangci-lint | 零警告 |
| 单元测试 | go test | 全部通过 |
| 测试覆盖率 | go test -cover | ≥70% |
| 依赖漏洞 | govulncheck | 零高危 |
| API 契约 | 基于 api/openapi.yaml 的契约测试 | 全部通过 |
| 编译 | go build | 无错误 |

## 目录结构

```
cmd/api/           # 主程序入口
internal/
  domain/          # 领域模型（无依赖）
  service/         # 业务逻辑
  repository/      # 数据访问
  handler/         # HTTP 处理
  router/          # 路由注册
  config/          # 配置加载
  middleware/      # 中间件
pkg/
  logger/          # 结构化日志
  db/              # 数据库连接
  f1api/           # 外部 F1 API 客户端
api/openapi.yaml   # API 契约（契约先行）
migrations/        # 数据库迁移
tests/             # 集成测试 / 契约测试
```

依赖方向：handler → service → repository → domain，禁止反向依赖。
