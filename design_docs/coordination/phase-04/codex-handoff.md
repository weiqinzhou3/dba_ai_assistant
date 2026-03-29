# Phase 04 - Codex Handoff

## 文档作用

由 Codex 在一个可评审轮次结束后填写，交给 Claude Code 做 review。

## 本轮交接

- 分支：`feat/p4-deep-agent-integration`
- 提交范围：
  - 最小 Deep Agent Skill SDK / HTTP client
  - request -> execute auto-chain orchestration
  - submit northbound principal 收敛
  - Phase 04 notes / local runbook / coordination 文档同步
- 主要变更：
  - `internal/skill/contracts.go`
    - 将两个高层 skill 的输入输出 contract 扩展为可运行版本
    - request skill 改为业务语义 selector 输入
    - execute skill 改为 order/task/trace/executor 输出
  - `internal/skill/service.go`
    - 新增最小 Skill SDK / HTTP client
    - `request_mysql_database_create` 通过 `POST /api/v1/action-requests` 发起 request
    - 在 `approval_required=false && status=APPROVED` 时可选自动继续 `execute_assistant_order`
    - auto-chain 继续使用同一认证主体调用 `/execute`
    - auto-chain 失败时保留 request 成功结果，并返回受控用户消息
  - `internal/api/server.go`
    - submit route 现在在 `X-Principal-ID` 存在时优先信任认证主体
    - body/header 不一致返回 `REQ_INVALID`
  - 新增 Phase 04 notes / runbook：
    - `design_docs/28-phase-04-deep-agent-integration-notes-v0.1.md`
    - `design_docs/29-phase-04-local-runbook-v0.1.md`
- 关键文件：
  - [contracts.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/skill/contracts.go)
  - [service.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/skill/service.go)
  - [integration_test.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/skill/integration_test.go)
  - [server.go](/Users/zqw/Desktop/Project/dba_ai_assistant/internal/api/server.go)
  - [28-phase-04-deep-agent-integration-notes-v0.1.md](/Users/zqw/Desktop/Project/dba_ai_assistant/design_docs/28-phase-04-deep-agent-integration-notes-v0.1.md)
  - [29-phase-04-local-runbook-v0.1.md](/Users/zqw/Desktop/Project/dba_ai_assistant/design_docs/29-phase-04-local-runbook-v0.1.md)
- 已执行验证：
  - `go test ./internal/skill -count=1`
  - `go test ./internal/api ./internal/application/actionrequest ./internal/skill -count=1`
  - `go test ./... -count=1`
- 未覆盖项：
  - 本轮没有引入真实 Deep Agent runtime / CLI，只交付最小 Skill SDK 与 northbound integration tests。
  - 默认自动化测试仍使用 fake MySQL admin，不直接连接真实 MySQL。
  - request 路径虽然已收敛 `principal_id` 到认证主体，但 request 权限模型仍沿用当前 static resolver / policy 行为，未在本轮重构。
- 请求 review 重点：
  - Deep Agent 是否仍严格只通过高层 skill / northbound Control API，而未绕过到 adapter。
  - auto-chain execute 是否仍体现为两次独立受控调用。
  - prod / approval 路径是否不会被 skill 偷跑 execute。
  - auto-chain principal 策略是否清晰，且未引入隐式 service principal。
  - 本轮是否严格未接 MCP、未新增动作、未改变既有控制层权威语义。
