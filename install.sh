#!/bin/sh
# AgentCRM 安装脚本
# 用法: curl -sfL https://github.com/AgentPal/AgentCRM/releases/latest/download/install.sh | sh

set -eu

REPO="AgentPal/AgentCRM"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# 检测平台
detect_platform() {
    OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
    ARCH="$(uname -m)"

    case "$OS" in
        linux)   PLATFORM="linux" ;;
        darwin)  PLATFORM="darwin" ;;
        mingw*|msys*|cygwin*) PLATFORM="windows" ;;
        *)       echo "不支持的操作系统: $OS"; exit 1 ;;
    esac

    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        *)          echo "不支持的架构: $ARCH"; exit 1 ;;
    esac

    echo "${PLATFORM}-${ARCH}"
}

PLATFORM=$(detect_platform)
echo "检测到平台: $PLATFORM"

# 获取最新版本
echo "获取最新版本..."
VERSION=$(curl -sfL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
if [ -z "$VERSION" ]; then
    echo "无法获取版本号，请检查网络连接"
    exit 1
fi
echo "最新版本: $VERSION"

# 下载
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$VERSION/agentcrm-${PLATFORM}.tar.gz"
echo "下载: $DOWNLOAD_URL"

TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

curl -sfL "$DOWNLOAD_URL" -o "$TMP_DIR/agentcrm.tar.gz"
tar xzf "$TMP_DIR/agentcrm.tar.gz" -C "$TMP_DIR"

# 安装
if [ ! -w "$INSTALL_DIR" ]; then
    echo "需要 sudo 权限安装到 $INSTALL_DIR"
    sudo mv "$TMP_DIR/agentcrm" "$INSTALL_DIR/agentcrm"
    sudo chmod +x "$INSTALL_DIR/agentcrm"
else
    mv "$TMP_DIR/agentcrm" "$INSTALL_DIR/agentcrm"
    chmod +x "$INSTALL_DIR/agentcrm"
fi

echo "安装完成！运行 'agentcrm init' 开始使用。"
