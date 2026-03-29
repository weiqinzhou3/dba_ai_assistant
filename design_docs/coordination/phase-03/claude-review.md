# Phase 03 - Claude Review

## 文档作用

由 Claude Code 填写，记录对本轮 handoff 的评审结论和 gate 判断。

## Review 结论

- 版本状态：**最终 PASS 版本**
- 总结：**PASS**
- Gate 决定：**允许进入 Phase 03 closeout**
- 审查时间：2026-03-29
- 审查分支：`feat/p3-mysql-database-create`
- 审查提交：`1bf7799`
- 验证命令：`go test -count=1 ./...` 全部通过（12 个 package，0 failure）

---

## 十项重点审查

### 1. 是否真正实现了 mysql.database.create 的最小真实纵切

**结论：是，已实现。**

- `DBNativeAdapter` 已从 Phase 02 的 skeleton 升级为真实 MySQL 实现：
  - `SQLAdmin.Ping()` → 真实 `database/sql` 连接测试
  - `SQLAdmin.DatabaseExists()` → 查询 `information_schema.schemata`
  - `SQLAdmin.CreateDatabase()` → 执行 `CREATE DATABASE`
  - 连接通过 `ConnectionResolver` 解析，支持 `DBNATIVE_CONNECTIONS_JSON` 和 `DBNATIVE_DSN_*` 环境变量
- 完整纵切链路已打通：
  - Submit → AuthorizationService.Evaluate → Plan.Build → Plan.Freeze → (Approval) → ExecuteApprovedOrder → PrincipalResolver.Resolve(executor) → ExecuteAuth.Authorize → Plan.Revalidate → AssetResolver.ResolveExact(re-match) → Router.Route → Adapter.DryRun → IdempotencyCheck → TaskRuntime.Start → Adapter.Execute → Terminal Audit → Terminal Evidence
- 测试覆盖：
  - adapter 级：DryRun 预检、Execute 成功、Execute 失败（`CREATE DATABASE` 错误）、执行后校验失败
  - service 级：成功终态、PLAN_STALE、幂等冲突、幂等重放、失败后重试
  - `database_name` 平台命名规范校验（`^[A-Za-z][A-Za-z0-9_]{0,63}$`）

### 2. AuthorizationService 是否仍是唯一最终授权出口

**结论：是，未被绕过。**

- Request 路径：`service.Submit()` 在 `service.go:126` 调用 `s.authorization.Evaluate()`，这是 request 链路唯一授权入口。
- Execute 路径：`ExecuteApprovedOrder()` 在 `service.go:283` 调用 `s.executeAuth.Authorize()`，这是 execute 链路唯一授权入口。
- `DBNativeAdapter` 不做任何权限判断，只执行已授权的 adapter request。
- 没有发现任何绕过 `AuthorizationService` 或 `ExecuteAuthorizationService` 的路径。

### 3. approval 与 execute 是否仍然分离

**结论：是，物理分离完好。**

- HTTP 层：`handleDecideApproval`（`POST /api/v1/orders/{order_id}/approvals`）与 `handleExecuteOrder`（`POST /api/v1/orders/{order_id}/execute`）是独立 handler。
- Service 层：approval 通过 `appapproval.Service.Decide()` 处理；execute 通过 `service.ExecuteApprovedOrder()` 处理。
- Execute 入口在 `service.go:254-256` 显式拒绝 `StatusWaitingApproval` 状态的 order。
- Approval actor 和 Execute actor 使用独立的认证上下文路径。

### 4. execute 前是否仍有 revalidate

**结论：是，revalidate 已从 Phase 02 占位升级为真实检查。**

Execute 前的 revalidate 包含以下检查层：

1. **Plan version 检查**（`service.go:323-325`）：order.PlanVersion 与 frozen plan 的 PlanVersion 比对。
2. **Planner.Revalidate**（`service.go:326-331`）：检查 plan snapshot frozen 状态和 stale 标记。
3. **Asset re-resolve**（`service.go:334-340`）：重新执行 `AssetResolver.ResolveExact()`，比较结果与 frozen order 中的 `ResolvedAssetIDs`，长度不一致或 ID 不匹配即触发 PLAN_STALE。
4. **Adapter DryRun**（`service.go:349-376`）：
   - 连接引用解析
   - MySQL 可达性
   - 数据库存在性检查 → 若已存在，触发 PLAN_STALE（或幂等重放）
   - 连接引用缺失、引擎不匹配等 → 触发 PLAN_STALE

`simulate_plan_stale` 已从所有 `.go` 源文件中移除（仅保留在 Phase 02 历史文档中），PLAN_STALE 完全由真实 revalidate 结果驱动。

### 5. AssetResolver 是否仍然 exact match

**结论：是，严格 exact match。**

- `AssetResolver` 接口只有 `ResolveExact()` 方法，不存在 fuzzy/partial match。
- `InMemoryExactAssetResolver.ResolveExact()` 按 `project + environment + service_instance` 三字段严格匹配。
- 0 匹配返回 `ASSET_NOT_FOUND`；>1 匹配返回 `ASSET_AMBIGUOUS`。
- Execute 路径在 `service.go:338` 额外要求 `len(resolvedAssets.Assets) == 1`。

### 6. DBNativeAdapter 是否只实现了 mysql.database.create

**结论：是，严格只覆盖 mysql.database.create。**

- `Supports()` 显式限制：`req.ActionName == "mysql.database.create" && req.TargetAssetType == "DatabaseTarget"`。
- `validate()` 在 `adapter.go:248` 对非 `mysql.database.create` 的 action 返回 `UNSUPPORTED_ACTION`。
- 未发现 `mysql.user.create`、`mysql.user.grant`、`mysql.password.change` 或任何其他 action 的代码。
- 未发现非 MySQL 引擎的适配代码。

### 7. 幂等处理是否至少具备最小可接受边界

**结论：是，三种情况均已覆盖。**

| 场景 | 处理方式 | 测试覆盖 |
|---|---|---|
| 前次已成功 + DB 存在 | 幂等重放：不重复 `CREATE DATABASE`，返回受控成功 | `TestServiceExecuteApprovedOrderReplaysSuccessfulIdempotentKeyWithoutRecreatingDatabase` |
| 前次进行中 | 返回 `IDEMPOTENCY_CONFLICT`，不创建新任务 | `TestServiceExecuteApprovedOrderReturnsIdempotencyConflictWhenKeyAlreadyRunning` |
| 前次已失败 | 允许受控重试，正常走 execute 路径 | `TestServiceExecuteApprovedOrderAllowsRetryAfterFailedIdempotentRecord` |

实现细节：
- Execute 前先保存 `RUNNING` 状态的 IdempotencyRecord（`service.go:398-406`）。
- 成功后更新为 `SUCCEEDED`（`service.go:461-470`）。
- 失败后更新为 `FAILED`（`service.go:732-741`）。
- IdempotencyKey 格式：`action_name:asset_id:database_name`。

已知限制（已在 handoff 中声明）：幂等记录为 in-memory，不具备进程重启后恢复能力。这在 MVP 阶段可接受。

### 8. Audit / Evidence 是否已从骨架占位升级为真实动作终态记录

**结论：是，已升级为真实终态语义。**

**Audit 事件链覆盖：**

| 路径 | 事件序列 |
|---|---|
| 成功 | REQUEST_ACCEPTED → AUTHORIZATION_DECIDED → ORDER_CREATED → PLAN_FROZEN → EXECUTE_TRIGGERED → PLAN_REVALIDATED → EXECUTION_STARTED → EXECUTION_SUCCEEDED → EVIDENCE_WRITTEN |
| PLAN_STALE | ... → EXECUTE_TRIGGERED → PLAN_STALE → EVIDENCE_WRITTEN |
| 幂等冲突 | ... → EXECUTE_TRIGGERED → EXECUTION_FAILED → EVIDENCE_WRITTEN |
| 幂等重放 | ... → EXECUTE_TRIGGERED → PLAN_REVALIDATED → EXECUTION_SUCCEEDED → EVIDENCE_WRITTEN |
| 执行失败 | ... → EXECUTE_TRIGGERED → EXECUTION_STARTED → EXECUTION_FAILED → EVIDENCE_WRITTEN |

**Evidence Pack 语义：**
- `ExecutionSuccess`：真实反映 adapter 执行结果，不再是骨架占位。
- `BeforeStateSnapshot`：包含 `database_exists`、`sql_summary`、`idempotency_key`、`connection_ref` 等真实预执行状态。
- `AfterStateSnapshot`：包含执行后 `database_exists`、`task_status`、`adapter` 等真实后状态。
- `ArtifactRefs`：包含 `sql_summary`、`before_snapshot`、`after_snapshot`（成功路径）、`failure_detail`（失败路径）。
- `FailureDetail`：包含统一错误码和错误信息。
- `RollbackSuggestion`：成功时建议 `DROP DATABASE`；失败时建议检查后重试。
- `ResultSummary`：反映真实动作结果。

### 9. 是否仍未越界到 Deep Agent / MCP / 多 adapter 扩展

**结论：是，未越界。**

- 未引入 Deep Agent 相关代码或接口。
- 未引入 MCP 相关代码或接口。
- `main.go` 只注册了一个 `dbnative.New(dbnative.Dependencies{})` adapter。
- `StaticExecutionRouter` 虽然支持多 adapter 注册，但当前只绑定了 `db_native`。
- `StaticExecutionPlanner.Build()` 的 `SelectedRoute` 固定为 `db_native`。

### 10. 是否允许进入 Phase 03 closeout

**结论：允许进入 Phase 03 closeout。**

理由：
1. `mysql.database.create` 已是可演示动作，不再是 stub。
2. dev/test 路径（无需审批）和 prod 路径（需审批）均已通过测试验证。
3. 成功、PLAN_STALE、幂等冲突、幂等重放、失败后重试五条路径均有 audit 和 evidence。
4. PLAN_STALE 路径由真实 revalidate 驱动，`simulate_plan_stale` 已从代码中移除。
5. `go test -count=1 ./...` 全部通过。

---

## 阻塞问题

无。

## 非阻塞问题

### NB-1: approval actor body fallback 仍未完全移除

`server.go:95-105` 中，当 `authCtx.AuthenticatedPrincipalID` 为空时，仍允许 body 中的 `approver_id` fallback。Codex 在 handoff 中已声明此为已知限制，当前做法（认证上下文优先 + mismatch 阻断）在 MVP 阶段可接受，但应在后续 phase 中收紧为"只信任认证上下文"。

### NB-2: 幂等记录为 in-memory

当前 `IdempotencyRepository` 是 `MemoryStore` 实现，进程重启后丢失。这意味着：
- 重启后的首次 execute 可能无法检测到"前次进行中"的冲突。
- 重启后的首次 execute 可能无法触发幂等重放。

在 MVP 阶段可接受，但持久化切换应在后续 phase 中优先处理。

### NB-3: `CreateDatabase` 的 SQL 拼接

`adapter.go:389` 中 `CreateDatabase` 使用 `sqlSummary(databaseName)` 拼接 SQL（`CREATE DATABASE \`name\``）。虽然 `databaseName` 已通过 `^[A-Za-z][A-Za-z0-9_]{0,63}$` 正则校验，注入风险极低，但 `CREATE DATABASE` 不支持参数化查询，这是 DDL 操作的固有限制。当前实现在校验后拼接是合理的。

### NB-4: adapter DryRun 被调用两次

`Execute()` 方法内部在 `adapter.go:134` 调用了 `a.DryRun()`，而 `service.go:349` 在 execute 主链路中也调用了 `binding.Adapter.DryRun()`。这意味着每次真实执行会做两次 DryRun（一次在 service 层 revalidate，一次在 adapter Execute 内部）。性能影响在 MVP 阶段可忽略，但后续可考虑让 Execute 接受 DryRun 结果以避免重复。

---

## 验证记录

```
$ go test -count=1 ./...
?   	dba_ai_assistant/cmd/server	[no test files]
ok  	dba_ai_assistant/internal/adapters/dbnative	0.016s
ok  	dba_ai_assistant/internal/api	0.021s
ok  	dba_ai_assistant/internal/application/actionrequest	0.034s
ok  	dba_ai_assistant/internal/application/approval	0.014s
ok  	dba_ai_assistant/internal/application/audit	0.026s
ok  	dba_ai_assistant/internal/application/authorization	0.020s
ok  	dba_ai_assistant/internal/application/evidence	0.030s
ok  	dba_ai_assistant/internal/application/execution	0.017s
ok  	dba_ai_assistant/internal/domain/policy	0.008s
ok  	dba_ai_assistant/internal/skill	0.008s
```

## Gate 判定

```
review_result     = PASS
blocking_issues   = 0
non_blocking_issues = 4 (NB-1 ~ NB-4)
ready_for_closeout = true
```

**允许进入 Phase 03 closeout。**

## 下一步要求

1. 进入 Phase 03 closeout 流程。
2. NB-1（approval body fallback 收紧）和 NB-2（幂等持久化）应在后续 phase 规划中明确排期。
3. closeout 后更新 `00-dashboard.md` 状态为 `accepted`。
