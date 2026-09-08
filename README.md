# 🏎️ 走近围场 · F1咨讯（F1-Guide）

从零开始，看懂F1的精彩。面向 F1 新手的中文咨讯与科普平台。

## 文档

- [项目研究报告](走近围场-F1咨讯-项目研究报告.md) —— 产品、内容、技术架构的完整设计（v1.0）
- [项目可行性方案](走近围场-F1咨讯-项目可行性方案.md) —— 决策文档：范围裁剪、排期、风险（v1.0）

## 技术栈

Go + chi + PostgreSQL + sqlx · OpenAPI 契约先行 · Go html/template 服务端渲染 · Jolpica 赛事数据（缓存 + 降级） · Docker · GitHub Actions CI

## 本地开发

```bash
# 1. 安装工具
make tools          # golangci-lint

# 2. 启动 PostgreSQL 并初始化
make docker-up      # 启动本地 PostgreSQL 17
make migrate-up     # 应用迁移（嵌入式，随二进制分发）
make seed           # 导入示例数据（占位内容）

# 3. 运行 API（:8080）
make run
```

集成测试（真实 PostgreSQL）：

```bash
export F1GUIDE_TEST_DATABASE_URL=postgres://f1guide:f1guide@localhost:5432/f1guide?sslmode=disable
go test -tags=integration ./tests/integration/...
```

### 网络与代理（国内网络）

本机开发依赖三类境外下载，国内直连时快时慢，踩坑记录如下：

| 依赖 | 问题 | 解法 |
| --- | --- | --- |
| Go 模块 | proxy.golang.org 超时 | 已持久配置 `GOPROXY=https://goproxy.cn,direct`（`go env -w`） |
| Colima VM 镜像 | GitHub CDN 限速（实测 ~28KB/s） | 挂本机代理重启：`HTTPS_PROXY=http://127.0.0.1:7897 colima start`（断点续传，端口按本机代理实际配置） |
| Docker Hub 镜像 | 目前直连可用 | 如变慢，给 colima 的 Docker daemon 配置镜像加速器 |

### 赛事数据接入（Jolpica）

赛程与积分榜来自 [Jolpica](https://api.jolpi.ca/ergast/f1/)（Ergast 数据镜像，免费、限速 500 req/h）：

- 本地内存 TTL 缓存：积分榜 30 分钟、赛程 1 小时（上游正赛后约 1 小时更新数据）
- 上游地址可用 `F1API_BASE_URL` 覆盖；国内联调时指向本地代理
- 降级策略：上游故障时页面渲染提示文案（首页/赛程/数据页均不中断），JSON API 返回 `502 upstream_unavailable`
- 中文映射（车队色/车队名/车手名/大奖赛名/赛道名）维护于 `internal/service/stats_mappings.go`，新增内容车手时同步补 `driverSlugs`

### 故障排查

- **8080 端口被占用**：`go run` 杀父进程会残留子二进制，用 `lsof -ti :8080 | xargs kill` 清理；本地冒烟建议 `go build -o /tmp/f1guide-api ./cmd/api && /tmp/f1guide-api` 直接管理进程
- **集成测试提示跳过**：未设置 `F1GUIDE_TEST_DATABASE_URL`，按上文导出后再跑

## 质量门禁（CI 自动执行）

| 检查 | 工具 | 标准 |
| --- | --- | --- |
| 静态分析 | golangci-lint | 零警告 |
| 单元测试 | go test | 全部通过 |
| 测试覆盖率 | go test -cover | ≥70%（口径 internal + pkg） |
| 契约测试 | kin-openapi 校验 api/openapi.yaml | 全部通过 |
| 集成测试 | 真实 PostgreSQL（CI service container） | 全部通过 |
| 依赖漏洞 | govulncheck | 零高危 |
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
  db/              # 数据库连接 + 嵌入式迁移（pkg/db/migrations/）
  f1api/           # 外部 F1 API 客户端
api/openapi.yaml   # API 契约（契约先行）
seeds/             # 本地开发示例数据
tests/             # 集成测试 / 契约测试
```

依赖方向：handler → service → repository → domain，禁止反向依赖。
