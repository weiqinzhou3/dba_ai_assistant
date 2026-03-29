# Phase 03 - Manual Verification Runbook v0.1

## 作用

给 Claude review 或人工验收使用，验证 `mysql.database.create` 在真实 MySQL 上的最小可执行闭环。

## 前置条件

1. 本地有一个可连接的 MySQL 实例
2. 连接账号具备：
   - 访问 `information_schema.schemata`
   - `CREATE DATABASE`
3. 服务启动前设置：

```bash
export DBNATIVE_CONNECTIONS_JSON='{
  "secret://db-targets/mysql-order-main-test":"root:root@tcp(127.0.0.1:3306)/mysql?parseTime=true",
  "secret://db-targets/mysql-order-main-prod":"root:root@tcp(127.0.0.1:3306)/mysql?parseTime=true"
}'
```

4. 启动服务：

```bash
go run ./cmd/server
```

## 场景 A：dev/test 直接 execute 成功

### 1. submit

```bash
curl -sS -X POST http://127.0.0.1:8080/api/v1/action-requests \
  -H 'Content-Type: application/json' \
  -d '{
    "principal_id":"u_requester",
    "action_hint":"mysql.database.create",
    "resource_selector":{
      "project":"order-platform",
      "environment":"test",
      "service_instance":"mysql-order-main"
    },
    "parameters":{
      "database_name":"phase3_demo_db"
    }
  }'
```

预期：

- `status = "APPROVED"`
- `approval_required = false`

### 2. explicit execute

```bash
curl -sS -X POST http://127.0.0.1:8080/api/v1/orders/ord_0001/execute \
  -H 'Content-Type: application/json' \
  -H 'X-Principal-ID: u_executor' \
  -H 'X-Roles: mysql_operator' \
  -d '{
    "reason":"manual execute"
  }'
```

预期：

- `status = "SUCCEEDED"`
- 返回 `task_id`

### 3. evidence

```bash
curl -sS http://127.0.0.1:8080/api/v1/evidence-packs/ord_0001
```

预期：

- `execution_success = true`
- `before_state_snapshot.database_exists = false`
- `after_state_snapshot.database_exists = true`
- `artifact_refs` 非空

### 4. audit

```bash
curl -sS http://127.0.0.1:8080/api/v1/audit-ledger/req_0001
```

预期至少包含：

- `EXECUTE_TRIGGERED`
- `PLAN_REVALIDATED`
- `EXECUTION_STARTED`
- `EXECUTION_SUCCEEDED`
- `EVIDENCE_WRITTEN`

## 场景 B：prod 审批后显式 execute

### 1. submit

```bash
curl -sS -X POST http://127.0.0.1:8080/api/v1/action-requests \
  -H 'Content-Type: application/json' \
  -d '{
    "principal_id":"u_requester",
    "action_hint":"mysql.database.create",
    "resource_selector":{
      "project":"order-platform",
      "environment":"prod",
      "service_instance":"mysql-order-main"
    },
    "parameters":{
      "database_name":"phase3_prod_db"
    }
  }'
```

预期：

- `status = "WAITING_APPROVAL"`
- `approval_required = true`

### 2. approval

```bash
curl -sS -X POST http://127.0.0.1:8080/api/v1/orders/ord_0002/approvals \
  -H 'Content-Type: application/json' \
  -H 'X-Principal-ID: u_approver' \
  -d '{
    "approver_id":"u_approver",
    "decision":"APPROVE",
    "comment":"approved in phase 03 manual run"
  }'
```

预期：

- `approval_status = "APPROVED"`
- `status = "APPROVED"`

### 3. explicit execute

沿用场景 A 的 execute 请求，但 `order_id` 换成 prod 工单。

## 场景 C：目标已存在触发 PLAN_STALE

### 1. 先在目标 MySQL 上手工建库

```sql
CREATE DATABASE `phase3_stale_db`;
```

### 2. submit 同名请求

与场景 A 类似，但 `database_name = phase3_stale_db`

### 3. execute

预期：

- 返回 `status = "PLAN_STALE"`
- 不返回新的 `task_id`
- evidence:
  - `execution_success = false`
  - `failure_detail.code = "PLAN_STALE"`

## 场景 D：幂等成功重放

### 1. 先成功执行一次 `phase3_replay_db`

### 2. 再提交第二个同名请求并 execute

预期：

- 不重复执行第二次 `CREATE DATABASE`
- 返回 `status = "SUCCEEDED"`
- evidence `result_summary` 提到 `idempotent replay`

## 场景 E：approval actor 认证边界

### 1. 发送 body / header 一致

应允许审批通过。

### 2. 发送 body / header 不一致

```bash
curl -sS -X POST http://127.0.0.1:8080/api/v1/orders/ord_xxxx/approvals \
  -H 'Content-Type: application/json' \
  -H 'X-Principal-ID: u_header' \
  -d '{
    "approver_id":"u_body",
    "decision":"APPROVE"
  }'
```

预期：

- HTTP `400`
- `error.code = "REQ_INVALID"`

## 收尾

验证结束后可手工清理：

```sql
DROP DATABASE IF EXISTS `phase3_demo_db`;
DROP DATABASE IF EXISTS `phase3_prod_db`;
DROP DATABASE IF EXISTS `phase3_stale_db`;
DROP DATABASE IF EXISTS `phase3_replay_db`;
```
