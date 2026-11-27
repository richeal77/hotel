#!/usr/bin/env bash
set -euo pipefail

fail() { printf 'ECS 管理端部署失败：%s\n' "$*" >&2; exit 1; }
[[ $# == 4 ]] || fail '缺少模式、版本、文件或地址类型'
mode=$1
release_id=$2
staged_file=$3
address_kind=$4
[[ $mode == production || $mode == test ]] || fail '模式只能是 production 或 test'
[[ $release_id =~ ^[A-Za-z0-9-]+$ ]] || fail '版本号格式错误'
[[ $address_kind == ip || $address_kind == domain ]] || fail '地址类型错误'
[[ -s $staged_file ]] || fail '上传的静态文件不存在或为空'

web_port=8083
api_port=8081
app_env=production
if [[ $mode == test ]]; then
  web_port=8084
  api_port=8082
  app_env=integration
fi

nginx_bin=${HOTEL_ADMIN_NGINX_BIN:-/www/server/nginx/sbin/nginx}
nginx_conf_root=${HOTEL_ADMIN_NGINX_CONF_ROOT:-/www/server/panel/vhost/nginx}
nginx_conf=$nginx_conf_root/hotel-admin-$mode.conf
root_dir=${HOTEL_ADMIN_OPT_ROOT:-/opt/hotel-admin}/$mode
command -v flock >/dev/null 2>&1 || fail 'ECS 缺少 flock'
command -v ss >/dev/null 2>&1 || fail 'ECS 缺少 ss'
[[ -x $nginx_bin ]] || fail "找不到宝塔 Nginx：$nginx_bin"
[[ -d ${nginx_conf%/*} ]] || fail "找不到宝塔 Nginx 配置目录：${nginx_conf%/*}"
mkdir -p "$root_dir/releases"
exec 9>"$root_dir/.deploy.lock"
flock -n 9 || fail "$mode 管理端正在部署，请等上一次结束"

if [[ -f $nginx_conf ]] && ! grep -Fq '# Managed by hotel deploy-admin-web-remote.sh' "$nginx_conf"; then
  fail "$nginx_conf 已存在且不是本脚本创建，未覆盖"
fi
if [[ -f $nginx_conf ]]; then
  grep -Fq "listen $web_port;" "$nginx_conf" || fail "$nginx_conf 的监听端口不符，未修改"
  grep -Fq "root $root_dir/current;" "$nginx_conf" || fail "$nginx_conf 的静态目录不符，未修改"
  grep -Fq "proxy_pass http://127.0.0.1:$api_port;" "$nginx_conf" || fail "$nginx_conf 的后端转发不符，未修改"
fi
if [[ ! -f $nginx_conf ]] && [[ -n $(ss -H -ltnp "sport = :$web_port") ]]; then
  fail "端口 $web_port 已被占用，未创建站点"
fi

release_dir="$root_dir/releases/$release_id"
[[ ! -e $release_dir ]] || fail "版本 $release_id 已存在"
mkdir -p "$release_dir"
tar -xzf "$staged_file" -C "$release_dir"
rm -f "$staged_file"
[[ -s $release_dir/index.html && -s $release_dir/config.js && -s $release_dir/app.js ]] || fail '管理端文件不完整'
grep -Fq "\"appEnv\":\"$app_env\"" "$release_dir/config.js" || fail '管理端模式不正确'
grep -Fq '"apiBaseUrl":"/api/v1"' "$release_dir/config.js" || fail '管理端 API 地址配置错误'

current="$root_dir/current"
previous=''
if [[ -L $current ]]; then previous=$(readlink "$current"); fi
activate() {
  local target=$1 link="$root_dir/.current-$release_id"
  ln -s "$target" "$link"
  if [[ $(uname -s) == Linux ]]; then mv -Tf "$link" "$current"; else mv -f "$link" "$current"; fi
}
backup_conf=''
if [[ -f $nginx_conf ]]; then
  backup_conf="$root_dir/.nginx-conf-$release_id.bak"
  cp -p "$nginx_conf" "$backup_conf"
fi

restore() {
  if [[ -n $previous ]]; then
    activate "$previous"
  else
    rm -f "$current"
  fi
  if [[ -n $backup_conf ]]; then cp -p "$backup_conf" "$nginx_conf"; else rm -f "$nginx_conf"; fi
  "$nginx_bin" -t >/dev/null 2>&1 && "$nginx_bin" -s reload >/dev/null 2>&1 || true
  fail '新站点未通过检查，已恢复上一版本；检查 Nginx 日志'
}

if [[ ! -f $nginx_conf ]]; then
cat > "$nginx_conf" <<EOF
# Managed by hotel deploy-admin-web-remote.sh
server {
    listen $web_port;
    server_name _;
    root $root_dir/current;
    index index.html;
    add_header X-Content-Type-Options nosniff always;

    location /api/v1/ {
        proxy_pass http://127.0.0.1:$api_port;
        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;
    }

    location = /config.js {
        add_header Cache-Control "no-store";
        try_files \$uri =404;
    }

    location / {
        try_files \$uri \$uri/ /index.html;
    }
}
EOF
fi
"$nginx_bin" -t || restore
activate "$release_dir"
"$nginx_bin" -s reload || restore

ready=false
for attempt in 1 2 3 4 5 6 7 8 9 10; do
  if curl -fsS --max-time 2 "http://127.0.0.1:$web_port/config.js" 2>/dev/null | grep -Fq "\"appEnv\":\"$app_env\""; then
    api_status=$(curl -sS -o /dev/null -w '%{http_code}' --max-time 2 \
      -H 'Content-Type: application/json' -d '{}' \
      "http://127.0.0.1:$web_port/api/v1/admin/auth/login" 2>/dev/null) || api_status=000
    if [[ $api_status == 401 || $api_status == 400 ]]; then ready=true; break; fi
  fi
  sleep 1
done
[[ $ready == true ]] || restore
if [[ $address_kind == ip ]] && command -v firewall-cmd >/dev/null 2>&1 && firewall-cmd --state >/dev/null 2>&1; then
  zone=$(firewall-cmd --get-zone-of-interface=eth0 2>/dev/null) || zone=public
  [[ -n $zone ]] || zone=public
  if ! firewall-cmd --zone="$zone" --query-port="$web_port/tcp" >/dev/null; then
    firewall-cmd --zone="$zone" --add-port="$web_port/tcp" >/dev/null || fail "站点已启动，但防火墙未放行 $web_port/tcp"
  fi
  if ! firewall-cmd --permanent --zone="$zone" --query-port="$web_port/tcp" >/dev/null; then
    firewall-cmd --permanent --zone="$zone" --add-port="$web_port/tcp" >/dev/null || fail "站点已启动，但防火墙未永久放行 $web_port/tcp"
  fi
fi
rm -f "$backup_conf"
printf 'ECS 管理端已启动：%s，端口 %s，API 端口 %s\n' "$mode" "$web_port" "$api_port"
