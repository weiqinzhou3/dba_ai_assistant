# Phase 04 - Codex Plan

## 文档作用

由 Codex 在进入 Phase 04 编码前填写，明确 Deep Agent 接入的边界、映射方式和验证方案。

## 当前 phase 目标

让 Deep Agent 通过 Skill / HTTP 安全接入 Control Layer，但不下放执行控制权。

## 必须回应事项

1. auto-chain execute 使用的 principal 策略是什么：
   - 本轮采用 **user principal 直接触发 execute** 的最小策略，不新引入代理执行身份。
   - `request_mysql_database_create` 在 `approval_required=false` 且 `status=APPROVED` 时，允许由 Skill 层自动继续调用 `execute_assistant_order`。
   - 第二次 `execute` 调用继续把同一认证主体通过 northbound header 传给 Control API，由既有 `ExecuteAuthorizationService` 做显式放行；当前静态实现只允许 `mysql_operator` / `platform_admin` / `control_executor`。
   - 本轮不会让 auto-chain 默默切换成隐式 service principal；`control_executor` 仅保留为既有 Control Layer 可接受角色，不作为本轮默认策略。
2. 如何证明 Agent 仍只是 northbound client，而不是事实上的控制器：
   - Skill 侧只允许调用 `POST /api/v1/action-requests` 与 `POST /api/v1/orders/{order_id}/execute`。
   - Skill 输入保持业务语义：`project + environment + service_instance + database_name`，不暴露 adapter、SQL、connection ref。
   - 所有 request / execute / approval / revalidate / audit / evidence 语义仍由 Control Layer 返回并裁决，Skill 只做 HTTP 映射、状态解释与可选 auto-chain。
3. 哪些验证能证明 prod 路径不会被 Agent 偷跑 execute：
   - skill 集成测试验证：当 submit 返回 `WAITING_APPROVAL` 或 `approval_required=true` 时，不会发起第二次 `/execute` 调用。
   - real HTTP integration test 验证：prod 资产路径只会创建 `WAITING_APPROVAL` order，Skill 输出等待审批消息，不产生 task。
   - non-prod 路径验证：auto-chain 发生时 Control API 仍保留两次独立 northbound 调用与既有 execute 权限校验。

## 本轮计划

- 当前分支：`feat/p4-deep-agent-integration`
- 本轮目标：
  - 基于既有 `mysql.database.create` MVP 与 northbound Control API，交付最小 Deep Agent Skill 接入层，使 Agent 可以通过高层 Skill 调用系统，并在无审批路径上做受控的 request -> execute 自动串联。
- 本轮范围：
  - 将 `internal/skill` 从 contract-only 升级为最小可运行 Skill SDK / HTTP client。
  - 固化两个高层 skill：
    - `request_mysql_database_create`
    - `execute_assistant_order`
  - request skill 负责：
    - 使用业务语义 selector 组装 `POST /api/v1/action-requests`
    - 透传认证主体到 submit / execute 两次 northbound 调用
    - 在 `approval_required=false` 且 `status=APPROVED` 时可选自动调用 `execute_assistant_order`
    - 返回统一用户消息、trace/order/task 引用与 auto-chain 结果
  - execute skill 负责：
    - 显式调用 `POST /api/v1/orders/{order_id}/execute`
    - 返回 task / trace / executor / order status 映射
  - 增加本地运行与联调文档，说明如何启动 Control API、如何用 Skill client 验证非审批与审批路径。
- 明确不做：
  - 不接 MCP。
  - 不新增动作。
  - 不让 Skill 直接碰 adapter、SQL、DSN、connection ref。
  - 不改变既有 `AuthorizationService` / `ExecuteAuthorizationService` / approval / audit / evidence 权威语义。
  - 不进入 Phase 05。
- 预计修改路径：
  - `internal/skill/`
  - `internal/api/`（仅当测试夹具或 API 映射需要最小补充时）
  - `design_docs/coordination/phase-04/`
  - `design_docs/coordination/00-dashboard.md`
  - `IMPLEMENTATION_STATUS.md`
  - `design_docs/28-phase-04-deep-agent-integration-notes-v0.1.md`
  - `design_docs/29-phase-04-local-runbook-v0.1.md`
- 计划验证命令：
  - `go test ./internal/skill -count=1`
  - `go test ./internal/api ./internal/application/actionrequest ./internal/skill -count=1`
  - `go test ./... -count=1`
- 请求 Claude review 的重点：
  - Deep Agent Skill 是否仍只通过高层 northbound API，而未越过 Control Layer。
  - auto-chain execute 是否仍体现为两次独立 northbound 调用。
  - prod / 审批路径是否明确不会被 Skill 偷跑 execute。
  - auto-chain 的 principal 策略是否明确且没有隐式身份漂移。
  - 本轮是否严格未接 MCP、未新增动作、未触碰 adapter 权威边界。
