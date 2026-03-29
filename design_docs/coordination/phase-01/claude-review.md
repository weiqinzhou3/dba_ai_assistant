# Phase 01 - Claude Review

## 文档作用

由 Claude Code 填写，记录对本轮 handoff 的评审结论和 gate 判断。

## Review 结论

- 总结：**PASS**
- Gate 决定：**允许继续。本轮可以进入 Phase 01 收口合并，随后进入 Phase 02。**

---

## 审查范围

本次审查覆盖：

1. 分支 `feat/p1-baseline-gap-and-guardrails` 相对 `docs/phase-00.5-closeout` 的全部 diff（1 commit: `18cb756`）
2. 正式文档 `design_docs/03~08`
3. Phase 01 规范 `design_docs/phases/phase-01-skeleton-guardrails.md`
4. Codex 协调文档 `codex-plan.md` / `codex-status.md` / `codex-handoff.md`
5. 新增文档 `22-phase-01-gap-analysis-v0.1.md` / `23-phase-01-skeleton-delta-v0.1.md`
6. 所有改动的 Go 源码与测试
7. `go test ./...` 执行结果（全部通过）

---

## 7 项重点审查

### 1. 是否真的先做了 baseline inventory + gap analysis

**结论：是，已完成。**

证据：

- `design_docs/22-phase-01-gap-analysis-v0.1.md` 详细记录了对 43 个 Go 文件 / 11 个测试文件的盘点结论。
- gap analysis 对照了正式文档 `03~08` 和 Phase 01 验收标准，逐项给出 `aligned` / `gap_fixed_in_this_round` / `partial_phase_01_aligned` 判断。
- `codex-plan.md` 第一节"已有产物盘点"回答了 Phase 01 要求的所有检查项，包括 domain objects、service interfaces、API skeleton、adapter SPI、已有测试。
- 工作顺序正确：先盘点 → 识别 gap → 补齐 contract + tests → 更新 status。

### 2. 是否没有越界进入真实 mysql.database.create

**结论：未越界。**

证据：

- `internal/adapters/dbnative/adapter.go:36-49` 的 `Execute(...)` 返回 `NOT_IMPLEMENTED_IN_PHASE_1`，明确拒绝真实执行。
- `DryRun(...)` 返回 `dry_run_stub` 模式，不连接真实 MySQL。
- 新增 `adapter_test.go` 两个测试锁定了上述语义：
  - `TestDBNativeAdapterDryRunReturnsSkeletonPreview` — 确认 dry run 仍是 stub。
  - `TestDBNativeAdapterExecuteRemainsPhaseOneSkeleton` — 确认 execute 必须失败。
- 没有引入任何 MySQL driver 依赖。
- `go.mod` 未新增外部模块。

### 3. AuthorizationService 是否仍是唯一授权出口

**结论：是。**

证据：

- `internal/application/authorization/contracts.go:29-31` 定义 `AuthorizationService` interface，保持唯一。
- `internal/application/actionrequest/service.go:101` 的 `Submit(...)` 只通过 `s.authorization.Evaluate(...)` 获取授权决策，没有自行拼接 `PolicyDecision + RiskDecision`。
- `ExecuteAuthorizationService` 作为独立的 execute 路径门禁（`contracts.go:33-35`），与 request 授权物理分开。
- 本轮没有新增任何绕过 `AuthorizationService` 的授权路径。

### 4. AssetResolver 是否仍是 exact match

**结论：是。**

证据：

- `internal/application/authorization/contracts.go:13-15` 仅暴露 `ResolveExact(...)`。
- 没有新增 `ResolveFuzzy` / `ResolveBestEffort` / `ResolveAlias` 等方法。
- 本轮代码 diff 未触碰 `AssetResolver` 接口或实现。

### 5. Approval 与 Execute 是否仍分离

**结论：是，且新增了测试锁定。**

证据：

- `internal/api/server.go:37-38` 保持两条独立路由：
  - `POST /api/v1/orders/{order_id}/approvals` → `handleDecideApproval`
  - `POST /api/v1/orders/{order_id}/execute` → `handleExecuteOrder`
- `handleDecideApproval` 调用 `s.approvals.Decide(...)`，不触碰 `s.actionRequests.ExecuteApprovedOrder(...)`。
- `handleExecuteOrder` 调用 `s.actionRequests.ExecuteApprovedOrder(...)`，不触碰 `s.approvals`。
- 新增测试 `TestServerRoutesApprovalsWithoutCallingExecute` 使用 `stubApprovalService` + `stubActionRequestService` 的 `executeCalled` flag，**硬编码锁定** approval 路由不会误触 execute 路径。

### 6. Audit/Evidence 是否已进入骨架

**结论：已进入 Phase 01 级骨架，边界表述准确。**

证据：

- `AuditService` interface 和 `MemoryService` 实现已存在。`Submit(...)` 已写入 `REQUEST_ACCEPTED` / `AUTHORIZATION_DECIDED` / `ORDER_CREATED` / `PLAN_FROZEN` 四个审计事件。
- `EvidenceService` interface 和 `MemoryService` 实现已存在。`Build(...)` 和 `GetByOrderID(...)` 已接入 `trace_id` 传递。
- 本轮新增 `audit/memory_service_test.go` 和 `evidence/memory_service_test.go`，锁定 `trace_id` 在 audit/evidence 查询链路中的传播。
- `IMPLEMENTATION_STATUS.md` 和 gap analysis 均正确表述了 Phase 01 含义："contract + query skeleton + trace contract 已固定"，**没有** 误报为"执行结束落账链路已完成"。
- execute 后的 success/failure/`PLAN_STALE` evidence build 明确标注为 Phase 02 工作。

### 7. 是否可以继续进入下一轮 Phase 01 收口，或者进入 Phase 02

**结论：可以进入 Phase 02。**

依据（对照 Phase 01 验收标准逐项确认）：

| 验收标准 | 状态 | 证据 |
|---|---|---|
| `go test ./...` 通过 | PASS | 11 个包全部通过 |
| `AuthorizationService` 已存在且调用方不自行拼接 | PASS | `service.go:101` 只调 `s.authorization.Evaluate(...)` |
| `AssetResolver` 只有 exact match contract | PASS | `contracts.go:14` 仅 `ResolveExact` |
| request / approval / execute 三条北向入口已分离 | PASS | `server.go:34-38` 三条独立路由 + 测试锁定 |
| 所有 northbound 返回对象带 `trace_id` | PASS | 本轮补齐全部出口 + `TestServerReturnsTraceIDOnNorthboundResponses` |
| 角色常量中包含 `control_executor` | PASS | 已在先前轮次确认 |
| `ApprovalPolicy` TTL 与 `ExecutePolicy` 已有明确模型 | PASS | 已在先前轮次确认 |
| Skill 输入输出 struct 已定义 | PASS | `skill/contracts.go` 含两个 skill + `trace_id` 映射 |
| Adapter SPI 已包含 `DryRun(...)` 与 `AdapterDryRunResult` | PASS | `execution/contracts.go:46-47` + `adapter_test.go` 锁定 |

Phase 01 进入下一阶段的三个条件：

1. **Guardrails 已落为测试** — PASS（13 个 guardrail test 覆盖）
2. **核心对象和接口已全部存在** — PASS（gap analysis 3.1~3.7 逐项确认）
3. **团队确认以 Go 模块化单体继续推进** — PASS（从 Phase 00.5 开始已确认）

---

## 阻塞问题

无。

---

## 非阻塞问题（建议性）

### NB-1: `design_docs/21-current-codebase-baseline-v0.1.md` 被删除

本轮删除了 Phase 00.5 产出的 `21-current-codebase-baseline-v0.1.md`。该文件的核心内容已被 `22-phase-01-gap-analysis-v0.1.md` 吸收和升级，因此信息未丢失。但作为 Phase 00.5 的正式产物，建议在 Phase 01 merge summary 中注明此文件被 `22` 替代，以便追溯。

严重级别：**informational**，不阻塞。

### NB-2: Phase 00.5 协调文档被删除

三个 Phase 00.5 协调文档被删除：

- `phase-00.5-claude-review.md`
- `phase-00.5-codex-roadmap-patch.md`
- `phase-00.5-merge-summary.md`

这些文档的删除可能是有意为之（Phase 00.5 已关闭，内容已合并到主链路文档中）。建议确认是刻意清理还是误删。若是刻意清理，在 merge summary 中记录即可。

严重级别：**informational**，不阻塞。

### NB-3: `approval` 包无测试文件

`internal/application/approval/` 目录当前无测试文件（`[no test files]`）。虽然 `NoopService` 的行为已被 `server_test.go` 间接测试（通过 `stubApprovalService`），但 approval 包本身未被直接测试。这不阻塞 Phase 01，但 Phase 02 接入共享状态机时需要补上。

严重级别：**low**，Phase 02 前置工作。

### NB-4: `MemoryAuditService.GetViewByRequestID` 的 trace_id 取值策略

当前实现（`audit/memory_service.go:43-48`）取事件流中第一个非空 `trace_id`。这在 Phase 01 是合理的（所有事件共享同一 trace），但 Phase 02 需要考虑跨 trace 场景（如 retry / re-execute）。建议在 Phase 02 plan 中标注。

严重级别：**low**，Phase 02 设计决策。

---

## Phase 01 验收标准对照表

| # | 验收标准 | 结论 |
|---|---|---|
| 1 | `go test ./...` 通过 | PASS |
| 2 | `AuthorizationService` 是唯一授权出口 | PASS |
| 3 | `AssetResolver` 仅 exact match | PASS |
| 4 | request / approval / execute 三条入口已分离 | PASS |
| 5 | northbound 返回对象带 `trace_id` | PASS |
| 6 | 角色常量含 `control_executor` | PASS |
| 7 | `ApprovalPolicy` TTL 与 `ExecutePolicy` 已建模 | PASS |
| 8 | Skill 输入输出 struct 已定义 | PASS |
| 9 | Adapter SPI 含 `DryRun(...)` 与 `AdapterDryRunResult` | PASS |

---

## 最终决定

```
Gate:    PASS
Action:  允许 Phase 01 收口合并，随后进入 Phase 02
```

Phase 02 启动前建议关注：

1. NB-1/NB-2 的删除文件在 merge summary 中做记录。
2. Phase 02 plan 中标注 approval 包测试补齐和 audit trace_id 策略演进。
3. Phase 02 的重点应是：persistence 接线 → approval 共享状态 → execute revalidate → TaskRuntime → evidence build，严格按 `IMPLEMENTATION_STATUS.md` 的"下一轮最合理的工作"推进。

---

## 审查元数据

- 审查人：Claude Code (Opus 4.6)
- 审查时间：2026-03-29
- 审查分支：`feat/p1-baseline-gap-and-guardrails` (commit `18cb756`)
- 基线分支：`docs/phase-00.5-closeout`
- 验证命令：`go test ./...` — 全部通过
