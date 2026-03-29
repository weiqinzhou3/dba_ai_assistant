# Phase 00.5 - Codex Roadmap Patch

## 0. 文档作用

本文档记录 Codex 针对 Claude Code roadmap review 的修补动作，用于把“原始 review 结论”与“当前仓库可放行状态”连接起来。

输入 review 来源：

- `design_docs/coordination/phase-00-claude-review-on-roadmap.md`

---

## 1. 原始 gate 结论

原始 Claude review 的结论是：

> PASS (with conditions)

其中有 2 个必须关闭的 HIGH 问题，若不关闭，不允许进入 Phase 01 编码。

---

## 2. Codex 修补范围

本轮修补只涉及 phase 文档、协作协议、协调文件和基线文档，不涉及业务代码实现。

修补涉及的核心文件：

1. `design_docs/phases/phase-01-skeleton-guardrails.md`
2. `design_docs/phases/phase-02-min-control-flow.md`
3. `design_docs/phases/phase-03-mysql-database-create.md`
4. `design_docs/phases/phase-04-deep-agent-integration.md`
5. `design_docs/coordination/decisions-log.md`
6. `design_docs/coordination/00-dashboard.md`
7. `design_docs/coordination/phase-01/codex-plan.md`
8. `design_docs/coordination/phase-02/codex-plan.md`
9. `design_docs/coordination/phase-03/codex-plan.md`
10. `design_docs/coordination/phase-04/codex-plan.md`
11. `design_docs/20-agent-collaboration-protocol-v0.1.md`
12. `design_docs/21-current-codebase-baseline-v0.1.md`

---

## 3. 问题关闭映射

### ISSUE-01 [HIGH] DryRun 覆盖缺失

已关闭，修补方式如下：

1. Phase 01 明确要求 Adapter SPI skeleton 包含 `DryRun(...)`、`AdapterDryRunResult`、`AdapterExecutionResult`
2. Phase 02 明确要求 `Plan Revalidate` 可选择调用 `Adapter.DryRun()`，本阶段允许 stub
3. Phase 03 明确要求 `DBNativeAdapter.DryRun()` 实现真实预检，至少覆盖：
   - 连接可达
   - 数据库名合法
   - 目标数据库不存在

关闭证据：

1. `design_docs/phases/phase-01-skeleton-guardrails.md`
2. `design_docs/phases/phase-02-min-control-flow.md`
3. `design_docs/phases/phase-03-mysql-database-create.md`

### ISSUE-02 [HIGH] 新路线图与已有代码关系未说明

已关闭，修补方式如下：

1. 在 `decisions-log.md` 中新增 Decision 1，明确路线图基于现有 Go skeleton 继续推进
2. 在 `phase-01/codex-plan.md` 中增加“已有产物盘点 + gap analysis”小节
3. 新增 `21-current-codebase-baseline-v0.1.md`，把当前代码树事实固定为正式基线证据

关闭证据：

1. `design_docs/coordination/decisions-log.md`
2. `design_docs/coordination/phase-01/codex-plan.md`
3. `design_docs/21-current-codebase-baseline-v0.1.md`

### ISSUE-03 [MEDIUM] RiskEngine 环境感知要求不显式

已记录并显式化：

1. Phase 02 范围中要求 `RiskEngine.Evaluate()` 按 asset environment 动态出 `R1/R2`
2. `phase-02/codex-plan.md` 要求 Codex 在开工前明确回应该策略

### ISSUE-04 [MEDIUM] 幂等三种情况未细化

已记录并显式化：

1. Phase 03 范围中明确三种幂等情况：
   - 前次已成功
   - 前次进行中
   - 前次已失败
2. `phase-03/codex-plan.md` 要求在实现前逐条回应

### ISSUE-05 [MEDIUM] PLAN_STALE evidence 不显式

已记录并显式化：

1. Phase 02 范围和验收标准中明确：
   - `PLAN_STALE` 路径必须生成 `EvidencePack`
   - `task_id = null`
   - 至少带 stale reason
2. `phase-02/codex-plan.md` 已增加对应回应位

### ISSUE-06 [MEDIUM] Phase 04 principal 策略不明确

已记录并显式化：

1. Phase 04 范围中明确：
   - user principal 直触发 execute 时必须被 `ExecutePolicy` 显式放行
   - 使用代理身份时必须是 `control_executor` 或等价受控 service principal
2. `phase-04/codex-plan.md` 已增加回应位

### ISSUE-07 [MEDIUM] 协作协议缺少快速修复机制

已关闭，修补方式如下：

1. `20-agent-collaboration-protocol-v0.1.md` 增加 review SLA
2. 增加分歧仲裁机制
3. 增加 hotfix / 快速修复路径
4. `00-dashboard.md` 增加 `Handoff At` 字段

### ISSUE-08 [LOW] Skill contract structs 未进入 Phase 01 验收

已记录并显式化：

1. Phase 01 验收标准中增加 skill contract structs 检查
2. `phase-01/codex-plan.md` 增加明确回应位

---

## 4. 当前 patch 结论

基于上述修补动作，可以确认：

1. 原始 Claude review 的两个阻塞性 HIGH 问题都已关闭。
2. MEDIUM 问题均已被写入对应 phase 文档或 codex-plan 回应位。
3. 协作协议与 dashboard 已具备继续推进所需的收口信息。
4. 本轮没有开始 Phase 01 业务编码，也没有改变主链路原则。

最终 patch 结论：

> Phase 00.5 的文档修补已足以支撑 Claude 的有条件放行结论转为正式可收口状态。
