#!/usr/bin/env bash
set -euo pipefail

repo_dir=$(cd "$(dirname "$0")/.." && pwd)
script="$repo_dir/scripts/deploy-api.sh"
tmp_dir=$(mktemp -d)
trap 'rm -rf "$tmp_dir"' EXIT

check_failure() {
  local label=$1 expected=$2
  shift 2
  if "$script" "$@" >"$tmp_dir/out" 2>&1; then
    echo "FAIL $label: command succeeded" >&2
    exit 1
  fi
  if ! grep -Fq "$expected" "$tmp_dir/out"; then
    echo "FAIL $label: expected '$expected'" >&2
    cat "$tmp_dir/out" >&2
    exit 1
  fi
}

check_failure 'missing mode' '必须指定 --mode' --address 39.105.122.224:8082 --dry-run
check_failure 'invalid mode' '模式只能是 test 或 production' --mode qa --address 39.105.122.224:8082 --dry-run
check_failure 'mismatched test port' '测试模式的 IP 地址必须使用 8082' --mode test --address 39.105.122.224:8081 --dry-run
check_failure 'mismatched production port' '线上模式的 IP 地址必须使用 8081' --mode production --address 39.105.122.224:8082 --dry-run
check_failure 'invalid address' '地址格式错误' --mode test --address 'bad host' --dry-run
check_failure 'invalid IPv4' 'IP 地址中的数字必须在 0 到 255 之间' --mode test --address 999.105.122.224:8082 --dry-run

"$script" --mode test --address 39.105.122.224:8082 --dry-run >"$tmp_dir/out"
grep -Fq '源码：当前工作区（含未提交改动）' "$tmp_dir/out"
grep -Fq '数据库：hotel_prod_test' "$tmp_dir/out"
grep -Fq '访问：http://39.105.122.224:8082/api/v1' "$tmp_dir/out"

"$script" --mode production --address api.example.com --dry-run >"$tmp_dir/out"
grep -Fq '源码：本地 main 已提交版本' "$tmp_dir/out"
grep -Fq '数据库：hotel_prod' "$tmp_dir/out"
grep -Fq '访问：https://api.example.com/api/v1' "$tmp_dir/out"
"$script" --mode production --address 39.105.122.224:8081 --dry-run >"$tmp_dir/out"
grep -Fq '访问：http://39.105.122.224:8081/api/v1' "$tmp_dir/out"

echo 'deploy-api CLI tests passed'

mkdir -p "$tmp_dir/build-fail-bin"
cat >"$tmp_dir/build-fail-bin/go" <<'EOF'
#!/usr/bin/env bash
exit 1
EOF
cat >"$tmp_dir/build-fail-bin/ssh" <<'EOF'
#!/usr/bin/env bash
touch "$SSH_CALLED_MARKER"
exit 1
EOF
chmod +x "$tmp_dir/build-fail-bin/go" "$tmp_dir/build-fail-bin/ssh"
if PATH="$tmp_dir/build-fail-bin:$PATH" SSH_CALLED_MARKER="$tmp_dir/ssh-called" \
    "$script" --mode test --address 39.105.122.224:8082 >"$tmp_dir/out" 2>&1; then
  echo 'FAIL rejected Go tests did not stop deployment' >&2
  exit 1
fi
test ! -e "$tmp_dir/ssh-called"

cat >"$tmp_dir/build-fail-bin/git" <<'EOF'
#!/usr/bin/env bash
for arg in "$@"; do
  if [[ $arg == fetch ]]; then touch "$FETCH_CALLED_MARKER"; exit 1; fi
done
exec "$REAL_GIT" "$@"
EOF
chmod +x "$tmp_dir/build-fail-bin/git"
real_git=$(command -v git)
if PATH="$tmp_dir/build-fail-bin:$PATH" REAL_GIT="$real_git" \
    SSH_CALLED_MARKER="$tmp_dir/ssh-called" FETCH_CALLED_MARKER="$tmp_dir/fetch-called" "$script" --mode production --address api.example.com >"$tmp_dir/out" 2>&1; then
  echo 'FAIL rejected Go tests did not stop production deployment' >&2
  exit 1
fi
grep -Fq '正在测试 Go 后端' "$tmp_dir/out"
test ! -e "$tmp_dir/fetch-called"
test ! -e "$tmp_dir/ssh-called"

remote_script="$repo_dir/scripts/deploy-api-remote.sh"
mkdir -p "$tmp_dir/fakebin" "$tmp_dir/opt" "$tmp_dir/etc" "$tmp_dir/data" "$tmp_dir/systemd"
cat >"$tmp_dir/fakebin/systemctl" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$SYSTEMCTL_LOG"
if [[ $1 == restart && ${FAIL_RESTART:-0} == 1 ]]; then exit 1; fi
if [[ $1 == is-active ]]; then [[ -f "$FAKE_SERVICE_STATE/$3" ]]; exit; fi
if [[ $1 == show ]]; then printf '12345\n'; exit; fi
if [[ $1 == restart ]]; then touch "$FAKE_SERVICE_STATE/$2"; fi
if [[ $1 == stop ]]; then rm -f "$FAKE_SERVICE_STATE/$2"; fi
exit 0
EOF
cat >"$tmp_dir/fakebin/ss" <<'EOF'
#!/usr/bin/env bash
if [[ ${FOREIGN_LISTENER:-0} == 1 ]]; then
  printf 'LISTEN 0 128 127.0.0.1:%s 0.0.0.0:* users:(("foreign",pid=99999,fd=3))\n' "${*: -1}"
elif [[ " $* " == *':8082'* && -f "$FAKE_SERVICE_STATE/hotel-api-test.service" ]] ||
     [[ " $* " == *':8081'* && -f "$FAKE_SERVICE_STATE/hotel-api-production.service" ]]; then
  printf 'LISTEN 0 128 127.0.0.1:8082 0.0.0.0:* users:(("hotel-api",pid=12345,fd=3))\n'
fi
EOF
cat >"$tmp_dir/fakebin/flock" <<'EOF'
#!/usr/bin/env bash
printf '%s\n' "$*" >> "$FLOCK_LOG"
exit 0
EOF
cat >"$tmp_dir/fakebin/curl" <<'EOF'
#!/usr/bin/env bash
if [[ " $* " == *'/auth/mock-login'* ]]; then printf '%s' "${MOCK_STATUS:-404}"; else printf '%s' "${LOGIN_STATUS:-400}"; fi
EOF
chmod +x "$tmp_dir/fakebin/systemctl" "$tmp_dir/fakebin/ss" "$tmp_dir/fakebin/flock" "$tmp_dir/fakebin/curl"
export PATH="$tmp_dir/fakebin:$PATH"
export SYSTEMCTL_LOG="$tmp_dir/systemctl.log"
export FLOCK_LOG="$tmp_dir/flock.log"
export FAKE_SERVICE_STATE="$tmp_dir/service-state"
mkdir -p "$FAKE_SERVICE_STATE"
export HOTEL_API_OPT_ROOT="$tmp_dir/opt"
export HOTEL_API_ENV_ROOT="$tmp_dir/etc"
export HOTEL_API_DATA_ROOT="$tmp_dir/data"
export HOTEL_API_SYSTEMD_ROOT="$tmp_dir/systemd"
export HOTEL_API_RUN_USER="$(id -un)"

printf 'binary-v1' > "$tmp_dir/staged-v1"
if bash "$remote_script" test missing-env "$tmp_dir/staged-v1" >"$tmp_dir/out" 2>&1; then
  echo 'FAIL test deployment accepted missing MySQL configuration' >&2
  exit 1
fi
grep -Fq '缺少' "$tmp_dir/out"
cat >"$tmp_dir/etc/test.env" <<'EOF'
APP_ENV=integration
API_ADDR=0.0.0.0:8082
MYSQL_DSN=hotel_api:test-only@tcp(127.0.0.1:3306)/hotel_prod_test?parseTime=true
WECHAT_APP_ID=test-only
WECHAT_APP_SECRET=test-only
EOF
chmod 600 "$tmp_dir/etc/test.env"
bash "$remote_script" test test-v1 "$tmp_dir/staged-v1" >"$tmp_dir/out"
test "$(readlink "$tmp_dir/opt/test/current")" = "$tmp_dir/opt/test/releases/test-v1"
grep -Fq -- '-n' "$FLOCK_LOG"

sed 's/APP_ENV=integration/APP_ENV=local-demo/' "$tmp_dir/etc/test.env" >"$tmp_dir/test-bad.env"
cp "$tmp_dir/test-bad.env" "$tmp_dir/etc/test.env"
printf 'wrong-mode' > "$tmp_dir/wrong-mode-staged"
if bash "$remote_script" test wrong-mode "$tmp_dir/wrong-mode-staged" >"$tmp_dir/out" 2>&1; then
  echo 'FAIL test deployment accepted demo mode' >&2
  exit 1
fi
grep -Fq 'APP_ENV=integration' "$tmp_dir/out"
sed 's/APP_ENV=local-demo/APP_ENV=integration/' "$tmp_dir/etc/test.env" >"$tmp_dir/test-good.env"
cp "$tmp_dir/test-good.env" "$tmp_dir/etc/test.env"
sed 's@/hotel_prod_test?@/hotel_prod?@' "$tmp_dir/etc/test.env" >"$tmp_dir/test-bad.env"
cp "$tmp_dir/test-bad.env" "$tmp_dir/etc/test.env"
printf 'wrong-db' > "$tmp_dir/wrong-db-staged"
if bash "$remote_script" test wrong-db "$tmp_dir/wrong-db-staged" >"$tmp_dir/out" 2>&1; then
  echo 'FAIL test deployment accepted production database' >&2
  exit 1
fi
grep -Fq 'hotel_prod_test' "$tmp_dir/out"
sed 's@/hotel_prod?@/hotel_prod_test?@' "$tmp_dir/etc/test.env" >"$tmp_dir/test-good.env"
cp "$tmp_dir/test-good.env" "$tmp_dir/etc/test.env"

printf 'mock-test' > "$tmp_dir/mock-test-staged"
if MOCK_STATUS=400 bash "$remote_script" test mock-test "$tmp_dir/mock-test-staged" >"$tmp_dir/out" 2>&1; then
  echo 'FAIL demo authentication was accepted in test' >&2
  exit 1
fi
test "$(readlink "$tmp_dir/opt/test/current")" = "$tmp_dir/opt/test/releases/test-v1"

printf 'foreign-v1' > "$tmp_dir/foreign-staged-v1"
if FOREIGN_LISTENER=1 bash "$remote_script" test foreign-v1 "$tmp_dir/foreign-staged-v1" >"$tmp_dir/out" 2>&1; then
  echo 'FAIL port owned by another process was accepted' >&2
  exit 1
fi
grep -Fq '端口 8082 已被其他进程占用' "$tmp_dir/out"
test "$(readlink "$tmp_dir/opt/test/current")" = "$tmp_dir/opt/test/releases/test-v1"

printf 'html-v3' > "$tmp_dir/staged-v3"
if LOGIN_STATUS=200 bash "$remote_script" test test-v3 "$tmp_dir/staged-v3" >"$tmp_dir/out" 2>&1; then
  echo 'FAIL unrelated HTTP 200 page was accepted as API readiness' >&2
  exit 1
fi
test "$(readlink "$tmp_dir/opt/test/current")" = "$tmp_dir/opt/test/releases/test-v1"
grep -Fq 'API_ADDR=0.0.0.0:8082' "$tmp_dir/etc/test.env"
grep -Fq 'hotel-api-test.service' "$SYSTEMCTL_LOG"

printf 'binary-v2' > "$tmp_dir/staged-v2"
if FAIL_RESTART=1 bash "$remote_script" test test-v2 "$tmp_dir/staged-v2" >"$tmp_dir/out" 2>&1; then
  echo 'FAIL restart failure was accepted' >&2
  exit 1
fi
test "$(readlink "$tmp_dir/opt/test/current")" = "$tmp_dir/opt/test/releases/test-v1"

cat >"$tmp_dir/etc/production.env" <<'EOF'
APP_ENV=production
API_ADDR=0.0.0.0:8081
MYSQL_DSN=hotel_api:test-only@tcp(127.0.0.1:3306)/hotel_prod?parseTime=true
WECHAT_APP_ID=test-only
WECHAT_APP_SECRET=test-only
EOF
chmod 600 "$tmp_dir/etc/production.env"
printf 'production-v1' > "$tmp_dir/prod-staged-v1"
bash "$remote_script" production prod-v1 "$tmp_dir/prod-staged-v1" >"$tmp_dir/out"
test "$(readlink "$tmp_dir/opt/production/current")" = "$tmp_dir/opt/production/releases/prod-v1"
grep -Fq 'hotel-api-production.service' "$SYSTEMCTL_LOG"

printf 'production-v2' > "$tmp_dir/prod-staged-v2"
if MOCK_STATUS=400 bash "$remote_script" production prod-v2 "$tmp_dir/prod-staged-v2" >"$tmp_dir/out" 2>&1; then
  echo 'FAIL demo authentication was accepted in production' >&2
  exit 1
fi
test "$(readlink "$tmp_dir/opt/production/current")" = "$tmp_dir/opt/production/releases/prod-v1"

echo 'deploy-api remote tests passed'
