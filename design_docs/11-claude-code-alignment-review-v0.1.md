# 11 — Claude Code Alignment Review v0.1

> **Author**: Claude Code (Opus 4.6)
> **Date**: 2026-03-28
> **Scope**: Review of `10-implementation-alignment-v0.1.md` against all design docs (00–09 + upm/*)
> **Purpose**: 供 Codex 在后续 Phase 1–5 执行前阅读，补齐 Phase 0 alignment 中的遗漏

---

## 1. 总体评价

你（Codex）在 `10-implementation-alignment-v0.1.md` 中对项目核心架构的理解是 **基本正确的**：

- 权威主链正确
- Agent-Execution 分离正确
- "审批 ≠ 执行" 正确
- Asset Resolver 精确匹配正确
- AuthorizationService 唯一权威正确
- Go 模块化单体方向正确

但以下 **11 处关键约束** 你未提及或理解不足。在继续 Phase 1–5 之前，请逐条确认你已理解，并在实现中体现。

---

## 2. 必须补齐的约束（按严重程度排序）

### 2.1 [高] ExecutePolicy 是独立策略对象

**来源**: `06-rbac-model-v0.1.md`

RBAC 模型定义了 5 层策略，其中 ExecutePolicy 是独立于 ActionPolicy 的：

- `ActionPolicy` 控制 "谁能请求哪些 Action"
- `ExecutePolicy` 控制 "谁能触发已审批 Order 的执行"

这两个是 **不同的权限判定**。你的 alignment 中只笼统提到了 execute 权限检查，但未将 ExecutePolicy 作为独立对象建模。

**要求**:
- Phase 1: `ExecutePolicy` 必须作为独立 domain object 出现在 `internal/domain/` 中
- Phase 2: `AuthorizationService` 在处理 execute 请求时，必须单独查询 ExecutePolicy，而非复用 ActionPolicy 的判定结果
- 允许执行的角色包括: `mysql_operator`, `platform_admin`, `control_executor`

---

### 2.2 [高] Adapter 幂等键设计

**来源**: `07-adapter-interface-v0.1.md`, `08-mysql-database-create-sequence.md`

每个 Adapter 调用必须携带幂等键，格式为：

```
<action_name>:<target_asset_id>:<distinguishing_params>
```

MVP 示例：
```
mysql.database.create:asset-mysql-prod-01:order_db
```

幂等冲突处理规则：
- 如果相同幂等键的前次执行 **已成功** → 返回 `AlreadyCompleted`，不重复执行
- 如果相同幂等键的前次执行 **进行中** → 返回 `InProgress`，不启动新任务
- 如果相同幂等键的前次执行 **已失败** → 允许重试

**要求**:
- Phase 1: `AdapterRequest` struct 必须包含 `IdempotencyKey string` 字段
- Phase 3: `DBNativeAdapter.Execute()` 实现中必须检查幂等键

---

### 2.3 [高] Adapter DryRun 能力

**来源**: `07-adapter-interface-v0.1.md`

Adapter SPI 定义了 4 个方法，不是 3 个：

```go
type Adapter interface {
    Type() AdapterType
    Supports(action string) bool
    DryRun(ctx context.Context, req AdapterRequest) (*DryRunResult, error)
    Execute(ctx context.Context, req AdapterRequest) (*AdapterResult, error)
}
```

`DryRun()` 的职责（以 DBNativeAdapter 为例）：
- 验证目标连接可达
- 验证 database name 合法性（字符集、长度、保留字）
- 验证目标 database 不存在（防止覆盖）
- **不执行任何写操作**

**要求**:
- Phase 1: Adapter interface 必须包含 `DryRun` 方法
- Phase 2: Control main chain 在生成 ExecutionPlan 前应调用 DryRun 进行预检
- Phase 3: DBNativeAdapter 实现真实的 DryRun 逻辑

---

### 2.4 [中] Approval TTL 语义与过期扫描

**来源**: `03-assistant-spec-v0.7.md`, `04-interface-design-v0.8.md`

审批不是无限期等待的。设计要求：

1. `ApprovalPolicy` 对象包含 `approval_ttl` 字段（例如 `72h`）
2. 如果 TTL 到期且未审批 → Order 状态变为 `APPROVAL_EXPIRED`
3. 必须有后台调度器（scheduler/cron）定期扫描过期审批
4. 过期时必须写入 `APPROVAL_EXPIRED` 审计事件
5. 过期的 Order 不可再被审批或执行

**要求**:
- Phase 1: `ApprovalPolicy` struct 包含 `TTL time.Duration`；`AssistantOrder` 的状态枚举包含 `APPROVAL_EXPIRED`
- Phase 2: 实现过期扫描逻辑（可以是简单的定时任务）
- Phase 4: 过期路径必须生成审计事件

---

### 2.5 [中] 风险等级映射规则

**来源**: `design_docs/upm/03-action-to-execution-mapping.md`, `08-mysql-database-create-sequence.md`

风险等级不是固定的，而是由 RiskEngine 根据上下文计算：

| 条件 | 风险等级 |
|------|---------|
| `mysql.database.create` 默认 | R1 (低风险写) |
| 目标环境为 production | 提升到 R2 (中风险写) |
| R2 及以上 | 需要审批 |
| R1 | 可自动执行（无需审批） |

**要求**:
- Phase 1: `RiskDecision` struct 中的 `risk_level` 不能是硬编码常量
- Phase 2: RiskEngine 必须接受 asset 的环境标签（`env: prod/staging/dev`）作为输入，动态计算风险等级

---

### 2.6 [中] control_executor 角色

**来源**: `06-rbac-model-v0.1.md`

系统中存在三类执行主体：

| 角色 | 说明 |
|------|------|
| `mysql_operator` | DBA 人工触发执行 |
| `platform_admin` | 平台管理员触发执行 |
| `control_executor` | **受控后端执行主体**（用于自动化流水线） |

`control_executor` 是关键角色——它允许系统在审批通过后，由自动化流程（而非人工）触发执行。这对于未来 Deep Agent 自动链式调用（request → approve → execute）至关重要。

**要求**:
- Phase 1: 角色枚举中必须包含 `control_executor`
- Phase 2: ExecutePolicy 判定逻辑必须识别此角色

---

### 2.7 [中] PLAN_STALE 路径的 Evidence Pack

**来源**: `03-assistant-spec-v0.7.md`, `08-mysql-database-create-sequence.md`

`PLAN_STALE` 是一个 **独立的终态**，不等于 `FAILED`。当执行前重新验证发现条件变化时：

1. Order 状态 → `PLAN_STALE`（不是 `FAILED`）
2. **必须生成 Evidence Pack**，其中 `task_id = null`（因为从未实际执行）
3. Evidence Pack 中必须记录 stale 的原因（例如 "asset offline", "permission revoked"）
4. 必须写入 `PLAN_STALE` 审计事件

**要求**:
- Phase 1: Order 状态枚举中 `PLAN_STALE` 与 `FAILED` 是两个独立值
- Phase 4: PLAN_STALE 路径必须有独立的 evidence 生成逻辑

---

### 2.8 [中] Skill 函数签名与北向 API 映射

**来源**: `04-interface-design-v0.8.md`

设计定义了两个 Skill（供 Deep Agent 调用）：

```
Skill 1: request_mysql_database_create
  输入: target_asset_id, database_name, character_set?, collation?
  输出: order_id, order_status, requires_approval (bool)

Skill 2: execute_assistant_order
  输入: order_id
  输出: task_id, task_status
```

Deep Agent 的链式调用逻辑：
- 调用 Skill 1 → 如果 `requires_approval = false` → 自动调用 Skill 2
- 调用 Skill 1 → 如果 `requires_approval = true` → 通知用户等待审批

Skill 是 Deep Agent 和 Control Layer 之间的 **契约接口**。HTTP API 是 Skill 的底层实现，但 Skill 的函数签名才是 Agent 看到的接口。

**要求**:
- Phase 1: 在 `internal/skill/` 或 `internal/api/` 中定义这两个 Skill 的输入输出 struct
- Phase 2: HTTP handler 实现时，确保响应体与 Skill 输出 struct 一致

---

### 2.9 [低] REQUEST_ACCEPTED 是第一个审计事件

**来源**: `08-mysql-database-create-sequence.md`

进入 Control Layer 后，**第一件事** 就是写入 `REQUEST_ACCEPTED` 审计事件，然后才进行 Principal 解析、Asset 解析等。这保证即使后续步骤全部失败，系统也有记录"收到了这个请求"。

**要求**:
- Phase 2: Control main chain 入口处，在任何业务逻辑之前写入 `REQUEST_ACCEPTED`

---

### 2.10 [低] UPM 逆向工程的设计启示

**来源**: `design_docs/upm/*`

你不需要依赖 UPM 运行时，但 UPM 的以下设计模式影响了本项目的 domain model：

- **Asset 分层**: Cluster → Zone → NodeGroup → Node → StorageClass（理解为什么 AssetResolver 需要精确匹配而非模糊搜索）
- **Service 聚合**: ServGroup → Serv → Unit（理解为什么 target_asset_id 指向的是一个 service 实例而非裸机器）
- **Order 模型**: OrderGroup → Order，带有 pre_state / current_state（理解为什么 AssistantOrder 需要快照执行时的条件）
- **Task 模型**: Task → Subtask，支持优先级、超时、心跳（理解为什么 ExecutionTask 需要 timeout 和 heartbeat）
- **CRD 适配**: tesseract-cube 和 kauntlet 的 CRD（理解未来 CRDAdapter 要对接的目标格式）

这些不需要你现在实现，但在设计 domain object 的字段时，请确保预留扩展点。

---

### 2.11 [低] 文档路径修正

你在 `09-codex-phased-execution-manual-v0.1.md` 中引用的路径是 `docs/`，但实际文件位于 `design_docs/`。请在后续工作中使用正确路径。

---

## 3. Phase Gate 补充检查项

在你原有的 Phase Gate 基础上，每个 Phase 增加以下检查：

### Phase 1 Gate 补充
- [ ] `ExecutePolicy` 作为独立 domain object 存在
- [ ] `AdapterRequest` 包含 `IdempotencyKey` 字段
- [ ] Adapter interface 包含 `DryRun` 方法签名
- [ ] `ApprovalPolicy` 包含 `TTL` 字段
- [ ] Order 状态枚举包含 `APPROVAL_EXPIRED` 和 `PLAN_STALE`（两个独立值）
- [ ] 角色枚举包含 `control_executor`
- [ ] Skill 输入输出 struct 已定义（`request_mysql_database_create`, `execute_assistant_order`）

### Phase 2 Gate 补充
- [ ] AuthorizationService 处理 execute 请求时查询 ExecutePolicy（非 ActionPolicy）
- [ ] RiskEngine 根据 asset 环境标签动态计算风险等级
- [ ] Control main chain 入口写入 `REQUEST_ACCEPTED` 审计事件
- [ ] DryRun 在 ExecutionPlan 生成前被调用
- [ ] 审批过期扫描逻辑存在（至少是 stub）

### Phase 3 Gate 补充
- [ ] DBNativeAdapter.Execute() 检查幂等键
- [ ] DBNativeAdapter.DryRun() 实现真实预检逻辑
- [ ] `control_executor` 角色可触发执行

### Phase 4 Gate 补充
- [ ] PLAN_STALE 路径生成 Evidence Pack（task_id = null）
- [ ] APPROVAL_EXPIRED 路径生成审计事件
- [ ] 成功路径、失败路径、PLAN_STALE 路径均有独立的 evidence 生成逻辑

---

## 4. 执行指令

Codex，请按以下顺序操作：

1. **阅读本文档全文**，逐条确认你理解每个约束
2. **更新你的 `10-implementation-alignment-v0.1.md`**，在末尾追加一个 "Alignment Addendum" 章节，逐条回应 §2 中的 11 个约束
3. **按原有 Phase 1–5 计划继续执行**，但在每个 Phase Gate 处增加 §3 中对应的检查项
4. 如果你对某个约束有疑问或认为设计文档之间存在矛盾，**在代码注释中标注 `// REVIEW:`**，不要自行决定

---

## 5. 不变量提醒

以下是贯穿所有 Phase 的不变量，再次强调：

1. **AuthorizationService 是唯一授权权威** — Policy + Risk + Exemptions 汇聚于此，其他任何组件不做授权判定
2. **审批 ≠ 执行** — 审批将 Order 移至 APPROVED；执行需要独立的 `POST /execute` 触发
3. **Asset Resolver 精确匹配** — 无模糊搜索、无猜测、无 fallback
4. **Evidence Pack 覆盖所有终态** — SUCCESS, FAILED, PLAN_STALE 均必须生成
5. **自审批禁止** — approver_id ≠ requester_id
6. **审计覆盖所有路径** — 包括正常路径、失败路径、过期路径、stale 路径
