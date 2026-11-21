# 后端部署：Mac 一条命令

脚本位置：[deploy-api.sh](../scripts/deploy-api.sh)。它只部署 Go 后端；不修改宝塔、Nginx、微信小程序或工作台。

## 两种模式

| 模式 | 代码来源 | ECS 服务 | 监听 | 数据 |
| --- | --- | --- | --- | --- |
| 测试 `test` | Mac 当前 `hotel` 目录，包含未提交文件 | `hotel-api-test` | `0.0.0.0:8082` | `/var/lib/hotel-api/test`，默认本地演示数据 |
| 线上 `production` | 执行时从 GitHub 获取的 `origin/main` | `hotel-api-production` | `127.0.0.1:8081` | 单独的线上 MySQL 数据库 |

脚本使用 `root@39.105.122.224:22`。Mac 需装好 Git、Go、SSH、SCP、curl，并能从终端 SSH 登录这台 ECS；ECS 需有 `flock` 和 `ss` 命令。首次连接会要求确认服务器指纹；使用密码登录时，上传和部署可能各提示一次密码。脚本不保存 SSH 密码。

## 先部署测试版

1. 在阿里云 ECS 安全组中，给 **8082/TCP** 添加一条只允许你当前公网 IP 访问的入方向规则。测试服务保留演示登录，不要把 8082 对全网开放。ECS 内若启用了防火墙，还需允许相同来源访问 8082。
2. 在 Mac 终端进入项目根目录：

   ```bash
   cd "/Users/huruiqi/Documents/民宿小程序/hotel"
   ```

3. 可先预演参数；预演不编译、不连接 ECS：

   ```bash
   ./scripts/deploy-api.sh --mode test --address 39.105.122.224:8082 --dry-run
   ```

4. 正式执行测试部署：

   ```bash
   ./scripts/deploy-api.sh --mode test --address 39.105.122.224:8082
   ```

成功时末行包含 `部署完成：test` 和 `http://39.105.122.224:8082/api/v1`。脚本会依次运行 Go 测试、编译 Linux x86_64 程序、上传、更新 `hotel-api-test` 服务、检查接口。首次部署会自动创建 `/etc/hotel-api/test.env`，默认使用 `APP_ENV=local-demo` 和独立 JSON 数据目录。以后如需测试真实微信/MySQL 集成，可单独改此文件为 `APP_ENV=integration`，并提供**独立测试库**和微信配置，不能共用线上库。

## 域名就绪后的线上版

前提：生产代码已合入 GitHub `main`，线上 MySQL 库及表结构已准备，备案域名已解析到 ECS，并配置了 HTTPS。当前 GitHub `main` 仍是演示版，**现在运行线上模式会明确停止**，不会把演示版作为线上服务启动。

线上凭据只保存在 ECS 的 `/etc/hotel-api/production.env`。该文件需由 root 创建，权限为 `600`，至少包含：

```text
APP_ENV=production
API_ADDR=127.0.0.1:8081
MYSQL_DSN=<线上数据库连接串>
WECHAT_APP_ID=<小程序 AppID>
WECHAT_APP_SECRET=<小程序 AppSecret>
```

数据库迁移需在首次上线或结构变化时**单独执行并先备份数据库**；部署脚本不会每次自动修改数据库。不要把真实凭据放入 Git、聊天消息或部署命令。

在宝塔现有 Nginx 里做**一次**域名 HTTPS 转发：把线上域名下的 `/api/v1/` 转到 `http://127.0.0.1:8081`。后续重复部署不必改 Nginx。修改 Nginx 前先备份配置。域名可用后运行：

```bash
./scripts/deploy-api.sh --mode production --address api.example.com
```

把 `api.example.com` 替换成真实的 API 域名，不要加 `https://` 或路径。线上模式不接受 IP 地址，因为后端只监听 ECS 本机，外部访问由 HTTPS Nginx 提供。若测试环境也有独立 HTTPS 域名，可以运行 `--mode test --address test-api.example.com`；Nginx 需事先把该域名转到 `127.0.0.1:8082`。

## 失败时看哪里

- Go 测试、编译或 SSH 失败：脚本停止，ECS 上正在运行的版本不变。
- 新程序在 ECS 启动失败，或线上版暴露演示登录接口：脚本恢复该环境的上一版，不动另一套服务。
- 程序已在 ECS 启动，但外部地址检查失败：程序保持运行；检查阿里云安全组、DNS、HTTPS 证书、宝塔 Nginx 转发。脚本会明确提示“服务已启动，但外部地址无法连接”。
- 8081 或 8082 已被其他程序占用，或同一模式已有部署正在运行：脚本停止，不替换当前服务。
- 在 ECS 上查看服务状态：`systemctl status hotel-api-test` 或 `systemctl status hotel-api-production`；查看最近日志：`journalctl -u hotel-api-test -n 50 --no-pager`。

脚本的版本文件放在 `/opt/hotel-api/<模式>/releases/`，`current` 指向正在运行的版本。重复执行只更新所选模式，不会自动推送 Git，也不会覆盖另一模式的数据。
