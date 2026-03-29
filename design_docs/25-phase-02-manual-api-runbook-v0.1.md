# Phase 02 Manual API Runbook v0.1

## 文档作用

提供 Phase 02 最小控制链路的手工 HTTP 演示步骤。

## 前置条件

1. 在仓库根目录启动服务：

```bash
go run ./cmd/server
```

2. 默认监听：

```text
http://127.0.0.1:8080
```

## 场景 A：prod 走审批后显式 execute

### 1. 提交 ActionRequest

```bash
curl -sS -X POST http://127.0.0.1:8080/api/v1/action-requests \
  -H 'Content-Type: application/json' \
  -d '{
    "principal_id": "u_requester",
    "action_hint": "mysql.database.create",
    "resource_selector": {
      "project": "order-platform",
      "environment": "prod",
      "service_instance": "mysql-order-main"
    },
    "parameters": {
      "database_name": "order_center"
    }
  }'
```

预期重点：

- `status = "WAITING_APPROVAL"`
- `approval_required = true`

### 2. 审批通过

```bash
curl -sS -X POST http://127.0.0.1:8080/api/v1/orders/ord_0001/approvals \
  -H 'Content-Type: application/json' \
  -d '{
    "approver_id": "u_approver",
    "decision": "APPROVE",
    "comment": "approved"
  }'
```

预期重点：

- `approval_status = "APPROVED"`
- `status = "APPROVED"`

### 3. 显式 execute

```bash
curl -sS -X POST http://127.0.0.1:8080/api/v1/orders/ord_0001/execute \
  -H 'Content-Type: application/json' \
  -H 'X-Principal-ID: u_executor' \
  -H 'X-Roles: mysql_operator' \
  -d '{
    "reason": "manual execute"
  }'
```

预期重点：

- `status = "EXECUTING"`
- `task_id != null`
- 这里只启动任务骨架，不代表真实数据库已经创建

### 4. 查询 Audit Ledger

```bash
curl -sS http://127.0.0.1:8080/api/v1/audit-ledger/req_0001
```

预期重点：

- `latest_order_status = "EXECUTING"`
- `latest_task_id = "task_ord_0001"`
- 事件流包含：
  - `REQUEST_ACCEPTED`
  - `AUTHORIZATION_DECIDED`
  - `ORDER_CREATED`
  - `PLAN_FROZEN`
  - `APPROVAL_CREATED`
  - `APPROVAL_APPROVED`
  - `EXECUTE_TRIGGERED`
  - `PLAN_REVALIDATED`
  - `EXECUTION_STARTED`
  - `EVIDENCE_WRITTEN`

### 5. 查询 EvidencePack

```bash
curl -sS http://127.0.0.1:8080/api/v1/evidence-packs/ord_0001
```

预期重点：

- `task_id = "task_ord_0001"`
- `execution_success = true`
- `result_summary` 明确写明 Phase 02 只启动 task skeleton，未做真实 DB 执行

## 场景 B：用占位开关演示 PLAN_STALE

### 1. 提交带 stale 模拟标记的 ActionRequest

```bash
curl -sS -X POST http://127.0.0.1:8080/api/v1/action-requests \
  -H 'Content-Type: application/json' \
  -d '{
    "principal_id": "u_requester",
    "action_hint": "mysql.database.create",
    "resource_selector": {
      "project": "order-platform",
      "environment": "test",
      "service_instance": "mysql-order-main"
    },
    "parameters": {
      "database_name": "order_center"
    },
    "request_context": {
      "simulate_plan_stale": true
    }
  }'
```

预期重点：

- `status = "APPROVED"`
- `approval_required = false`

### 2. execute

```bash
curl -sS -X POST http://127.0.0.1:8080/api/v1/orders/ord_0002/execute \
  -H 'Content-Type: application/json' \
  -H 'X-Principal-ID: u_executor' \
  -H 'X-Roles: mysql_operator' \
  -d '{
    "reason": "manual stale probe"
  }'
```

预期重点：

- `status = "PLAN_STALE"`
- `task_id = null`

### 3. 查询 stale EvidencePack

```bash
curl -sS http://127.0.0.1:8080/api/v1/evidence-packs/ord_0002
```

预期重点：

- `task_id = null`
- `execution_success = false`
- `failure_detail.reason = "simulated by request_context.simulate_plan_stale"`
- `result_summary = "计划失效，未启动执行任务"`

## 已知边界

1. 本 runbook 只验证控制流接线，不验证真实 MySQL side effect。
2. approval 请求当前仍通过请求体传 `approver_id`，正式认证收口留在后续 phase。
3. `simulate_plan_stale` 只是 Phase 02 的手工演示开关，不属于正式产品语义。
