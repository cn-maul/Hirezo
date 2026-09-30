# Hirezo 教师人员管理系统 · 设计文档

> 版本 v1.0 · 本文件与实现同步维护。

## 1. 项目概述

学校教师信息管理工具。以「教师」为管理对象，记录基本信息、学历背景与任教资质。

- **目标**：单二进制、零外部依赖、启动即用；数据落在本地 SQLite 文件，备份即拷贝。
- **范围（一期）**：人员管理（列表 / 搜索 / 筛选 / 新增 / 编辑 / 删除 / 详情 / 导出）、学科与学历字典管理、管理员登录与改密。
- **范围（二期）**：简历识别录入（docx/pdf → 大模型抽取 → 人工修正 → 入库），见第 10 章。
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
| 大模型接入 | `github.com/cn-maul/rosetta` v0.5.1 | 统一 OpenAI Chat / Responses / Anthropic Messages 三协议，零第三方依赖，Go 1.27 |
| PDF 文本抽取 | `github.com/ledongthuc/pdf` | 纯 Go 文本层抽取；无文本层的扫描件转 `pdftoppm` 渲染 |
| 前端 | Vue 3 + TypeScript + Vite | 复用 tix 界面风格 |
| 简历原件预览 | `docx-preview`（前端） | docx 在识别页左侧渲染版式；懒加载独立分块，仅识别页加载 |
| 样式 | 自实现 CSS（CSS 变量设计令牌，HeroUI 风格） | 无业务组件库、无 Tailwind |
| 路由 / 数据 | `vue-router` + 自写 `useQuery` composable | 轻量、零额外依赖 |
| HTTP | `axios`（统一封装于 `src/api/client.ts`） | |
| 静态资源 | `embed.FS`（`//go:embed web/dist`） | 前端产物编译进二进制 |

依赖总量：后端 5 个直接依赖（sqlite / crypto / excelize / rosetta / ledongthuc-pdf），前端 5 个直接依赖（vue / vue-router / lucide-vue-next / axios / docx-preview）。

## 3. 系统架构

### 3.1 目录结构

```
Hirezo/
├── main.go          # 入口：flag → openDB/initDB/migrateDB → setupDefaultAdmin
│                    #      → app 构造 → 中间件链 → 优雅关闭；路由注册、SPA 回退、安全头
├── api.go           # 通用响应/参数解析 + teachers 与 dictionaries 处理器 + 导出 + 校验
├── auth.go          # 会话存储、登录/登出/状态、改密、设置接口
├── resume.go        # 简历识别：上传校验、docx/pdf 解析、模型抽取、字段闸门、草稿入库
├── llm.go           # 模型配置读写（settings 表）+ rosetta client 构造
├── store.go         # 限流器、数据模型、建表与迁移、字典/教师/用户/设置/草稿 数据访问
├── embed.go         #（无独立文件，embed 声明在 main.go）
├── web/             # 前端工程
│   ├── src/
│   │   ├── api/        # client 封装 + auth/teachers/dicts/settings/resume
│   │   ├── components/ # 通用组件（Table/Dialog/Button…）+ teachers/ 筛选与表单
│   │   ├── composables/ # useAuth / useTheme / useToast / useQuery / useForm
│   │   ├── pages/      # Login / TeacherList / TeacherDetail / Resume / Settings
│   │   ├── styles/     # tokens.css（设计令牌）/ base.css / utilities.css
│   │   ├── lib/        # validation、utils、version
│   │   └── router.ts
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

- `securityHeaders`：`X-Content-Type-Options: nosniff`、`X-Frame-Options: DENY`、`Referrer-Policy: same-origin`、CSP（`default-src 'self'` 等；`frame-src 'self' blob:` 是简历原件预览用 blob iframe 渲染的前提）。
- `authMiddleware`：路径以 `/api/` 开头且不在 `publicAPI` 白名单内时校验会话，未通过返回 401。
- `publicAPI` 白名单：`/api/health`、`/api/login`、`/api/logout`、`/api/auth/status`、`/api/settings`。简历识别与模型配置接口均需登录。

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
  resume_file  TEXT    NOT NULL DEFAULT '',      -- 入库简历原件的文件名（仅文件名，位于 resumes 目录），空 = 手工录入无原件
  created_at   TEXT    NOT NULL,
  updated_at   TEXT    NOT NULL
);
CREATE INDEX idx_teachers_subject   ON teachers(subject);
CREATE INDEX idx_teachers_education ON teachers(education);
CREATE INDEX idx_teachers_name      ON teachers(name);

-- 设置（键值对）
CREATE TABLE settings (key TEXT PRIMARY KEY, value TEXT NOT NULL);

-- 简历识别草稿（中间数据，入库成功或手动删除后移除）
CREATE TABLE resume_drafts (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  file_name  TEXT    NOT NULL,
  file_type  TEXT    NOT NULL,                    -- 'docx' | 'pdf'
  file_size  INTEGER NOT NULL DEFAULT 0,
  page_count INTEGER NOT NULL DEFAULT 0,
  raw_text   TEXT    NOT NULL,                    -- 归一化文本（纯扫描件为空）
  payload    TEXT    NOT NULL,                    -- 抽取结果 JSON（fields + evidence + confidence + warnings）
  created_at TEXT    NOT NULL
);
```

**设计要点**

- **字典存 `name` 作引用值**（沿用 tix 分类的字符串引用模式）：≤500 人的量级下按显示名关联最简，筛选与导出无需 join。
- **改名级联**：字典项改名时在同一事务内级联 `UPDATE teachers`，不产生孤儿数据。
- **删除保护**：字典项被教师引用时返回 409，提示先处理关联教师（避免历史数据悬空）。
- **`enabled` 只影响前端下拉选项**；后端校验「值存在于字典」即可，允许教师保留已停用的学科值。
- **性别**为固定枚举，不进字典。
- **年龄**直接存整数（0 = 未填），按需求 18–100；后续如需精算可加 `birth_date`（待定项）。
- **硬删除**：`DELETE FROM teachers WHERE id=?`，无回收站。
- **简历原件的存放**：`teachers.resume_file` 只存文件名，不存路径（防止目录搬迁后数据失效）；物理文件在 `resumes/{教师ID}.{ext}`。识别阶段的原件先落 `resumes/drafts/{草稿ID}.{ext}`，入库时 move 进永久目录，丢弃草稿或进程重启时清理临时目录（`clearDraftResumeFiles`）。数据库与文件系统不做事务，写 `resume_file` 失败只记日志，不影响人员入库。

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
| DELETE | `/api/teachers/{id}` | 硬删除 → `{data:{ok:true}}`，同时清理其归档简历原件 |
| GET | `/api/teachers/{id}/resume` | 下载入库简历原件（二进制流，`attachment; filename*=UTF-8''{姓名}-简历.{ext}`）；无原件 404，未登录 401 |
| GET | `/api/dictionaries/{kind}` | `kind ∈ subject\|education` → `{data:[...]}` |
| POST | `/api/dictionaries/{kind}` | 新增 → 201 `{data:Dictionary}` |
| PUT | `/api/dictionaries/{kind}/{id}` | 部分更新（指针字段）→ `{data:Dictionary}` |
| DELETE | `/api/dictionaries/{kind}/{id}` | 未被引用 → `{data:{ok:true}}`；被引用 → 409 |
| GET | `/api/settings` | 公开，仅返回白名单键（`site_name`） |
| PUT | `/api/settings` | 管理员，白名单键写入 |
| POST | `/api/resume/parse` | multipart `file` → 同步解析+模型抽取 → 201 `{data:Draft}`；未配置模型 503 |
| GET | `/api/resume/drafts` | 草稿列表 → `{items,total,page,size}` |
| GET | `/api/resume/drafts/{id}` | 草稿详情 → `{data:Draft}` |
| DELETE | `/api/resume/drafts/{id}` | 丢弃草稿 → `{data:{ok:true}}` |
| POST | `/api/resume/drafts/{id}/commit` | `{fields:{...}}`（人工修正后）→ 201 `{data:Teacher}` 并删除草稿 |
| GET | `/api/settings/llm` | 管理员，模型配置回显（API Key 掩码为 `****xxxx`） |
| PUT | `/api/settings/llm` | 管理员，写入 endpoint/protocol/model/api_key（api_key 留空表示不改） |

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
| `/teachers/:id` | 人员详情 | 只读字段（与核对表单同样两列成行：姓名/性别、年龄/电话、学历/院校、学科/专业，资格证与备注整幅；≤720px 退回单列）+ 下载简历原件（有归档原件时）+ 编辑/删除入口 |
| `/resume` | 简历识别 | 无页头标题：选择文件后进入左右等高双栏（左=原件预览，右=识别进度 → 完成后右侧就地变成核对表单，两栏各占一屏高度，栏内独立滚动） |
| `/settings` | 设置布局 | 子路由重定向到 general |
| `/settings/general` | 通用 | 站点名称 |
| `/settings/dicts` | 字典管理 | 学科 / 学历两个 Tab |
| `/settings/llm` | 模型配置 | 端点 / 协议 / 模型 / API Key |
| `*` | 404 | |

**守卫**：`RequireAuth` 调 `GET /api/auth/status`，未登录跳 `/login?next=…`。`client.ts` 拦截器对 401 统一跳登录。

### 7.2 页面与组件

```
Layout.vue            侧边栏（人员管理 / 简历识别 / 系统设置）+ 顶栏（主题、退出、账号弹窗）
PageHeader.vue        标题 + 主操作按钮
DataTable.vue         DataTable / PaginationBar（通用）
TeacherFilters.vue    姓名搜索（300ms 防抖 + URL 同步）+ 学科/性别/学历/证书 下拉 + 导出
TeacherForm.vue       人员表单（字段清单驱动的双列网格：姓名/性别、年龄/电话、学历/院校、学科/专业成行，备注与资格证整幅）；弹窗与识别页共用
TeacherFormDialog.vue 新增/编辑弹窗：只负责 Dialog 外壳，表单交给 TeacherForm
DeleteConfirm.vue     危险操作二次确认
AccountDialog.vue     修改密码
Dicts.vue             字典管理（Tab 切换 kind）
LLM.vue               模型配置（端点/协议/模型/API Key + 连通性说明）
Resume.vue            简历识别：拖拽上传 → 「左预览 / 右进度」等高双栏（docx 用 docx-preview 渲染、PDF 用 blob 原生 iframe）→ 识别完成右侧内嵌核对表单
```

**列表列**：姓名（跳详情）/ 性别 / 年龄 / 学科（色胶囊）/ 教师资格证 / 联系电话 / 学历 / 毕业院校 / 专业 / 录入时间 / 操作（查看 · 编辑 · 删除）。支持多选批量删除。

**交互细节**：关键字防抖 300ms 并同步到 URL（`?keyword=`，刷新与分享可保留）；筛选变化回第一页并清空多选；删除当前页最后一条且不在第一页时自动回退一页。

### 7.3 样式与状态

- 设计令牌在 `src/styles/tokens.css` 的 CSS 变量（`--accent`、`--track`、`--r-*`、`--hairline` 等），亮/暗主题由 `composables/useTheme.ts` 切换 `.dark`。
- 组件样式为自实现 CSS（`src/styles/base.css` / `utilities.css`）+ 少量自定义类（`.cat-pill`、`.surface-card`、`.glass-nav`），参考 HeroUI 设计语言（Apple 风格、oklch 色板、柔和 accent、圆角与阴影层级）。
- 状态：`composables/useQuery.ts` 管理服务端缓存与失效（按 key 缓存 + `invalidateQueries`）；表单用自写 `useForm` composable（提交整体校验、出错字段再输入时清除）。
- 轻提示用自写 `useToast` composable + `ToastHost.vue`。

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
| 简历原件目录 | `-resume-dir` flag | 数据库同目录 `resumes/`（草稿临时文件在其 `drafts/` 子目录） |

- **embed 前置条件**：`web/dist` 必须存在，否则 `go build` 失败；先跑前端构建。
- **预压缩**：Vite 插件在构建期生成 `.gz` / `.br`，运行时按 `Accept-Encoding` 优先返回；`index.html` 不走预压缩（需动态注入站点名到 `<title>`）。
- **缓存**：`assets/` 下 hashed 资源 `immutable` 长缓存，`index.html` `no-cache`。
- **数据备份**：停机复制 `hirezo.db`（WAL 模式含 `-wal`/`-shm` 伴生文件），或在线 `VACUUM INTO` 快照。**简历原件不在库内**，需连同 `resumes/` 目录一起拷贝，否则详情页「下载简历」会 404。
- **Docker**：三阶段构建（node 构建前端 → go 编译 → alpine 运行）。运行镜像必须带 `ca-certificates`（否则 HTTPS 调不通大模型接口）与 `poppler-utils`（提供 `pdftoppm`，扫描件识别用），两者均随 Dockerfile 一起装好；二进制自行部署时缺失 `pdftoppm` 只影响扫描件，会明确报错。运行阶段 `ENV PORT=8882` + `EXPOSE 8882`，`-db /data/hirezo.db`，数据（库、`.hirezo-secret`、`resumes/`）全在 `/data` 卷里。
- **镜像流水线**：`.github/workflows/docker.yml` 在推送 `v*` 标签（或手动触发）时构建 `linux/amd64` 单架构镜像并推 `ghcr.io/<owner>/<repo>`；镜像 tag 读 `web/package.json` 的 `version`（界面右上角同源），标签与版本号不一致直接 `::error::` 失败，标签触发时额外推 `:latest`。推送鉴权用仓库自带的 `GITHUB_TOKEN` + `permissions: packages: write`，不落任何 PAT。
- **构建上下文**：`.dockerignore` 排除 `*.db`、`.hirezo-secret`、`resumes/`、`web/dist`、`node_modules`，避免本地数据与简历原件被送进 daemon/runner。
- **模型配置**：存 `settings` 表（见 10.4），改完即生效，无需重启。

## 9. 错误处理与日志

- 标准库 `log` 输出启动、迁移、异常信息。
- 前端 axios 拦截器统一提取 `error.message`（中文），401 全局跳登录，其余 toast 展示。
- panic 恢复：Go 的 `http.Server` 对单请求 panic 不会中断整个服务（net/http 内置 recover 并断开连接），日志可见堆栈。

## 10. 简历识别模块（M4）

### 10.1 流程

```
[前端] 选择 .docx / .pdf（≤10MB）→ 立即进入「左原件预览 / 右识别进度」双栏
   ↓ POST /api/resume/parse（multipart，同步等待）
[服务端] 魔数校验 → 抽取文本 / 渲染页图 → 构造 prompt → rosetta.Chat → 容错解析 JSON
        → 字段闸门（正则 + 字典白名单）→ 写 resume_drafts + 原件落 resumes/drafts/{草稿ID}.{ext}
        → 201 {data:Draft}
[前端] 识别完成时右侧分栏就地换成 TeacherForm（fieldMeta 高亮预填，原件预览保持在左栏）→ 人工修正
   ↓ POST /api/resume/drafts/{id}/commit {fields}
[服务端] validateTeacher → createTeacher → 草稿原件 move 到 resumes/{教师ID}.{ext} 并写 resume_file
        → 删除草稿 → 201 {data:Teacher}
[前端] 人员详情页「下载简历」→ GET /api/teachers/{id}/resume（服务端鉴权后 ServeContent）
```

模型调用是同步阻塞的（通常 3–30s），因此 `/api/resume/parse` 走 90s 上下文超时，服务端 `WriteTimeout` 相应放宽到 120s。

### 10.2 输入与解析

| 类型 | 判定 | 解析方式 |
|---|---|---|
| `.docx` | 魔数 `PK\x03\x04` | `archive/zip` + `encoding/xml` 读 `word/document.xml`；段落逐行，**表格渲染为 `标签 \| 值` 行**喂模型 |
| `.pdf`（有文本层） | 魔数 `%PDF` | `ledongthuc/pdf` 抽取文本 |
| `.pdf`（扫描件） | 文本层为空 | `pdftoppm -png -r 150` 逐页渲染 → base64 → `rosetta.UserImage` |
| 其他 | — | 400 拒绝（`.doc`、jpg/png 一律不收） |

- **判定一律看魔数**，扩展名仅作提示；大小上限 10MB，页数上限 10，超出返回 400。
- **原件归档**：解析全程在内存中进行，识别成功后把原件写入 `resumes/drafts/{草稿ID}.{ext}`（0600）；入库成功才 move 到 `resumes/{教师ID}.{ext}` 并记 `resume_file`。丢弃草稿、覆盖重传、进程启动时都会清理 `drafts/`（`clearDraftResumeFiles`），删除教师时清理对应归档文件。扫描件渲染另需 `os.MkdirTemp` 存放页图，请求结束即整体删除。
- **归档失败不阻断录入**：写临时原件或 move 失败只记日志，人员照常入库，`resume_file` 留空（详情页不显示下载按钮）。
- 服务器缺 `pdftoppm` 且遇到扫描件 → 500 明确提示「缺少 poppler-utils」，**不静默降级**（启动日志会打印检测结果）。

### 10.3 抽取契约与字段闸门

系统提示强制只输出纯 JSON；服务端容错解析（剥离 ```json 围栏、取首个 `{` 到末个 `}`），**不使用 `response_format` / `Extra`**（三协议不兼容）。

```json
{
  "fields": {
    "name": { "value": "张三", "evidence": "张三  男  28岁", "confidence": 0.98 },
    "phone": { "value": "13800138000", "evidence": "电话：13800138000", "confidence": 0.95 }
  },
  "summary": "中学数学教师，5 年教龄"
}
```

模型输出之后、入库之前，在**本地**过一道闸门（防幻觉 + 字典一致性）：

| 字段 | 规则 | 不通过处理 |
|---|---|---|
| `gender` | 男/男性 → `male`，女/女性 → `female` | 清空 + warning |
| `age` | 18–100 整数；契约「未知给 0」→ 0 且 evidence 为空视为**未知**，不算错 | 越界清空 + error；0 但有证据 → warn |
| `phone` | `^1[3-9]\d{9}$` | 清空 + warning |
| `has_cert` | 有/具备/是/✓ → 1；无/没有/否 → 0 | 清空 + warning |
| `subject` | 归一化（去空白/「学科」后缀/近义词）后须命中 `dictionaries(kind='subject')` | 留空 + warning「学科不在字典中，请下拉选择」 |
| `education` | 同上，命中 `kind='education'` | 留空 + warning |
| `name` | TrimSpace 后 1–50 字符 | warning（提交时 `validateTeacher` 兜底拦截） |
| `university` / `major` / `remark` | 长度上限（100/100/500） | 截断 + warning |

- 闸门返回 `warnings: [{field, message, level}]`，`level ∈ warn|error`；**清空的字段一定同时给 warning**，不会静默丢数据。
- **取值为空但 evidence 非空**（模型按「不在列表给空值」契约清空了取值，但确实识别到了内容）→ 必定 warn：学科/学历给「不在字典中，请下拉选择」，手机号含数字给 error、不含给 warn，其余给「识别到X相关内容，但未能写入取值，请手动补全」。
- **反向一致性**：`subject` / `education` 的取值必须能在其 evidence 中找到（去空白比对），否则 warn「取值与识别依据不一致」——用来抓住「简历写篮球教练、模型却填体育」这类脑补。只提示不拦截，取值照写，交人工核对。
- 字典不匹配**一律留空 + 标黄**，禁止把模型的自由文本写进 `subject`/`education` 列。
- `confidence < 0.6` 的字段前端标黄提示复核，不影响入库（入库只看 `validateTeacher`）。

### 10.4 模型接入（Rosetta）

| 配置项 | settings 键 | 说明 |
|---|---|---|
| 端点 | `llm_endpoint` | 如 `https://api.deepseek.com/v1`；裸主机自动补 `/v1`，不允许 query 串 |
| 协议 | `llm_protocol` | `openai-chat`（默认）/ `openai-responses` / `anthropic` |
| 模型 | `llm_model` | 如 `deepseek-chat`、`qwen-vl-max` |
| API Key | `llm_api_key` | 只写不读：接口回显 `****后4位`，公开 `/api/settings` 不含任何 `llm_*` 键 |

- `rosetta.NewClient(WithEndpoint, WithAPIKey, WithProtocol)`；endpoint/key 任一为空 → 503「模型尚未配置」。
- `Client` 每次请求现建（配置改了立刻生效），并发安全由 SDK 保证。
- 错误映射：`*rosetta.APIError` → 502（`Retryable` 时提示「上游繁忙，请重试」）；本地哨兵 `ErrInvalidRequest` → 502 配置/请求不被上游接受；90s 超时 → 504。

### 10.5 接口与权限

- `parse` / `drafts*` / `teachers/{id}/resume` 全部需登录会话（不在 `publicAPI` 白名单内）；`settings/llm` 需管理员。
- 提交入库完全复用 `validateTeacher` + `createTeacher`，不新增写入逻辑；`commit` 成功后删除对应草稿，并把草稿原件 move 成 `resumes/{教师ID}.{ext}`。
- 下载按文件名从 `resumeDir` 取，路径只由「教师 ID + 库里存的扩展名」拼出，不接受用户输入的路径片段；教师不存在、`resume_file` 为空或物理文件缺失，一律 404。
- 删除教师（单条与批量）后清理其归档原件；清理失败只记日志，不影响删除结果。
- 草稿列表按 id 倒序分页，复用 `pagination`。

### 10.6 前端

- `/resume`：无页头标题（省掉一行说明文字），拖拽/点击上传（accept 仅 `.docx,.pdf`，本地先校验扩展名与 10MB）→ 立即切换到识别会话双栏（整块 `margin-top: -16px` 吃掉一半上留白好让位给内容，`height: calc(100vh - 112px)` = 顶栏 64 + 上留白 16 + 下留白 32，两栏等高且刚好占满一屏、底部仍留一圈白边而不是贴住页面下缘；`grid-template-rows: minmax(0, 1fr)` 让栅格行不随内容长高，栏内独立滚动而不是把整页撑长；列宽 `minmax(0, 1fr) clamp(420px, 42%, 560px)`，右栏保底 420px 才排得下两列，其余宽度都留给原件预览；≤900px 退回上下堆叠）：
  - 左：原件预览（不显示文件名标题）。PDF 用 `URL.createObjectURL` + 原生 `<iframe>`；docx 动态 `import('docx-preview')` 渲染（独立懒加载分块，只有识别页才加载），离开页面/结束会话时 `revokeObjectURL` 并清空容器。
  - 右：识别状态（转圈 + 「AI 正在识别…」+ 文件类型/大小 + 已用时计时）、失败时显示服务端错误文案与「重试 / 放弃」；识别成功后右栏直接换成内嵌的 `TeacherForm`，不再有弹窗层级，也不加「识别结果 · 请核对」这类小标题——表单第一行就是姓名。点开学科/学历下拉时 `TeacherForm.revealSelect` 会把该字段滚到面板上沿，避免 300px 弹层被滚动容器裁掉。
- `TeacherForm` 的字段顺序即双列栅格的成行关系：`FIELDS` 按 姓名/性别、年龄/联系电话、学历/毕业院校、学科/专业 排成四行两列，备注与资格证用 `full` 占整行（`grid-column: 1 / -1`）；下拉在表单里改成 `display: block` + 触发器 `width: 100%`，与输入框右边缘对齐。新增/编辑弹窗共用同一份顺序，窄视口（<640px）自动退回单列。
  - 「返回上传」即结束本次核对，回到上传区重新选文件；不再展示草稿列表。
- `fieldMeta: Record<field, {evidence, confidence, level}>`：识别依据与置信度**不再逐项显示**，「识别原文」折叠区也已去掉（页面上每项下方只在校验不通过时有一行错误），核对原件靠左侧预览；`level=warn` 黄框、`level=error` 红框仍按字段着色，warning 汇总为表单顶部提示。证据与原文仍留在草稿库里，只是不再渲染。
- 草稿列表（旧「识别记录」区）已从页面移除：`GET/DELETE /api/resume/drafts*` 接口保留（入库仍按草稿 ID 提交），未核对的草稿原件由进程启动时的 `clearDraftResumeFiles` 清理。
- 未配置模型时 `/resume` 顶部显示配置引导条，点击跳 `/settings/llm`，上传按钮同步禁用。
- 人员详情页在 `teacher.resume_file` 非空时显示「下载简历」，走 axios `responseType: 'blob'` 携会话下载后触发保存，失败 toast 提示（直接 `<a href>` 导航时 401 会被存成错误文件）。

### 10.7 隐私与安全

- 简历正文与模型返回**不写日志**；数据库只存归一化文本与 JSON。
- **原件属个人敏感信息**：`resumes/` 目录 0700、文件 0600，只按库内文件名读取，不做目录列表、不提供公开访问；备份范围因此包含该目录，同时该目录已进 `.gitignore`，原件不会随代码入库。
- 上传大小上限 10MB；类型按魔数判定；`ParseMultipartForm` 显式限额。
- 模型配置仅管理员可读写；API Key 永不回显原文。

## 11. 里程碑

| 阶段 | 内容 | 状态 |
|---|---|---|
| M1 骨架 | 从 tix 裁剪出 Go 工程 + 登录/会话 + embed 静态服务 + 前端布局与登录页 | ✅ |
| M2 人员管理 | teachers CRUD + 分页/搜索/筛选 + 弹窗表单 + 详情页 + 前后端校验 | ✅ |
| M3 完善 | 字典管理页 + xlsx 导出 + 批量删除 + 测试与打磨 | ✅ |
| M4 简历识别 | docx/pdf 解析 + Rosetta 抽取 + 字段闸门 + 草稿高亮表单 + 入库 | ✅ |
| M5 识别体验与归档 | 识别页左右分栏（原件预览 + 进度）+ 完成后右栏内嵌核对表单 + 简历原件归档与详情页下载 | ✅ |

## 12. 从 tix 裁剪的记录

**删除**：工单（tickets/comments 表与全部 handler）、统计看板、消息推送（`notify.go`）、外部集成 API Key、多用户管理与指派、游客公开查询、公开提交页、CSV 导出（改为 xlsx）。

**保留改造**：分类 → 字典（加 `kind`）、列表分页筛选模式、会话/限流/安全头、设置（`site_name`）、改密、SPA 回退与预压缩、`DataTable`/`Dialog`/`DeleteConfirm` 等通用组件。

**待定项**

1. ~~简历录入模块的数据模型与流程（二期）~~ → 已定稿，见第 10 章。
2. 是否补字段：工号、职称、入职日期 / 在职状态、身份证号。
3. 年龄 → 出生日期 是否切换。
4. 统计看板 / 操作日志 是否需要。
5. Excel 导入（含重复识别策略）。
6. 会话是否改为落库（当前重启失效）。
7. `.doc` / 图片简历的识别（当前明确不支持，需转换工具链时再议）。
8. 重复简历识别（同名同手机号时提示已存在）。
