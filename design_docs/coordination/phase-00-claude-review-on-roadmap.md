# Phase 00 — Claude Code Review on Roadmap & Collaboration Protocol

> **Reviewer**: Claude Code (Opus 4.6)
> **Date**: 2026-03-29
> **Scope**: Review of `phases/phase-01` through `phase-05` + `20-agent-collaboration-protocol-v0.1.md`
> **Baseline**: `03-assistant-spec-v0.7.md`, `04-interface-design-v0.8.md`, `05-control-layer-schema-v0.1.md`, `06-rbac-model-v0.1.md`, `07-adapter-interface-v0.1.md`, `08-mysql-database-create-sequence.md`

---

## 总评: PASS (with conditions)

Phase 路线图整体结构合理，协作协议机制清晰。可以进入 Phase 01 编码，但必须先解决下方 2 个 HIGH 级问题，并在对应 Phase 的 codex-plan.md 中明确回应 MEDIUM 级问题。

---

## 1. Phase 顺序审查

### 结论: 正确

| Phase | 核心目标 | 是否符合正式文档 | 说明 |
|---|---|---|---|
| 01 | 骨架与门禁 | 符合 | doc 05 §8 "先把对象模型固化，再写 handler"；doc 03 §18 "优先先做 Control Layer 的核心对象和控制流程" |
| 02 | 最小控制链路 | 符合 | doc 08 "让实现者按阶段落地，而不是一上来写一个大 handler" |
| 03 | mysql.database.create | 符合 | doc 03 §14 MVP 范围；doc 07 §3 "MVP 只实现 db_native" |
| 04 | Deep Agent 接入 | 符合 | Agent 在 Control Layer 固化之后才接入，符合 "Agent 与执行权分离" 原则 |
| 05 | MCP 接入 | 符合 | doc 07 §8.3 "现阶段先打通 Control Layer 主链路...后续再把 MCP 作为第二条执行路线接入" |

Phase 顺序严格遵循了设计文档的 "先控制面，再执行面，再 Agent 面，再多通道" 的递进逻辑。

---

## 2. 主链路遵守审查

### 结论: 正确

所有 5 个 Phase 文档都围绕权威主链路展开：

```
ActionRequest -> Principal/Asset/Policy/Risk -> AuthorizationDecision
  -> AssistantOrder -> ExecutionPlan -> Approval(if needed) -> explicit execute
  -> Plan Re-validate -> ExecutionTask -> Audit -> Evidence
```

- Phase 01 在目标中明确列出完整主链路
- Phase 02 按链路节点逐步接线
- Phase 03 在链路上接入真实 Adapter
- Phase 04 在链路北向增加 Skill 层
- Phase 05 在链路南向增加 MCP 通道

未发现任何 Phase 试图绕过或重排主链路节点。

---

## 3. 审批与 Execute 分离审查

### 结论: 正确

- Phase 01 禁止事项: "不让审批通过后自动执行"
- Phase 02 范围: approval 和 execute 各有独立小节
- Phase 02 验收标准 3: "WAITING_APPROVAL 状态下 execute 返回 APPROVAL_REQUIRED"
- Phase 04 禁止事项: "不在审批通过后默认自动执行"
- Phase 04 验收标准 2: "Agent 可自动串联 execute，但控制层仍体现为两次独立调用"

分离语义贯穿所有 Phase，未发现混写。

---

## 4. Deep Agent 位置审查

### 结论: 正确

Deep Agent 被放在 Phase 04，位于 Control Layer 骨架 (Phase 01) + 最小链路 (Phase 02) + 真实执行 (Phase 03) 之后。

Phase 04 明确约束:
- "不让 Deep Agent 直接调用 Adapter"
- "不让 Deep Agent 接触真实凭证或 connection ref"
- "不把 Policy / Risk / Approval 逻辑搬到 Agent 侧"

这完全符合 doc 03 §4.1 "Agent 与执行权分离" 和 doc 04 §1.1 "Deep Agent 只发起标准动作申请，不直接持有底层执行权限"。

---

## 5. MCP 位置审查

### 结论: 正确

MCP 在 Phase 05，是最后一个 Phase。Phase 02 禁止事项明确 "不接入 MCP / CRD / gRPC / K8s / Shell / VM 多通道"。Phase 05 的验收标准要求 MCP 路线也必须经过完整 control chain 验证。

---

## 6. Phase 01 范围审查

### 结论: 正确

Phase 01 的范围严格限定为骨架与门禁:
- 只有 domain objects / interfaces / SPI / guardrail tests
- 禁止真实执行、真实 MySQL、多 Adapter、fuzzy search
- 验收标准是 `go test ./...` 通过和接口存在性检查

---

## 7. 关键问题列表

### ISSUE-01 [HIGH] DryRun 在所有 Phase 文档中缺失

**现象**: Adapter SPI 的 `DryRun()` 方法在 doc 04 §6.2 和 doc 07 §4 中被明确定义为必要接口，但 5 个 Phase 文档中**没有任何一个**提及 DryRun。

**影响**:
- Phase 01 可能不会在 Adapter SPI skeleton 中包含 DryRun
- Phase 02 不知道 DryRun 应在 Plan Re-validate 流程中的什么位置被调用
- Phase 03 不知道 DBNativeAdapter.DryRun() 应做哪些具体检查

**正式文档要求**:
- doc 04 §6.2: `DryRun(ctx context.Context, req AdapterExecutionRequest) (AdapterDryRunResult, error)` 是 SPI 的一部分
- doc 04 §6.5: MVP 中 DB-native Adapter 的 DryRun "只校验连接、目标存在性、数据库名规范合法性"
- doc 07 §6: AdapterDryRunResult 有明确的字段定义 (supported, ready, issues, rendered_preview)

**修改建议**:
1. Phase 01 范围中增加: "Adapter SPI 骨架包含 DryRun 方法签名与 AdapterDryRunResult 类型"
2. Phase 02 范围中增加: "Plan Revalidate 可选择性地调用 Adapter.DryRun() 作为预检手段之一（Phase 02 可为 stub）"
3. Phase 03 范围中增加: "DBNativeAdapter.DryRun() 实现真实预检: 连接可达、数据库名合法、目标数据库不存在"

---

### ISSUE-02 [HIGH] 新 Phase 路线图与已有代码的关系未说明

**现象**: 仓库中已存在 39 个 Go 文件（来自之前按 doc 09 旧计划执行的 Phase 0 + Phase 1 产物），包括 domain objects、application services、adapter stubs、API skeleton、tests 等。新的 Phase 路线图（phases/phase-01 至 phase-05）和协作协议（doc 20）均未提及这些已有代码。

**影响**:
- Codex 不知道 Phase 01 是从零开始还是基于已有代码继续
- 已有代码已实现了大量 Phase 01 要求的骨架（ExecutePolicy、DryRun 签名、IdempotencyKey、ApprovalPolicy.TTL 等），但是否完全对齐新 Phase 01 的验收标准尚不确定
- 如果从零开始，已有代码的测试和结构会被浪费；如果继续构建，需要先做一次 gap analysis

**修改建议**:
1. 在 `coordination/decisions-log.md` 中新增 Decision 1，明确声明: "新 Phase 路线图基于已有代码继续 / 或从干净状态重新开始"
2. 如果继续使用已有代码，在 Phase 01 的 `codex-plan.md` 中增加一个 "已有产物盘点" 小节，列出已存在的 domain objects、interfaces、tests，然后标注哪些满足新验收标准、哪些需要补充
3. 如果重新开始，在 decisions-log 中记录原因

---

### ISSUE-03 [MEDIUM] Phase 02 未明确 RiskEngine 环境感知要求

**现象**: Phase 02 范围提到 "RiskEngine 接入统一 repository"，但未说明 RiskEngine 必须根据 asset 环境标签动态计算风险等级。

**正式文档要求**:
- doc 08 §4.2 Step 5: Risk Engine 需要判断 "是否 prod"、"目标是否高敏"、"风险等级 R1/R2/R3"
- doc 06 §4.6: mysql_operator + mysql.database.create + R1 -> allow; + R2 -> require_approval
- doc 06 §6.5: "dev/test + R1 -> allow; prod + R2 -> require approval"

**影响**: Codex 可能把 RiskEngine 实现为 hardcoded R1，导致 Phase 02 验收标准 2 ("prod 场景可走到 WAITING_APPROVAL") 无法通过。

**修改建议**: Phase 02 范围中增加: "RiskEngine.Evaluate() 必须根据 asset 的 environment 字段动态计算风险等级（dev/test -> R1, prod -> R2）"

---

### ISSUE-04 [MEDIUM] Phase 03 幂等冲突处理规则未细化

**现象**: Phase 03 范围说 "幂等键检查与冲突处理"，但未列出具体处理规则。

**正式文档要求** (doc 07 §7.5):
- 相同幂等键前次执行**已成功** -> 返回幂等成功（AlreadyCompleted），不重复执行
- 相同幂等键前次执行**进行中** -> 返回幂等冲突（InProgress），不启动新任务
- 相同幂等键前次执行**已失败** -> 允许重试

**影响**: Codex 可能只实现 "已存在则跳过" 这种简化逻辑，遗漏 InProgress 冲突检测和失败重试路径。

**修改建议**: Phase 03 范围中增加幂等处理的三种情况及其期望行为。或至少在 codex-plan.md 中明确回应。

---

### ISSUE-05 [MEDIUM] Phase 02 PLAN_STALE 路径的 Evidence 要求不够显式

**现象**: Phase 02 范围说 "EvidencePack 最小结构与查询接口"，验收标准说 "Plan Revalidate 可以触发 PLAN_STALE 且不创建任务"。但未明确要求 PLAN_STALE 路径也生成 Evidence Pack。

**正式文档要求** (doc 03 §9.4, doc 08 §9.2):
- PLAN_STALE 路径: `task_id = null`，失败原因写明 "计划失效，未启动执行任务"
- doc 05 §4.13: "PLAN_STALE 场景也要生成失败证据"

**影响**: Phase 02 可能只为成功/失败路径生成 evidence，而 PLAN_STALE 路径缺失。Phase 03 入口条件之一 ("审计和证据在真实执行路径中得到验证") 可能因此不严格。

**修改建议**: Phase 02 验收标准中增加: "PLAN_STALE 路径生成 EvidencePack（task_id = null），至少包含 stale reason"

---

### ISSUE-06 [MEDIUM] Phase 04 auto-chain 应明确 control_executor 角色使用场景

**现象**: Phase 04 范围提到 "approval_required=false 且 status=APPROVED 时允许自动串联 execute"，但未说明此时 execute 调用使用的 principal 角色应为什么。

**正式文档要求** (doc 06 §4.8, §6.7):
- `control_executor` 是 "受控后台执行主体"，用于自动化流水线
- Phase 04 的 Agent 自动串联 execute 本质上就是 "受控后台执行" 场景

**影响**: Codex 可能让 Agent 直接复用 request 阶段的 user principal 来触发 execute，而不是使用 `control_executor` 角色。这虽然在低风险场景下可能被 ExecutePolicy 放行，但概念上不够准确。

**修改建议**: Phase 04 范围中增加: "Agent 自动串联 execute 时，应通过 auth context 中的 principal 身份触发。如果是 user principal 直接触发，需确保 ExecutePolicy 允许该 principal 在该 environment 执行。如果使用代理身份，需明确使用 `control_executor` 或等价受控 service principal。"

---

### ISSUE-07 [MEDIUM] 协作协议缺少 "紧急修复" 机制

**现象**: doc 20 的交接流程是: codex-plan -> codex-status -> codex-handoff -> claude-review -> codex-fix-response -> merge-summary。完整一轮至少需要 6 个文件交接。

**影响**: 如果在 Phase 中期发现严重 bug（如主链路被破坏），仍需走完整流程。对于单行 hotfix 级别的修复，协议开销过大。

**修改建议**: 在 doc 20 中增加一个 §8.5 "快速修复路径": 当修改范围 <= 3 个文件且不涉及接口/状态机变更时，允许 Codex 在 codex-status.md 中标注 `hotfix: true` 并直接提交，Claude 在下一次常规 review 中追溯确认。

---

### ISSUE-08 [LOW] Phase 01 验收标准未提及 Skill contract structs

**现象**: Phase 01 范围第 8 点提到 "northbound DTO / skill contract"，但验收标准 7 条中没有一条检查 Skill 输入输出 struct 是否存在。

**正式文档要求** (doc 04 §4.1-4.3):
- `request_mysql_database_create` 和 `execute_assistant_order` 的输入输出模型已明确定义

**修改建议**: Phase 01 验收标准中增加: "Skill 输入输出 struct 已定义（request_mysql_database_create Input/Output, execute_assistant_order Input/Output）"

---

### ISSUE-09 [LOW] Phase 02 验收标准未明确 REQUEST_ACCEPTED 必须是第一条审计事件

**现象**: Phase 02 验收标准 7 说 "REQUEST_ACCEPTED 是第一条审计事件"，这个表述已经存在且正确。

**确认**: 无需修改。doc 08 §3.2 step 5 "追加 REQUEST_ACCEPTED 审计事件" 和 doc 05 §4.1 "进入控制主链路后，必须立刻写第一条 AuditEvent" 的要求已被准确反映。

---

### ISSUE-10 [LOW] Dashboard 模板文件内容为占位符

**现象**: `coordination/phase-01/` 下的 6 个文件（codex-plan.md 等）目前是空模板。

**确认**: 这是预期行为。协作协议定义了由 Codex 在 Phase 开始时填写。无需修改。

---

## 8. 协作协议审查

### 8.1 文件交接机制

| 检查项 | 状态 |
|--------|------|
| 交接完全基于本地文件 | 通过 |
| 每个 Phase 有固定 6 个交接文件 | 通过 |
| Dashboard 作为唯一总览页 | 通过 |
| 状态枚举清晰（planned -> in_progress -> awaiting_review -> changes_required -> accepted -> merged） | 通过 |
| decisions-log 为 append-only | 通过 |
| 进入下一阶段需要 5 个前置条件 | 通过 |
| 阻塞退出条件明确（审批与 execute 混写、Agent 获取凭证等） | 通过 |

### 8.2 职责边界

| 检查项 | 状态 |
|--------|------|
| Codex 负责产出与修复 | 通过 |
| Claude Code 负责门禁与评审 | 通过 |
| 双方均不得重新定义正式领域语义 | 通过 |
| 正式文档冲突时先写 decisions-log | 通过 |
| Codex 不能自己给自己做 gate 审批 | 通过 |

### 8.3 存在的协议风险

除 ISSUE-07（紧急修复机制缺失）外：

- 协议未定义 Claude Code review 的 **SLA**（Codex 提交 handoff 后多久必须有 review）。在实际协作中，如果 Claude review 延迟，Codex 可能空等。建议在 dashboard 中增加一个 "handoff_at" 时间戳列。
- 协议未定义当 Claude 与 Codex 对某个设计点存在分歧时的 **仲裁机制**。建议: 分歧写入 decisions-log，由项目 owner（user）最终裁定。

---

## 9. Gate 结论

### 是否允许进入 Phase 01 编码？

**允许，但需先完成以下前置动作：**

1. **[必须]** 在 Phase 01 文档或 codex-plan.md 中补充 DryRun 在 Adapter SPI 骨架中的位置（ISSUE-01）
2. **[必须]** 在 decisions-log 中记录新 Phase 路线图与已有代码的关系（ISSUE-02）
3. **[建议]** 在 Phase 02 的 codex-plan.md 中回应 ISSUE-03（RiskEngine 环境感知）和 ISSUE-05（PLAN_STALE evidence）
4. **[建议]** 在 Phase 03 的 codex-plan.md 中回应 ISSUE-04（幂等三种情况）

以上 "必须" 项完成后，Phase 01 编码可以开始。"建议" 项可以在对应 Phase 的 codex-plan.md 中回应，不阻塞 Phase 01。

---

## 10. 附录：审查覆盖矩阵

| 审查维度 | 结论 | 关键证据 |
|----------|------|----------|
| Phase 顺序合理性 | PASS | 骨架 -> 链路 -> 执行 -> Agent -> MCP，符合 doc 03/04/07 |
| 主链路遵守 | PASS | 所有 Phase 都围绕权威链路展开 |
| 审批/Execute 分离 | PASS | 所有 Phase 禁止事项和验收标准均体现 |
| Deep Agent 位置 | PASS | Phase 04，在 Control Layer 之后 |
| MCP 位置 | PASS | Phase 05，最后接入 |
| Phase 01 只做骨架 | PASS | 禁止真实执行/MySQL/多 Adapter |
| mysql.database.create 在 Phase 03 | PASS | Phase 01/02 明确禁止 |
| 协作协议文件交接 | PASS | 6 个固定文件 + dashboard + decisions-log |
| 协作协议职责边界 | PASS | Codex 产出，Claude 门禁，双方均不改语义 |
| DryRun 覆盖 | FAIL | 5 个 Phase 均未提及 |
| 已有代码衔接 | FAIL | 新旧计划关系未说明 |
| RiskEngine 环境感知 | WARN | Phase 02 隐含但未显式 |
| 幂等冲突细节 | WARN | Phase 03 提及但未细化三种情况 |
| PLAN_STALE evidence | WARN | Phase 02 提及状态但未要求 evidence |
| control_executor in Phase 04 | WARN | 未说明 auto-chain execute 的 principal 策略 |
