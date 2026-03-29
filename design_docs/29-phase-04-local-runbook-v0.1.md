# Phase 04 - Local Runbook v0.1

## 目标

说明如何在本地启动既有 Control API，并通过 Phase 04 的 skill client 联调：

- non-prod auto-chain success
- prod waiting approval

## 1. 启动 Control API

在项目根目录执行：

```bash
go run ./cmd/server
```

默认监听：

```text
http://localhost:8080
```

注意：

- 本地 server 仍使用 in-memory store
- `cmd/server` 默认注册的是既有 `DBNativeAdapter`
- 若未配置真实 MySQL 连接，适合先跑 skill 单测 / API 联调逻辑，不适合做真实数据库创建验证

## 2. 运行 Phase 04 skill 测试

优先执行：

```bash
go test ./internal/skill -count=1
```

若要连同 northbound Control API 相关回归一起执行：

```bash
go test ./internal/api ./internal/application/actionrequest ./internal/skill -count=1
```

## 3. 最小本地调用示例

可用一个最小 Go 片段直接调用 skill client：

```go
package main

import (
	"context"
	"fmt"

	appauth "dba_ai_assistant/internal/application/authorization"
	"dba_ai_assistant/internal/domain/principal"
	"dba_ai_assistant/internal/skill"
)

func main() {
	client, err := skill.NewService(skill.Dependencies{
		BaseURL: "http://localhost:8080",
	})
	if err != nil {
		panic(err)
	}

	out, err := client.RequestMySQLDatabaseCreate(context.Background(), appauth.AuthContext{
		AuthenticatedPrincipalID: "u_requester",
		Roles:                    []string{principal.RoleMySQLOperator},
		Source:                   "deep_agent",
	}, skill.RequestMySQLDatabaseCreateInput{
		Project:         "order-platform",
		Environment:     "test",
		ServiceInstance: "mysql-order-main",
		DatabaseName:    "order_center",
		Reason:          "local phase-04 smoke",
		AutoExecute:     true,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", out)
}
```

## 4. 期望联调结果

### 场景 A: non-prod auto-chain

输入：

- `environment = test`
- `roles = [mysql_operator]`
- `AutoExecute = true`

期望：

- request 返回 `APPROVED`
- skill 自动再调用一次 `execute_assistant_order`
- 最终输出包含：
  - `auto_execute_triggered = true`
  - `auto_execute_result.task_id != ""`
  - `order_status = SUCCEEDED`

### 场景 B: prod waiting approval

输入：

- `environment = prod`
- `roles = [mysql_operator]`
- `AutoExecute = true`

期望：

- request 返回 `WAITING_APPROVAL`
- skill 不会调用 `/execute`
- 最终输出包含：
  - `approval_required = true`
  - `auto_execute_triggered = false`
  - `order_status = WAITING_APPROVAL`

## 5. 常见问题

### `EXECUTOR_NOT_ALLOWED`

含义：

- request 已成功
- 但 auto-chain 的第二次 execute 调用未通过既有 `ExecuteAuthorizationService`

处理：

- 不重发 request
- 让具备执行角色的主体显式调用 `execute_assistant_order`

### `APPROVAL_REQUIRED`

含义：

- 当前工单仍需要审批

处理：

- 先完成 `/approvals`
- 再显式执行 `execute_assistant_order`

### `PLAN_STALE`

含义：

- execute 前 revalidate 未通过

处理：

- 不复用旧工单
- 按既有控制层语义重新发起新的 request
