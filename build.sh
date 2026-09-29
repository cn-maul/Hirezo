#!/usr/bin/env bash
# ============================================
#   Hirezo 教师管理系统 - Linux 构建
#   Usage:
#     ./build.sh                  # 默认: 构建前端 + 编译当前平台
#     ./build.sh linux amd64      # 交叉编译到 linux/amd64
#     ./build.sh windows amd64    # 交叉编译到 windows/amd64
# ============================================
set -euo pipefail

cd "$(dirname "$0")"

BINARY="hirezo"
GOOS="${1:-}"
GOARCH="${2:-}"

# 指定目标平台时拼接文件名
if [ -n "$GOOS" ] && [ -n "$GOARCH" ]; then
    BINARY="hirezo-${GOOS}-${GOARCH}"
    if [ "$GOOS" = "windows" ]; then
        BINARY="${BINARY}.exe"
    fi
    export GOOS GOARCH
    echo "  目标平台: ${GOOS}/${GOARCH}"
fi

echo "============================================"
echo "  Hirezo 教师管理系统 - 构建"
echo "============================================"
echo ""

# ---- 确认 pnpm 可用（nvm 的登录脚本会自动加载）----
# pnpm 11 默认会在运行任何命令前检查项目依赖完整性，但非交互（无 TTY）时
# 子进程 install 会失败；此处关闭该校验（已被验证过的等价方式）。
export pnpm_config_verify_deps_before_run=false
if ! command -v pnpm >/dev/null 2>&1; then
    export NVM_DIR="${NVM_DIR:-$HOME/.config/nvm}"
    if [ -s "$NVM_DIR/nvm.sh" ]; then
        # shellcheck disable=SC1091
        . "$NVM_DIR/nvm.sh" >/dev/null 2>&1
    fi
fi
if ! command -v pnpm >/dev/null 2>&1; then
    echo "[失败] 未找到 pnpm，请先安装：npm install -g pnpm"
    exit 1
fi

# ---- [1/2] 构建前端 ----
echo "[1/2] 构建前端 ..."
if [ ! -d "web/node_modules/vite" ]; then
    echo "    安装依赖 ..."
    pnpm --dir web install --ignore-scripts --force
fi
pnpm --dir web build
echo "    前端构建完成"
echo ""

# ---- [2/2] 编译 Go 可执行文件 ----
echo "[2/2] 编译 Go 可执行文件 ..."
CGO_ENABLED=0 go build -ldflags="-s -w" -o "$BINARY" .
echo ""

echo "[成功] 已生成 ${BINARY}"
echo "       $(ls -lh "$BINARY" | awk '{print $5}')  前端已嵌入单文件，无需静态托管"
echo ""
echo "运行方式："
echo "  ./${BINARY}                     # 默认 :8080"
echo "  ./${BINARY} -addr :8888 -db /path/to/hirezo.db"
echo "  PORT=8888 ./${BINARY}"
echo ""
echo "开发模式（前端分仓，热更新）："
echo "  pnpm --dir web dev  # Vite :5173（代理 /api 到 :8080）"
echo "  go run .            # Go 监听 :8080"
