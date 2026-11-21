# 第二阶段验收执行记录（2026-11-17）

依据：[第二阶段验收用例](phase2-acceptance-cases.md)。本次没有修改业务代码、重置数据、重启服务或执行备份恢复。运行证据只代表 2026-11-17 当前可访问的环境；不把单元测试、静态核对或 `local-demo` 当成真实微信/MySQL 集成通过。

2026-11-18 复测：同一 `8082` 地址再次 `curl`，退出码 7，约 18 ms 后报告 `Couldn't connect to server`；仍未取得 HTTP 响应，以下阻塞状态不变。

## 环境门槛

| 操作 | 预期 | 实际与证据 | 判定 |
| --- | --- | --- | --- |
| 探测本机 `127.0.0.1:8080` 的 `POST /api/v1/auth/wx-login`（空 JSON，无真实 code） | integration 路由存在 | HTTP 404；同机 `GET /api/v1/admin/demo/state` 为 HTTP 401，而非 404，表明演示路由存在；与 `apps/api/internal/app/router.go` 中条件注册相符 | 本机是演示模式，不能用作第二阶段环境 |
| 连接部署文档所列测试地址 `http://39.105.122.224:8082/api/v1/admin/demo/state` | 可连接独立测试服务并进一步核对模式 | `curl` 退出码 7：`Failed to connect ... port 8082`；无 HTTP 响应 | 阻塞 |
| 用现有 SSH 公钥访问该测试 ECS | 能只读核对服务环境 | `Permission denied (publickey,gssapi-keyex,gssapi-with-mic,password)`；没有权限核对 `APP_ENV`、MySQL、版本或备份 | 阻塞 |
| 确认独立 HTTPS API 域名、后台地址、真实 AppID 及测试 MySQL | 环境配置已核验 | 仓库未记录已确认的测试 HTTPS 域名；`docs/deployment-api.md` 指出 test 默认 `local-demo`/JSON，需单独改为 integration；目前无法核验实际远端配置 | 阻塞 |

## 用例执行状态

| 用例 | 操作范围 | 实际 | 判定 |
| --- | --- | --- | --- |
| P2-01 | 独立环境、路由、AppID/库边界 | 本机只证实是演示模式；远端连不上 | 阻塞 |
| P2-02 至 P2-04 | 真机真实微信登录、手机号授权、入会/券、重复入会 | 没有可达 integration API 和目标体验版真机证据；未尝试伪造微信 code | 阻塞 |
| P2-05 至 P2-09 | 两晚预订、后台确认/入住/离店、返住金、券核销、看板 | 未向演示或未知远端写入业务数据；无法取真实订单/会员 ID 与前后计数 | 阻塞 |
| P2-10 至 P2-16 | 拒绝授权、越权、幂等、重复核销/离店、脱敏与注销 | 无可用集成环境和两名受控测试会员/员工角色；仅完成下方代码静态核对 | 阻塞 |
| P2-17 至 P2-20 | API/MySQL 重启、备份、隔离库恢复及环境复核 | 未获独立测试库和操作权限，未执行任何重启、备份或恢复 | 阻塞 |

## 可执行的本地验证（不计入集成验收）

| 操作 | 结果 | 证据界限 |
| --- | --- | --- |
| `GOCACHE=/tmp/hotel-go-build go test ./...`（`apps/api`） | 退出码 0，`internal/app` 通过 | 测试使用替身或本地存储；不证明真实微信、MySQL、远端部署或恢复。原命令因沙箱无权写默认 Go 缓存失败，改用 `/tmp` 后通过。 |
| `pnpm --dir apps/miniapp test` | 26/26 通过 | 不证明体验版真机登录、手机号授权或实际页面链路。 |
| `pnpm --dir apps/admin-web test` | 6/6 通过 | 不证明后台已连接 integration 或角色操作可用。 |

## 静态预检发现（待集成复现，不冒充运行失败）

| 对应用例 | 发现与最短复现条件 | 位置 | 归属 |
| --- | --- | --- | --- |
| P2-05、P2-07 | 两晚订单按单晚价格创建；创建时已将参考价写入 `actual_amount_cent`。在可用环境用“大床房”预订两晚，检查参考价和实收字段即可复现，预期分别为 16000 分与未填写。 | `apps/api/internal/app/service.go` 的 `CreateBooking`；`apps/api/internal/app/models.go` 的 `BookingOrder` | 后端 AI |
| P2-06 | 现有后台路由只有订单列表和状态变更，没有设计要求的受控 `GET /admin/bookings/{order_id}/contact`。在 integration 用前台身份请求该路径，预期单笔电话+审计，当前路由预计 404。 | `apps/api/internal/app/router.go` 的 `adminAuthed` 订单路由；`docs/superpowers/specs/2026-11-17-wechat-offline-booking-design.md` 第 5.3 节 | 后端 AI + 管理后台 AI（同一后台代码仓也可交后端 AI） |
| P2-13 | `POST /bookings` 没有读取或传递幂等标识，`CreateBooking` 每次追加订单；相同请求重放预计生成两个 ID。需在独立库用同一会员和 key 实际验证计数。 | `apps/api/internal/app/router.go` 的订房路由；`apps/api/internal/app/service.go` 的 `CreateBooking` | 后端 AI；小程序 AI 补前端防连点 |
| P2-14 | 离店请求模型没有 `offline_payment_confirmed` 和幂等标识；现有服务在状态改为离店时直接标已付并发权益，若未传正数实收则沿用建单时的估算额。可在隔离测试库构造未确认收款的订单再离店复现。 | `apps/api/internal/app/models.go` 的 `UpdateBookingStatusRequest`；`apps/api/internal/app/service.go` 的 `UpdateBookingStatus`；`apps/admin-web/src/app.js` 的状态操作 | 后端 AI + 管理后台 AI（或负责后台的后端 AI） |

以上四项与新设计文档一致，但新设计仍是待交付要求；当前 `docs/contracts/openapi.yaml` 也尚未收录联系电话单笔读取、幂等键和结清确认的新契约。应先由负责 API 的 AI 同步契约与实现，再做集成环境失败复现及回归。当前结论是**第二阶段未验收，不能放行**，不是“所有链路均已运行失败”。

## 继续执行的最少前置条件

1. 提供独立、可从本机访问的 integration 测试 HTTPS API 和管理后台地址，并只读确认 `APP_ENV=integration`、独立 MySQL、目标 AppID、部署版本；不要在聊天中发送密码或 AppSecret。
2. 给测试用的老板/前台/财务账号及两名受控微信号准备授权路径；真机 `wx.login`/手机号授权必须由账号持有人在体验版操作一次，提供脱敏截图和对应时间/会员 ID。其余接口与后台核对可由我继续执行。
3. 若要执行 P2-17 至 P2-20，授权仅针对独立测试服务和测试库的重启、只读核验、备份及恢复到**新隔离库**；先确认不会影响生产或现有业务数据。
