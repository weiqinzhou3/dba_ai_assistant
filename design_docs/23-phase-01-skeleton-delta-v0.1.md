# Phase 01 Skeleton Delta v0.1

## 0. 文档定位

本文档记录本轮相对 Phase 00.5 基线新增或收口的 Phase 01 级别改动。

重点是说明：

1. 哪些变更属于“骨架与门禁补齐”
2. 哪些内容被明确保留为后续 phase 工作

---

## 1. 代码侧 delta

### 1.1 northbound trace_id contract 收口

本轮新增或补齐 `trace_id` 的对象：

1. `internal/domain/order/types.go`
2. `internal/domain/task/types.go`
3. `internal/application/approval/contracts.go`
4. `internal/application/audit/contracts.go`
5. `internal/application/evidence/contracts.go`
6. `internal/skill/contracts.go`

配套最小接线：

1. `ActionRequestService.Submit(...)` 写入 `AssistantOrder.trace_id`
2. `NoopTaskRuntime.Start(...)` 预留 `ExecutionTask.trace_id`
3. `NoopApprovalService.Create(...)` 回填 `trace_id`
4. `MemoryAuditService.GetViewByRequestID(...)` 从事件流回填 `trace_id`
5. `MemoryEvidenceService.Build(...)` / `GetByOrderID(...)` 保留 `trace_id`

### 1.2 guardrail tests 补强

本轮新增测试覆盖：

1. northbound query / approval / audit / evidence 响应都带 `trace_id`
2. `ActionRequestService.GetOrder(...)` 返回提交时生成的 `trace_id`
3. skill output mapping 保留 `trace_id`
4. approval route 不触发 execute 路径
5. `DBNativeAdapter.DryRun(...)` 仍是 stub preview
6. `DBNativeAdapter.Execute(...)` 仍是 Phase 01 skeleton，不进入真实执行

---

## 2. 文档侧 delta

本轮新增：

1. `design_docs/22-phase-01-gap-analysis-v0.1.md`
2. `design_docs/23-phase-01-skeleton-delta-v0.1.md`

本轮更新：

1. `design_docs/coordination/phase-01/codex-plan.md`
2. `design_docs/coordination/phase-01/codex-status.md`
3. `design_docs/coordination/phase-01/codex-handoff.md`
4. `design_docs/coordination/00-dashboard.md`
5. `IMPLEMENTATION_STATUS.md`

---

## 3. 明确保留到后续 phase 的内容

以下内容未在本轮推进，且这是刻意边界，不是遗漏：

1. `ApprovalService` 共享状态与真实状态迁移
2. execute 前 `Plan Revalidate`
3. `PLAN_STALE` 状态推进与证据生成
4. `ExecutionTask` 创建与运行时推进
5. append-only audit repository
6. 真实 `DBNativeAdapter` MySQL 连接与 `CREATE DATABASE`

---

## 4. 总结

本轮 delta 的本质是：

> 把已有 skeleton 从“结构大体正确”收口到“Phase 01 约束被代码和测试明确锁定”，同时避免越界进入 Phase 02/03。
