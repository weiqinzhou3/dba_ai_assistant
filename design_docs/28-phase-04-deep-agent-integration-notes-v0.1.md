# Phase 04 - Deep Agent Integration Notes v0.1

## 目标

在不改变既有 Control Layer 权威语义的前提下，让 Deep Agent 通过两个高层 skill 安全调用系统：

- `request_mysql_database_create`
- `execute_assistant_order`

本轮严格不接 MCP，也不新增业务动作。

## 本轮设计

### 1. Skill 只走 northbound Control API

Skill 侧只允许调用：

- `POST /api/v1/action-requests`
- `POST /api/v1/orders/{order_id}/execute`

Skill 不直接接触：

- adapter
- SQL
- `connection_ref`
- DSN / 凭证

因此 Deep Agent 仍然只是 northbound client，而不是事实上的控制器。

### 2. Skill 输入保持业务语义

`request_mysql_database_create` 输入只接受：

- `project`
- `environment`
- `service_instance`
- `database_name`
- 可选 `conversation_id` / `message_id` / `reason`

它不会接受：

- `asset_id`
- `adapter_type`
- 任何底层连接参数

这保证 `AssetResolver` 仍然是唯一 exact match 命中入口。

### 3. auto-chain execute 采用同一 user principal

本轮 auto-chain 策略固定为：

1. request skill 先提交 `POST /api/v1/action-requests`
2. 若响应满足：
   - `approval_required = false`
   - `status = APPROVED`
3. skill 才会继续用**同一认证主体**调用 `execute_assistant_order`

第二次 `execute` 仍通过既有 northbound header 传递：

- `X-Principal-ID`
- `X-Roles`

因此 execute 放行仍由既有 `ExecuteAuthorizationService` 权威裁决。  
本轮没有默认切换成代理身份，也没有把 auto-chain 藏进 Control Layer 内部。

### 4. prod / 审批路径不会被偷跑 execute

当 submit 返回：

- `approval_required = true`
或
- `status = WAITING_APPROVAL`

skill 直接停止在 request 结果，不会发起第二次 `/execute` 调用。

因此 prod 路径仍必须满足：

- request
- approval
- explicit execute

控制层语义保持不变。

### 5. submit 的 principal 可信传递最小收敛

为避免 skill/body 自填 `principal_id` 漂移，本轮把 submit handler 收敛为：

- 若存在 `X-Principal-ID`
  - body `principal_id` 为空：使用认证主体
  - body `principal_id` 与认证主体不一致：返回 `REQ_INVALID`

这样 skill 侧可以安全地把认证主体写入 northbound header，并保持 body/header 一致。

## 本轮交付

### 代码

- `internal/skill/contracts.go`
- `internal/skill/service.go`
- `internal/api/server.go`

### 测试

- `internal/skill/contracts_test.go`
- `internal/skill/service_test.go`
- `internal/skill/integration_test.go`
- `internal/api/server_test.go`

## 已验证路径

1. non-prod 请求可经 `request_mysql_database_create` 自动串联 `execute_assistant_order`
2. prod 请求会停在 `WAITING_APPROVAL`
3. auto-chain 遇到 `EXECUTOR_NOT_ALLOWED` 时保留 request 成功结果，并返回受控用户提示
4. submit body/header `principal_id` 不一致时返回 `REQ_INVALID`

## 明确不做

- 不接 MCP
- 不新增 `mysql.user.create` 等动作
- 不让 Deep Agent 直接调用 adapter
- 不让 Deep Agent 直接执行 SQL
- 不把 Policy / Risk / Approval / Audit 逻辑搬到 skill 层
