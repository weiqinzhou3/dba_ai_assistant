# Agent Collaboration Protocol v0.1

## 0. 文档定位

本文档定义 Codex 与 Claude Code 在本项目中的本地协作协议。它不引入新的业务语义，只规定谁负责什么、通过哪些文件交接、何时允许进入下一阶段。

## 1. 协作原则

1. 正式语义以 `design_docs/03` 到 `design_docs/10` 为准。
2. 所有 phase 都必须围绕 Control Layer / Action / Principal / Asset / Authorization / Approval / Audit / Evidence / Adapter 主链路展开。
3. 任何一方都不得绕过 `AuthorizationService`、审批边界、显式 execute、append-only audit 和对称 evidence。
4. 若发现正式文档冲突，先写入 review 或 decisions log，再决定是否继续，不得自行改写语义。

## 2. Codex 负责什么

Codex 负责：

1. 进入某个 phase 前写本 phase 的 `codex-plan.md`。
2. 按当前 phase 的范围实施代码或文档变更。
3. 在工作中持续更新 `codex-status.md`。
4. 完成一个可评审轮次后写 `codex-handoff.md`。
5. 收到 Claude review 后，在 `codex-fix-response.md` 中逐条回应并说明修复证据。
6. review 通过并准备收口时，写 `merge-summary.md`。

Codex 不负责：

1. 重新定义正式领域语义。
2. 跳过 phase 边界擅自扩 scope。
3. 自己给自己做最终 gate 审批。

## 3. Claude Code 负责什么

Claude Code 负责：

1. 依据正式文档对当前 phase 产物做 review。
2. 在 `claude-review.md` 中写明阻塞问题、非阻塞问题、风险和 gate 结论。
3. 检查本轮变更是否越过当前 phase 范围。
4. 检查主链路是否被破坏，尤其是：
   - approval 是否被混入 execute
   - execute 是否仍由独立边界触发
   - Asset Resolver 是否保持 exact match
   - 审计与证据是否保留对称性

Claude Code 不负责：

1. 直接改写正式 spec。
2. 在没有 review 记录的情况下口头放行下一阶段。

## 4. 本地文件交接机制

所有交接通过 `design_docs/coordination/` 目录完成。

### 4.1 全局文件

- `00-dashboard.md`
  - 项目总览页，显示每个 phase 当前状态、分支和交接文件入口。
- `decisions-log.md`
  - 记录跨 phase 的追加型关键决策。

### 4.2 每个 phase 的固定文件

- `codex-plan.md`
  - 本 phase 的实施计划、边界、验证计划。
- `codex-status.md`
  - 过程状态、已完成项、未完成项、当前阻塞。
- `codex-handoff.md`
  - 一轮实现后的交接说明、变更清单、验证结果、请求 review 的重点。
- `claude-review.md`
  - Claude 的评审结论、阻塞项、非阻塞项、gate 决定。
- `codex-fix-response.md`
  - Codex 对 review 的逐条回应和修复证据。
- `merge-summary.md`
  - 本 phase 合入前的收口总结、残余风险、下一阶段建议。

## 5. 项目 dashboard 如何更新

1. Codex 开始某 phase 时，把 dashboard 对应行状态更新为 `in_progress`，并写入当前分支。
2. Codex 提交 handoff 后，把状态更新为 `awaiting_review`。
3. Claude review 若发现阻塞问题，把状态更新为 `changes_required`。
4. Codex 完成修复并更新 fix response 后，再次把状态更新为 `awaiting_review`。
5. Claude review 通过后，把状态更新为 `accepted`。
6. merge 完成后，把状态更新为 `merged`。
7. Codex 提交 handoff 时，同时填写 dashboard 中的 `Handoff At` 时间戳。

## 5.1 Review SLA

默认 review SLA：

1. Codex 把状态更新为 `awaiting_review` 后，Claude Code 应在 24 小时内给出首次 review 结果。
2. 若 24 小时内未完成 review，Codex 应在 `codex-status.md` 中记录 pending review 风险。
3. SLA 超时不构成自动放行；没有 `claude-review.md` 的正式结论，仍不得进入下一阶段。

## 6. 每轮完成后谁写什么

1. Codex 写：
   - `codex-status.md`
   - `codex-handoff.md`
   - 如有修复，则写 `codex-fix-response.md`
2. Claude Code 写：
   - `claude-review.md`
3. 收口阶段由 Codex 写：
   - `merge-summary.md`
4. 任何跨 phase 决策，由提出者追加到 `decisions-log.md`，并由另一方确认。

## 6.1 分歧仲裁机制

1. 若 Claude Code 与 Codex 对某个设计点存在分歧，先把分歧事实写入对应 phase 的 review / fix-response 文件。
2. 若该分歧影响后续 phase 或正式边界，必须追加写入 `decisions-log.md`。
3. 最终仲裁人是项目 owner，也就是当前用户；在 owner 明确裁定前，不得把争议点当作已关闭结论。

## 7. 阶段推进规则

只有同时满足以下条件，才允许进入下一阶段：

1. 当前 phase 的 `merge-summary.md` 已写完。
2. `claude-review.md` 的结论为通过，不存在未关闭阻塞项。
3. `00-dashboard.md` 中当前 phase 状态为 `accepted` 或 `merged`。
4. 当前 phase 的验收标准已被验证，并在交接文件中留下证据。
5. 当前 phase 的禁止事项没有被违反；若违反，必须先记录在 decisions log 并重新做 gate 判断。

## 8. 何时写 decisions log

出现以下情况时必须写入 `decisions-log.md`：

1. 正式文档存在冲突，需要明确以哪条为准。
2. 当前 phase 必须临时扩 scope，且会影响后续 phase。
3. 接口、状态机、权限模型、审计模型发生跨 phase 影响的变更。
4. 协作协议本身需要调整。

## 8.1 快速修复路径

当同时满足以下条件时，可以走快速修复路径：

1. 修改范围不超过 3 个文件。
2. 不涉及接口签名、状态机、权限模型、审计模型或 phase 边界变更。
3. 目标是修复局部错误、错字、链接、模板疏漏或明显的低风险文档问题。

快速修复规则：

1. Codex 在对应 `codex-status.md` 中标记 `hotfix: true` 并说明原因。
2. Codex 可以直接提交该小修复，但必须在下一次常规 handoff 中补充说明。
3. Claude Code 在下一次常规 review 中追溯确认该 hotfix 是否合理。
4. 若修复实际超出上述边界，Claude Code 可以在 review 中要求回退到常规流程。

## 9. 何时禁止进入下一阶段

出现以下任一情况，禁止进入下一阶段：

1. 当前 phase 存在未关闭阻塞 review。
2. 当前 phase 尚未形成 `merge-summary.md`。
3. 只完成成功路径，失败路径或 `PLAN_STALE` 路径仍空缺。
4. 发现审批与 execute 被混写。
5. 发现 Agent 直接获得 Adapter 或真实凭证访问。
6. 发现 Asset Resolver 出现 fuzzy / guess / auto-pick 行为。

## 10. 结论

本协议的核心目标只有一个：

> 让 Codex 负责产出与修复，让 Claude Code 负责门禁与评审，双方通过本地文件完成可追踪、可复盘、可继续推进的 phase 协作。
