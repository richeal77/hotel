#!/usr/bin/env bash
set -euo pipefail

die() { printf '管理端部署失败：%s\n' "$*" >&2; exit 1; }
info() { printf '%s\n' "$*"; }

usage() {
  cat <<'EOF'
用法：
  ./scripts/deploy-admin-web.sh --mode production --address 39.105.122.224:8083
  ./scripts/deploy-admin-web.sh --mode test --address 39.105.122.224:8084
  ./scripts/deploy-admin-web.sh --mode production --address admin.example.com

production 使用本地 main 已提交版本，连接生产后端 8081。
test 使用当前工作区，连接测试后端 8082。
IP:端口使用 HTTP；域名使用 HTTPS，域名的 Nginx/证书需预先配置。
--dry-run 只显示计划，不编译或连接 ECS。
EOF
}

mode=''
address=''
dry_run=false
while (($#)); do
  case $1 in
    --mode|--address)
      (($# >= 2)) || die "$1 缺少参数"
      if [[ $1 == --mode ]]; then mode=$2; else address=$2; fi
      shift 2 ;;
    --dry-run) dry_run=true; shift ;;
    --help|-h) usage; exit 0 ;;
    *) die "未知参数：$1" ;;
  esac
done

[[ $mode == production || $mode == test ]] || die '必须指定 --mode production 或 --mode test'
[[ -n $address ]] || die '必须指定 --address'

if [[ $mode == production ]]; then
  web_port=8083
  api_port=8081
  app_env=production
  source_label='本地 main 已提交版本'
else
  web_port=8084
  api_port=8082
  app_env=integration
  source_label='当前工作区（含未提交改动）'
fi

if [[ $address =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}:([0-9]+)$ ]]; then
  [[ ${BASH_REMATCH[2]} == "$web_port" ]] || die "$mode 模式的 IP 地址必须使用 $web_port 端口"
  IFS=. read -r a b c d <<< "${address%:*}"
  for octet in "$a" "$b" "$c" "$d"; do
    ((10#$octet <= 255)) || die 'IP 地址格式错误'
  done
  external_url="http://$address"
  address_kind=ip
elif [[ $address =~ ^([A-Za-z0-9]([A-Za-z0-9-]*[A-Za-z0-9])?\.)+[A-Za-z]([A-Za-z0-9-]*[A-Za-z0-9])?$ ]]; then
  external_url="https://$address"
  address_kind=domain
else
  die '地址格式错误：填写 IP:端口 或域名，不加协议和路径'
fi

repo_dir=$(cd "$(dirname "$0")/.." && pwd)
target=root@39.105.122.224
info "模式：$mode"
info "源码：$source_label"
info "管理端：$external_url"
info "ECS Nginx 端口：${web_port}；后端 API 端口：${api_port}"
if [[ $dry_run == true ]]; then
  info '预演结束：没有编译或连接服务器。'
  exit 0
fi

for command in git node ssh scp curl tar mktemp; do
  command -v "$command" >/dev/null 2>&1 || die "Mac 缺少命令：$command"
done
tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/hotel-admin-deploy.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT

if [[ $mode == production ]]; then
  git -C "$repo_dir" rev-parse --verify refs/heads/main >/dev/null || die '本地不存在 main 分支'
  source_root="$tmp_dir/source"
  mkdir -p "$source_root"
  info '正在读取本地 main 已提交版本（不访问 GitHub）…'
  git -C "$repo_dir" archive refs/heads/main | tar -x -C "$source_root"
  source_ref=$(git -C "$repo_dir" rev-parse --short=12 refs/heads/main)
else
  source_root=$repo_dir
  source_ref="local-$(date -u +%Y%m%d%H%M%S)"
fi

web_dir="$source_root/apps/admin-web"
[[ -f $web_dir/scripts/build.mjs ]] || die '源码缺少管理端构建程序'
info '正在测试管理端…'
(cd "$web_dir" && node --test ./test/*.test.mjs)
info '正在构建管理端静态文件…'
(cd "$web_dir" && APP_ENV="$app_env" ADMIN_API_BASE_URL=/api/v1 node ./scripts/build.mjs)
grep -Fq "\"appEnv\":\"$app_env\"" "$web_dir/dist/config.js" || die '构建环境配置不正确'
grep -Fq '"apiBaseUrl":"/api/v1"' "$web_dir/dist/config.js" || die '管理端 API 地址不是同源 /api/v1'
tar -czf "$tmp_dir/admin-web.tar.gz" -C "$web_dir/dist" .

release_id="$(date -u +%Y%m%d%H%M%S)-$$"
remote_tmp="/tmp/hotel-admin-${mode}-${release_id}.tar.gz"
info '正在上传管理端到 ECS；如果提示密码，请输入 ECS root 密码。'
scp -P 22 "$tmp_dir/admin-web.tar.gz" "$target:$remote_tmp"
info '上传完成，正在更新 ECS Nginx 站点；如果再次提示密码，请再次输入。'
ssh -p 22 "$target" bash -s -- "$mode" "$release_id" "$remote_tmp" "$address_kind" < "$repo_dir/scripts/deploy-admin-web-remote.sh"

info "正在检查外部访问：$external_url"
index_file="$tmp_dir/index.html"
config_file="$tmp_dir/config.js"
curl -fsS --connect-timeout 5 --max-time 15 "$external_url/" -o "$index_file" || {
  if [[ $address_kind == ip ]]; then die "站点已更新，但外部访问失败：检查阿里云安全组是否放行 $web_port/tcp"; fi
  die '站点已更新，但外部访问失败：检查域名、HTTPS 证书和宝塔 Nginx 配置'
}
grep -Fq '高碑店尚客优管理后台' "$index_file" || die '外部地址没有返回管理端网页'
curl -fsS --connect-timeout 5 --max-time 15 "$external_url/config.js" -o "$config_file" || die '外部地址没有返回管理端配置'
grep -Fq "\"appEnv\":\"$app_env\"" "$config_file" || die '外部地址返回了错误模式的管理端配置'
api_status=$(curl -sS -o /dev/null -w '%{http_code}' --connect-timeout 5 --max-time 15 \
  -H 'Content-Type: application/json' -d '{}' "$external_url/api/v1/admin/auth/login") || die '管理端可访问，但后端 API 转发失败'
[[ $api_status == 401 || $api_status == 400 ]] || die "管理端 API 返回 HTTP ${api_status}，预期为 401 或 400"
info "管理端部署完成：${mode}，版本 ${source_ref}，地址 $external_url"
if [[ $address_kind == ip ]]; then
  info '当前为 HTTP，仅用于访问验收；管理员输入真实密码前需配置 HTTPS。'
fi
