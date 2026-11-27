# 后端部署：Mac 一条命令

脚本位置：[deploy-api.sh](../scripts/deploy-api.sh)。它只部署 Go 后端；不修改宝塔、Nginx、微信小程序或工作台。

## 两种模式

| 模式 | 代码来源 | ECS 服务 | 监听 | 数据 |
| --- | --- | --- | --- | --- |
| 测试 `test` | Mac 当前 `hotel` 目录，包含未提交文件 | `hotel-api-test` | `0.0.0.0:8082` | MySQL `hotel_prod_test`，`APP_ENV=integration` |
| 线上 `production` | 本地 `main` 分支的已提交版本；不连接 GitHub | `hotel-api-production` | `0.0.0.0:8081` | MySQL `hotel_prod`，`APP_ENV=production` |

脚本使用 `root@39.105.122.224:22`。Mac 需装好 Git、Go、SSH、SCP、curl，并能从终端 SSH 登录这台 ECS；ECS 需有 `flock` 和 `ss` 命令。首次连接会要求确认服务器指纹；使用密码登录时，上传和部署可能各提示一次密码。脚本不保存 SSH 密码。

## 先部署测试版

1. 在宝塔创建独立测试库 `hotel_prod_test` 并执行与线上库相同的建表 SQL。在 ECS 创建 `/etc/hotel-api/test.env`，权限为 `600`，内容如下（填入真实值；数据库密码不要提交到 Git）：

   ```text
   APP_ENV=integration
   API_ADDR=0.0.0.0:8082
   MYSQL_DSN=hotel_test_user:测试库密码@tcp(127.0.0.1:3306)/hotel_prod_test?parseTime=true
   WECHAT_APP_ID=小程序AppID
   WECHAT_APP_SECRET=小程序AppSecret
   ```

   测试环境使用真实微信登录和独立 MySQL，不提供模拟登录。若此前已有 `APP_ENV=local-demo` 的 `test.env`，须改为上述配置。测试库账号只授权 `hotel_prod_test`，不能共用线上库账号。
2. 在阿里云 ECS 安全组中，给 **8082/TCP** 添加一条只允许你当前公网 IP 访问的入方向规则。不要把 8082 对全网开放。ECS 内若启用了防火墙，还需允许相同来源访问 8082。
3. 在 Mac 终端进入项目根目录：

   ```bash
   cd "/Users/huruiqi/Documents/民宿小程序/hotel"
   ```

4. 可先预演参数；预演不编译、不连接 ECS：

   ```bash
   ./scripts/deploy-api.sh --mode test --address 39.105.122.224:8082 --dry-run
   ```

5. 正式执行测试部署：

   ```bash
   ./scripts/deploy-api.sh --mode test --address 39.105.122.224:8082
   ```

成功时末行包含 `部署完成：test` 和 `http://39.105.122.224:8082/api/v1`。脚本会依次运行 Go 测试、编译 Linux x86_64 程序、上传、更新 `hotel-api-test` 服务、检查接口。`test.env` 缺失、数据库名不是 `hotel_prod_test`、仍使用 `local-demo`，或模拟登录接口存在时，部署会失败。公网 IP 的 HTTP 地址仅供受限来源验收，不要用于正式小程序。

## 部署线上版：先用 IP，域名就绪后改用域名

前提：生产代码已提交到本地 `main` 分支，线上 MySQL 库及表结构已准备。当前仓库没有 `master` 分支；脚本直接打包本地 `main` 的提交，不切换分支、不读取工作区未提交内容，也不访问 GitHub。脚本会检查提交是否包含生产模式和真实登录实现；生产环境还需事先手动创建以下私密配置文件。

线上凭据只保存在 ECS 的 `/etc/hotel-api/production.env`。该文件需由 root 创建，权限为 `600`，至少包含：

```text
APP_ENV=production
API_ADDR=0.0.0.0:8081
MYSQL_DSN=hotel_api:线上库密码@tcp(127.0.0.1:3306)/hotel_prod?parseTime=true
WECHAT_APP_ID=<小程序 AppID>
WECHAT_APP_SECRET=<小程序 AppSecret>
```

数据库迁移需在首次上线或结构变化时**单独执行并先备份数据库**；部署脚本不会每次自动修改数据库。不要把真实凭据放入 Git、聊天消息或部署命令。
`MYSQL_DSN` 中的密码是宝塔创建 `hotel_api` 数据库账号时保存的密码，不是宝塔面板密码或 MySQL root 密码。编辑的是 ECS 上的 `/etc/hotel-api/production.env`，不是 Mac 的代码文件。

备案和域名 HTTPS 尚未就绪时，先确认阿里云安全组已放行 **8081/TCP**。ECS 的 `firewalld` 若正在运行，还要在 ECS 终端一次性执行：

```bash
firewall-cmd --permanent --zone=public --add-port=8081/tcp
firewall-cmd --reload
firewall-cmd --zone=public --query-port=8081/tcp
```

最后一条应输出 `yes`。部署脚本不会修改阿里云安全组或 ECS 防火墙。网络规则准备好后，在 Mac 终端运行：

```bash
./scripts/deploy-api.sh --mode production --address 39.105.122.224:8081
```

这个地址使用明文 HTTP；尚未配置 HTTPS 时，只用它检查部署是否连通，不在上面输入真实账号密码。若现阶段确需让任何公网客户端连接 8081，安全组来源可设为 `0.0.0.0/0`；这会向所有公网 IP 开放该端口。

域名就绪后，在宝塔现有 Nginx 里做**一次** HTTPS 转发：把线上域名下的 `/api/v1/` 转到 `http://127.0.0.1:8081`。后续重复部署不必改 Nginx。修改 Nginx 前先备份配置。此时关闭安全组的公网 8081 入站规则；ECS 上执行 `firewall-cmd --permanent --zone=public --remove-port=8081/tcp && firewall-cmd --reload`，之后运行：

```bash
./scripts/deploy-api.sh --mode production --address api.example.com
```

把 `api.example.com` 替换成真实的 API 域名，不要加 `https://` 或路径。`--address` 只决定外部检查地址，IP 用 HTTP，域名用 HTTPS；两种方式都使用同一个线上服务和数据库。若测试环境也有独立 HTTPS 域名，可以运行 `--mode test --address test-api.example.com`；Nginx 需事先把该域名转到 `127.0.0.1:8082`。

## 失败时看哪里

- Go 测试、编译或 SSH 失败：脚本停止，ECS 上正在运行的版本不变。
- 新程序在 ECS 启动失败，或线上版暴露演示登录接口：脚本恢复该环境的上一版，不动另一套服务。
- 程序已在 ECS 启动，但外部地址检查失败：程序保持运行。IP 地址检查 ECS 的 `firewalld` 和阿里云安全组；域名检查 DNS、HTTPS 证书、宝塔 Nginx 转发与 443 端口。脚本会明确提示“服务已启动，但外部地址无法连接”。
- 8081 或 8082 已被其他程序占用，或同一模式已有部署正在运行：脚本停止，不替换当前服务。
- 在 ECS 上查看服务状态：`systemctl status hotel-api-test` 或 `systemctl status hotel-api-production`；查看最近日志：`journalctl -u hotel-api-test -n 50 --no-pager`。

脚本的版本文件放在 `/opt/hotel-api/<模式>/releases/`，`current` 指向正在运行的版本。重复执行只更新所选模式，不会自动推送 Git，也不会覆盖另一模式的数据。
