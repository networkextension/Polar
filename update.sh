
#!/usr/bin/env bash
set -euo pipefail

# update.sh
# - 检查本地 polar 版本（polar-vX.Y.Z-<os>-<arch>.tar.gz）
# - 获取 GitHub release latest
# - 如果远程版本较新，则下载并解压

REPO="networkextension/Polar-"
BIN_PREFIX="polar"
LOCAL_DIR="."
DOWNLOAD_DIR="${LOCAL_DIR}/.polar_update"

# 获取平台
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
if [ "$OS" = "darwin" ]; then
  OS="darwin"
elif [ "$OS" = "linux" ]; then
  OS="linux"
else
  echo "Unsupported OS: $OS" >&2
  exit 1
fi

case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;; 
  aarch64|arm64) ARCH="arm64" ;; 
  *) echo "Unsupported arch: $ARCH" >&2; exit 1 ;;
esac

# 本地最新版本（优先最新时间）
local_file=$(ls -1t "${LOCAL_DIR}/${BIN_PREFIX}-v"*"-${OS}-${ARCH}.tar.gz" 2>/dev/null | head -n1 || true)
local_version="0.0.0"
if [ -n "$local_file" ]; then
  local_version=$(basename "$local_file" | sed -E "s/^${BIN_PREFIX}-v([0-9]+\.[0-9]+\.[0-9]+)-${OS}-${ARCH}\.tar\.gz$/\1/")
fi

echo "local version: $local_version, platform: $OS/$ARCH"

# 获取 latest release
latest_info=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest")
if [ -z "$latest_info" ]; then
  echo "Failed to fetch latest release info" >&2
  exit 1
fi
latest_tag=$(echo "$latest_info" | grep -oP '"tag_name"\s*:\s*"\K[^\"]+')
latest_version="${latest_tag#v}"

echo "remote latest: $latest_tag ($latest_version)"

ver_ge() {
  local v1=$1 v2=$2
  IFS=. read -r -a a <<< "$v1"
  IFS=. read -r -a b <<< "$v2"
  for i in 0 1 2; do
    ai=${a[i]:-0}
    bi=${b[i]:-0}
    if (( ai > bi )); then return 0; fi
    if (( ai < bi )); then return 1; fi
  done
  return 0
}

if ver_ge "$local_version" "$latest_version"; then
  if [ "$local_version" = "$latest_version" ]; then
    echo "Already latest ($latest_version)"
    exit 0
  fi
  echo "Local version $local_version is newer than remote $latest_version, skip."
  exit 0
fi

asset_name="${BIN_PREFIX}-${latest_tag}-${OS}-${ARCH}.tar.gz"
download_url=$(echo "$latest_info" | grep -oP '"browser_download_url"\s*:\s*"\K[^\"]+' | grep "/${asset_name}$" | head -n1)
if [ -z "$download_url" ]; then
  echo "No asset found for ${asset_name}" >&2
  exit 1
fi

mkdir -p "$DOWNLOAD_DIR"
output_file="$DOWNLOAD_DIR/${asset_name}"

echo "Downloading $download_url to $output_file"
wget -qO "$output_file" "$download_url"

if [ ! -f "$output_file" ]; then
  echo "Download failed" >&2
  exit 1
fi

echo "Extracting to ${LOCAL_DIR}"
tar -xzf "$output_file" -C "$LOCAL_DIR"

echo "Done: ${latest_tag} (${OS}/${ARCH})"