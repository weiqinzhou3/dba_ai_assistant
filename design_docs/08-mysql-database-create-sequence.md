# mysql.database.create Sequence v0.1

## 0. 文档定位

本文档把 `mysql.database.create` 作为 MVP 纵切片，按阶段拆成：

1. 请求受理阶段
2. 授权决策阶段
3. 审批阶段（如需要）
4. execute 阶段
5. 执行与回写阶段
6. 审计与证据阶段

目标是让实现者按阶段落地，而不是一上来写一个大 handler。

---

## 1. 动作定义

### 标准动作名
`mysql.database.create`

### 业务语义
在一个已纳管 MySQL 目标上创建逻辑数据库。

### 一期命名规范
- 以字母开头
- 仅允许字母/数字/下划线
- 最大 64 个 ASCII 字符

建议 regex：

```text
^[A-Za-z][A-Za-z0-9_]{0,63}$
```

---

## 2. 入口与边界

### 上层入口
- Skill：`request_mysql_database_create`

### 北向 API
- `POST /api/v1/action-requests`

### 审批 API
- `POST /api/v1/orders/{order_id}/approvals`

### execute API
- `POST /api/v1/orders/{order_id}/execute`

### 查询 API
- `GET /api/v1/orders/{order_id}`
- `GET /api/v1/tasks/{task_id}`
- `GET /api/v1/audit-ledger/{request_id}`
- `GET /api/v1/evidence-packs/{order_id}`

---

## 3. 阶段一：请求受理

## 3.1 输入

示例：

```json
{
  "principal_id": "u_1001",
  "action_hint": "mysql.database.create",
  "resource_selector": {
    "project": "order-platform",
    "environment": "prod",
    "service_instance": "mysql-order-main"
  },
  "parameters": {
    "database_name": "order_center"
  },
  "request_context": {
    "source": "deep_agent",
    "conversation_id": "conv_123",
    "message_id": "msg_456"
  }
}
```

## 3.2 处理步骤
1. 生成 `request_id`
2. 生成 `trace_id`
3. 校验请求体 schema
4. 创建 `ActionRequest`
5. 追加 `REQUEST_ACCEPTED` 审计事件

## 3.3 输出
- 成功：进入授权链路
- 失败：返回 `REQ_INVALID`

---

## 4. 阶段二：授权决策

## 4.1 主链路

```text
ActionRequest
  -> Action Normalizer
  -> Principal Resolver
  -> Asset Resolver (exact)
  -> Policy Engine
  -> Risk Engine
  -> AuthorizationService
```

## 4.2 处理步骤

### Step 1: Action Normalizer
- 归一化为 `mysql.database.create`
- 校验 `database_name` 命名规范

### Step 2: Principal Resolver
- 从认证上下文 / principal_id 解析 Principal

### Step 3: Asset Resolver
- 使用 `project + environment + service_instance`
- 严格精确匹配
- 0 个 -> `ASSET_NOT_FOUND`
- 多个 -> `ASSET_AMBIGUOUS`

### Step 4: Policy Engine
检查：
- 是否允许发起 `mysql.database.create`
- 是否命中资源范围

### Step 5: Risk Engine
判断：
- 是否 prod
- 目标是否高敏
- 风险等级 R1/R2/R3

### Step 6: AuthorizationService
输出唯一权威结论：

- `ALLOW_NO_APPROVAL`
- `ALLOW_WITH_APPROVAL`
- `DENY`

## 4.3 输出对象
- `PolicyDecision`
- `RiskDecision`
- `AuthorizationDecision`

---

## 5. 阶段三：工单与计划生成

## 5.1 处理步骤
1. 创建 `AssistantOrder`
2. 生成 `ExecutionPlan`
3. 冻结 `ExecutionPlanSnapshot`
4. 写 `ORDER_CREATED` 和 `PLAN_FROZEN` 审计事件

## 5.2 计划最小步骤
1. `validate_target`
2. `check_database_not_exists`
3. `create_database`
4. `verify_database_created`

## 5.3 状态分支

### 分支 A：无需审批
- `approval_status = NOT_REQUIRED`
- `order.status = APPROVED`

### 分支 B：需要审批
- 创建 `ApprovalRecord`
- `approval_status = WAITING_APPROVAL`
- `order.status = WAITING_APPROVAL`

---

## 6. 阶段四：审批（仅 prod / 高风险）

## 6.1 输入
- `order_id`
- `approver_id`
- `decision`
- `comment`

## 6.2 校验
1. 审批人身份合法
2. 审批人具备审批角色
3. 禁止自我审批
4. 审批未过期
5. 工单当前处于 `WAITING_APPROVAL`

## 6.3 状态结果
- `APPROVE` -> `approval_status = APPROVED`, `order.status = APPROVED`
- `REJECT` -> `approval_status = REJECTED`, `order.status = REJECTED`
- 过期 -> `approval_status = EXPIRED`, `order.status = EXPIRED`

## 6.4 注意
审批通过不自动执行。

---

## 7. 阶段五：显式 execute

## 7.1 输入
- `order_id`
- execute 调用人的认证上下文
- 可选 `reason`

## 7.2 必做校验
1. execute 调用主体从认证上下文解析
2. 具备 ExecutePolicy
3. 工单当前为 `APPROVED`
4. 计划存在且版本一致
5. 执行前做 `Plan Re-validate`

## 7.3 Re-validate 最小检查项
1. 目标 Asset 仍唯一命中
2. 连接引用仍可解析
3. 数据库仍不存在
4. 幂等键无运行中冲突
5. 计划版本仍匹配

## 7.4 Re-validate 结果

### 通过
- 创建 `ExecutionTask`
- 进入 `EXECUTING`

### 失败
- `plan_status = STALE`
- `order.status = PLAN_STALE`
- 不创建任务
- 写 `PLAN_STALE` 审计事件
- 生成失败证据包

---

## 8. 阶段六：执行

## 8.1 Execution Router
MVP 固定：
- `mysql.database.create` -> `DBNativeAdapter`

## 8.2 DB-native Adapter 最小流程

### Step 1: validate_target
- 校验连接存在
- 校验目标引擎为 MySQL

### Step 2: check_database_not_exists
- 查询目标 schema 是否存在
- 若已存在，返回幂等成功或冲突结论，按策略处理

### Step 3: create_database
- 生成 SQL 摘要
- 执行 `CREATE DATABASE`

### Step 4: verify_database_created
- 再次查询 schema 是否存在

## 8.3 ExecutionTask / Step 状态推进
- Task: `PENDING -> RUNNING -> SUCCEEDED/FAILED`
- Step: `PENDING -> RUNNING -> SUCCEEDED/FAILED/SKIPPED`

---

## 9. 阶段七：审计与证据

## 9.1 审计事件最小集
- `REQUEST_ACCEPTED`
- `AUTHORIZATION_DECIDED`
- `ORDER_CREATED`
- `PLAN_FROZEN`
- `APPROVAL_CREATED`（如有）
- `APPROVAL_APPROVED` / `APPROVAL_REJECTED` / `APPROVAL_EXPIRED`
- `EXECUTE_TRIGGERED`
- `PLAN_REVALIDATED`
- `PLAN_STALE`（如有）
- `EXECUTION_STARTED`
- `EXECUTION_SUCCEEDED` / `EXECUTION_FAILED`
- `EVIDENCE_WRITTEN`

## 9.2 证据包最小内容

### 成功路径
- 请求摘要
- 目标实例标识
- before: `database_exists=false`
- after: `database_exists=true`
- SQL 摘要
- 执行时间
- 审批引用（如有）

### 失败路径
- 请求摘要
- 失败详情
- before/after 快照
- 日志引用
- 回滚建议

### PLAN_STALE 路径
- `task_id = null`
- 失败原因写明“计划失效，未启动执行任务”

---

## 10. 状态机摘要

## 10.1 无审批路径（dev/test）
```text
ActionRequest
  -> AuthorizationDecision(ALLOW_NO_APPROVAL)
  -> AssistantOrder(APPROVED)
  -> explicit execute
  -> ExecutionTask
  -> SUCCEEDED / FAILED
```

## 10.2 有审批路径（prod）
```text
ActionRequest
  -> AuthorizationDecision(ALLOW_WITH_APPROVAL)
  -> AssistantOrder(WAITING_APPROVAL)
  -> ApprovalRecord(APPROVED)
  -> AssistantOrder(APPROVED)
  -> explicit execute
  -> ExecutionTask
  -> SUCCEEDED / FAILED
```

## 10.3 计划失效路径
```text
APPROVED
  -> execute
  -> re-validate failed
  -> PLAN_STALE
  -> no task created
  -> failure evidence written
```

---

## 11. MVP 交付验收点

实现 `mysql.database.create` 时，至少验收以下内容：

1. 能通过 `POST /api/v1/action-requests` 创建动作请求
2. 能精确命中资产
3. 能正确输出 `AuthorizationDecision`
4. prod 路径能进入 `WAITING_APPROVAL`
5. 审批通过后不会自动执行
6. 只能通过显式 `execute` 启动执行
7. execute 前必须 re-validate
8. 能通过 `DBNativeAdapter` 完成数据库创建
9. 成功路径有审计和证据
10. 失败路径也有审计和证据
11. `PLAN_STALE` 路径可见且不落成普通 `FAILED`

---

## 12. 对 Coding Agent 的直接要求

1. 不要把这一整条链路揉成一个 handler。
2. 先打通对象流，再填充数据库细节。
3. 先做一条最小纵切片，不扩散到 `mysql.user.create` 等其他动作。
4. 如果某一步没有实现，也要把接口和状态先建出来。
