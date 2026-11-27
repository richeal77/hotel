# M1 房型报价与库存独立验收（2026-11-26）

**结论：不允许 M1 签收。** 本次只完成冻结契约校验、代码静态核对、三端本地测试和 JSON 模式后端 API 测试；没有独立 MySQL 空库、已部署的三端联调环境或小程序真机预览证据。未测试项不作通过结论。未修改业务代码、冻结契约或数据库。

## 版本与执行证据

| 端 | 代码版本/状态 | 实际执行 | 结果与界限 |
| --- | --- | --- | --- |
| 契约 | `0.4.0-draft`；OpenAPI SHA-256 `ddcfb4aca96059ff725a47facf4acb4defdceff43ab84de50b61f3cdc67912b5`；样例 SHA-256 `2d9f710b71e22f663e8abeb7ad86ad96dd793e5dae631c08cafe8f5779b7a8a0`，均与冻结记录一致 | `ruby docs/contracts/validate-room-benefits.rb` | 通过：60 paths，12 examples；这是契约结构校验。 |
| 后端 | HEAD `98321ed` 加未提交的 `apps/api` 改动及新增 `configured*.go`、迁移 002；尚无可指认的交付 commit | `GOCACHE=/tmp/hotel-m1-go-build GOFLAGS=-count=1 go test ./... -v`；`go test -race -count=1 ./internal/app -run 'TestConfiguredBooking' -v` | 两命令退出 0。`TestConfiguredMySQLFirstInitialization` 因缺 `API_TEST_MYSQL_DSN` **SKIP**，不能计通过。 |
| 管理端 | HEAD `98321ed` 加未提交的 `apps/admin-web` 改动和 `operations.js`；尚无交付 commit | `pnpm --dir apps/admin-web test` | 11/11 通过；含接口映射、409 提示与构建测试，未见浏览器三端联调。 |
| 小程序 | HEAD `98321ed` 加未提交的 `apps/miniapp` 改动和 `booking-flow.js`；尚无交付 commit | `pnpm --dir apps/miniapp test` | 28/28 通过；含报价/提交字段及过期冲突逻辑，未见微信体验版真机预览。 |

## M1 用例判定

| 用例 | 证据 | 判定 |
| --- | --- | --- |
| 后台配置房型、基础房价与默认库存 | 后端 `TestConfiguredBookingCancellationReleasesEachNight` 通过；测试创建房型和 7900 分价格，默认库存 1 | JSON 后端 API 通过；管理端页面未测 |
| 两晚报价、订一间、后台查单、取消后两晚各归还一间 | 同一测试报价 15800 分，提交后两晚 `reserved_quantity=1`，重开 JSON 仓库仍能查订单，取消后均为 0 | JSON 后端 API 通过；三端端到端未测 |
| 相同/不同幂等键重复取消 | 同一测试分别以 key 8、8、9 重放，库存维持 0 | JSON 后端 API 通过 |
| 任一晚关房、报价后关房 | `TestConfiguredBookingRejectsRoomClosedAfterQuote` 通过，提交得到 409 `ROOM_CLOSED`，两晚未占库存 | JSON 后端 API 通过；管理端页面未测 |
| 最后一间并发 | `TestConfiguredBookingConcurrentLastRoom` 两个独立报价并发提交，断言一笔 200、一笔 409，库存只占 1；`-race` 通过 | JSON 单进程通过；MySQL 多实例未测 |
| 报价到期、改价 | 10 分钟时提交获 409 `QUOTE_EXPIRED`；未发现独立的改价后重报价格端到端用例 | 到期通过；改价未测 |
| 开关关闭不退回旧入口 | 取消后将 `CONFIGURED_BOOKING_ENABLED=false`，以 integration 模式路由测试旧 `/bookings`、`/members/join` 和新提交均 409 `FEATURE_DISABLED`；旧正式订单仍可查/取消 | API 模拟模式通过；实际 integration 部署未测 |
| no-show 12:59/13:00、补跑 | `TestConfiguredNoShowReleasesAtDeadlineOnce` 在 12:59 不释放，13:00 连续运行两次后两晚归还；入住与 no-show 冲突测试通过 | JSON 后端 API 通过；真实调度/进程重启补跑未测 |
| JSON demo / 隔离 MySQL 一致性 | JSON 测试通过；MySQL 初始化测试跳过 | **未测/阻断** |

上述后端 API 证据来自仓库内 `httptest` 用例（`apps/api/internal/app/configured_test.go`），由我独立执行并核对断言；它不等于浏览器/小程序/真实数据库的三端联调。

## 契约与架构差异

| 差异 | 级别 | 复现/证据 | 修订建议与责任 |
| --- | --- | --- | --- |
| 资源不存在返回 404 `NOT_FOUND`，冻结契约未在所有适用资源操作逐项声明 | M1 契约偏差，集成前修订 | 对不存在房型/价格/订单 ID 请求；后端响应与 OpenAPI 各路径响应定义比对。当前只做静态核对，运行矩阵未测 | 契约负责人下一版逐项补 404，三端重新确认；后端补响应一致性测试 |
| 库存 GET 实现要求 `from`、`to` 同时存在，冻结契约未标必填 | M1 客户端兼容风险 | 省略任一查询参数请求 `/admin/room-types/{id}/inventory`，预期实现拒绝；当前仅静态核对 | 契约负责人在下一版标 `required: true`；管理端保持同时传参 |
| 报价有效期 10 分钟、日期跨度最多一年在实现中生效，但冻结契约未声明 | M1 边界行为未冻结 | 10 分钟到期已有 API 测试；一年跨度尚未单独验证 | 下一版写明 TTL、时间边界与最大跨度并加边界测试；后端/契约负责人 |
| 迁移 002 将 M1 状态放进 `configured_booking_store` 单行 JSON，并用全局 `store_revision` 锁；报价单独表 | **生产架构阻断项**，不影响本地 JSON 功能结论 | `apps/api/internal/app/migrations/002_configured_booking.sql` 和 `apps/api/internal/app/mysql_repository.go`；与拆分房型/房价/库存/订单表的原方案不同 | 架构/总控评估写放大、锁竞争、索引查询、备份恢复和兼容回滚，决定限规模接受或要求规范化迁移；验收 AI 不批准生产 |

## 复测前置

1. 三端各提供可定位的交付 commit 或冻结工作树快照，及管理端页面操作、小程序体验版截图/录屏；当前单元测试通过不能代替交付。
2. 提供**一次性空库**的 `API_TEST_MYSQL_DSN`（经安全渠道注入环境，不贴进聊天或仓库），确认允许迁移和写入；再跑首次初始化、双晚库存、并发、重启和 JSON/MySQL 对比。不能指向现有验收库或生产库。
3. 提供独立 integration HTTPS API、后台地址、测试 AppID、受控员工/会员身份和开关状态；执行实际后台→小程序→后台→取消链路及改价、关房、no-show 调度补跑。真机需账号持有人授权。

**复现责任端**：两晚库存、幂等、并发、no-show 与 MySQL 是后端；后台页面配置和订单查询是管理端；报价/下单/取消、开关关闭提示是真机小程序；契约差异由契约负责人协调三端，迁移架构由总控/架构负责人审批。M2 权益发放不在本次 M1 签收范围。
