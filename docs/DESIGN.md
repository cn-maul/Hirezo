# Hirezo 教师人员管理系统 · 设计文档

> 版本 v1.0 · 本文件与实现同步维护。

## 1. 项目概述

学校教师信息管理工具。以「教师」为管理对象，记录基本信息、学历背景与任教资质。

- **目标**：单二进制、零外部依赖、启动即用；数据落在本地 SQLite 文件，备份即拷贝。
- **范围（一期）**：人员管理（列表 / 搜索 / 筛选 / 新增 / 编辑 / 删除 / 详情 / 导出）、学科与学历字典管理、管理员登录与改密。
- **二期规划**：简历录入模块（暂未设计）。
- **非目标**：多角色 RBAC、招聘流程流转、统计看板、Excel 导入、操作审计、多管理员、消息推送、对外集成 API。

## 2. 技术选型

本项目复用工单系统 `tix` 的工程骨架与界面风格，裁剪掉工单业务后改造为教师管理。

| 层 | 选型 | 说明 |
|---|---|---|
| 语言 | Go 1.27 | |
| Web 框架 | 标准库 `net/http` + `ServeMux`（Go 1.22+ 路由模式） | 零第三方路由 |
| 数据库 | SQLite（`modernc.org/sqlite`） | 纯 Go 实现，无 CGO，`CGO_ENABLED=0` 可交叉编译 |
| 密码 | `golang.org/x/crypto/bcrypt` | 库中只存哈希 |
| Excel 导出 | `github.com/xuri/excelize/v2` | 纯 Go；仅导出，导入属二期 |
| 前端 | React 19 + TypeScript + Vite | 复用 tix 界面风格 |
| 样式 | Tailwind CSS v4 + shadcn 风格组件 + CSS 变量设计令牌 | 无业务组件库 |
| 路由 / 数据 | `react-router-dom` + `@tanstack/react-query` | |
| HTTP | `axios`（统一封装于 `src/api/client.ts`） | |
| 静态资源 | `embed.FS`（`//go:embed web/dist`） | 前端产物编译进二进制 |

依赖总量：后端 3 个直接依赖（sqlite / crypto / excelize），前端为 tix 原有依赖集，未新增。

## 3. 系统架构

### 3.1 目录结构

```
Hirezo/
├── main.go          # 入口：flag → openDB/initDB/migrateDB → setupDefaultAdmin
│                    #      → app 构造 → 中间件链 → 优雅关闭；路由注册、SPA 回退、安全头
├── api.go           # 通用响应/参数解析 + teachers 与 dictionaries 处理器 + 导出 + 校验
├── auth.go          # 会话存储、登录/登出/状态、改密、设置接口
├── store.go         # 限流器、数据模型、建表与迁移、字典/教师/用户/设置 数据访问
├── embed.go         #（无独立文件，embed 声明在 main.go）
├── web/             # 前端工程
│   ├── src/
│   │   ├── api/        # client 封装 + auth/teachers/dicts/settings
│   │   ├── components/ # 通用组件（Table/Dialog/Button…）+ teachers/ 筛选与表单
│   │   ├── pages/      # Login / TeacherList / TeacherDetail / Settings
│   │   ├── lib/        # validation、theme、utils
│   │   └── router.tsx
│   ├── dist/           # 构建产物（被 embed）
│   └── vite.config.ts  # dev 时 proxy /api → :8080
├── docs/DESIGN.md
├── build.sh / build.bat
└── go.mod
```

### 3.2 请求链路

```
浏览器 ─ 静态资源(embed) / /api/*
   → securityHeaders → authMiddleware（publicAPI 白名单外校验会话）
   → handler（参数解析与校验）→ store（参数化 SQL）→ hirezo.db
   → 统一 JSON 信封返回
```

### 3.3 中间件

- `securityHeaders`：`X-Content-Type-Options: nosniff`、`X-Frame-Options: DENY`、`Referrer-Policy: same-origin`、CSP（`default-src 'self'` 等）。
- `authMiddleware`：路径以 `/api/` 开头且不在 `publicAPI` 白名单内时校验会话，未通过返回 401。
- `publicAPI` 白名单：`/api/health`、`/api/login`、`/api/logout`、`/api/auth/status`、`/api/settings`。

## 4. 数据库设计

迁移方式：**无版本号的幂等代码迁移**。启动时 `initDB` 建表（`CREATE TABLE IF NOT EXISTS`）→ `migrateDB` 执行若干幂等步骤（`hasColumn` 探测、空表才写种子、`password NOT LIKE '$2%'` 才升级）。重复执行安全，无需迁移工具。

```sql
-- 管理员（首期仅 admin 一行，结构预留扩展）
CREATE TABLE users (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  username     TEXT NOT NULL UNIQUE,
  password     TEXT NOT NULL,                    -- bcrypt
  display_name TEXT NOT NULL,
  role         TEXT NOT NULL DEFAULT 'operator', -- 单管理员场景恒为 admin
  created_at   TEXT NOT NULL
);

-- 通用字典：学科与学历共用一张表，按 kind 区分
CREATE TABLE dictionaries (
  id      INTEGER PRIMARY KEY AUTOINCREMENT,
  kind    TEXT    NOT NULL,                      -- 'subject' | 'education'
  name    TEXT    NOT NULL,
  color   TEXT    NOT NULL DEFAULT '#2563eb',
  sort    INTEGER NOT NULL DEFAULT 0,
  enabled INTEGER NOT NULL DEFAULT 1
);
CREATE UNIQUE INDEX idx_dictionaries_kind_name ON dictionaries(kind, name);

-- 人员（教师）
CREATE TABLE teachers (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  name         TEXT    NOT NULL,
  gender       TEXT    NOT NULL DEFAULT 'male',  -- 'male' | 'female'
  age          INTEGER NOT NULL DEFAULT 0,       -- 0 = 未填
  subject      TEXT    NOT NULL DEFAULT '',      -- → dictionaries(kind='subject').name
  has_cert     INTEGER NOT NULL DEFAULT 0,       -- 教师资格证 0/1
  phone        TEXT    NOT NULL DEFAULT '',
  education    TEXT    NOT NULL DEFAULT '',      -- → dictionaries(kind='education').name
  university   TEXT    NOT NULL DEFAULT '',      -- 毕业院校
  major        TEXT    NOT NULL DEFAULT '',      -- 专业
  remark       TEXT    NOT NULL DEFAULT '',
  created_at   TEXT    NOT NULL,
  updated_at   TEXT    NOT NULL
);
CREATE INDEX idx_teachers_subject   ON teachers(subject);
CREATE INDEX idx_teachers_education ON teachers(education);
CREATE INDEX idx_teachers_name      ON teachers(name);

-- 设置（键值对）
CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);
```

**设计要点**

- **字典存 `name` 作引用值**（沿用 tix 分类的字符串引用模式）：≤500 人的量级下按显示名关联最简，筛选与导出无需 join。
- **改名级联**：字典项改名时在同一事务内级联 `UPDATE teachers`，不产生孤儿数据。
- **删除保护**：字典项被教师引用时返回 409，提示先处理关联教师（避免历史数据悬空）。
- **`enabled` 只影响前端下拉选项**；后端校验「值存在于字典」即可，允许教师保留已停用的学科值。
- **性别**为固定枚举，不进字典。
- **年龄**直接存整数（0 = 未填），按需求 18–100；后续如需精算可加 `birth_date`（待定项）。
- **硬删除**：`DELETE FROM teachers WHERE id=?`，无回收站。

### 种子数据

仅当 `dictionaries` 表为空时写入：

- 学科（13）：语文、数学、英语、物理、化学、生物、政治、历史、地理、音乐、体育、美术、信息技术
- 学历（5）：中专、大专、本科、硕士、博士

每项带默认颜色，排序为写入顺序。用户改动过不重置。

## 5. API 设计

### 5.1 约定

- 前缀 `/api`，除导出外均为 JSON。
- 鉴权：登录后下发 `HttpOnly` Cookie，服务端校验内存会话。
- 响应信封：

```json
成功（单条）  { "data": { ... } }
成功（列表）  { "items": [ ... ], "total": 12, "page": 1, "size": 20 }
成功（布尔）  { "ok": true }
失败         { "error": { "code": 400, "message": "姓名不能为空" } }
```

| HTTP | 场景 |
|---|---|
| 400 | 参数/业务校验失败 |
| 401 | 未登录或会话过期；用户名密码错误 |
| 403 | 需要管理员权限 |
| 404 | 资源不存在 / 未知 `/api` 路径 |
| 405 | 方法不支持 |
| 409 | 字典项被引用，不可删除 |
| 429 | 登录尝试过于频繁 |
| 500 | 服务器内部错误 |

错误码即 HTTP 状态码（沿用 tix 约定，非业务码段）。

### 5.2 端点

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/health` | 健康检查（公开） |
| POST | `/api/login` | `{username,password}` → 下发 Cookie，返回 `{data:{ok,user}}` |
| POST | `/api/logout` | 撤销会话并清 Cookie |
| GET | `/api/auth/status` | `{data:{ok,user}}`，前端路由守卫依赖 |
| PUT | `/api/profile/password` | `{old_password,new_password}`，改密后吊销其他会话 |
| GET | `/api/teachers` | 列表，参数见 5.3，返回 `{items,total,page,size}` |
| POST | `/api/teachers` | 新增 → 201 `{data:Teacher}` |
| GET | `/api/teachers/export` | 同筛选参数 → `teachers-YYYYMMDD.xlsx` |
| POST | `/api/teachers/batch-delete` | `{ids:[...]}` → `{data:{ok,deleted}}` |
| GET | `/api/teachers/{id}` | 详情 → `{data:Teacher}` |
| PUT | `/api/teachers/{id}` | 修改 → `{data:Teacher}` |
| DELETE | `/api/teachers/{id}` | 硬删除 → `{data:{ok:true}}` |
| GET | `/api/dictionaries/{kind}` | `kind ∈ subject\|education` → `{data:[...]}` |
| POST | `/api/dictionaries/{kind}` | 新增 → 201 `{data:Dictionary}` |
| PUT | `/api/dictionaries/{kind}/{id}` | 部分更新（指针字段）→ `{data:Dictionary}` |
| DELETE | `/api/dictionaries/{kind}/{id}` | 未被引用 → `{data:{ok:true}}`；被引用 → 409 |
| GET | `/api/settings` | 公开，仅返回白名单键（`site_name`） |
| PUT | `/api/settings` | 管理员，白名单键写入 |

> 路由字面量段优先于通配符：`GET /api/teachers/export` 会先于 `GET /api/teachers/{id}` 匹配。

### 5.3 列表查询参数

| 参数 | 取值 | 默认 |
|---|---|---|
| `page` | ≥1 | 1 |
| `size` | 1–100 | 20 |
| `keyword` | 姓名模糊匹配（LIKE，通配符已转义） | 空 |
| `subject` | 字典名精确匹配 | 空 |
| `gender` | `male` / `female` / 空 | 空 |
| `education` | 字典名精确匹配 | 空 |
| `hasCert` | `1` / `0` / 空（空 = 不筛） | 空 |
| `order` | `asc` / `desc`（按 id，即录入先后） | `desc` |

非法的 `gender` / `hasCert` / `order` → 400。`order` 是唯一进入 SQL 拼接的值，**必须走白名单校验**（防注入），其余一律 `?` 占位符。

### 5.4 字段校验（前后端一致）

| 字段 | 规则 |
|---|---|
| name | 必填，TrimSpace 后 1–50 字符 |
| gender | 必填，`male` / `female` |
| age | 可空（0）；非空须 18–100 整数 |
| subject | 可空；非空须存在于 `dictionaries(kind='subject')` |
| has_cert | `0` / `1` |
| phone | 可空；非空须匹配 `^1[3-9]\d{9}$` |
| education | 可空；非空须存在于 `dictionaries(kind='education')` |
| university / major | 可空，各 ≤100 字符 |
| remark | 可空，≤500 字符 |
| 字典 name | 必填，TrimSpace 后 1–32 字符，同 kind 内不重复 |
| 字典 color | `#RRGGBB` |

## 6. 认证与安全

1. **登录**：`apiLogin` → 登录限流（同 IP 每分钟 10 次，超限 429）→ bcrypt 校验 → 签发 32 字节随机 token（内存会话）→ 下发 Cookie。
2. **Cookie**：`hirezo_session`，`HttpOnly`、`SameSite=Lax`、`Path=/`；`Secure` 按传输层动态判定（直连 TLS 或 `-trust-proxy` 且 `X-Forwarded-Proto: https`）。
3. **会话**：存内存 map，TTL 7 天；**服务重启即失效**（单管理员内网场景可接受，重启需重新登录）。`requireAuth` 会回查用户，角色以数据库为准。
4. **改密**：校验旧密码 → 更新为 bcrypt → `revokeUserExcept` 保留当前会话、吊销其他端。
5. **初始化**：首次启动若 `users` 无 `admin`，按 `-password` / `HIREZO_PASSWORD`（默认 `admin123`）写入并打印到启动日志。
6. **限流**：内存滑动窗口，超 4096 个 key 触发整体清扫，防伪造 XFF 造成内存无界增长。
7. **注入防护**：全部参数化查询；表名/列名走白名单常量；`ORDER BY` 走白名单；`LIKE` 关键字经 `escapeLike` 转义。
8. **其他**：请求体上限 1MB；安全响应头（见 3.3）；导出走 axios 携带会话下载（`<a download>` 直接导航时 401 会被存成文件）。

## 7. 前端设计

### 7.1 路由

| 路径 | 页面 | 说明 |
|---|---|---|
| `/login` | 登录 | 无侧边栏 |
| `/` | → `/teachers` | 重定向 |
| `/teachers` | 人员列表 | 搜索 + 筛选 + 表格 + 分页 + 新增/导出 |
| `/teachers/:id` | 人员详情 | 只读字段 + 编辑/删除入口 |
| `/settings` | 设置布局 | 子路由重定向到 general |
| `/settings/general` | 通用 | 站点名称 |
| `/settings/dicts` | 字典管理 | 学科 / 学历两个 Tab |
| `/settings/data` | 数据与备份 | 说明页 |
| `*` | 404 | |

**守卫**：`RequireAuth` 调 `GET /api/auth/status`，未登录跳 `/login?next=…`。`client.ts` 拦截器对 401 统一跳登录。

### 7.2 页面与组件

```
Layout.tsx            侧边栏（人员管理 / 系统设置）+ 顶栏（主题、退出、账号弹窗）
PageHeader.tsx        标题 + 主操作按钮
Table.tsx             DataTable / PaginationBar（通用）
TeacherFilters.tsx    姓名搜索（300ms 防抖 + URL 同步）+ 学科/性别/学历/证书 下拉 + 导出
TeacherFormDialog.tsx 新增/编辑弹窗（同一组件，2 列网格）
DeleteConfirm.tsx     危险操作二次确认
AccountDialog.tsx     修改密码
Dicts.tsx             字典管理（Tab 切换 kind）
```

**列表列**：姓名（跳详情）/ 性别 / 年龄 / 学科（色胶囊）/ 教师资格证 / 联系电话 / 学历 / 毕业院校 / 专业 / 录入时间 / 操作（查看 · 编辑 · 删除）。支持多选批量删除。

**交互细节**：关键字防抖 300ms 并同步到 URL（`?keyword=`，刷新与分享可保留）；筛选变化回第一页并清空多选；删除当前页最后一条且不在第一页时自动回退一页。

### 7.3 样式与状态

- 设计令牌在 `src/index.css` 的 CSS 变量（`--accent`、`--track`、`--r-*`、`--hairline` 等），亮/暗主题由 `lib/theme.tsx` 切换 `data-theme`。
- 组件样式以 Tailwind 工具类 + 少量自定义类（`.cat-pill`、`.surface-card`、`.glass-nav`）组合。
- 状态：`@tanstack/react-query` 管理服务端缓存与失效；表单用自写 `useFormState` Hook（提交整体校验、出错字段再输入时清除）。
- 轻提示用 `sonner` toast。

## 8. 构建与部署

```bash
./build.sh            # pnpm --dir web build → CGO_ENABLED=0 go build -ldflags="-s -w"
./hirezo              # 默认 http://localhost:8080

# 开发模式（前后端分离，热更新）
pnpm --dir web dev    # Vite :5173，proxy /api → :8080
go run .
```

| 配置 | 方式 | 默认 |
|---|---|---|
| 端口 | `PORT` 环境变量 / `-addr` flag | `8080` |
| 数据库路径 | `-db` flag | `hirezo.db` |
| 初始密码 | `-password` flag / `HIREZO_PASSWORD` | `admin123` |
| 信任代理头 | `-trust-proxy` flag | 关（直连部署勿开） |

- **embed 前置条件**：`web/dist` 必须存在，否则 `go build` 失败；先跑前端构建。
- **预压缩**：Vite 插件在构建期生成 `.gz` / `.br`，运行时按 `Accept-Encoding` 优先返回；`index.html` 不走预压缩（需动态注入站点名到 `<title>`）。
- **缓存**：`assets/` 下 hashed 资源 `immutable` 长缓存，`index.html` `no-cache`。
- **数据备份**：停机复制 `hirezo.db`（WAL 模式含 `-wal`/`-shm` 伴生文件），或在线 `VACUUM INTO` 快照。
- **Docker**：三阶段构建（node 构建前端 → go 编译 → scratch 运行）。

## 9. 错误处理与日志

- 标准库 `log` 输出启动、迁移、异常信息。
- 前端 axios 拦截器统一提取 `error.message`（中文），401 全局跳登录，其余 toast 展示。
- panic 恢复：Go 的 `http.Server` 对单请求 panic 不会中断整个服务（net/http 内置 recover 并断开连接），日志可见堆栈。

## 10. 里程碑

| 阶段 | 内容 | 状态 |
|---|---|---|
| M1 骨架 | 从 tix 裁剪出 Go 工程 + 登录/会话 + embed 静态服务 + 前端布局与登录页 | ✅ |
| M2 人员管理 | teachers CRUD + 分页/搜索/筛选 + 弹窗表单 + 详情页 + 前后端校验 | ✅ |
| M3 完善 | 字典管理页 + xlsx 导出 + 批量删除 + 测试与打磨 | ✅ |
| M4 二期 | 简历录入模块 | ⏳ 待设计 |

## 11. 从 tix 裁剪的记录

**删除**：工单（tickets/comments 表与全部 handler）、统计看板、消息推送（`notify.go`）、外部集成 API Key、多用户管理与指派、游客公开查询、公开提交页、CSV 导出（改为 xlsx）。

**保留改造**：分类 → 字典（加 `kind`）、列表分页筛选模式、会话/限流/安全头、设置（`site_name`）、改密、SPA 回退与预压缩、`DataTable`/`Dialog`/`DeleteConfirm` 等通用组件。

**待定项**

1. 简历录入模块的数据模型与流程（二期）。
2. 是否补字段：工号、职称、入职日期 / 在职状态、身份证号。
3. 年龄 → 出生日期 是否切换。
4. 统计看板 / 操作日志 是否需要。
5. Excel 导入（含重复识别策略）。
6. 会话是否改为落库（当前重启失效）。
