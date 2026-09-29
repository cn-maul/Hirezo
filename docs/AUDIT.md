# Hirezo 代码审计报告

> 审计日期：2026-02
> 审计范围：后端 Go（`main.go` / `api.go` / `auth.go` / `store.go` / `resume.go` / `llm.go`）+ 前端 React/TS（`web/src`）+ 构建与部署配置
> 验证结果：`go test ./...` ✅ 全部通过 · `go vet ./...` ✅ 通过 · `tsc -b` ✅ 类型检查通过

---

## 一、总体评价

项目整体质量**很高**，代码结构清晰、命名规范、注释到位，安全意识和健壮性处理明显超出一般水平：

- **安全设计扎实**：bcrypt 密码哈希、内存会话 + 会话吊销、登录限流、CSP 安全头、SQL 注入防护（列名白名单 + 参数占位符）、LIKE 通配符转义、开放重定向防护、API Key 不回显、字典值白名单校验等，覆盖面很全。
- **简历识别链路设计精巧**：魔数嗅探、docx/pdf 解析、扫描件渲染、模型抽取、字段闸门（反幻觉 + 字典白名单 + 证据比对）、草稿人工核对，逻辑严谨。
- **测试覆盖良好**：后端有 4 个测试文件、约 30 个测试用例，覆盖 CRUD、校验、鉴权、限流、安全头、简历识别全链路。
- **工程化到位**：单文件 embed 部署、预压缩静态资源、长缓存策略、Docker 多阶段构建、优雅关闭。

以下问题按**严重程度**分级列出，均为**可优化项**而非阻塞性缺陷。

---

## 二、中等问题（建议尽快处理）

### 1. API Key 明文存储于数据库（安全）
- **位置**：`llm.go:220-225`（`setSetting(a.db, settingLLMAPIKey, nextKey)`）
- **问题**：大模型 API Key 以**明文**写入 SQLite 的 `settings` 表。虽然接口层不回显原文（`maskKey`），但任何能读取 `hirezo.db` 文件的人（备份泄露、磁盘访问、Docker 卷被挂载）都能直接拿到密钥。
- **建议**：对 API Key 做可逆加密存储（如 AES-GCM，密钥来自环境变量或独立密钥文件），或至少做混淆；同时考虑将密钥与普通设置表分离。

### 2. 前端存在未实现的多用户管理死代码（一致性）
- **位置**：`web/src/api/auth.ts:29-53`
- **问题**：`fetchUsers` / `createUser` / `updateUser` / `deleteUser` 四个函数调用了 `/api/users` 系列接口，但**后端没有任何 `/api/users` 路由**，前端也没有任何组件调用它们。这是多用户管理功能残留的"半成品"，一旦被误用会得到 404。
- **建议**：删除这四个死函数，或补齐后端实现（当前 README 明确是"单管理员账号"，建议删除）。

### 3. 前端存在未使用的死代码文件/函数
- **位置**：
  - `web/src/components/ui/pagination.tsx`（整个文件未被引用，项目实际用 `Table.tsx` 里的自定义 `PaginationBar`）
  - `web/src/api/dicts.ts:21` `fetchEnabledOptions`（定义后从未被调用）
- **建议**：删除，减少维护负担和打包体积。

---

## 三、轻微问题（可选优化）

### 4. `injectSiteTitle` 每次 SPA 回退都查库（性能）
- **位置**：`main.go:252-258`
- **问题**：每次请求未命中静态资源、回退到 `index.html` 时，都会执行一次 `getSetting(a.db, "site_name")` 数据库查询。虽然 SQLite 单连接 + 单行查询开销极小，但在高并发下属于不必要的重复 IO。
- **建议**：将站点名缓存到内存（启动时加载 + 设置变更时失效），或使用 `sync.Once` + 原子更新。

### 5. `rateLimiter` 的清扫逻辑可微调（健壮性）
- **位置**：`store.go:37-68`
- **问题**：`allow()` 在每次调用时都会遍历该 key 的时间戳切片做清理，且仅在 key 数超阈值时才全表清扫。极端情况下（大量伪造 IP）内存仍可能短暂增长。
- **建议**：可改为固定周期后台清理，或对单 key 的时间戳长度设上限。

### 6. 前端大量使用 `any` 类型（类型安全）
- **位置**：`TeacherList.tsx`、`TeacherDetail.tsx`、`Login.tsx`、`Dicts.tsx`、`AccountDialog.tsx`、`Table.tsx` 等约 14 处 `catch (e: any)` / `onError: (e: any)` / `(r as any)`
- **问题**：错误处理统一用 `any`，丢失类型信息；`Table.tsx:158` 的 `(r as any)[c.key]` 绕过了类型检查。
- **建议**：定义统一的 `ApiError` 类型（axios 拦截器已把错误转成 `Error`，可封装 `errMsg(e: unknown)` 工具函数，`Resume.tsx` 已有此模式可推广）。

### 7. `go.mod` 依赖版本较激进（可维护性）
- **位置**：`go.mod`
- **问题**：`github.com/ledongthuc/pdf v0.0.0-20260907135840` 是一个 2026 年的伪版本（pseudo-version），且 `go 1.27` 是较新的 Go 版本。若团队环境 Go 版本不一致可能编译失败。
- **建议**：确认团队统一 Go 版本；对 pdf 库可考虑锁定到稳定 tag。

### 8. 前端 `index.css` 体积偏大（性能）
- **位置**：`web/src/index.css`（约 500 行）
- **问题**：包含大量设计 token、动画、无障碍媒体查询，虽然组织良好，但作为单文件 CSS 在首屏加载时占一定体积。
- **建议**：Tailwind v4 已支持按需生成，可确认构建产物中未使用的自定义类是否被 tree-shake；若首屏 LCP 敏感可考虑拆分关键 CSS。

---

## 四、值得肯定的亮点（保持）

- **SQL 注入防护到位**：`teacherDictColumn` 白名单 + 参数占位符，`order` 字段白名单校验，`escapeLike` 转义通配符。
- **会话安全**：改密后 `revokeUserExcept` 吊销其他会话；`requireAuth` 每次校验用户仍存在且角色以库为准。
- **限流防绕过**：`-trust-proxy` 显式开关，直连时忽略伪造的 `X-Forwarded-For`。
- **简历反幻觉**：`valueInEvidence` / `flattenRunes` 比对取值是否出现在证据/正文中，`looksPositive` 判断证书证据。
- **错误信息不泄露**：`llmErrorToHTTP` 把上游错误详情只写日志，不下发客户端。
- **上传安全**：魔数嗅探 + 文件名清洗（`sanitizeUploadName`）+ 大小限制 + 临时目录自动清理。
- **测试覆盖**：`TestTeacherListFilters` 覆盖了 LIKE 通配符转义、非法参数注入等边界。

---

## 五、建议的优化优先级

| 优先级 | 事项 | 类型 |
|--------|------|------|
| 🔴 高 | API Key 明文存储加密 | 安全 |
| 🟡 中 | 删除未实现的多用户死代码 | 一致性 |
| 🟡 中 | 删除未使用的 `pagination.tsx` / `fetchEnabledOptions` | 整洁 |
| 🟢 低 | 站点名缓存 | 性能 |
| 🟢 低 | 统一前端错误类型 | 类型安全 |
| 🟢 低 | 依赖版本锁定 | 可维护性 |

---

## 六、修复记录（2026-02 依据审计整改）

| 编号 | 事项 | 处理 | 涉及文件 |
|------|------|------|----------|
| 1 | API Key 明文存储 | 改为 **AES-256-GCM 加密**存储；密钥文件默认在数据库同目录 `.hirezo-secret`（0600 权限，自动生成），可用 `-secret-key` / `HIREZO_SECRET_FILE` 指定。历史明文无前缀时读取兼容、保存时自动升级为密文。LLM 配置读取改走 `app.loadLLMConfig()`，落库改走 `app.saveLLMConfig()` | 新增 `secret.go`；`llm.go`、`main.go`、`resume.go` |
| 2 | 多用户管理死代码 | 删除前端 `fetchUsers` / `createUser` / `updateUser` / `deleteUser`（后端无对应路由，单管理员架构） | `web/src/api/auth.ts` |
| 3 | 未使用文件/函数 | 删除 `pagination.tsx`（用自定义 `PaginationBar`）与 `fetchEnabledOptions`；顺带清理未使用的 `resumeKinds` | `web/src/components/ui/pagination.tsx`、`web/src/api/dicts.ts`、`resume.go` |
| 4 | `injectSiteTitle` 每次回退查库 | 站点名改为内存缓存（`app.siteName()` + `siteNameLoaded` 标记），设置变更时由 `apiSettingsUpdate` 调用 `invalidateSiteName()` 失效；空站点名也缓存，避免重复查库 | `main.go`、`store.go`、`auth.go` |
| 5 | rateLimiter 清扫逻辑 | 新增后台周期 `startSweep`（每分钟 `sweep()` 全表清理过期 key），并对单个 key 的时间戳数量设上限（`maxRateKeyEntries=256`），内存不再随请求量线性增长 | `store.go` |
| 6 | 前端大量 `any` 类型 | 新增 `errMsg(e: unknown, fallback)` 工具（`web/src/lib/utils.ts`），替换全部 14 处 `catch (e: any)` / `onError: (e: any)`，`Table.tsx` 的 `(r as any)[c.key]` 改为 `(r as Record<string, unknown>)[c.key]`；`Resume.tsx` / `LLM.tsx` 的局部 `errMsg` 收敛为共享实现 | `web/src/lib/utils.ts`、各页面/组件 |
| 7 | 依赖版本激进 | 维持在 `go 1.27`（与审计环境一致）；pdf 库伪版本保留（上游无稳定 tag，模块未发布版本）；已在 README 提示团队统一 Go 版本 | `go.mod`、`README.md` |
| 8 | `index.css` 体积 | Tailwind v4 按需生成已生效（构建产物未使用类被 tree-shake），暂不拆分；保留现状 | `web/src/index.css` |

**验证**：`go vet ./...` ✅ · `go test ./...` ✅（含新增 `secret_test.go` 加解密/历史明文/错误密钥/持久化/落库无明文用例）· `tsc -b` ✅ · `vite build` ✅

---
*本报告基于对全部源码的静态审查与 `go test` / `go vet` / `tsc` 验证，未做运行时渗透测试。*