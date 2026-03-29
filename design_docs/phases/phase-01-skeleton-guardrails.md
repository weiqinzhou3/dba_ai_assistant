# Phase 01 - Skeleton And Guardrails

## 目标

固化后续所有实现必须依赖的对象、接口、状态机和门禁，确保系统从第一天开始就围绕 `ActionRequest -> AuthorizationDecision -> AssistantOrder -> ExecutionPlan -> Approval/Execute -> ExecutionTask -> Audit/Evidence` 主链路建设。

## 范围

1. 模块化单体目录结构与 Go module 边界。
2. 核心领域对象：
   - `ActionRequest`
   - `Principal`
   - `ResolvedAssetSet`
   - `PolicyDecision`
   - `RiskDecision`
   - `AuthorizationDecision`
   - `AssistantOrder`
   - `ExecutionPlan`
   - `ApprovalRecord`
   - `ExecutionTask`
   - `ExecutionStep`
   - `AuditEvent`
   - `EvidencePack`
3. 核心 service interface、repository interface、adapter SPI。
4. `AuthorizationService` 作为唯一权威授权出口。
5. `ExecuteAuthorizationService` / `ExecutePolicy` 的独立建模。
6. `AssetResolver.ResolveExact(...)` guardrail。
7. 审批 API 与 execute API 的物理分离。
8. northbound DTO / skill contract / 错误码 / `trace_id` 传递。
9. Adapter SPI 骨架必须包含 `DryRun(...)` 方法签名，以及 `AdapterDryRunResult` / `AdapterExecutionResult` 类型。

## 禁止事项

1. 不实现完整 `mysql.database.create` 真实执行。
2. 不连接真实 MySQL。
3. 不实现多个 Adapter。
4. 不引入 fuzzy asset search。
5. 不让审批通过后自动执行。
6. 不把 request 权限、approval 权限和 execute 权限混成一个模型。

## 产物

1. 可编译的 skeleton。
2. 统一的 domain / application / persistence / adapter contract。
3. `DBNativeAdapter` 接口占位。
4. request / approval / execute / query API skeleton。
5. guardrail tests。
6. `IMPLEMENTATION_STATUS.md` 更新记录。

## 验收标准

1. `go test ./...` 通过。
2. `AuthorizationService` 已存在，且调用方不再自行拼接 `PolicyDecision + RiskDecision`。
3. `AssetResolver` 只有 exact match contract。
4. request / approval / execute 三条北向入口已分离。
5. 所有 northbound 返回对象带 `trace_id`。
6. 角色常量中已包含 `control_executor`。
7. `ApprovalPolicy` TTL 与 `ExecutePolicy` 已有明确模型。
8. Skill 输入输出 struct 已定义，至少覆盖：
   - `request_mysql_database_create`
   - `execute_assistant_order`
9. Adapter SPI 已显式包含 `DryRun(...)` 与 `AdapterDryRunResult`。

## 风险点

1. 只做 contract、不做共享状态接线，容易形成“结构正确、运行链路未通”的假完成。
2. 如果状态机和错误码没有在此阶段固化，后续 Phase 2/3 会反复返工。
3. 当前仓库 git/worktree 结构不标准，协作时容易误把内层 `.git` 当普通目录处理。

## 进入下一阶段条件

1. Guardrails 已落为测试，不只停留在文档。
2. 核心对象和接口已全部存在。
3. 团队确认以 Go 模块化单体继续推进，不再讨论切语言或先拆微服务。

## 推荐 branch 名

`phase/01-skeleton-guardrails`

## 推荐 commit message 模式

1. `feat(domain): add control-layer core models`
2. `feat(api): separate request approval execute endpoints`
3. `test(guardrail): lock exact asset and execute policy rules`
4. `docs(status): update implementation status for phase 01`
