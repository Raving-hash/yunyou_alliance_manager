# 云游联盟管理系统 — 开发文档

## 项目概述

联盟管理工具，管理员上传 CSV 文件（含用户名、武勋、繁荣数据），系统按日期存储快照，生成趋势图表。

- 后端：Go + Gin + GORM + SQLite
- 前端：Vue 3 + Element Plus + ECharts
- 无外部数据库依赖，单文件 SQLite 存储

---

## 快速启动

```bash
# 一键启动前后端（开发模式）
bash dev.sh

# 单独重启后端
bash restart-backend.sh

# 单独重启前端
bash restart-frontend.sh

# 停止所有
bash stop.sh
```

进程 PID 和日志存放在 `.pids/` 目录：
- `.pids/backend.pid` / `.pids/backend.log`
- `.pids/frontend.pid` / `.pids/frontend.log`

> **注意**：`dev.sh` 使用 `go build` 编译后再运行二进制，不用 `go run`。
> 原因：`go run` 会 fork 出子进程，PID 文件记录的是父进程，停止父进程不会停止实际监听端口的子进程，导致端口冲突。

访问地址：
- 后端 API：http://localhost:8080
- 前端页面：http://localhost:5173

---

## 项目结构

```
yunyoumanager/
├── main.go                          # 入口：DB 初始化 + Gin 启动
├── config/config.go                 # 端口(:8080)、DB路径(data/yunyou.db)
├── internal/
│   ├── model/model.go               # GORM 数据模型
│   ├── repository/repository.go     # 所有数据库操作
│   ├── handler/
│   │   ├── upload.go                # POST /api/upload
│   │   ├── member.go                # GET /api/members, /api/members/overview
│   │   └── chart.go                 # GET /api/chart/:username, /api/chart/alliance
│   ├── router/router.go             # 路由注册 + CORS
│   └── testutil/testutil.go         # 测试用内存 SQLite DB
├── cmd/seed/main.go                 # 生成测试数据（30人×7日）
├── frontend/
│   ├── src/
│   │   ├── api.js                   # Axios 封装，baseURL: http://localhost:8080
│   │   ├── App.vue                  # 根组件
│   │   └── components/
│   │       ├── UploadPanel.vue      # 上传表单
│   │       ├── MemberTable.vue      # 成员总览表格
│   │       └── MemberChart.vue      # ECharts 图表
│   └── vite.config.js
└── data/yunyou.db                   # SQLite 数据库（首次启动自动创建）
```

---

## 数据模型

```go
// 成员表，用户名唯一
type Member struct {
    ID        uint
    Username  string    // uniqueIndex
    CreatedAt time.Time
}

// 每日快照，(member_id, date) 唯一约束
type DailyRecord struct {
    ID            uint
    MemberID      uint
    Date          time.Time
    MilitaryMerit int64  // 武勋
    Prosperity    int64  // 繁荣
}
```

唯一约束通过 SQL 索引而非 GORM tag 实现（GORM 复合唯一索引在 SQLite 的 upsert 中不可靠）：
```sql
CREATE UNIQUE INDEX IF NOT EXISTS idx_daily_member_date ON daily_records(member_id, date)
```

---

## API 接口

| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/api/upload` | 上传 CSV，写入当日快照 |
| GET | `/api/members` | 成员列表（按用户名排序） |
| GET | `/api/members/overview` | 成员总览（含武勋增长率、与联盟均值差值） |
| GET | `/api/chart/:username` | 个人武勋/繁荣/武勋率时间序列 |
| GET | `/api/chart/alliance` | 联盟每日武勋总量 + 增量 |

### POST /api/upload

- 表单字段：`file`（CSV 文件）、`date`（可选，YYYY-MM-DD，默认今天）
- CSV 必须含表头行，列名必须包含 `用户名`、`武勋`、`繁荣`（列顺序不限）
- 同一日期重复上传会覆盖已有数据（upsert）
- 返回：`{ message, date, imported, skipped }`

### GET /api/members/overview

查询参数：`?search=xxx`（模糊匹配用户名，大小写不敏感）

返回：
```json
{
  "alliance_avg_growth_rate": 52.3,   // 所有成员增长率均值，数据不足时为 null
  "members": [
    {
      "Username": "凌霄剑客",
      "CurrentMilitary": 85000,
      "CurrentProsperity": 13200,
      "GrowthRate": 78.5,             // 近3日武勋增量的增长率，数据不足时为 null
      "GrowthRateDiff": 26.2,         // GrowthRate - alliance_avg
      "Recent": [...]                 // 近3日数据（倒序）
    }
  ]
}
```

> `alliance_avg_growth_rate` 始终基于全量成员计算，不受 `search` 参数影响。

### GET /api/chart/:username

```json
{
  "member": { "username": "凌霄剑客", ... },
  "dates": ["2026-04-22", ...],
  "military": [60000, 65000, ...],
  "prosperity": [13000, 12800, ...],
  "military_delta": [null, 5000, ...],  // 第一天为 null
  "military_rate": [null, 0.38, ...]    // 武勋增量/繁荣，繁荣为0时为 null
}
```

### GET /api/chart/alliance

```json
{
  "dates": ["2026-04-22", ...],
  "total_military": [6148386, ...],
  "total_prosperity": [383482, ...],
  "military_delta": [null, 99195, ...]
}
```

---

## 增长率计算逻辑

需要至少 3 天数据。设最近 3 天武勋为 d0（最新）、d1、d2：

```
delta0 = d0 - d1   （最近一日增量）
delta1 = d1 - d2   （前一日增量）
GrowthRate = (delta0 - delta1) / |delta1| × 100%
```

含义：武勋日增量本身的增长率（>0 说明越打越多，<0 说明在下降）。

---

## 测试

```bash
# 运行全部测试
go test ./...

# 运行单个包的测试
go test ./internal/handler/...
go test ./internal/repository/...

# 带详细输出
go test -v ./...
```

测试采用内存 SQLite（`":memory:"`），每个测试用例独立 DB，互不干扰。测试文件：

| 文件 | 覆盖内容 |
|------|---------|
| `internal/repository/repository_test.go` | UpsertMember、UpsertDailyRecord、ListMembers、GetChartData、GetMembersOverview、GetAllianceDailyTotals |
| `internal/handler/upload_test.go` | 正常上传、列顺序、重复上传覆盖、空用户名跳过、默认日期、缺字段报错 |
| `internal/handler/member_test.go` | List、Overview 搜索过滤、联盟均值不受搜索影响、GrowthRateDiff、无增长数据时均值为 null |
| `internal/handler/chart_test.go` | 成员不存在 404、单日无 delta、多日 delta 和 rate、繁荣为0时 rate 为 null、联盟汇总 |

---

## 生成测试数据

```bash
go run cmd/seed/main.go
```

写入 30 个中文名成员、7 日数据。武勋初始 5~30 万，每日增量 500~7500（20% 概率"摆烂"仅增 0~200）。数据库须已存在（先运行主程序或 `dev.sh`）。

---

## 前端说明

### 图表组件（MemberChart.vue）

- 联盟武勋图：`onMounted` 时立即 `echarts.init`（div 始终在 DOM），数据回来后 `setOption`
  - 武勋总量（左轴）+ 每日增量（右轴），双 Y 轴避免数量级差距导致增量不可见
  - 武勋以**万**为单位显示
- 个人图表：选择成员后动态 init，切换成员用 `setOption` 覆盖

> 个人武勋图表使用 `v-if/v-else` 控制渲染（选人后才 init），而联盟图使用 `v-show` + 始终存在的 div，两者处理方式不同，原因：联盟图在 `onMounted` 时就需要 init。

### API 封装（api.js）

当前 `baseURL` 硬编码为 `http://localhost:8080`，生产部署时需改为实际地址（或改用相对路径 + Nginx 反代）。

---

## 常见问题

**前端 5173 无法访问**

前端进程可能意外退出。用 `bash restart-frontend.sh` 重启，或直接：
```bash
cd frontend && npm run dev &
```
如果报 npm 依赖错误（`rolldown` native binding 缺失），执行：
```bash
cd frontend && rm -rf node_modules package-lock.json && npm install
```

**端口被占用**

`dev.sh` 和 `restart-backend.sh` 内置了 `lsof -ti:<port> | xargs kill -9` 兜底清理，一般自动处理。

**重复上传同一天数据**

系统会覆盖该成员当天的数据（upsert 语义），不会产生重复行。
