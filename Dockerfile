# ============================================
# Hirezo 教师管理系统 - Docker 镜像（两阶段：Node 构建前端 → Go 编译含 embed 的可执行文件）
#   docker build -t hirezo .
#   docker run -d -p 8080:8080 -v hirezo-data:/data --name hirezo hirezo
# ============================================

# 阶段 1：构建 React 前端
FROM node:22-alpine AS frontend
WORKDIR /web
# pnpm 11 默认会在运行任何命令前检查依赖完整性（无 TTY 时子进程 install 会失败），此处关闭
ENV pnpm_config_verify_deps_before_run=false
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN corepack enable && pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

# 阶段 2：Go 交叉编译（embed web/dist）
FROM golang:1.27-alpine AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /web/dist web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o hirezo .

# 运行阶段：alpine（需要 CA 证书访问大模型 HTTPS 接口，
# 以及 poppler-utils 提供 pdftoppm 识别扫描件 PDF）
FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata poppler-utils
COPY --from=builder /build/hirezo /hirezo
ENV TZ=Asia/Shanghai
WORKDIR /data
EXPOSE 8080
VOLUME ["/data"]
ENTRYPOINT ["/hirezo"]
CMD ["-db", "/data/hirezo.db"]
