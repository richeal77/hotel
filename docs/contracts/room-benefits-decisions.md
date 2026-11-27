# D1-D7 Decision Register

Contract: `0.4.0-draft`; source: `docs/房型价格与优惠活动配置PRD.md` sections 2 and 8, and `docs/superpowers/plans/2026-11-25-room-pricing-promotions.md`. "门店已答复" records the PRD's reported answer, not a new production sign-off. Test data and launch configuration still require separate approval.

| ID | 门店确认结果（按 PRD 记录） | 联调测试口径 | 上线前状态 |
| --- | --- | --- | --- |
| D1 | 一单一房型一间、可连住；日库存缺省继承房型默认，特惠与标准独立；下单占量，取消或未入住释放 | 两晚各占一间；关任一晚不可订；最后一间并发只成一单；重复取消不二次释放 | 规则已答复；真实房型、价格、默认房量待门店签字 |
| D2 | 新人礼主动领取一次，默认一项，可配项数或全部；微信及验证手机号各限一次；名额和期限按注册时快照，默认七天 | 注册只建资格；同手机号/微信并发只一份；第七天截止前可领，截止后 409；改版不改既有资格 | 规则已答复；活动配置待签字 |
| D3 | 仅自营订单标记已入住时可选多项；库存不足跳过该项但入住继续；实物短缺线下处理 | 入住选两项、一项售罄，返回 skipped_campaigns；订单仍入住；重复操作不重发 | 规则已答复；实物名单及库存待签字 |
| D4 | 券与实物可用于新人、入住、推荐；推荐双方独立配置；返住金固定额，入住发、后续单抵扣；首版仅新规则 | 158 元房费减 10 元券和 5 元旧返住金，应付 143 元；本单新发 3 元不能抵本单；双侧奖励分别开关 | 规则已答复；超额抵扣风险试运行观察，金额待签字 |
| D5 | 固定截止或相对天数；下单按入住日查有效；已领规则保持；次日 13:00 判未入住；迟到/订错线下处理 | 12:59 不转、13:00 仅转一次并释放占用；已入住不转；到期时刻不可用 | 规则已答复；门店时区按 Asia/Shanghai |
| D6 | 首次上线不迁移历史权益；首版明细及订单、权益流水导出；推荐关系有据可查 | 空生产库初始化；列表/CSV 同筛选同数量；手机号脱敏；导出记审计 | 规则已答复；试运行字段可调整，生产数据范围待审批 |
| D7 | 先做后台全量读写日志 | 登录、读列表、失败写、幂等重放、读日志自身逐次留痕；请求体不含密钥/全手机号 | 仅范围已答复；留存、导出、小程序侧、故障时是否 503 为试运行待办，不得当成生产定稿 |

## Contract Notes

- Date-only fields use `Asia/Shanghai`; stay interval is `[checkin_date, checkout_date)`. Timestamps are RFC3339 UTC. Money is integer cents.
- Omitted optional fields mean "leave unchanged" for PATCH. `sellable_quantity: null` means inherit the room default; `0` means explicit zero. `entitlement: null` means no current entitlement. Grant amount/quantity fields are required but nullable when inapplicable.
- All new writes require UUID `Idempotency-Key`; same key and normalized request replay the original success. A different body returns 409 `IDEMPOTENCY_CONFLICT`. Version mismatch returns 409 `STALE_VERSION` with `current_version`.
- New operations in OpenAPI are `planned`; old `/bookings/my` and `/admin/bookings` response expansion, server implementation and acceptance are not complete. Do not treat this draft as deployed.
