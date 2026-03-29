# Agent Coordination Dashboard

## 作用

这是 Codex / Claude Code 协作的唯一总览页。所有 phase 状态、当前分支、最近交接文件、评审结论和下一阶段门禁，都必须在这里可见。

## 更新规则

1. Codex 在开始一个 phase、更新状态、提交 handoff 时更新对应行。
2. Claude Code 在完成 review 后更新评审结论与 gate 状态。
3. 进入下一阶段前，必须先在本文件把上一阶段状态改为 `accepted`。
4. Codex 提交 handoff 时必须填写 `Handoff At` 时间戳，用于跟踪 review SLA。

## 当前总览

## Phase 00.5 Closeout

| Stage | 目标摘要 | 状态 | 当前分支 | Baseline | Codex Patch | Claude Review | Merge Summary | Gate |
|---|---|---|---|---|---|---|---|---|
| Phase 00.5 | 路线图、协作协议与现有代码基线收口 | accepted | `docs/phase-00.5-closeout` | [../21-current-codebase-baseline-v0.1.md](../21-current-codebase-baseline-v0.1.md) | [phase-00.5-codex-roadmap-patch.md](phase-00.5-codex-roadmap-patch.md) | [phase-00.5-claude-review.md](phase-00.5-claude-review.md) | [phase-00.5-merge-summary.md](phase-00.5-merge-summary.md) | Phase 00.5 已关闭，允许进入 Phase 01 |

当前 gate 结论：

> Phase 00.5 已关闭。Phase 01 已基于最新 `main` 完成 closeout-v2，并被标记为 `accepted`。
>
> `ready_for_next_phase = true`
>
> `next_phase = phase-02`

| Phase | 目标摘要 | 状态 | 当前分支 | Handoff At | Codex Plan | Codex Status | Codex Handoff | Claude Review | Codex Fix | Merge Summary |
|---|---|---|---|---|---|---|---|---|---|---|
| Phase 01 | 骨架与门禁 | accepted | `docs/phase-01-closeout-v2` | 2026-03-29 16:18:18 CST | [phase-01/codex-plan.md](phase-01/codex-plan.md) | [phase-01/codex-status.md](phase-01/codex-status.md) | [phase-01/codex-handoff.md](phase-01/codex-handoff.md) | [phase-01/claude-review.md](phase-01/claude-review.md) | [phase-01/codex-fix-response.md](phase-01/codex-fix-response.md) | [phase-01/merge-summary.md](phase-01/merge-summary.md) |
| Phase 02 | 最小控制链路 | planned |  |  | [phase-02/codex-plan.md](phase-02/codex-plan.md) | [phase-02/codex-status.md](phase-02/codex-status.md) | [phase-02/codex-handoff.md](phase-02/codex-handoff.md) | [phase-02/claude-review.md](phase-02/claude-review.md) | [phase-02/codex-fix-response.md](phase-02/codex-fix-response.md) | [phase-02/merge-summary.md](phase-02/merge-summary.md) |
| Phase 03 | mysql.database.create MVP | planned |  |  | [phase-03/codex-plan.md](phase-03/codex-plan.md) | [phase-03/codex-status.md](phase-03/codex-status.md) | [phase-03/codex-handoff.md](phase-03/codex-handoff.md) | [phase-03/claude-review.md](phase-03/claude-review.md) | [phase-03/codex-fix-response.md](phase-03/codex-fix-response.md) | [phase-03/merge-summary.md](phase-03/merge-summary.md) |
| Phase 04 | Deep Agent 接入 | planned |  |  | [phase-04/codex-plan.md](phase-04/codex-plan.md) | [phase-04/codex-status.md](phase-04/codex-status.md) | [phase-04/codex-handoff.md](phase-04/codex-handoff.md) | [phase-04/claude-review.md](phase-04/claude-review.md) | [phase-04/codex-fix-response.md](phase-04/codex-fix-response.md) | [phase-04/merge-summary.md](phase-04/merge-summary.md) |
| Phase 05 | MCP 接入 | planned |  |  | [phase-05/codex-plan.md](phase-05/codex-plan.md) | [phase-05/codex-status.md](phase-05/codex-status.md) | [phase-05/codex-handoff.md](phase-05/codex-handoff.md) | [phase-05/claude-review.md](phase-05/claude-review.md) | [phase-05/codex-fix-response.md](phase-05/codex-fix-response.md) | [phase-05/merge-summary.md](phase-05/merge-summary.md) |

## 状态枚举

- `planned`: 文档已建，但还未进入实际工作
- `in_progress`: Codex 已开始本 phase
- `awaiting_review`: Codex 已提交 handoff，等待 Claude review
- `changes_required`: Claude review 给出阻塞问题，等待 Codex 修复
- `accepted`: Claude review 已通过，允许进入 merge / 下一阶段
- `merged`: 该 phase 已合入主干
