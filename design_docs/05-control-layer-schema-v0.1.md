# Assistant Control Layer Schema v0.1

## 0. 文档定位

本文档在以下正式文档基础上展开：

- `00-project-context.md`
- `01-glossary.md`
- `03-assistant-spec-v0.7.md`
- `04-interface-design-v0.8.md`

本文档只回答一件事：

> Assistant Control Layer 作为企业控制后端，内部到底有哪些核心对象、这些对象分别承担什么职责、它们如何串成一条受控执行链路。

本文档是**面向实现的对象模型文档**，不是代码生成说明书，也不是数据库建表脚本。

---

## 1. 设计目标

Control Layer 必须做到以下几点：

1. 不让 Deep Agent 直接拿到底层执行权。
2. 统一承接动作、资产、权限、风险、审批、执行、审计、证据。
3. 让同一个 Action 同时兼容：
   - MCP 执行路径
   - DB-native / CRD / gRPC / K8s / Shell 等自研执行路径
4. 让执行过程天然可审计、可追踪、可复核。
5. 让高风险动作天然具备审批与显式 execute 触发边界。

---

## 2. Control Layer 的最小职责边界

Control Layer 必须承担：

- Action 标准化
- Principal 识别
- Asset 精确解析
- 基础权限判断
- 风险判断
- 审批编排
- Order 管理
- Plan 生成与冻结
- Execute 前复核
- 执行路由
- 任务运行时管理
- 审计事件追加
- 证据包归档

Control Layer 不承担：

- 自然语言生成
- 通用对话管理
- Agent 记忆管理
- Deep Agent 的推理本身
- 底层执行器的具体实现细节

---

## 3. 核心对象总览

```text
ActionRequest
  -> AuthorizationDecision
  -> AssistantOrder
     -> ExecutionPlan
     -> ApprovalRecord (0..n)
     -> ExecutionTask (0..1 for MVP)
        -> ExecutionStep (1..n)
     -> AuditEvent (1..n)
     -> EvidencePack (1..n)
```

配套上下文对象：

- `Principal`
- `ResolvedAssetSet`
- `PolicyDecision`
- `RiskDecision`
- `PlanValidationResult`
- `AdapterBinding`

---

## 4. 核心对象定义

## 4.1 ActionRequest

### 作用
表示一次由上层 Skill/HTTP API 提交进来的标准动作申请。

### 为什么需要
因为 Deep Agent 不应该直接创建工单或直接执行。  
它先提交一个“动作申请”，后续是否被放行，由 Control Layer 决定。

### 建议字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `request_id` | string | 全局唯一请求 ID |
| `trace_id` | string | 全链路追踪 ID |
| `principal_id` | string | 发起主体 ID |
| `action_name` | string | 标准动作名，如 `mysql.database.create` |
| `resource_selector` | json | 资源选择器 |
| `parameters` | json | 动作参数 |
| `request_context` | json | 会话、消息来源、原因等上下文 |
| `source` | string | `deep_agent` / `api` / `ui` |
| `status` | enum | `ACCEPTED` / `REJECTED` |
| `created_at` | datetime | 创建时间 |

### 关键约束
1. `ActionRequest` 不是最终执行授权。
2. `resource_selector` 只能是业务语义，不允许裸 IP、裸凭证。
3. 进入控制主链路后，必须立刻写第一条 `AuditEvent`。

---

## 4.2 Principal

### 作用
表示“谁在发起这个动作”。

### 为什么需要
没有 Principal，就无法做权限判断、审批约束、审计落账。

### 建议字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `principal_id` | string | 主体 ID |
| `principal_type` | enum | `human` / `service` / `agent_proxy` |
| `user_id` | string | 用户标识 |
| `display_name` | string | 展示名 |
| `roles` | string[] | 角色集合 |
| `groups` | string[] | 所属组 |
| `scopes` | json | 可操作资源范围 |
| `approval_roles` | string[] | 可承担审批职责的角色 |
| `policy_exemptions` | string[] | 豁免标记 |
| `is_active` | bool | 是否可用 |

### 关键约束
1. Principal 必须从认证上下文解析，而不是信任客户端自填。
2. execute 触发时也必须重新解析 Principal。
3. `approver_id == created_by` 必须被禁止。

---

## 4.3 ResolvedAssetSet

### 作用
表示 `Asset Resolver` 对 selector 做精确匹配后的结果集合。

### 为什么需要
因为控制层不能对“订单库主库”这种自然语言直接执行，必须先变成受控对象。

### 建议字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `asset_ids` | string[] | 命中的资产 ID 集合 |
| `assets` | json[] | 资产详情 |
| `matched_exactly` | bool | 是否精确命中 |
| `asset_type` | string | 目标资产类型 |
| `resolved_at` | datetime | 解析时间 |

### 关键约束
1. MVP 中只能命中 1 个资产。
2. 0 个返回 `ASSET_NOT_FOUND`。
3. 多个返回 `ASSET_AMBIGUOUS`。
4. Resolver 不允许模糊匹配、最佳猜测、自动选一个。

---

## 4.4 PolicyDecision

### 作用
表示基础权限与资源范围校验结果。

### 为什么需要
把“有没有基础权限”和“风险多高”分开，才能让授权链路清晰。

### 建议字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `basic_allow` | bool | 是否具备基础动作权限 |
| `scope_allowed` | bool | 是否命中允许的资源范围 |
| `matched_roles` | string[] | 命中的角色 |
| `decision` | enum | `ALLOW` / `DENY` |
| `deny_reasons` | string[] | 拒绝原因 |
| `approval_exemption_flags` | string[] | 审批豁免标签 |

---

## 4.5 RiskDecision

### 作用
表示动作在当前资产和环境下的风险判断结果。

### 建议字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `risk_level` | enum | `R0` / `R1` / `R2` / `R3` |
| `decision` | enum | `ALLOW` / `REQUIRE_APPROVAL` / `DENY` |
| `reasons` | string[] | 风险原因 |
| `sensitivity_snapshot` | json | 目标敏感度快照 |

### 说明
RiskDecision 不单独决定最终执行。  
最终必须由 `AuthorizationDecision` 统一承接。

---

## 4.6 AuthorizationDecision

### 作用
Control Layer 对外唯一权威的最终授权结论。

### 为什么需要
禁止上层自己拼 `PolicyDecision + RiskDecision`，避免逻辑分叉。

### 建议字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `authorized` | bool | 是否允许进入下一阶段 |
| `final_decision` | enum | `ALLOW_NO_APPROVAL` / `ALLOW_WITH_APPROVAL` / `DENY` |
| `approval_required` | bool | 是否需要审批 |
| `risk_level` | enum | 最终风险等级 |
| `policy_decision` | string | 基础策略结论 |
| `risk_decision` | string | 风险结论 |
| `effective_exemptions` | string[] | 生效的豁免 |
| `deny_reasons` | string[] | 拒绝原因 |
| `approval_policy_ref` | string | 命中的审批策略 ID |

### 强制约束
1. 最终放行/拒绝只能看这个对象。
2. 它必须在 `Policy -> Risk` 之后生成。
3. 审批要求必须在这里明确落定。

---

## 4.7 AssistantOrder

### 作用
正式进入控制流后的工单对象，是 Control Layer 的业务聚合根。

### 建议字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `order_id` | string | 工单 ID |
| `request_id` | string | 来源请求 ID |
| `action_name` | string | 标准动作名 |
| `resolved_assets` | string[] | 命中的资产 ID |
| `risk_level` | enum | 最终风险等级 |
| `approval_required` | bool | 是否需要审批 |
| `approval_status` | enum | `NOT_REQUIRED` / `WAITING_APPROVAL` / `APPROVED` / `REJECTED` / `EXPIRED` |
| `status` | enum | `DRAFT` / `WAITING_APPROVAL` / `APPROVED` / `EXECUTING` / `SUCCEEDED` / `FAILED` / `PLAN_STALE` / `POLICY_REJECTED` / `CANCELLED` |
| `plan_id` | string | 绑定的冻结计划 ID |
| `plan_version` | int | 计划版本 |
| `created_by` | string | 发起人 |
| `last_execute_triggered_by` | string | 最近 execute 触发主体 |
| `created_at` | datetime | 创建时间 |
| `updated_at` | datetime | 更新时间 |

### 关键约束
1. 审批对象是 `AssistantOrder + ExecutionPlanSnapshot`。
2. 审批通过不自动执行。
3. execute 必须是单独接口。
4. `PLAN_STALE` 不是 `FAILED`。

---

## 4.8 ExecutionPlan

### 作用
表示审批前可审阅、审批后可复核、执行前可 re-validate 的冻结计划。

### 建议字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `plan_id` | string | 计划 ID |
| `order_id` | string | 所属工单 |
| `plan_version` | int | 版本号 |
| `plan_status` | enum | `DRAFT` / `FROZEN` / `REVALIDATED` / `STALE` / `CONSUMED` |
| `selected_route` | string | 默认执行路线 |
| `adapter_chain` | string[] | 候选适配器链 |
| `steps` | json[] | 执行步骤定义 |
| `rollback_strategy` | string | 回滚策略 |
| `idempotency_strategy` | string | 幂等策略 |
| `snapshot_frozen` | bool | 是否已冻结审批快照 |
| `validated_at` | datetime | 最近复核时间 |
| `stale_reason` | string | 失效原因 |

### 关键约束
1. 审批前必须先生成并冻结。
2. 审批人看到的是 PlanSnapshot，不是原始 SQL。
3. execute 前只允许 re-validate，不默认重建新计划。
4. 计划失效后，旧工单不继续执行。

---

## 4.9 ApprovalRecord

### 作用
记录审批动作及审批时看到的风险/计划快照。

### 建议字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `approval_id` | string | 审批记录 ID |
| `order_id` | string | 所属工单 |
| `approver_id` | string | 审批人 |
| `decision` | enum | `APPROVE` / `REJECT` |
| `comment` | string | 审批意见 |
| `risk_snapshot` | json | 审批时风险快照 |
| `plan_id` | string | 审批时的计划 ID |
| `plan_version` | int | 审批时计划版本 |
| `approved_at` | datetime | 审批时间 |

### 关键约束
1. 禁止自我审批。
2. 审批过期要有独立终态。
3. 审批通过不自动产生任务。

---

## 4.10 ExecutionTask

### 作用
表示一次真实执行任务。

### 建议字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `task_id` | string | 任务 ID |
| `order_id` | string | 所属工单 |
| `action_name` | string | 动作名 |
| `status` | enum | `PENDING` / `RUNNING` / `SUCCEEDED` / `FAILED` / `TIMEOUT` / `CANCELLED` |
| `started_at` | datetime | 开始时间 |
| `ended_at` | datetime | 结束时间 |
| `heartbeat_at` | datetime | 心跳时间 |

### 说明
MVP 可先限定一个工单最多一个任务。  
后续复杂动作再扩展成多任务。

---

## 4.11 ExecutionStep

### 作用
表示任务中的单个执行步骤。

### 建议字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `step_id` | string | 步骤 ID |
| `task_id` | string | 所属任务 |
| `priority` | int | 顺序 |
| `adapter_type` | string | 执行器类型 |
| `operation` | string | 具体操作名 |
| `status` | enum | `PENDING` / `RUNNING` / `SUCCEEDED` / `FAILED` / `SKIPPED` |
| `timeout_seconds` | int | 超时 |
| `error_message` | string | 失败信息 |

---

## 4.12 AuditEvent

### 作用
作为 append-only 审计源记录。

### 最小事件集建议
- `REQUEST_ACCEPTED`
- `AUTHORIZATION_DECIDED`
- `ORDER_CREATED`
- `PLAN_FROZEN`
- `APPROVAL_CREATED`
- `APPROVAL_APPROVED`
- `APPROVAL_REJECTED`
- `APPROVAL_EXPIRED`
- `EXECUTE_TRIGGERED`
- `PLAN_REVALIDATED`
- `PLAN_STALE`
- `EXECUTION_STARTED`
- `EXECUTION_SUCCEEDED`
- `EXECUTION_FAILED`
- `EVIDENCE_WRITTEN`

### 建议字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `event_id` | string | 事件 ID |
| `request_id` | string | 请求 ID |
| `order_id` | string | 工单 ID |
| `task_id` | string | 任务 ID |
| `event_type` | string | 事件类型 |
| `principal_id` | string | 发起主体 |
| `approval_actor_id` | string | 审批人 |
| `execute_actor_id` | string | 执行触发人 |
| `raw_user_prompt` | string | 原始输入 |
| `normalized_action` | string | 归一化动作 |
| `resolved_asset_ids` | string[] | 资产 ID |
| `risk_level` | string | 风险等级 |
| `policy_decision` | string | 策略结论 |
| `authorization_decision` | string | 最终授权结论 |
| `approval_status` | string | 审批状态 |
| `order_status` | string | 工单状态 |
| `selected_adapter` | string | 实际执行器 |
| `execution_summary` | string | 执行摘要 |
| `success` | bool | 是否成功 |
| `error_code` | string | 错误码 |
| `error_message` | string | 错误信息 |
| `trace_id` | string | trace ID |
| `created_at` | datetime | 创建时间 |

---

## 4.13 EvidencePack

### 作用
固化一次动作的前后状态、审批、执行摘要和失败说明。

### 建议字段

| 字段 | 类型 | 说明 |
|---|---|---|
| `evidence_id` | string | 证据包 ID |
| `order_id` | string | 所属工单 |
| `task_id` | string | 所属任务，可为空 |
| `artifact_refs` | json[] | 外部制品引用 |
| `request_summary` | string | 请求摘要 |
| `before_state_snapshot` | json | 执行前状态 |
| `after_state_snapshot` | json | 执行后状态 |
| `approval_refs` | string[] | 审批引用 |
| `execution_success` | bool | 是否成功 |
| `failure_detail` | json | 失败详情 |
| `result_summary` | string | 结果摘要 |
| `rollback_suggestion` | string | 回滚建议 |

### 关键约束
1. 成功和失败都必须生成。
2. `PLAN_STALE` 场景也要生成失败证据。
3. 失败时 `failure_detail` 不得为空。

---

## 5. 对象状态与主链路

## 5.1 主链路

```text
ActionRequest
  -> Principal / Asset / Policy / Risk
  -> AuthorizationDecision
  -> AssistantOrder
  -> ExecutionPlan
  -> ApprovalRecord (if needed)
  -> explicit execute
  -> Plan Re-validate
  -> ExecutionTask / ExecutionStep
  -> AuditEvent*
  -> EvidencePack
```

## 5.2 关键状态边界
1. `APPROVED` 不是执行中。
2. `WAITING_APPROVAL` 不可 execute。
3. `PLAN_STALE` 表示计划失效，不代表执行失败。
4. `EXECUTING` 状态下重复 execute 必须幂等返回现有任务。
5. `SUCCEEDED` 后不能再新建任务。

---

## 6. MVP：mysql.database.create 的对象要求

MVP 至少需要以下对象真正落地：

- `ActionRequest`
- `Principal`
- `ResolvedAssetSet`
- `PolicyDecision`
- `RiskDecision`
- `AuthorizationDecision`
- `AssistantOrder`
- `ExecutionPlan`
- `ApprovalRecord`（prod 路径）
- `ExecutionTask`
- `ExecutionStep`
- `AuditEvent`
- `EvidencePack`

对于这个动作，最小步骤建议是：

1. `validate_target`
2. `check_database_not_exists`
3. `create_database`
4. `verify_database_created`

---

## 7. 建表建议（实现提示，不是强制）

如果你先用单库 MVP，建议至少拆这些表：

- `principals`
- `roles`
- `principal_roles`
- `assets`
- `action_requests`
- `assistant_orders`
- `execution_plans`
- `approval_records`
- `execution_tasks`
- `execution_steps`
- `audit_events`
- `evidence_packs`

后续再补：
- `action_policies`
- `resource_scope_policies`
- `risk_policies`
- `approval_policies`

---

## 8. 对 Coding Agent 的直接要求

1. 先把对象模型固化，再写 handler。
2. 所有执行能力都要围绕这些对象流转。
3. 不允许跳过 `AuthorizationDecision` 直接创建任务。
4. 不允许跳过 `ExecutionPlan` 直接调 Adapter。
5. 不允许只写成功审计，不写失败审计。
6. 不允许只写成功证据，不写失败证据。
7. `request -> approval -> execute` 必须拆成受控边界，不可揉成一个 handler。
