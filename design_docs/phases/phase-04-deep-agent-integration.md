# Phase 04 - Deep Agent Integration

## 目标

让 Deep Agent 通过 Skill / HTTP 安全接入已经成形的 Control Layer，但不把任何控制权下放给 Agent。

## 范围

1. 固化两个最小 skill：
   - `request_mysql_database_create`
   - `execute_assistant_order`
2. Deep Agent 到 Control API 的输入输出映射。
3. Agent 侧最小编排规则：
   - `approval_required=false` 且 `status=APPROVED` 时允许自动串联 execute
   - `approval_required=true` 时只返回等待审批
4. auth context 与 `principal_id` 的可信传递。
5. 面向用户的消息模板、错误解释与状态反馈。

## 禁止事项

1. 不让 Deep Agent 直接调用 Adapter。
2. 不让 Deep Agent 接触真实凭证或 connection ref。
3. 不把 Policy / Risk / Approval 逻辑搬到 Agent 侧。
4. 不在审批通过后默认自动执行。
5. 不把自然语言 alias 直接送入核心 `AssetResolver`。

## 产物

1. Skill contract 与 HTTP 映射实现。
2. Deep Agent 最小调用示例。
3. request / execute 两次独立调用的演示链路。
4. 常见失败路径的用户消息模板。

## 验收标准

1. Agent 能成功调用 `request_mysql_database_create`。
2. dev/test 无审批场景下，Agent 可自动串联 execute，但控制层仍体现为两次独立调用。
3. prod 场景下，Agent 只返回等待审批，不会偷跑 execute。
4. `order_id`、`task_id`、`trace_id` 可在会话层安全引用。
5. Agent 接入没有改变任何 Control Layer 授权、审批、审计语义。

## 风险点

1. Agent 自动串联 execute 可能与重试机制叠加，造成重复触发。
2. auth context 在 skill 层处理不严，容易出现会话身份与执行身份不一致。
3. 用户消息模板若过度简化，可能掩盖 `APPROVAL_REQUIRED`、`PLAN_STALE` 等关键状态。

## 进入下一阶段条件

1. Deep Agent 已明确作为 northbound client，而不是事实上的控制器。
2. request / execute 分离语义在 Agent 接入后仍保持清晰。
3. 自动串联 execute 只发生在正式允许的低风险路径。

## 推荐 branch 名

`phase/04-deep-agent-minimal`

## 推荐 commit message 模式

1. `feat(skill): add deep agent skill mappings`
2. `feat(agent): add request then execute orchestration rules`
3. `docs(agent): add deep agent integration examples`
4. `test(agent): verify approval and no-approval control paths`

