#!/usr/bin/env bash
set -euo pipefail

repo_dir=$(cd "$(dirname "$0")/.." && pwd)
script="$repo_dir/scripts/deploy-admin-web.sh"
remote="$repo_dir/scripts/deploy-admin-web-remote.sh"
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT

"$script" --mode production --address 39.105.122.224:8083 --dry-run > "$tmp_dir/out"
grep -Fq '后端 API 端口：8081' "$tmp_dir/out"
"$script" --mode test --address 39.105.122.224:8084 --dry-run > "$tmp_dir/out"
grep -Fq '后端 API 端口：8082' "$tmp_dir/out"
if "$script" --mode production --address 39.105.122.224:8084 --dry-run > "$tmp_dir/out" 2>&1; then
  echo 'FAIL: production accepted test port' >&2
  exit 1
fi
grep -Fq '必须使用 8083' "$tmp_dir/out"

mkdir -p "$tmp_dir/fakebin" "$tmp_dir/nginx-conf" "$tmp_dir/opt" "$tmp_dir/site"
cat > "$tmp_dir/fakebin/nginx" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$FAKE_NGINX_LOG"
exit 0
EOF
cat > "$tmp_dir/fakebin/ss" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
cat > "$tmp_dir/fakebin/flock" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
cat > "$tmp_dir/fakebin/curl" <<'EOF'
#!/usr/bin/env bash
if [[ " $* " == *'/config.js'* ]]; then
  printf 'window.ADMIN_CONFIG = Object.freeze({"appEnv":"production","apiBaseUrl":"/api/v1","proxyEnabled":false});\n'
else
  printf '%s' "${FAKE_API_STATUS:-401}"
fi
EOF
chmod +x "$tmp_dir/fakebin/"*
export PATH="$tmp_dir/fakebin:$PATH"
export FAKE_NGINX_LOG="$tmp_dir/nginx.log"
export HOTEL_ADMIN_NGINX_BIN="$tmp_dir/fakebin/nginx"
export HOTEL_ADMIN_NGINX_CONF_ROOT="$tmp_dir/nginx-conf"
export HOTEL_ADMIN_OPT_ROOT="$tmp_dir/opt"
printf '<title>高碑店尚客优管理后台</title>\n' > "$tmp_dir/site/index.html"
printf 'window.ADMIN_CONFIG = Object.freeze({"appEnv":"production","apiBaseUrl":"/api/v1","proxyEnabled":false});\n' > "$tmp_dir/site/config.js"
printf 'console.log("admin");\n' > "$tmp_dir/site/app.js"
tar -czf "$tmp_dir/staged-v1.tar.gz" -C "$tmp_dir/site" .
bash "$remote" production prod-v1 "$tmp_dir/staged-v1.tar.gz" domain > "$tmp_dir/out"
test "$(readlink "$tmp_dir/opt/production/current")" = "$tmp_dir/opt/production/releases/prod-v1"
grep -Fq 'proxy_pass http://127.0.0.1:8081;' "$tmp_dir/nginx-conf/hotel-admin-production.conf"
grep -Fq -- '-s reload' "$FAKE_NGINX_LOG"
cp "$tmp_dir/nginx-conf/hotel-admin-production.conf" "$tmp_dir/original-nginx.conf"

tar -czf "$tmp_dir/staged-v2.tar.gz" -C "$tmp_dir/site" .
if FAKE_API_STATUS=500 bash "$remote" production prod-v2 "$tmp_dir/staged-v2.tar.gz" domain > "$tmp_dir/out" 2>&1; then
  echo 'FAIL: unhealthy API was accepted' >&2
  exit 1
fi
test "$(readlink "$tmp_dir/opt/production/current")" = "$tmp_dir/opt/production/releases/prod-v1"
cmp -s "$tmp_dir/original-nginx.conf" "$tmp_dir/nginx-conf/hotel-admin-production.conf"

printf 'unmanaged site\n' > "$tmp_dir/nginx-conf/hotel-admin-test.conf"
tar -czf "$tmp_dir/staged-test.tar.gz" -C "$tmp_dir/site" .
if bash "$remote" test test-v1 "$tmp_dir/staged-test.tar.gz" domain > "$tmp_dir/out" 2>&1; then
  echo 'FAIL: unmanaged Nginx config was overwritten' >&2
  exit 1
fi
grep -Fq 'unmanaged site' "$tmp_dir/nginx-conf/hotel-admin-test.conf"

cat > "$tmp_dir/fakebin/scp" <<'EOF'
#!/usr/bin/env bash
[[ -s $3 ]] || exit 1
printf '%s\n' "$*" > "$FAKE_SCP_LOG"
EOF
cat > "$tmp_dir/fakebin/ssh" <<'EOF'
#!/usr/bin/env bash
cat >/dev/null
printf '%s\n' "$*" > "$FAKE_SSH_LOG"
EOF
cat > "$tmp_dir/fakebin/curl" <<'EOF'
#!/usr/bin/env bash
out=''
url=''
while (($#)); do
  if [[ $1 == -o ]]; then out=$2; shift 2; continue; fi
  if [[ $1 == http* ]]; then url=$1; fi
  shift
done
if [[ $url == */config.js ]]; then
  printf 'window.ADMIN_CONFIG = Object.freeze({"appEnv":"production","apiBaseUrl":"/api/v1","proxyEnabled":false});\n' > "$out"
elif [[ $url == */api/v1/admin/auth/login ]]; then
  printf '401'
else
  printf '<title>高碑店尚客优管理后台</title>\n' > "$out"
fi
EOF
chmod +x "$tmp_dir/fakebin/scp" "$tmp_dir/fakebin/ssh" "$tmp_dir/fakebin/curl"
export FAKE_SCP_LOG="$tmp_dir/scp.log"
export FAKE_SSH_LOG="$tmp_dir/ssh.log"
"$script" --mode production --address 39.105.122.224:8083 > "$tmp_dir/out"
grep -Fq '管理端部署完成：production' "$tmp_dir/out"
grep -Fq 'root@39.105.122.224:' "$FAKE_SCP_LOG"
grep -Fq 'production' "$FAKE_SSH_LOG"

echo 'deploy-admin-web tests passed'
