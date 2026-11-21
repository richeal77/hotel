#!/usr/bin/env bash
set -euo pipefail

fail() { printf 'ECS 部署失败：%s\n' "$*" >&2; exit 1; }
[[ $# == 3 ]] || fail '需要模式、版本号和已上传的文件路径'
mode=$1
release_id=$2
staged_file=$3
[[ $mode == test || $mode == production ]] || fail '模式只能是 test 或 production'
[[ $release_id =~ ^[A-Za-z0-9-]+$ ]] || fail '版本号格式错误'
[[ -s $staged_file ]] || fail '上传的程序不存在或为空'

opt_root=${HOTEL_API_OPT_ROOT:-/opt/hotel-api}
env_root=${HOTEL_API_ENV_ROOT:-/etc/hotel-api}
data_root=${HOTEL_API_DATA_ROOT:-/var/lib/hotel-api}
systemd_root=${HOTEL_API_SYSTEMD_ROOT:-/etc/systemd/system}
run_user=${HOTEL_API_RUN_USER:-hotel-api-$mode}
service="hotel-api-$mode.service"
port=8082
addr='0.0.0.0:8082'
if [[ $mode == production ]]; then
  port=8081
  addr='127.0.0.1:8081'
fi

command -v flock >/dev/null 2>&1 || fail 'ECS 缺少 flock 命令（安装 util-linux 后重试）'
command -v ss >/dev/null 2>&1 || fail 'ECS 缺少 ss 命令（安装 iproute 后重试）'
mkdir -p "$opt_root"
exec 9>"$opt_root/.deploy-$mode.lock"
flock -n 9 || fail "$mode 环境正在部署；请等上一次部署结束后重试"

service_pid() {
  local pid
  pid=$(systemctl show --property=MainPID --value "$service" 2>/dev/null) || return 1
  [[ $pid =~ ^[1-9][0-9]*$ ]] || return 1
  printf '%s' "$pid"
}

listener_belongs_to_service() {
  local pid listeners
  pid=$(service_pid) || return 1
  listeners=$(ss -H -ltnp "sport = :$port") || return 1
  [[ $listeners == *"pid=$pid,"* ]]
}

existing_listener=$(ss -H -ltnp "sport = :$port") || fail "无法检查端口 $port 的占用情况"
if [[ -n $existing_listener ]] && ! listener_belongs_to_service; then
  fail "端口 $port 已被其他进程占用；请先确认原服务和端口配置"
fi

mkdir -p "$opt_root/$mode/releases" "$env_root" "$data_root/$mode" "$systemd_root"
chmod 700 "$env_root"

if ! id -u "$run_user" >/dev/null 2>&1; then
  useradd --system --user-group --shell /sbin/nologin "$run_user"
fi
chown "$run_user" "$data_root/$mode"
chmod 750 "$data_root/$mode"

env_file="$env_root/$mode.env"
if [[ $mode == test && ! -f $env_file ]]; then
  cat >"$env_file" <<EOF
APP_ENV=local-demo
API_ADDR=$addr
API_DATA_DIR=$data_root/$mode
EOF
  chmod 600 "$env_file"
fi
[[ -f $env_file ]] || fail "缺少 ${env_file}；先配置生产数据库和微信凭据"
env_mode=$(stat -c %a "$env_file" 2>/dev/null || stat -f %Lp "$env_file" 2>/dev/null)
[[ $env_mode == 600 ]] || fail "$env_file 权限必须是 600"
grep -Fxq "API_ADDR=$addr" "$env_file" || fail "$env_file 必须设置 API_ADDR=$addr"
if [[ $mode == production ]]; then
  grep -Fxq 'APP_ENV=production' "$env_file" || fail "$env_file 必须设置 APP_ENV=production"
  for key in MYSQL_DSN WECHAT_APP_ID WECHAT_APP_SECRET; do
    grep -Eq "^${key}=.+$" "$env_file" || fail "$env_file 缺少 $key"
  done
else
  grep -Eq '^APP_ENV=(local-demo|integration)$' "$env_file" || fail "$env_file 只允许 APP_ENV=local-demo 或 integration"
fi

release_dir="$opt_root/$mode/releases/$release_id"
[[ ! -e $release_dir ]] || fail "版本 $release_id 已存在"
mkdir -p "$release_dir"
install -m 755 "$staged_file" "$release_dir/hotel-api"
rm -f "$staged_file"

unit_file="$systemd_root/$service"
if [[ -f $unit_file ]] && ! grep -Fq '# Managed by hotel deploy-api-remote.sh' "$unit_file"; then
  fail "$unit_file 已存在且不是本脚本创建的服务；为避免覆盖，部署已停止"
fi
cat >"$unit_file" <<EOF
# Managed by hotel deploy-api-remote.sh
[Unit]
Description=Hotel API ($mode)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$run_user
WorkingDirectory=$data_root/$mode
EnvironmentFile=$env_file
ExecStart=$opt_root/$mode/current/hotel-api
Restart=on-failure
RestartSec=2
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
EOF
chmod 644 "$unit_file"
systemctl daemon-reload
systemctl enable "$service" >/dev/null

current="$opt_root/$mode/current"
previous=''
if [[ -L $current ]]; then previous=$(readlink "$current"); fi
next_link="$opt_root/$mode/.current-$release_id"
activate() {
  ln -s "$1" "$next_link"
  if [[ $(uname -s) == Linux ]]; then
    mv -Tf "$next_link" "$current"
  else
    mv -f "$next_link" "$current"
  fi
}
activate "$release_dir"

rollback() {
  if [[ -n $previous ]]; then
    activate "$previous"
    if ! systemctl restart "$service" >/dev/null 2>&1; then
      fail "已恢复上一版本文件，但旧服务重启失败；立即查看 journalctl -u $service -n 50"
    fi
  else
    systemctl stop "$service" >/dev/null 2>&1 || true
    rm -f "$current"
  fi
  fail "新版本未通过启动检查，已恢复上一版本；查看 journalctl -u $service -n 50"
}

systemctl restart "$service" || rollback

ready=false
for attempt in 1 2 3 4 5 6 7 8 9 10; do
  status=$(curl -sS -o /dev/null -w '%{http_code}' --max-time 2 \
    -H 'Content-Type: application/json' -d '{}' \
    "http://127.0.0.1:$port/api/v1/admin/auth/login" 2>/dev/null) || status=000
  if [[ $status == 400 || $status == 401 ]] && systemctl is-active --quiet "$service" && listener_belongs_to_service; then
    ready=true
    break
  fi
  sleep 1
done
[[ $ready == true ]] || rollback

if [[ $mode == production ]]; then
  mock_status=$(curl -sS -o /dev/null -w '%{http_code}' --max-time 2 \
    -H 'Content-Type: application/json' -d '{}' \
    "http://127.0.0.1:$port/api/v1/auth/mock-login" 2>/dev/null) || mock_status=000
  [[ $mock_status == 404 ]] || rollback
fi

printf 'ECS 服务已启动：%s，版本 %s，本机端口 %s\n' "$service" "$release_id" "$port"
