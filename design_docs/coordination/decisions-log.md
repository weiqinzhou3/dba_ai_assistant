# Decisions Log

## 作用

这是跨 phase 的追加型决策日志。只记录会影响后续阶段边界、接口、状态机、权限语义、审计语义或协作流程的决定。

## 记录规则

1. 采用 append-only 方式追加，不回写旧结论。
2. 只有会影响后续协作和实现方向的决定才记入本文件。
3. 若只是某一轮临时实现细节，不进入本日志，写入对应 phase 的状态或 handoff 文件即可。

## 当前状态

### Decision 1

- Date: 2026-03-29
- Phase: Phase 00 / Phase 01 pre-gate
- Proposed by: Codex
- Confirmed by: Claude Code review `phase-00-claude-review-on-roadmap.md`
- Decision: 新 Phase 路线图基于仓库中已有 Go skeleton、既有测试和现有正式文档继续推进，不从干净状态重写。
- Reason: 当前仓库已存在大量 Phase 01 级产物，包括 domain objects、application services、API skeleton、adapter stubs 和 tests。重开新骨架会浪费已存在的实现与验证结果。
- Impacted files / modules:
  - `cmd/`
  - `internal/`
  - `IMPLEMENTATION_STATUS.md`
  - `design_docs/phases/phase-01-skeleton-guardrails.md`
  - `design_docs/coordination/phase-01/codex-plan.md`
- Affected future phases: Phase 01 到 Phase 05
- Follow-up action:
  - Phase 01 开始前，Codex 必须先做“已有产物盘点 + gap analysis”
  - 盘点结果写入 `design_docs/coordination/phase-01/codex-plan.md`
  - 只有在 gap analysis 完成后，才允许把 dashboard 状态改为 `in_progress`

## 模板

### Decision N

- Date:
- Phase:
- Proposed by:
- Confirmed by:
- Decision:
- Reason:
- Impacted files / modules:
- Affected future phases:
- Follow-up action:
