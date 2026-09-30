# Hirezo 教师管理

> 面向学校教师人员的档案管理系统：录入与维护教师信息，按学科 / 性别 / 学历 / 资格证筛选，一键导出 Excel；支持上传简历自动识别入库。

**架构**：后端 Go（标准库 `net/http` + SQLite，纯 Go 驱动、无 CGO），提供 JSON REST API；前端 Vue 3 + TypeScript + Vite + vue-router，样式为自实现 CSS（设计令牌 + 亮/暗主题，无组件库、无 Tailwind），构建为 SPA 并通过 `embed.FS` 由 Go 单文件托管。**单管理员账号**。核心功能零外部依赖；仅「简历识别」按需调用你配置的大模型接口（经 `rosetta` SDK 支持 OpenAI Chat / Responses / Anthropic Messages 三协议任选）。

## 快速开始

```bash
./build.sh          # 构建前端 + 编译（默认当前平台）
./hirezo            # 默认 http://localhost:8080
```

- **访问地址**：`http://localhost:8080`，默认账号 `admin` / `admin123`（可用 `-password` 或 `HIREZO_PASSWORD` 覆盖，登录后请在右上角头像 → 账号设置中修改密码）
- **反向代理部署**：加 `-trust-proxy` 后才会读取 `X-Forwarded-For` 头获取真实 IP；直连部署时不要开，否则限流可被伪造头绕过

开发模式（前端分仓，热更新）：

```bash
pnpm --dir web dev    # Vite :5173（代理 /api 到 :8080）
go run .              # Go 监听 :8080
```

## 功能概览

- **人员管理**：姓名 / 性别 / 年龄 / 学科 / 教师资格证 / 电话 / 学历 / 毕业院校 / 专业 / 备注；表格 + 弹窗表单新增编辑，人员详情页字段两列成行，单条与批量硬删除
- **简历识别**：上传 `.docx` / `.pdf`（≤10MB）后进入左右双栏——左侧简历原件预览，右侧识别进度，识别完成后右侧就地变成核对表单（不额外弹窗，短字段两列一排）；需要复核的字段按黄/红框标出并在顶部汇总告警，手机号 / 年龄 / 资格证走本地正则闸门，学科与学历只认字典白名单，扫描件 PDF 自动逐页渲染（需要 `pdftoppm`）
- **简历原件归档**：经识别入库的人员，其简历原文件保存在服务器 `resumes/` 目录（默认与数据库同目录），人员详情页可直接「下载简历」取回原件；手工录入无原件则不显示该按钮
- **模型配置**：设置 → 模型配置，填端点 / 协议 / 模型 / API Key 即可（支持 DeepSeek、Qwen、Moonshot、Claude、Ollama、vLLM 等）；密钥以 AES-256-GCM 加密后存本地数据库（密钥文件默认在数据库同目录 `.hirezo-secret`，可用 `-secret-key` 或 `HIREZO_SECRET_FILE` 指定），任何接口不回显原文；备份时请连同密钥文件一起备份
- **组合筛选与搜索**：姓名关键字（防抖 + URL 同步）、学科、性别、学历、资格证，分页浏览
- **Excel 导出**：跟随当前筛选条件导出 `.xlsx`（excelize 生成，非 CSV）
- **字典管理**：学科与学历为可配置字典（名称、颜色、排序、启用停用）；重命名会级联更新人员记录，被引用时禁止删除并返回 409
- **通用设置**：站点名称（左上角与登录页同步显示）
- **账号安全**：bcrypt 存储密码、内存会话 Cookie（重启失效）、登录限流、改密后其它会话立即失效
- **SQLite 单文件存储**：数据即 `hirezo.db`（WAL 模式下另有 `-wal`/`-shm` 伴生文件），停机复制即可备份，也可用 `VACUUM INTO` 做在线快照；仅简历原件存在库外的 `resumes/` 目录，备份时一并拷走

## 简历识别

1. 侧边栏「简历识别」→ 拖入或选择简历（按文件内容判定类型，扩展名不作数）。
2. 页面立刻变成左右两栏：左边是简历原件预览（Word 内嵌渲染、PDF 用浏览器原生查看器），右边显示识别中与已用时；服务端解析（docx 走 zip+xml，PDF 走文本层；扫描件用 `pdftoppm` 渲染成图片交给模型）→ 调用模型抽取 → 本地闸门校验 → 存草稿。
3. 识别完成后右侧直接显示核对表单：短字段两列一排（姓名/性别、年龄/联系电话、学历/毕业院校、学科/专业），备注与教师资格证整幅；黄/红框标出需要复核的字段并在表单顶部汇总原因，学科、学历不在字典中的留空并提示下拉选择；对照原件就看左侧的简历。
4. 核对无误后「确认入库」，走与手工录入完全相同的校验；点「返回上传」即回到上传区，重新选一份简历即可。
5. 入库成功的简历原件会归档到 `resumes/` 目录，人员在详情页可下载原件；删除人员即删除归档文件。

**运行依赖**（可选）：

| 依赖 | 用途 | 缺失时 |
|------|------|--------|
| `pdftoppm`（poppler-utils） | 渲染扫描件 / 图片型 PDF | 明确报错，文本层 PDF 与 docx 不受影响 |
| 大模型 API | 字段抽取 | 页面提示未配置，跳转设置页 |

## 配置项

| 配置 | 方式 | 默认 | 说明 |
|------|------|------|------|
| 监听地址 | `-addr` flag / `PORT` 环境变量 | `:8080` | |
| 数据库路径 | `-db` flag | `hirezo.db` | WAL 模式，另有 `-wal`/`-shm` 伴生文件 |
| 初始密码 | `-password` flag / `HIREZO_PASSWORD` | `admin123` | 仅首次创建 admin 时生效，登录后请改密 |
| 信任代理头 | `-trust-proxy` flag | 关 | 仅反代理后开启，否则限流可被伪造 `X-Forwarded-For` 绕过 |
| 密钥文件 | `-secret-key` flag / `HIREZO_SECRET_FILE` | 数据库同目录 `.hirezo-secret` | API Key 的 AES-256-GCM 加密密钥，0600 权限自动生成 |
| 简历原件目录 | `-resume-dir` flag | 数据库同目录 `resumes/` | 识别入库的简历原文件归档处，草稿临时文件在其 `drafts/` 子目录（启动时清理） |

## 部署

**拉取镜像**（已含 `poppler-utils`（`pdftoppm`，识别扫描件必需）与 CA 证书，`linux/amd64`）：

```bash
docker pull ghcr.io/cn-maul/hirezo:0.3.0
docker run -d -p 8882:8882 -v hirezo-data:/data --name hirezo ghcr.io/cn-maul/hirezo:0.3.0
```

监听端口由镜像里的 `PORT=8882` 决定（`-addr` 默认跟随 `PORT`），换端口同时改 `-p` 和 `-e PORT`。`/data` 卷内是 `hirezo.db`、`.hirezo-secret` 和 `resumes/`（简历原件），备份这三样就是全量备份。

**镜像构建**：推 `v*` 标签（如 `git tag v0.3.0 && git push origin v0.3.0`）后由 GitHub Actions（`.github/workflows/docker.yml`）构建并推送到 ghcr，镜像 tag 取 `web/package.json` 的 version，标签与版本号不一致时流水线直接报错退出；打 tag 触发时额外推 `:latest`，手动运行 workflow 只推版本号。推送用的是仓库自带的 `GITHUB_TOKEN`（workflow 里声明了 `packages: write`），不需要配置任何 PAT。

**本地构建**：`docker build -t hirezo .`

**预编译二进制 + 最小镜像**（不想在容器里跑构建时）：

```bash
./build.sh linux amd64                       # 得到 hirezo-linux-amd64
docker build -t hirezo -f Dockerfile.binary . # 该文件需与二进制同目录
```

**手工部署**：`./build.sh` 产出单文件（Windows 用 `build.bat`），拷到目标机直接运行；数据即 `hirezo.db`，连同 `.hirezo-secret` 与 `resumes/` 目录（简历原件）一起备份。

## 详细文档

| 文档 | 说明 |
|------|------|
| [设计文档](docs/DESIGN.md) | 需求、技术选型、数据模型、API 约定、简历识别与字段闸门、构建部署 |
| [审计报告](docs/AUDIT.md) | 代码审计结论与整改记录（含 API Key 加密、会话与限流加固） |

> 审计与设计文档中涉及前端的章节以 v0.2.0 的 Vue 3 迁移为准；迁移前的 React 文件路径仅作历史记录。
