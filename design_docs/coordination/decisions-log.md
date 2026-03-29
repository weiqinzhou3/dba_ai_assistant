# Decisions Log

## 作用

这是跨 phase 的追加型决策日志。只记录会影响后续阶段边界、接口、状态机、权限语义、审计语义或协作流程的决定。

## 记录规则

1. 采用 append-only 方式追加，不回写旧结论。
2. 只有会影响后续协作和实现方向的决定才记入本文件。
3. 若只是某一轮临时实现细节，不进入本日志，写入对应 phase 的状态或 handoff 文件即可。

## 当前状态

当前无全局决策记录。首条记录请直接复制以下模板并追加到文末。

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

