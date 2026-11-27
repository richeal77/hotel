# 管理端部署

在 Mac 的 `hotel` 目录执行一条命令。脚本会测试、构建、上传管理端静态文件，在 ECS 上首次创建独立的宝塔 Nginx 配置，以后只切换静态文件版本，并检查网页和 API 转发。

| 模式 | 源码 | 管理端端口 | 转发后端 |
| --- | --- | --- | --- |
| `production` | 本地 `main` 已提交版本 | `8083` | `127.0.0.1:8081` |
| `test` | Mac 当前工作区 | `8084` | `127.0.0.1:8082` |

先用 IP 验收生产管理端：

```bash
cd "/Users/huruiqi/Documents/民宿小程序/hotel"
./scripts/deploy-admin-web.sh --mode production --address 39.105.122.224:8083
```

测试环境后端和数据库配置好后，可执行：

```bash
./scripts/deploy-admin-web.sh --mode test --address 39.105.122.224:8084
```

脚本使用 `root@39.105.122.224:22`，可能分别为上传和远端部署提示 SSH 密码。ECS 上 `firewalld` 正在运行时，脚本会为所选 IP 模式端口添加临时和永久放行。**阿里云安全组仍需一次性添加对应端口的 TCP 入方向规则**；脚本没有阿里云账号凭据，无法修改安全组。

IP 端口使用明文 HTTP，只用来确认网页可打开和 API 转发正常。管理员输入真实密码前，应完成备案和 HTTPS。域名及证书配置好后，宝塔 Nginx 的域名站点需将 `/` 指向 `/opt/hotel-admin/production/current`，将 `/api/v1/` 转发到 `http://127.0.0.1:8081`。后续可用域名验收：

```bash
./scripts/deploy-admin-web.sh --mode production --address admin.example.com
```

域名模式只上传管理端并验证域名，不代替证书和域名站点配置。域名可用后关闭公网 8083 入站规则。管理端当前还没有员工账号管理页面/API；`boss` 账号已可登录现有功能，其他账号仍需通过 ECS 上的 `cmd/staff` 命令维护。
