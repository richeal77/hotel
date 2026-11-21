#!/usr/bin/env bash
set -euo pipefail

usage() {
  cat <<'EOF'
用法：
  scripts/deploy-api.sh --mode test --address 39.105.122.224:8082
  scripts/deploy-api.sh --mode production --address api.example.com

选项：
  --mode       test 部署当前本地代码；production 部署 GitHub origin/main
  --address    当前环境的外部地址：测试可用 IP:8082，域名使用 HTTPS
  --dry-run    只显示部署计划，不编译、不连接服务器
  --help       显示帮助
EOF
}

die() { printf '错误：%s\n' "$*" >&2; exit 1; }
info() { printf '%s\n' "$*"; }
require_command() { command -v "$1" >/dev/null 2>&1 || die "缺少命令：$1"; }

mode=''
address=''
dry_run=false
while (($#)); do
  case "$1" in
    --mode|--address)
      (($# >= 2)) || die "$1 缺少参数"
      if [[ $1 == --mode ]]; then mode=$2; else address=$2; fi
      shift 2
      ;;
    --dry-run) dry_run=true; shift ;;
    --help|-h) usage; exit 0 ;;
    *) die "未知参数：$1" ;;
  esac
done

[[ -n $mode ]] || die '必须指定 --mode'
[[ $mode == test || $mode == production ]] || die '模式只能是 test 或 production'
[[ -n $address ]] || die '必须指定 --address'

if [[ $mode == test ]]; then port=8082; source_label='当前工作区（含未提交改动）'
else port=8081; source_label='GitHub origin/main'; fi

if [[ $address =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}:([0-9]+)$ ]]; then
  address_port=${BASH_REMATCH[2]}
  [[ $address_port == "$port" ]] || {
    if [[ $mode == test ]]; then die '测试模式的 IP 地址必须使用 8082'; fi
    die '线上模式必须使用 HTTPS 域名'
  }
  [[ $mode == test ]] || die '线上模式必须使用 HTTPS 域名'
  IFS=. read -r octet1 octet2 octet3 octet4 <<< "${address%:*}"
  for octet in "$octet1" "$octet2" "$octet3" "$octet4"; do
    ((10#$octet <= 255)) || die '地址格式错误：IP 地址中的数字必须在 0 到 255 之间'
  done
  external_url="http://$address"
elif [[ $address =~ ^([A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?\.)+[A-Za-z]([A-Za-z0-9-]*[A-Za-z0-9])?$ ]]; then
  external_url="https://$address"
else
  die '地址格式错误：填写 IP:端口 或域名，不加 http://、路径和空格'
fi

repo_dir=$(cd "$(dirname "$0")/.." && pwd)
target='root@39.105.122.224'
info "模式：${mode}（端口 ${port}）"
info "源码：$source_label"
info "服务器：$target:22"
info "访问：$external_url/api/v1"

if [[ $dry_run == true ]]; then
  info '预演结束：没有编译或连接服务器。'
  exit 0
fi

for command in git go ssh scp curl tar mktemp; do require_command "$command"; done
tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/hotel-api-deploy.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT

if [[ $mode == production ]]; then
  info '正在获取 GitHub main 的最新提交…'
  git -C "$repo_dir" fetch origin main
  source_root="$tmp_dir/source"
  mkdir -p "$source_root"
  git -C "$repo_dir" archive origin/main | tar -x -C "$source_root"
  [[ -f "$source_root/apps/api/cmd/server/main.go" ]] || die 'origin/main 缺少后端源码'
  grep -Eq 'mode[[:space:]]*==[[:space:]]*"production"' "$source_root/apps/api/cmd/server/main.go" || die 'GitHub main 尚不支持 APP_ENV=production；线上部署已停止'
  [[ -f "$source_root/apps/api/internal/app/live_auth.go" ]] || die 'GitHub main 尚无真实登录实现；线上部署已停止'
  source_ref=$(git -C "$repo_dir" rev-parse --short=12 origin/main)
else
  source_root=$repo_dir
  source_ref="local-$(date -u +%Y%m%d%H%M%S)"
fi

info "正在测试 Go 后端（${source_ref}）…"
(cd "$source_root/apps/api" && GOCACHE="$tmp_dir/go-cache" go test ./...)

binary="$tmp_dir/hotel-api"
info '正在编译 Linux x86_64 程序…'
(cd "$source_root/apps/api" && GOCACHE="$tmp_dir/go-cache" CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o "$binary" ./cmd/server)
[[ -s $binary ]] || die '编译结果为空'

release_id="$(date -u +%Y%m%d%H%M%S)-$$"
remote_tmp="/tmp/hotel-api-${mode}-${release_id}"
info '正在上传程序到 ECS…'
scp -P 22 "$binary" "$target:$remote_tmp"

info "正在更新 ECS 上的 $mode 服务…"
ssh -p 22 "$target" bash -s -- "$mode" "$release_id" "$remote_tmp" < "$repo_dir/scripts/deploy-api-remote.sh"

info "正在检查外部地址：$external_url"
status=$(curl -sS -o /dev/null -w '%{http_code}' --connect-timeout 5 --max-time 15 \
  -H 'Content-Type: application/json' -d '{}' "$external_url/api/v1/admin/auth/login") || die "服务已启动，但外部地址无法连接：${external_url}；检查安全组、DNS、证书或 Nginx"
[[ $status == 400 || $status == 401 ]] || die "服务已启动，但外部接口返回 HTTP ${status}；检查 Nginx 转发"
info "部署完成：${mode}，版本 ${source_ref}，接口 $external_url/api/v1"
