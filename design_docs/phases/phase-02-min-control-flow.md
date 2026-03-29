# Phase 02 - Minimal Control Flow

## 目标

打通最小可运行控制链路，使系统在不依赖真实 MySQL 写操作的前提下，能够完成 request、authorization、order、approval、execute、plan revalidate、task 启动骨架、audit/evidence 基础落账。

## 范围

1. `POST /api/v1/action-requests` 的真实链路接线。
2. `PrincipalResolver`、`AssetResolver`、`PolicyEngine`、`RiskEngine`、`AuthorizationService` 接入统一 repository。
3. `RiskEngine.Evaluate()` 必须根据 asset 的 environment 与敏感度动态计算风险等级，至少满足：
   - dev/test -> `R1`
   - prod -> `R2`
   - 高敏目标至少触发 require approval
4. `AssistantOrder` 与 `ExecutionPlan` 共享持久化。
5. `ApprovalService` 实现：
   - `WAITING_APPROVAL -> APPROVED`
   - `WAITING_APPROVAL -> REJECTED`
   - `WAITING_APPROVAL -> EXPIRED`
6. `POST /api/v1/orders/{order_id}/execute` 接入：
   - execute auth
   - `ExecutePolicy`
   - `Plan Revalidate`
   - 可选择调用 `Adapter.DryRun()` 作为预检手段之一（本阶段允许 stub）
   - task 创建骨架
7. append-only `AuditEvent` 最小写入。
8. `EvidencePack` 最小结构与查询接口，且 `PLAN_STALE` 路径必须支持 `task_id = null`。

## 禁止事项

1. 不做真实 `CREATE DATABASE`。
2. 不扩散到其他 MySQL 动作。
3. 不接入 MCP / CRD / gRPC / K8s / Shell / VM 多通道。
4. 不把审批 TTL 写成隐式常量。
5. 不把 `PLAN_STALE` 视为普通 `FAILED`。

## 产物

1. 可运行的最小控制链路。
2. request / approval / execute / query API。
3. append-only 审计仓储基础实现。
4. 最小 `AuditLedgerView`。
5. `PLAN_STALE`、`REJECTED`、`EXPIRED` 可见状态。

## 验收标准

1. dev/test 场景可走到 `APPROVED`，并通过显式 execute 进入任务骨架。
2. prod 场景可走到 `WAITING_APPROVAL -> APPROVED -> explicit execute`。
3. `WAITING_APPROVAL` 状态下 execute 返回 `APPROVAL_REQUIRED`。
4. `EXECUTING` 状态重复 execute 不产生第二个任务。
5. `Plan Revalidate` 可以触发 `PLAN_STALE` 且不创建任务。
6. 审批过期扫描最少以 stub 方式可执行。
7. `REQUEST_ACCEPTED` 是第一条审计事件。
8. `PLAN_STALE` 路径生成 `EvidencePack`，至少包含 stale reason，且 `task_id = null`。

## 风险点

1. 如果 repository 形状不稳，Phase 3 接真实执行时仍会大改 application service。
2. 如果 Phase 2 只做“内存假跑通”，团队会误判主链路已经完成。
3. execute auth、approval auth、request auth 混用会直接破坏控制边界。

## 进入下一阶段条件

1. request / approval / execute 三条链路共享同一批 order/plan/task/audit repository。
2. `AuthorizationDecision` 与 execute auth decision 都已成为各自路径的唯一权威结果。
3. `PLAN_STALE`、`EXPIRED`、`REJECTED` 均有真实状态推进与审计记录。
4. 最小控制链路已经可演示。

## 推荐 branch 名

`phase/02-minimal-control-chain`

## 推荐 commit message 模式

1. `feat(control): wire request authorization order plan flow`
2. `feat(approval): add approval transitions and expiry scan`
3. `feat(execute): add execute authorization and revalidate gate`
4. `feat(audit): add append-only audit event repository`
